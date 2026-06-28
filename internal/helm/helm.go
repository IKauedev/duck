package helm

import (
	"fmt"

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
	svc := service{bin: cfg.HelmBin, runner: run}
	return cli.Command{
		Name:        "helm",
		Aliases:     []string{"h"},
		Description: "Executa tarefas comuns de Helm",
		Usage:       "helm <comando> [argumentos]",
		Children: []cli.Command{
			{Name: "version", Description: "Mostra a versao do Helm", Usage: "helm version", Run: svc.noExtra("version", "--short")},
			{Name: "list", Aliases: []string{"ls"}, Description: "Lista releases instaladas", Usage: "helm list [args...]", Run: svc.withArgs("list")},
			{Name: "status", Description: "Mostra status de uma release", Usage: "helm status <release> [args...]", Run: svc.withArgs("status")},
			{Name: "install", Description: "Instala um chart", Usage: "helm install <release> <chart> [args...]", Run: svc.withArgs("install")},
			{Name: "upgrade", Description: "Atualiza ou instala um chart", Usage: "helm upgrade <release> <chart> [args...]", Run: svc.withArgs("upgrade")},
			{Name: "uninstall", Aliases: []string{"delete", "del"}, Description: "Remove uma release", Usage: "helm uninstall <release> [args...]", Run: svc.uninstall},
			{Name: "template", Description: "Renderiza templates localmente", Usage: "helm template <release> <chart> [args...]", Run: svc.withArgs("template")},
			{Name: "lint", Description: "Valida um chart", Usage: "helm lint <chart> [args...]", Run: svc.withArgs("lint")},
			{Name: "repo", Description: "Gerencia repositorios Helm", Usage: "helm repo <args...>", Run: svc.withArgs("repo")},
			{Name: "raw", Description: "Envia argumentos diretamente para helm", Usage: "helm raw <args...>", Run: svc.raw},
		},
		Examples: []string{
			"helm list -A",
			"helm upgrade --install api ./chart",
			"h template api ./chart",
		},
	}
}

func (s service) uninstall(ctx cli.Context, args []string) error {
	if len(args) == 0 {
		return cli.UsageError("informe ao menos uma release")
	}
	if !hasForce(args) && !ctx.Options.Force {
		ok, err := prompt.Confirm("Remover release(s) Helm informada(s)? [s/N] ")
		if err != nil {
			return err
		}
		if !ok {
			fmt.Println("Cancelado.")
			return nil
		}
	}
	return s.run(append([]string{"uninstall"}, args...), runner.InteractiveOptions())
}

func (s service) raw(_ cli.Context, args []string) error {
	if len(args) == 0 {
		return cli.UsageError("informe argumentos para helm")
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
		helmArgs := append([]string{}, prefix...)
		helmArgs = append(helmArgs, extra...)
		return s.run(helmArgs, runner.InteractiveOptions())
	}
}

func (s service) run(args []string, opts runner.Options) error {
	return s.runner.Run(s.bin, args, opts)
}

func hasForce(args []string) bool {
	for _, arg := range args {
		switch arg {
		case "--force", "-f", "--yes", "-y":
			return true
		}
	}
	return false
}
