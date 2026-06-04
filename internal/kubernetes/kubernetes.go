package kubernetes

import (
	"fmt"
	"strings"

	"duck/internal/cli"
	"duck/internal/config"
	"duck/internal/prompt"
	"duck/internal/runner"
)

type service struct {
	bin    string
	runner runner.Runner
}

func Command(cfg config.Config, run runner.Runner) cli.Command {
	svc := service{bin: cfg.KubectlBin, runner: run}
	return cli.Command{
		Name:        "kube",
		Aliases:     []string{"k"},
		Description: "Executa tarefas Kubernetes com kubectl",
		Usage:       "kube <comando> [argumentos]",
		Children: []cli.Command{
			{Name: "status", Description: "Mostra contexto e recursos basicos", Usage: "kube status", Run: svc.status},
			{Name: "contexts", Description: "Lista contextos", Usage: "kube contexts", Run: svc.noExtra("config", "get-contexts")},
			{Name: "use", Description: "Troca o contexto atual", Usage: "kube use <context>", Run: svc.useContext},
			{Name: "ns", Aliases: []string{"namespaces"}, Description: "Lista namespaces", Usage: "kube ns", Run: svc.noExtra("get", "namespaces")},
			{Name: "pods", Aliases: []string{"po"}, Description: "Lista pods", Usage: "kube pods [-n namespace]", Run: svc.getResource("pods")},
			{Name: "svc", Aliases: []string{"services"}, Description: "Lista services", Usage: "kube svc [-n namespace]", Run: svc.getResource("services")},
			{Name: "deploy", Aliases: []string{"deployments"}, Description: "Lista deployments", Usage: "kube deploy [-n namespace]", Run: svc.getResource("deployments")},
			{Name: "logs", Description: "Exibe logs de um pod", Usage: "kube logs <pod> [-n namespace] [-f] [--tail N]", Run: svc.logs},
			{Name: "exec", Description: "Executa comando em um pod", Usage: "kube exec <pod> [-n namespace] -- <cmd>", Run: svc.exec},
			{Name: "describe", Description: "Descreve recurso", Usage: "kube describe <tipo> <nome> [-n namespace]", Run: svc.describe},
			{Name: "apply", Description: "Aplica manifestos", Usage: "kube apply -f <arquivo>", Run: svc.withArgs("apply")},
			{Name: "delete", Description: "Remove recursos ou manifestos", Usage: "kube delete [-f arquivo|tipo nome] [-n namespace] [-f|--force]", Run: svc.delete},
			{Name: "raw", Description: "Envia argumentos diretamente para kubectl", Usage: "kube raw <kubectl args...>", Run: svc.raw},
		},
		Examples: []string{
			"kube status",
			"k pods -n default",
			"k logs api-123 -n apps --tail 100",
			"k raw get nodes",
		},
	}
}

func (s service) status(_ cli.Context, args []string) error {
	if len(args) > 0 {
		return cli.UsageError("status nao recebe argumentos")
	}

	fmt.Println("Contexto atual:")
	if err := s.run([]string{"config", "current-context"}, runner.DefaultOptions()); err != nil {
		return err
	}

	fmt.Println()
	fmt.Println("Namespaces:")
	return s.run([]string{"get", "namespaces"}, runner.DefaultOptions())
}

func (s service) useContext(_ cli.Context, args []string) error {
	if len(args) != 1 {
		return cli.UsageError("informe exatamente um contexto")
	}
	return s.run([]string{"config", "use-context", args[0]}, runner.DefaultOptions())
}

func (s service) getResource(resource string) func(cli.Context, []string) error {
	return func(_ cli.Context, args []string) error {
		kubeArgs := []string{"get", resource}
		namespace, rest, err := namespaceArgs(args)
		if err != nil {
			return err
		}
		if len(rest) > 0 {
			return cli.UsageError("argumentos invalidos: " + strings.Join(rest, " "))
		}
		kubeArgs = appendNamespace(kubeArgs, namespace)
		return s.run(kubeArgs, runner.DefaultOptions())
	}
}

func (s service) logs(_ cli.Context, args []string) error {
	if len(args) == 0 {
		return cli.UsageError("informe o pod")
	}

	kubeArgs := []string{"logs", args[0]}
	namespace := ""
	for i := 1; i < len(args); i++ {
		switch args[i] {
		case "-n", "--namespace":
			if i+1 >= len(args) {
				return cli.UsageError(args[i] + " precisa de um valor")
			}
			namespace = args[i+1]
			i++
		case "-f", "--follow":
			kubeArgs = append(kubeArgs, "--follow")
		case "--tail":
			if i+1 >= len(args) {
				return cli.UsageError("--tail precisa de um valor")
			}
			kubeArgs = append(kubeArgs, "--tail", args[i+1])
			i++
		default:
			return cli.UsageError("opcao invalida para logs: " + args[i])
		}
	}

	kubeArgs = appendNamespace(kubeArgs, namespace)
	return s.run(kubeArgs, runner.DefaultOptions())
}

