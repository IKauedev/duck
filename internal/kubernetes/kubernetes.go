package kubernetes

import (
	"bufio"
	"fmt"
	"net"
	"net/url"
	"os"
	"strings"

	"github.com/IKauedev/duck/internal/cli"
	"github.com/IKauedev/duck/internal/config"
	"github.com/IKauedev/duck/internal/prompt"
	"github.com/IKauedev/duck/internal/runner"
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
			{Name: "ctx", Description: "Lista contextos", Usage: "kube ctx", Run: svc.noExtra("config", "get-contexts")},
			{Name: "use", Description: "Troca o contexto atual", Usage: "kube use <context>", Run: svc.useContext},
			{Name: "ns", Aliases: []string{"namespaces"}, Description: "Lista namespaces", Usage: "kube ns", Run: svc.noExtra("get", "namespaces")},
			{Name: "pods", Aliases: []string{"po"}, Description: "Lista pods", Usage: "kube pods [-n namespace]", Run: svc.getResource("pods")},
			{Name: "svc", Aliases: []string{"services"}, Description: "Lista services", Usage: "kube svc [-n namespace]", Run: svc.getResource("services")},
			{Name: "deploy", Aliases: []string{"deployments"}, Description: "Lista deployments", Usage: "kube deploy [-n namespace]", Run: svc.getResource("deployments")},
			{Name: "events", Description: "Lista eventos ordenados por criacao", Usage: "kube events [-n namespace]", Run: svc.events},
			{Name: "logs", Description: "Exibe logs de um pod", Usage: "kube logs <pod> [-n namespace] [-f] [--tail N]", Run: svc.logs},
			{Name: "exec", Description: "Executa comando em um pod", Usage: "kube exec <pod> [-n namespace] -- <cmd>", Run: svc.exec},
			{Name: "shell", Description: "Abre shell em um pod", Usage: "kube shell <pod> [-n namespace] [shell]", Run: svc.shell},
			{Name: "debug", Description: "Mostra describe, eventos e logs recentes de um pod", Usage: "kube debug <pod> [-n namespace]", Run: svc.debug},
			{Name: "restart", Description: "Reinicia um deployment", Usage: "kube restart <deployment> [-n namespace]", Run: svc.restart},
			{Name: "scale", Description: "Altera replicas de um deployment", Usage: "kube scale <deployment> <replicas> [-n namespace]", Run: svc.scale},
			{Name: "image", Description: "Atualiza imagem de um deployment", Usage: "kube image <deployment> <container=image> [-n namespace]", Run: svc.image},
			{Name: "wait", Description: "Aguarda rollout de um deployment", Usage: "kube wait <deployment> [-n namespace]", Run: svc.waitDeployment},
			{Name: "port-forward", Description: "Abre port-forward para recurso", Usage: "kube port-forward <recurso> <porta> [-n namespace]", Run: svc.portForward},
			{Name: "curl", Description: "Testa URL/porta a partir de um pod temporario no cluster", Usage: "kube curl <url> [--port porta] [-n namespace] [--timeout segundos] [--insecure]", Run: svc.curl},
			{Name: "curl-many", Description: "Testa varias URLs a partir do cluster", Usage: "kube curl-many <arquivo> [-n namespace]", Run: svc.curlMany},
			{Name: "dns", Description: "Testa DNS a partir do cluster", Usage: "kube dns <host> [-n namespace]", Run: svc.dns},
			{Name: "tcp", Description: "Testa TCP a partir do cluster", Usage: "kube tcp <host> <port> [-n namespace]", Run: svc.tcp},
			{Name: "ingress", Description: "Lista ingresses", Usage: "kube ingress [-n namespace]", Run: svc.getResource("ingress")},
			{Name: "resources", Description: "Mostra requests e limits de pods", Usage: "kube resources [-n namespace]", Run: svc.resources},
			{Name: "failed", Description: "Lista pods com erro", Usage: "kube failed [-n namespace]", Run: svc.failed},
			{Name: "clean-failed", Description: "Remove pods falhos com confirmacao", Usage: "kube clean-failed [-n namespace] [-f|--force]", Run: svc.cleanFailed},
			{Name: "top-pods", Description: "Mostra uso de recursos dos pods", Usage: "kube top-pods [-n namespace]", Run: svc.topResource("pods")},
			{Name: "top-nodes", Description: "Mostra uso de recursos dos nodes", Usage: "kube top-nodes", Run: svc.noExtra("top", "nodes")},
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

func (s service) events(_ cli.Context, args []string) error {
	namespace, rest, err := namespaceArgs(args)
	if err != nil {
		return err
	}
	if len(rest) > 0 {
		return cli.UsageError("argumentos invalidos: " + strings.Join(rest, " "))
	}

	kubeArgs := []string{"get", "events", "--sort-by=.metadata.creationTimestamp"}
	kubeArgs = appendNamespace(kubeArgs, namespace)
	return s.run(kubeArgs, runner.DefaultOptions())
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

func (s service) shell(_ cli.Context, args []string) error {
	if len(args) == 0 {
		return cli.UsageError("informe o pod")
	}
	pod := args[0]
	namespace, rest, err := namespaceArgs(args[1:])
	if err != nil {
		return err
	}
	shell := "sh"
	if len(rest) > 1 {
		return cli.UsageError("shell aceita no maximo um shell opcional")
	}
	if len(rest) == 1 {
		shell = rest[0]
	}
	kubeArgs := []string{"exec", "-it", pod}
	kubeArgs = appendNamespace(kubeArgs, namespace)
	kubeArgs = append(kubeArgs, "--", shell)
	return s.run(kubeArgs, runner.InteractiveOptions())
}

func (s service) debug(_ cli.Context, args []string) error {
	if len(args) == 0 {
		return cli.UsageError("informe o pod")
	}
	namespace, rest, err := namespaceArgs(args[1:])
	if err != nil {
		return err
	}
	if len(rest) > 0 {
		return cli.UsageError("argumentos invalidos: " + strings.Join(rest, " "))
	}

	fmt.Println("==> describe")
	describeArgs := appendNamespace([]string{"describe", "pod", args[0]}, namespace)
	if err := s.run(describeArgs, runner.DefaultOptions()); err != nil {
		return err
	}
	fmt.Println()
	fmt.Println("==> logs recentes")
	logArgs := appendNamespace([]string{"logs", args[0], "--tail", "100"}, namespace)
	return s.run(logArgs, runner.DefaultOptions())
}

func (s service) restart(_ cli.Context, args []string) error {
	if len(args) == 0 {
		return cli.UsageError("informe o deployment")
	}

	namespace, rest, err := namespaceArgs(args[1:])
	if err != nil {
		return err
	}
	if len(rest) > 0 {
		return cli.UsageError("argumentos invalidos: " + strings.Join(rest, " "))
	}

	kubeArgs := []string{"rollout", "restart", "deployment/" + args[0]}
	kubeArgs = appendNamespace(kubeArgs, namespace)
	return s.run(kubeArgs, runner.DefaultOptions())
}

func (s service) scale(_ cli.Context, args []string) error {
	if len(args) < 2 {
		return cli.UsageError("informe deployment e replicas")
	}
	namespace, rest, err := namespaceArgs(args[2:])
	if err != nil {
		return err
	}
	if len(rest) > 0 {
		return cli.UsageError("argumentos invalidos: " + strings.Join(rest, " "))
	}
	kubeArgs := []string{"scale", "deployment/" + args[0], "--replicas", args[1]}
	kubeArgs = appendNamespace(kubeArgs, namespace)
	return s.run(kubeArgs, runner.DefaultOptions())
}

func (s service) image(_ cli.Context, args []string) error {
	if len(args) < 2 {
		return cli.UsageError("informe deployment e container=image")
	}
	namespace, rest, err := namespaceArgs(args[2:])
	if err != nil {
		return err
	}
	if len(rest) > 0 {
		return cli.UsageError("argumentos invalidos: " + strings.Join(rest, " "))
	}
	kubeArgs := []string{"set", "image", "deployment/" + args[0], args[1]}
	kubeArgs = appendNamespace(kubeArgs, namespace)
	return s.run(kubeArgs, runner.DefaultOptions())
}

func (s service) waitDeployment(_ cli.Context, args []string) error {
	if len(args) == 0 {
		return cli.UsageError("informe o deployment")
	}
	namespace, rest, err := namespaceArgs(args[1:])
	if err != nil {
		return err
	}
	if len(rest) > 0 {
		return cli.UsageError("argumentos invalidos: " + strings.Join(rest, " "))
	}
	kubeArgs := []string{"rollout", "status", "deployment/" + args[0]}
	kubeArgs = appendNamespace(kubeArgs, namespace)
	return s.run(kubeArgs, runner.InteractiveOptions())
}

func (s service) portForward(_ cli.Context, args []string) error {
	if len(args) < 2 {
		return cli.UsageError("informe recurso e porta")
	}

	namespace, rest, err := namespaceArgs(args[2:])
	if err != nil {
		return err
	}
	if len(rest) > 0 {
		return cli.UsageError("argumentos invalidos: " + strings.Join(rest, " "))
	}

	kubeArgs := []string{"port-forward", args[0], args[1]}
	kubeArgs = appendNamespace(kubeArgs, namespace)
	return s.run(kubeArgs, runner.InteractiveOptions())
}

func (s service) curl(_ cli.Context, args []string) error {
	opts, err := parseCurlArgs(args)
	if err != nil {
		return err
	}

	target, err := normalizeCurlURL(opts.url, opts.port)
	if err != nil {
		return err
	}

	kubeArgs := []string{
		"run",
		"duck-curl",
		"--rm",
		"-i",
		"--restart=Never",
		"--image=" + opts.image,
	}
	kubeArgs = appendNamespace(kubeArgs, opts.namespace)
	kubeArgs = append(kubeArgs, "--", "curl", "-v", "--connect-timeout", opts.timeout)
	if opts.insecure {
		kubeArgs = append(kubeArgs, "-k")
	}
	kubeArgs = append(kubeArgs, target)
	return s.run(kubeArgs, runner.InteractiveOptions())
}

func (s service) curlMany(_ cli.Context, args []string) error {
	namespace, rest, err := namespaceArgs(args)
	if err != nil {
		return err
	}
	if len(rest) != 1 {
		return cli.UsageError("use: curl-many <arquivo> [-n namespace]")
	}
	file, err := os.Open(rest[0])
	if err != nil {
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		target := strings.TrimSpace(scanner.Text())
		if target == "" || strings.HasPrefix(target, "#") {
			continue
		}
		fmt.Println("==>", target)
		if err := s.curl(cli.Context{}, appendNamespace([]string{target}, namespace)); err != nil {
			return err
		}
	}
	return scanner.Err()
}

func (s service) dns(_ cli.Context, args []string) error {
	namespace, rest, err := namespaceArgs(args)
	if err != nil {
		return err
	}
	if len(rest) != 1 {
		return cli.UsageError("use: dns <host> [-n namespace]")
	}
	return s.runDebugPod(namespace, []string{"nslookup", rest[0]})
}

func (s service) tcp(_ cli.Context, args []string) error {
	namespace, rest, err := namespaceArgs(args)
	if err != nil {
		return err
	}
	if len(rest) != 2 {
		return cli.UsageError("use: tcp <host> <port> [-n namespace]")
	}
	return s.runDebugPod(namespace, []string{"sh", "-c", "nc -vz " + rest[0] + " " + rest[1]})
}

func (s service) resources(_ cli.Context, args []string) error {
	namespace, rest, err := namespaceArgs(args)
	if err != nil {
		return err
	}
	if len(rest) > 0 {
		return cli.UsageError("argumentos invalidos: " + strings.Join(rest, " "))
	}
	kubeArgs := []string{"get", "pods", "-o", "custom-columns=NAME:.metadata.name,CPU_REQ:.spec.containers[*].resources.requests.cpu,MEM_REQ:.spec.containers[*].resources.requests.memory,CPU_LIM:.spec.containers[*].resources.limits.cpu,MEM_LIM:.spec.containers[*].resources.limits.memory"}
	kubeArgs = appendNamespace(kubeArgs, namespace)
	return s.run(kubeArgs, runner.DefaultOptions())
}

func (s service) failed(_ cli.Context, args []string) error {
	namespace, rest, err := namespaceArgs(args)
	if err != nil {
		return err
	}
	if len(rest) > 0 {
		return cli.UsageError("argumentos invalidos: " + strings.Join(rest, " "))
	}
	kubeArgs := []string{"get", "pods", "--field-selector=status.phase!=Running,status.phase!=Succeeded"}
	kubeArgs = appendNamespace(kubeArgs, namespace)
	return s.run(kubeArgs, runner.DefaultOptions())
}

func (s service) cleanFailed(_ cli.Context, args []string) error {
	force, rest := stripForce(args)
	namespace, rest, err := namespaceArgs(rest)
	if err != nil {
		return err
	}
	if len(rest) > 0 {
		return cli.UsageError("argumentos invalidos: " + strings.Join(rest, " "))
	}
	if !force {
		ok, err := prompt.Confirm("Remover pods falhos? [s/N] ")
		if err != nil {
			return err
		}
		if !ok {
			fmt.Println("Cancelado.")
			return nil
		}
	}
	kubeArgs := []string{"delete", "pods", "--field-selector=status.phase!=Running,status.phase!=Succeeded"}
	kubeArgs = appendNamespace(kubeArgs, namespace)
	return s.run(kubeArgs, runner.DefaultOptions())
}

func (s service) runDebugPod(namespace string, command []string) error {
	kubeArgs := []string{"run", "duck-netcheck", "--rm", "-i", "--restart=Never", "--image=busybox:1.36"}
	kubeArgs = appendNamespace(kubeArgs, namespace)
	kubeArgs = append(kubeArgs, "--")
	kubeArgs = append(kubeArgs, command...)
	return s.run(kubeArgs, runner.InteractiveOptions())
}

func (s service) topResource(resource string) func(cli.Context, []string) error {
	return func(_ cli.Context, args []string) error {
		namespace, rest, err := namespaceArgs(args)
		if err != nil {
			return err
		}
		if len(rest) > 0 {
			return cli.UsageError("argumentos invalidos: " + strings.Join(rest, " "))
		}

		kubeArgs := []string{"top", resource}
		kubeArgs = appendNamespace(kubeArgs, namespace)
		return s.run(kubeArgs, runner.DefaultOptions())
	}
}

type curlOptions struct {
	url       string
	port      string
	namespace string
	timeout   string
	image     string
	insecure  bool
}

func parseCurlArgs(args []string) (curlOptions, error) {
	opts := curlOptions{timeout: "10", image: "curlimages/curl:8.10.1"}
	rest := make([]string, 0, len(args))

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "-n", "--namespace":
			if i+1 >= len(args) {
				return opts, cli.UsageError(args[i] + " precisa de um valor")
			}
			opts.namespace = args[i+1]
			i++
		case "--port":
			if i+1 >= len(args) {
				return opts, cli.UsageError("--port precisa de um valor")
			}
			opts.port = args[i+1]
			i++
		case "--timeout":
			if i+1 >= len(args) {
				return opts, cli.UsageError("--timeout precisa de um valor")
			}
			opts.timeout = args[i+1]
			i++
		case "--image":
			if i+1 >= len(args) {
				return opts, cli.UsageError("--image precisa de um valor")
			}
			opts.image = args[i+1]
			i++
		case "-k", "--insecure":
			opts.insecure = true
		default:
			rest = append(rest, args[i])
		}
	}
	if len(rest) != 1 {
		return opts, cli.UsageError("use: kube curl <url> [--port porta] [-n namespace] [--timeout segundos] [--insecure]")
	}
	opts.url = rest[0]
	if opts.namespace == "" {
		opts.namespace = defaultNamespace()
	}
	return opts, nil
}

func normalizeCurlURL(rawURL string, port string) (string, error) {
	if !strings.Contains(rawURL, "://") {
		rawURL = "http://" + rawURL
	}
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return "", err
	}
	if parsed.Host == "" {
		return "", fmt.Errorf("url invalida: %s", rawURL)
	}
	if port != "" && parsed.Port() == "" {
		parsed.Host = net.JoinHostPort(parsed.Hostname(), port)
	}
	return parsed.String(), nil
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
	namespace := defaultNamespace()
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

func defaultNamespace() string {
	settings, err := config.LoadSettings()
	if err != nil {
		return ""
	}
	return settings["kube.namespace"]
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
