package docker

import (
	"fmt"

	"duck/internal/cli"
	"duck/internal/config"
	"duck/internal/prompt"
	"duck/internal/runner"
)

type service struct {
	bin        string
	composeBin string
	runner     runner.Runner
}

func Command(cfg config.Config, run runner.Runner) cli.Command {
	svc := service{bin: cfg.DockerBin, composeBin: cfg.DockerComposeBin, runner: run}
	return cli.Command{
		Name:        "docker",
		Aliases:     []string{"d"},
		Description: "Executa tarefas Docker",
		Usage:       "docker <comando> [argumentos]",
		Children:    svc.commands(),
		Examples: []string{
			"docker status",
			"docker ps -a",
			"d logs api --tail 100",
			"d compose up -d",
		},
	}
}

func LegacyCommands(cfg config.Config, run runner.Runner) []cli.Command {
	svc := service{bin: cfg.DockerBin, composeBin: cfg.DockerComposeBin, runner: run}
	commands := svc.commands()
	legacy := commands[1:]
	for index := range legacy {
		legacy[index].Category = "Atalhos Docker diretos"
	}
	return legacy
}

func (s service) commands() []cli.Command {
	return []cli.Command{
		{Name: "status", Description: "Mostra se o Docker esta acessivel", Usage: "docker status", Run: s.status},
		{Name: "ps", Aliases: []string{"containers"}, Description: "Lista containers", Usage: "docker ps [-a|--all]", Run: s.ps},
		{Name: "images", Aliases: []string{"imgs"}, Description: "Lista imagens", Usage: "docker images", Run: s.noExtra("images")},
		{Name: "volumes", Aliases: []string{"vols"}, Description: "Lista volumes", Usage: "docker volumes", Run: s.noExtra("volume", "ls")},
		{Name: "networks", Aliases: []string{"nets"}, Description: "Lista redes", Usage: "docker networks", Run: s.noExtra("network", "ls")},
		{Name: "start", Description: "Inicia containers", Usage: "docker start <container...>", Run: s.withArgs("start")},
		{Name: "stop", Description: "Para containers", Usage: "docker stop <container...>", Run: s.withArgs("stop")},
		{Name: "restart", Description: "Reinicia containers", Usage: "docker restart <container...>", Run: s.withArgs("restart")},
		{Name: "logs", Description: "Exibe logs de um container", Usage: "docker logs <container> [-f|--follow] [--tail N]", Run: s.logs},
		{Name: "shell", Aliases: []string{"sh"}, Description: "Abre shell no container", Usage: "docker shell <container> [sh|bash|ash|powershell]", Run: s.shell},
		{Name: "exec", Description: "Executa comando no container", Usage: "docker exec <container> -- <comando...>", Run: s.exec},
		{Name: "rm", Aliases: []string{"remove"}, Description: "Remove containers", Usage: "docker rm [-f|--force] <container...>", Run: s.removeContainers},
		{Name: "rmi", Aliases: []string{"remove-image"}, Description: "Remove imagens", Usage: "docker rmi [-f|--force] <image...>", Run: s.removeImages},
		{Name: "pull", Description: "Baixa uma imagem", Usage: "docker pull <image>", Run: s.withArgs("pull")},
		{Name: "run", Description: "Executa docker run", Usage: "docker run <docker run args...>", Run: s.withArgs("run")},
		{Name: "compose", Aliases: []string{"c"}, Description: "Executa Docker Compose", Usage: "docker compose <args...>", Run: s.compose},
		{Name: "compose-ps", Description: "Lista servicos do Compose", Usage: "docker compose-ps [args...]", Run: s.composeWith("ps")},
		{Name: "compose-logs", Description: "Exibe logs do Compose", Usage: "docker compose-logs [args...]", Run: s.composeWith("logs")},
		{Name: "compose-stop", Description: "Para servicos do Compose", Usage: "docker compose-stop [servico...]", Run: s.composeWith("stop")},
		{Name: "compose-restart", Description: "Reinicia servicos do Compose", Usage: "docker compose-restart [servico...]", Run: s.composeWith("restart")},
		{Name: "compose-down", Description: "Remove containers e rede do Compose", Usage: "docker compose-down [args...]", Run: s.composeWith("down")},
		{Name: "compose-rm", Description: "Remove containers parados do Compose", Usage: "docker compose-rm [-f] [servico...]", Run: s.composeRemove},
		{Name: "prune", Description: "Limpa recursos nao utilizados", Usage: "docker prune [containers|images|volumes|networks|system] [-f|--force]", Run: s.prune},
		{Name: "raw", Description: "Envia argumentos diretamente para Docker", Usage: "docker raw <docker args...>", Run: s.raw},
	}
}

