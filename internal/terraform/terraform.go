package terraform

import (
	"fmt"
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
	svc := service{bin: cfg.TerraformBin, runner: run}
	return cli.Command{
		Name:        "terraform",
		Aliases:     []string{"tf"},
		Description: "Executa tarefas comuns de Terraform",
		Usage:       "terraform <comando> [argumentos]",
		Children: []cli.Command{
			{Name: "version", Description: "Mostra a versao do Terraform", Usage: "terraform version", Run: svc.noExtra("version")},
			{Name: "init", Description: "Inicializa o diretorio de trabalho", Usage: "terraform init [args...]", Run: svc.withArgs("init")},
			{Name: "plan", Description: "Gera plano de execucao", Usage: "terraform plan [args...]", Run: svc.withArgs("plan")},
			{Name: "apply", Description: "Aplica alteracoes de infraestrutura", Usage: "terraform apply [args...]", Run: svc.apply},
			{Name: "destroy", Description: "Destroi recursos gerenciados", Usage: "terraform destroy [args...]", Run: svc.destroy},
			{Name: "validate", Description: "Valida arquivos Terraform", Usage: "terraform validate [args...]", Run: svc.withArgs("validate")},
			{Name: "fmt", Description: "Formata arquivos Terraform", Usage: "terraform fmt [args...]", Run: svc.withArgs("fmt")},
			{Name: "output", Description: "Mostra outputs do state", Usage: "terraform output [args...]", Run: svc.withArgs("output")},
			{Name: "show", Description: "Mostra state ou plano", Usage: "terraform show [args...]", Run: svc.withArgs("show")},
			{Name: "state", Description: "Gerencia state remoto/local", Usage: "terraform state <args...>", Run: svc.withArgs("state")},
			{Name: "workspace", Description: "Gerencia workspaces", Usage: "terraform workspace <args...>", Run: svc.withArgs("workspace")},
			{Name: "import", Description: "Importa recurso existente", Usage: "terraform import <args...>", Run: svc.withArgs("import")},
			{Name: "providers", Description: "Mostra providers configurados", Usage: "terraform providers [args...]", Run: svc.withArgs("providers")},
			{Name: "check", Description: "Executa fmt -check e validate", Usage: "terraform check", Run: svc.check},
			{Name: "raw", Description: "Envia argumentos diretamente para terraform", Usage: "terraform raw <args...>", Run: svc.raw},
		},
		Examples: []string{
			"terraform init",
			"terraform plan",
			"terraform apply -auto-approve",
			"tf check",
		},
	}
}

func (s service) apply(ctx cli.Context, args []string) error {
	if !hasAutoApprove(args) && !ctx.Options.Force {
		ok, err := prompt.Confirm("Executar terraform apply? [s/N] ")
		if err != nil {
			return err
		}
		if !ok {
			fmt.Println("Cancelado.")
			return nil
		}
	}
	return s.run(append([]string{"apply"}, args...), runner.InteractiveOptions())
}

func (s service) destroy(ctx cli.Context, args []string) error {
	if !hasAutoApprove(args) && !ctx.Options.Force {
		ok, err := prompt.Confirm("Executar terraform destroy? Esta acao e destrutiva. [s/N] ")
		if err != nil {
			return err
		}
		if !ok {
			fmt.Println("Cancelado.")
			return nil
		}
	}
	return s.run(append([]string{"destroy"}, args...), runner.InteractiveOptions())
}

func (s service) check(ctx cli.Context, args []string) error {
	if len(args) > 0 {
		return cli.UsageError("check nao recebe argumentos")
	}
	steps := [][]string{
		{"fmt", "-check", "-recursive"},
		{"validate"},
	}
	for _, step := range steps {
		fmt.Println("==>", "terraform", strings.Join(step, " "))
		if err := s.run(step, runner.DefaultOptions()); err != nil {
			return err
		}
	}
	return nil
}

func (s service) raw(_ cli.Context, args []string) error {
	if len(args) == 0 {
		return cli.UsageError("informe argumentos para terraform")
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
		tfArgs := append([]string{}, prefix...)
		tfArgs = append(tfArgs, extra...)
		return s.run(tfArgs, runner.InteractiveOptions())
	}
}

func (s service) run(args []string, opts runner.Options) error {
	return s.runner.Run(s.bin, args, opts)
}

func hasAutoApprove(args []string) bool {
	for _, arg := range args {
		if arg == "-auto-approve" {
			return true
		}
	}
	return false
}