func (s service) exec(_ cli.Context, args []string) error {
	if len(args) < 2 {
		return cli.UsageError("use: exec <pod> [-n namespace] -- <cmd>")
	}

	pod := args[0]
	namespace := ""
	cmdStart := -1
	for i := 1; i < len(args); i++ {
		switch args[i] {
		case "-n", "--namespace":
			if i+1 >= len(args) {
				return cli.UsageError(args[i] + " precisa de um valor")
			}
			namespace = args[i+1]
			i++
		case "--":
			cmdStart = i + 1
			i = len(args)
		default:
			if cmdStart == -1 {
				cmdStart = i
				i = len(args)
			}
		}
	}
	if cmdStart < 0 || cmdStart >= len(args) {
		return cli.UsageError("informe o comando para executar")
	}

	kubeArgs := []string{"exec", "-it", pod}
	kubeArgs = appendNamespace(kubeArgs, namespace)
	kubeArgs = append(kubeArgs, "--")
	kubeArgs = append(kubeArgs, args[cmdStart:]...)
	return s.run(kubeArgs, runner.InteractiveOptions())
}

func (s service) describe(_ cli.Context, args []string) error {
	if len(args) < 2 {
		return cli.UsageError("informe tipo e nome do recurso")
	}

	namespace, rest, err := namespaceArgs(args[2:])
	if err != nil {
		return err
	}
	if len(rest) > 0 {
		return cli.UsageError("argumentos invalidos: " + strings.Join(rest, " "))
	}

	kubeArgs := []string{"describe", args[0], args[1]}
	kubeArgs = appendNamespace(kubeArgs, namespace)
	return s.run(kubeArgs, runner.DefaultOptions())
}

func (s service) delete(_ cli.Context, args []string) error {
	force, kubeArgs := stripForce(args)
	if len(kubeArgs) == 0 {
		return cli.UsageError("informe o recurso ou manifesto para remover")
	}

	if !force {
		ok, err := prompt.Confirm("Remover recursos Kubernetes informados? [s/N] ")
		if err != nil {
			return err
		}
		if !ok {
			fmt.Println("Cancelado.")
			return nil
		}
	}

	return s.run(append([]string{"delete"}, kubeArgs...), runner.DefaultOptions())
}

func (s service) raw(_ cli.Context, args []string) error {
	if len(args) == 0 {
		return cli.UsageError("informe argumentos para kubectl")
	}
	return s.run(args, runner.InteractiveOptions())
}

func (s service) noExtra(args ...string) func(cli.Context, []string) error {
	return func(_ cli.Context, extra []string) error {
		if len(extra) > 0 {
			return cli.UsageError("este comando nao recebe argumentos extras")
		}
		return s.run(args, runner.DefaultOptions())
	}
}

func (s service) withArgs(prefix ...string) func(cli.Context, []string) error {
	return func(_ cli.Context, extra []string) error {
		if len(extra) == 0 {
			return cli.UsageError("argumentos obrigatorios ausentes")
		}
		kubeArgs := append([]string{}, prefix...)
		kubeArgs = append(kubeArgs, extra...)
		return s.run(kubeArgs, runner.DefaultOptions())
	}
}

func (s service) run(args []string, opts runner.Options) error {
	return s.runner.Run(s.bin, args, opts)
}

func namespaceArgs(args []string) (string, []string, error) {
	namespace := ""
	rest := make([]string, 0, len(args))

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "-n", "--namespace":
			if i+1 >= len(args) {
				return "", nil, cli.UsageError(args[i] + " precisa de um valor")
			}
			namespace = args[i+1]
			i++
		default:
			rest = append(rest, args[i])
		}
	}

	return namespace, rest, nil
}

func appendNamespace(args []string, namespace string) []string {
	if namespace == "" {
		return args
	}
	return append(args, "-n", namespace)
}

func stripForce(args []string) (bool, []string) {
	force := false
	rest := make([]string, 0, len(args))

	for _, arg := range args {
		switch arg {
		case "--force", "--yes", "-y":
			force = true
		default:
			rest = append(rest, arg)
		}
	}

	return force, rest
}