func (s service) status(_ cli.Context, args []string) error {
	if len(args) > 0 {
		return cli.UsageError("status nao recebe argumentos")
	}

	fmt.Println("Docker encontrado.")
	if err := s.run([]string{"version", "--format", "Cliente: {{.Client.Version}} | Servidor: {{.Server.Version}}"}, runner.DefaultOptions()); err != nil {
		return fmt.Errorf("nao foi possivel falar com o Docker daemon: %w", err)
	}

	fmt.Println()
	fmt.Println("Containers:")
	return s.run([]string{"ps", "-a", "--format", "table {{.Names}}\t{{.Status}}\t{{.Image}}\t{{.Ports}}"}, runner.DefaultOptions())
}

func (s service) ps(_ cli.Context, args []string) error {
	dockerArgs := []string{"ps", "--format", "table {{.Names}}\t{{.Status}}\t{{.Image}}\t{{.Ports}}"}
	for _, arg := range args {
		switch arg {
		case "-a", "--all":
			dockerArgs = append(dockerArgs, "-a")
		default:
			return cli.UsageError("opcao invalida para ps: " + arg)
		}
	}

	return s.run(dockerArgs, runner.DefaultOptions())
}

func (s service) logs(_ cli.Context, args []string) error {
	if len(args) == 0 {
		return cli.UsageError("informe o container")
	}

	dockerArgs := []string{"logs"}
	container := args[0]
	for i := 1; i < len(args); i++ {
		switch args[i] {
		case "-f", "--follow":
			dockerArgs = append(dockerArgs, "--follow")
		case "--tail":
			if i+1 >= len(args) {
				return cli.UsageError("--tail precisa de um valor")
			}
			dockerArgs = append(dockerArgs, "--tail", args[i+1])
			i++
		default:
			return cli.UsageError("opcao invalida para logs: " + args[i])
		}
	}

	dockerArgs = append(dockerArgs, container)
	return s.run(dockerArgs, runner.DefaultOptions())
}

func (s service) shell(_ cli.Context, args []string) error {
	if len(args) == 0 {
		return cli.UsageError("informe o container")
	}
	if len(args) > 2 {
		return cli.UsageError("shell aceita apenas container e shell opcional")
	}

	shell := "sh"
	if len(args) == 2 {
		shell = args[1]
	}

	return s.run([]string{"exec", "-it", args[0], shell}, runner.InteractiveOptions())
}

func (s service) exec(_ cli.Context, args []string) error {
	if len(args) < 2 {
		return cli.UsageError("use: exec <container> -- <comando...>")
	}

	cmdStart := 1
	if args[1] == "--" {
		cmdStart = 2
	}
	if cmdStart >= len(args) {
		return cli.UsageError("informe o comando para executar")
	}

	dockerArgs := append([]string{"exec", "-it", args[0]}, args[cmdStart:]...)
	return s.run(dockerArgs, runner.InteractiveOptions())
}

func (s service) removeContainers(_ cli.Context, args []string) error {
	force, targets := parseForceTargets(args)
	if len(targets) == 0 {
		return cli.UsageError("informe ao menos um container")
	}

	if !force {
		ok, err := prompt.Confirm("Remover containers informados? [s/N] ")
		if err != nil {
			return err
		}
		if !ok {
			fmt.Println("Cancelado.")
			return nil
		}
	}

	dockerArgs := []string{"rm"}
	if force {
		dockerArgs = append(dockerArgs, "-f")
	}
	dockerArgs = append(dockerArgs, targets...)
	return s.run(dockerArgs, runner.DefaultOptions())
}

