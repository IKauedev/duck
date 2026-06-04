package app

import (
	"fmt"
	"strings"

	"duck/internal/aws"
	"duck/internal/cli"
	"duck/internal/config"
	"duck/internal/docker"
	"duck/internal/golang"
	"duck/internal/install"
	"duck/internal/kubernetes"
	"duck/internal/runner"
	"duck/internal/wsl"
)

const Name = "duck"

func Run(args []string) int {
	cfg := config.Load()
	run := runner.New()
	commands := Commands(cfg, run)
	return cli.Run(Name, commands, args)
}

func Commands(cfg config.Config, run runner.Runner) []cli.Command {
	commands := []cli.Command{
		{
			Name:        "status",
			Description: "Mostra status das ferramentas usadas pelo Duck",
			Usage:       "status",
			Run:         status(cfg, run),
		},
		install.Command(),
		install.SetupCommand(),
		wsl.Command(cfg, run),
		docker.Command(cfg, run),
		golang.Command(cfg, run),
		kubernetes.Command(cfg, run),
		aws.Command(cfg, run),
	}

	commands = append(commands, docker.LegacyCommands(cfg, run)...)
	return commands
}

func status(cfg config.Config, run runner.Runner) func(cli.Context, []string) error {
	return func(_ cli.Context, args []string) error {
		if len(args) > 0 {
			return cli.UsageError("status nao recebe argumentos")
		}

		checkTool("Go", cfg.GoBin, []string{"version"}, run)
		wsl.Check(cfg, run)
		checkTool("Docker", cfg.DockerBin, []string{"version", "--format", "{{.Client.Version}}"}, run)
		checkTool("Kubernetes", cfg.KubectlBin, []string{"config", "current-context"}, run)
		checkTool("AWS", cfg.AWSBin, []string{"--version"}, run)
		return nil
	}
}

func checkTool(label, bin string, args []string, run runner.Runner) {
	output, err := run.Output(bin, args)
	if err != nil {
		message := strings.TrimSpace(output)
		if message == "" {
			message = err.Error()
		}
		fmt.Printf("%-12s indisponivel: %s\n", label, message)
		return
	}

	fmt.Printf("%-12s ok: %s", label, output)
}