func (s service) removeImages(_ cli.Context, args []string) error {
	force, targets := parseForceTargets(args)
	if len(targets) == 0 {
		return cli.UsageError("informe ao menos uma imagem")
	}

	if !force {
		ok, err := prompt.Confirm("Remover imagens informadas? [s/N] ")
		if err != nil {
			return err
		}
		if !ok {
			fmt.Println("Cancelado.")
			return nil
		}
	}

	dockerArgs := []string{"rmi"}
	if force {
		dockerArgs = append(dockerArgs, "-f")
	}
	dockerArgs = append(dockerArgs, targets...)
	return s.run(dockerArgs, runner.DefaultOptions())
}

func (s service) prune(_ cli.Context, args []string) error {
	resource := "system"
	force := false

	for _, arg := range args {
		switch arg {
		case "-f", "--force", "-y", "--yes":
			force = true
		case "containers", "container":
			resource = "container"
		case "images", "image":
			resource = "image"
		case "volumes", "volume":
			resource = "volume"
		case "networks", "network":
			resource = "network"
		case "system", "all":
			resource = "system"
		default:
			return cli.UsageError("recurso invalido para prune: " + arg)
		}
	}

	if !force {
		ok, err := prompt.Confirm("Esta acao pode remover recursos Docker nao utilizados. Continuar? [s/N] ")
		if err != nil {
			return err
		}
		if !ok {
			fmt.Println("Cancelado.")
			return nil
		}
	}

	dockerArgs := []string{resource, "prune"}
	if force {
		dockerArgs = append(dockerArgs, "-f")
	}
	return s.run(dockerArgs, runner.DefaultOptions())
}

func (s service) compose(_ cli.Context, args []string) error {
	if len(args) == 0 {
		return cli.UsageError("argumentos obrigatorios ausentes")
	}
	return s.runCompose(args, runner.InteractiveOptions())
}

func (s service) composeWith(command string) func(cli.Context, []string) error {
	return func(_ cli.Context, args []string) error {
		composeArgs := append([]string{command}, args...)
		return s.runCompose(composeArgs, runner.InteractiveOptions())
	}
}

func (s service) composeRemove(_ cli.Context, args []string) error {
	force, targets := parseForceTargets(args)
	if !force {
		ok, err := prompt.Confirm("Remover containers parados do Docker Compose? [s/N] ")
		if err != nil {
			return err
		}
		if !ok {
			fmt.Println("Cancelado.")
			return nil
		}
	}

	composeArgs := []string{"rm"}
	if force {
		composeArgs = append(composeArgs, "-f")
	}
	composeArgs = append(composeArgs, targets...)
	return s.runCompose(composeArgs, runner.DefaultOptions())
}

func (s service) raw(_ cli.Context, args []string) error {
	if len(args) == 0 {
		return cli.UsageError("informe argumentos para o Docker")
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
		dockerArgs := append([]string{}, prefix...)
		dockerArgs = append(dockerArgs, extra...)
		return s.run(dockerArgs, runner.InteractiveOptions())
	}
}

func (s service) run(args []string, opts runner.Options) error {
	return s.runner.Run(s.bin, args, opts)
}

func (s service) runCompose(args []string, opts runner.Options) error {
	if _, err := s.runner.Output(s.bin, []string{"compose", "version"}); err == nil {
		dockerArgs := append([]string{"compose"}, args...)
		return s.runner.Run(s.bin, dockerArgs, opts)
	}

	if _, err := s.runner.Output(s.composeBin, []string{"version"}); err == nil {
		return s.runner.Run(s.composeBin, args, opts)
	}

	return fmt.Errorf("Docker Compose nao encontrado; instale o plugin 'docker compose' ou o binario '%s'", s.composeBin)
}

func parseForceTargets(args []string) (bool, []string) {
	force := false
	targets := make([]string, 0, len(args))

	for _, arg := range args {
		switch arg {
		case "-f", "--force", "-y", "--yes":
			force = true
		default:
			targets = append(targets, arg)
		}
	}

	return force, targets
}
