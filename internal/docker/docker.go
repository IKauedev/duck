package docker

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/IKauedev/duck/internal/cli"
	"github.com/IKauedev/duck/internal/config"
	"github.com/IKauedev/duck/internal/prompt"
	"github.com/IKauedev/duck/internal/runner"
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
			"docker status api",
			"docker ps -a",
			"d logs api --tail 100",
			"d up -d",
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
		{Name: "status", Description: "Mostra Docker e status de containers", Usage: "docker status [container...]", Run: s.status},
		{Name: "ps", Aliases: []string{"containers"}, Description: "Lista containers", Usage: "docker ps [-a|--all]", Run: s.ps},
		{Name: "pick", Description: "Seleciona um container interativamente", Usage: "docker pick [logs|shell|start|stop|restart|inspect]", Run: s.pick},
		{Name: "find", Description: "Busca containers e imagens por nome", Usage: "docker find <termo>", Run: s.find},
		{Name: "stats", Description: "Mostra uso de recursos dos containers", Usage: "docker stats [container...]", Run: s.stats},
		{Name: "ports", Description: "Mostra portas publicadas de um container", Usage: "docker ports <container>", Run: s.ports},
		{Name: "inspect", Description: "Mostra resumo de um container", Usage: "docker inspect <container>", Run: s.inspectSummary},
		{Name: "health", Description: "Mostra healthcheck dos containers", Usage: "docker health [container...]", Run: s.health},
		{Name: "wait-healthy", Description: "Aguarda container ficar healthy", Usage: "docker wait-healthy <container> [--timeout segundos]", Run: s.waitHealthy},
		{Name: "cp-from", Description: "Copia arquivo do container para host", Usage: "docker cp-from <container> <origem> <destino>", Run: s.copyFrom},
		{Name: "cp-to", Description: "Copia arquivo do host para container", Usage: "docker cp-to <container> <origem> <destino>", Run: s.copyTo},
		{Name: "size", Description: "Lista uso de disco de containers e imagens", Usage: "docker size", Run: s.size},
		{Name: "open", Description: "Abre shell no container detectando bash/sh", Usage: "docker open <container>", Run: s.openShell},
		{Name: "env", Description: "Lista variaveis de ambiente do container", Usage: "docker env <container>", Run: s.containerEnv},
		{Name: "top", Description: "Lista processos do container", Usage: "docker top <container>", Run: s.containerTop},
		{Name: "backup-volume", Description: "Faz backup de volume para tar.gz", Usage: "docker backup-volume <volume> <arquivo.tar.gz>", Run: s.backupVolume},
		{Name: "restore-volume", Description: "Restaura volume a partir de tar.gz", Usage: "docker restore-volume <volume> <arquivo.tar.gz>", Run: s.restoreVolume},
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
		{Name: "rm-all", Aliases: []string{"remove-all"}, Description: "Remove todos os containers", Usage: "docker rm-all [-f|--force]", Run: s.removeAllContainers},
		{Name: "clean-all", Aliases: []string{"cleanup", "reset"}, Description: "Limpa todo o ambiente Docker", Usage: "docker clean-all [-f|--force]", Run: s.cleanAll},
		{Name: "clean-images", Description: "Remove imagens Docker nao usadas", Usage: "docker clean-images [-f|--force]", Run: s.cleanImages},
		{Name: "clean-volumes", Description: "Remove volumes Docker nao usados", Usage: "docker clean-volumes [-f|--force]", Run: s.cleanVolumes},
		{Name: "rmi", Aliases: []string{"remove-image"}, Description: "Remove imagens", Usage: "docker rmi [-f|--force] <image...>", Run: s.removeImages},
		{Name: "pull", Description: "Baixa uma imagem", Usage: "docker pull <image>", Run: s.withArgs("pull")},
		{Name: "run", Description: "Executa docker run", Usage: "docker run <docker run args...>", Run: s.withArgs("run")},
		{Name: "up", Description: "Sobe servicos do Docker Compose", Usage: "docker up [args...]", Run: s.composeWith("up")},
		{Name: "down", Description: "Remove servicos do Docker Compose", Usage: "docker down [args...]", Run: s.composeWith("down")},
		{Name: "compose", Aliases: []string{"c"}, Description: "Executa Docker Compose", Usage: "docker compose <args...>", Run: s.compose},
		{Name: "compose-find", Description: "Procura arquivo Docker Compose na pasta atual", Usage: "docker compose-find", Run: s.composeFind},
		{Name: "compose-status", Description: "Mostra status dos servicos do Compose", Usage: "docker compose-status [args...]", Run: s.composeWith("ps")},
		{Name: "compose-up", Description: "Sobe servicos do Compose", Usage: "docker compose-up [args...]", Run: s.composeWith("up")},
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
	fmt.Println("Docker encontrado.")
	if err := s.run([]string{"version", "--format", "Cliente: {{.Client.Version}} | Servidor: {{.Server.Version}}"}, runner.DefaultOptions()); err != nil {
		return fmt.Errorf("nao foi possivel falar com o Docker daemon: %w", err)
	}

	fmt.Println()
	if len(args) > 0 {
		fmt.Println("Containers informados:")
		dockerArgs := []string{"inspect", "--format", "{{.Name}} | Status: {{.State.Status}} | Running: {{.State.Running}} | Health: {{if .State.Health}}{{.State.Health.Status}}{{else}}n/a{{end}} | Image: {{.Config.Image}}"}
		dockerArgs = append(dockerArgs, args...)
		return s.run(dockerArgs, runner.DefaultOptions())
	}

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

func (s service) pick(_ cli.Context, args []string) error {
	action := ""
	if len(args) > 1 {
		return cli.UsageError("use: docker pick [logs|shell|start|stop|restart|inspect]")
	}
	if len(args) == 1 {
		if !validPickAction(args[0]) {
			return cli.UsageError("acao invalida para pick: " + args[0])
		}
		action = args[0]
	}

	output, err := s.runner.Output(s.bin, []string{"ps", "-a", "--format", "{{.Names}}\t{{.Status}}\t{{.Image}}"})
	if err != nil {
		return err
	}
	containers := parseContainerOptions(output)
	if len(containers) == 0 {
		fmt.Println("Nenhum container encontrado.")
		return nil
	}

	result, err := tea.NewProgram(newPickModel(containers, action)).Run()
	if err != nil {
		return err
	}
	model, ok := result.(pickModel)
	if !ok || model.cancelled || model.selected < 0 {
		fmt.Println("Cancelado.")
		return nil
	}

	selectedAction := model.action
	if selectedAction == "" {
		selectedAction = pickActions[model.actionCursor]
	}
	return s.runPickedAction(selectedAction, containers[model.selected].name)
}

func (s service) runPickedAction(action string, container string) error {
	switch action {
	case "logs":
		return s.run([]string{"logs", "--tail", "100", container}, runner.DefaultOptions())
	case "shell":
		return s.run([]string{"exec", "-it", container, "sh"}, runner.InteractiveOptions())
	case "start", "stop", "restart":
		return s.run([]string{action, container}, runner.DefaultOptions())
	case "inspect":
		return s.run([]string{"inspect", "--format", "Nome: {{.Name}}\nImagem: {{.Config.Image}}\nStatus: {{.State.Status}}\nRunning: {{.State.Running}}\nStarted: {{.State.StartedAt}}\nIP: {{range .NetworkSettings.Networks}}{{.IPAddress}} {{end}}\nPortas: {{json .NetworkSettings.Ports}}", container}, runner.DefaultOptions())
	default:
		return cli.UsageError("acao invalida para pick: " + action)
	}
}

func (s service) find(_ cli.Context, args []string) error {
	if len(args) != 1 {
		return cli.UsageError("informe exatamente um termo de busca")
	}

	term := args[0]
	fmt.Println("Containers:")
	if err := s.run([]string{"ps", "-a", "--filter", "name=" + term, "--format", "table {{.Names}}\t{{.Status}}\t{{.Image}}"}, runner.DefaultOptions()); err != nil {
		return err
	}

	fmt.Println()
	fmt.Println("Imagens:")
	return s.run([]string{"images", "--filter", "reference=*" + term + "*", "--format", "table {{.Repository}}\t{{.Tag}}\t{{.ID}}\t{{.Size}}"}, runner.DefaultOptions())
}

func (s service) stats(_ cli.Context, args []string) error {
	dockerArgs := []string{"stats"}
	dockerArgs = append(dockerArgs, args...)
	return s.run(dockerArgs, runner.InteractiveOptions())
}

func (s service) ports(_ cli.Context, args []string) error {
	if len(args) != 1 {
		return cli.UsageError("informe exatamente um container")
	}
	return s.run([]string{"port", args[0]}, runner.DefaultOptions())
}

func (s service) inspectSummary(_ cli.Context, args []string) error {
	if len(args) != 1 {
		return cli.UsageError("informe exatamente um container")
	}
	return s.run([]string{"inspect", "--format", "Nome: {{.Name}}\nImagem: {{.Config.Image}}\nStatus: {{.State.Status}}\nRunning: {{.State.Running}}\nStarted: {{.State.StartedAt}}\nIP: {{range .NetworkSettings.Networks}}{{.IPAddress}} {{end}}\nPortas: {{json .NetworkSettings.Ports}}", args[0]}, runner.DefaultOptions())
}

func (s service) health(_ cli.Context, args []string) error {
	dockerArgs := []string{"inspect", "--format", "{{.Name}} | Status: {{.State.Status}} | Health: {{if .State.Health}}{{.State.Health.Status}}{{else}}n/a{{end}}"}
	if len(args) == 0 {
		output, err := s.runner.Output(s.bin, []string{"ps", "-aq"})
		if err != nil {
			return err
		}
		args = nonEmptyLines(output)
	}
	if len(args) == 0 {
		fmt.Println("Nenhum container encontrado.")
		return nil
	}
	dockerArgs = append(dockerArgs, args...)
	return s.run(dockerArgs, runner.DefaultOptions())
}

func (s service) waitHealthy(_ cli.Context, args []string) error {
	timeout := 60 * time.Second
	targets := make([]string, 0, len(args))
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--timeout":
			if i+1 >= len(args) {
				return cli.UsageError("--timeout precisa de um valor")
			}
			seconds, err := time.ParseDuration(args[i+1] + "s")
			if err != nil {
				return cli.UsageError("--timeout precisa ser numero de segundos")
			}
			timeout = seconds
			i++
		default:
			targets = append(targets, args[i])
		}
	}
	if len(targets) != 1 {
		return cli.UsageError("use: wait-healthy <container> [--timeout segundos]")
	}
	deadline := time.Now().Add(timeout)
	for {
		output, err := s.runner.Output(s.bin, []string{"inspect", "--format", "{{if .State.Health}}{{.State.Health.Status}}{{else}}{{.State.Status}}{{end}}", targets[0]})
		if err != nil {
			return err
		}
		status := strings.TrimSpace(output)
		if status == "healthy" || status == "running" {
			fmt.Println(targets[0], status)
			return nil
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("timeout aguardando %s ficar healthy, status atual: %s", targets[0], status)
		}
		time.Sleep(2 * time.Second)
	}
}

func (s service) copyFrom(_ cli.Context, args []string) error {
	if len(args) != 3 {
		return cli.UsageError("use: cp-from <container> <origem> <destino>")
	}
	return s.run([]string{"cp", args[0] + ":" + args[1], args[2]}, runner.DefaultOptions())
}

func (s service) copyTo(_ cli.Context, args []string) error {
	if len(args) != 3 {
		return cli.UsageError("use: cp-to <container> <origem> <destino>")
	}
	return s.run([]string{"cp", args[1], args[0] + ":" + args[2]}, runner.DefaultOptions())
}

func (s service) size(_ cli.Context, args []string) error {
	if len(args) > 0 {
		return cli.UsageError("size nao recebe argumentos")
	}
	if err := s.run([]string{"system", "df", "-v"}, runner.DefaultOptions()); err != nil {
		return err
	}
	return nil
}

func (s service) openShell(_ cli.Context, args []string) error {
	if len(args) != 1 {
		return cli.UsageError("informe exatamente um container")
	}
	shell := "sh"
	if _, err := s.runner.Output(s.bin, []string{"exec", args[0], "bash", "-lc", "true"}); err == nil {
		shell = "bash"
	}
	return s.run([]string{"exec", "-it", args[0], shell}, runner.InteractiveOptions())
}

func (s service) containerEnv(_ cli.Context, args []string) error {
	if len(args) != 1 {
		return cli.UsageError("informe exatamente um container")
	}
	return s.run([]string{"exec", args[0], "env"}, runner.DefaultOptions())
}

func (s service) containerTop(_ cli.Context, args []string) error {
	if len(args) != 1 {
		return cli.UsageError("informe exatamente um container")
	}
	return s.run([]string{"top", args[0]}, runner.DefaultOptions())
}

func (s service) backupVolume(_ cli.Context, args []string) error {
	if len(args) != 2 {
		return cli.UsageError("use: backup-volume <volume> <arquivo.tar.gz>")
	}
	archive, err := filepath.Abs(args[1])
	if err != nil {
		return err
	}
	return s.run([]string{"run", "--rm", "-v", args[0] + ":/volume:ro", "-v", filepath.Dir(archive) + ":/backup", "alpine", "tar", "czf", "/backup/" + filepath.Base(archive), "-C", "/volume", "."}, runner.DefaultOptions())
}

func (s service) restoreVolume(_ cli.Context, args []string) error {
	if len(args) != 2 {
		return cli.UsageError("use: restore-volume <volume> <arquivo.tar.gz>")
	}
	archive, err := filepath.Abs(args[1])
	if err != nil {
		return err
	}
	return s.run([]string{"run", "--rm", "-v", args[0] + ":/volume", "-v", filepath.Dir(archive) + ":/backup", "alpine", "sh", "-c", "cd /volume && tar xzf /backup/" + filepath.Base(archive)}, runner.DefaultOptions())
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

func (s service) removeAllContainers(_ cli.Context, args []string) error {
	force, rest := parseForceTargets(args)
	if len(rest) > 0 {
		return cli.UsageError("rm-all aceita apenas -f, --force, -y ou --yes")
	}

	return s.removeAllContainersWithConfirm(force, "Remover todos os containers, incluindo containers em execucao? [s/N] ")
}

func (s service) cleanAll(_ cli.Context, args []string) error {
	force, rest := parseForceTargets(args)
	if len(rest) > 0 {
		return cli.UsageError("clean-all aceita apenas -f, --force, -y ou --yes")
	}

	if !force {
		ok, err := prompt.Confirm("Limpar todo o ambiente Docker? Isto remove containers, imagens nao usadas, volumes, redes e cache. [s/N] ")
		if err != nil {
			return err
		}
		if !ok {
			fmt.Println("Cancelado.")
			return nil
		}
	}

	if err := s.removeAllContainersWithConfirm(true, ""); err != nil {
		return err
	}

	return s.run([]string{"system", "prune", "-a", "--volumes", "-f"}, runner.DefaultOptions())
}

func (s service) cleanImages(_ cli.Context, args []string) error {
	force, rest := parseForceTargets(args)
	if len(rest) > 0 {
		return cli.UsageError("clean-images aceita apenas -f, --force, -y ou --yes")
	}

	if !force {
		ok, err := prompt.Confirm("Remover imagens Docker nao usadas? [s/N] ")
		if err != nil {
			return err
		}
		if !ok {
			fmt.Println("Cancelado.")
			return nil
		}
	}

	return s.run([]string{"image", "prune", "-a", "-f"}, runner.DefaultOptions())
}

func (s service) cleanVolumes(_ cli.Context, args []string) error {
	force, rest := parseForceTargets(args)
	if len(rest) > 0 {
		return cli.UsageError("clean-volumes aceita apenas -f, --force, -y ou --yes")
	}

	if !force {
		ok, err := prompt.Confirm("Remover volumes Docker nao usados? [s/N] ")
		if err != nil {
			return err
		}
		if !ok {
			fmt.Println("Cancelado.")
			return nil
		}
	}

	return s.run([]string{"volume", "prune", "-f"}, runner.DefaultOptions())
}

func (s service) removeAllContainersWithConfirm(force bool, message string) error {
	output, err := s.runner.Output(s.bin, []string{"ps", "-aq"})
	if err != nil {
		return err
	}

	targets := nonEmptyLines(output)
	if len(targets) == 0 {
		fmt.Println("Nenhum container encontrado.")
		return nil
	}

	if !force {
		ok, err := prompt.Confirm(message)
		if err != nil {
			return err
		}
		if !ok {
			fmt.Println("Cancelado.")
			return nil
		}
	}

	dockerArgs := append([]string{"rm", "-f"}, targets...)
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

func (s service) composeFind(_ cli.Context, args []string) error {
	if len(args) > 0 {
		return cli.UsageError("compose-find nao recebe argumentos")
	}

	path, ok := findComposeFile(".")
	if !ok {
		return fmt.Errorf("arquivo compose nao encontrado na pasta atual ou em pastas acima")
	}

	fmt.Println(path)
	return nil
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

type containerOption struct {
	name   string
	status string
	image  string
}

type pickModel struct {
	containers   []containerOption
	action       string
	selected     int
	cursor       int
	actionCursor int
	stage        int
	cancelled    bool
}

var (
	pickActions       = []string{"logs", "shell", "start", "stop", "restart", "inspect"}
	pickTitleStyle    = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("229"))
	pickSelectedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("229")).Background(lipgloss.Color("62")).Padding(0, 1)
	pickMutedStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
)

func newPickModel(containers []containerOption, action string) pickModel {
	return pickModel{containers: containers, action: action, selected: -1}
}

func (m pickModel) Init() tea.Cmd {
	return nil
}

func (m pickModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	key, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, nil
	}
	switch key.String() {
	case "ctrl+c", "esc", "q":
		m.cancelled = true
		return m, tea.Quit
	case "up", "k":
		if m.stage == 0 && m.cursor > 0 {
			m.cursor--
		}
		if m.stage == 1 && m.actionCursor > 0 {
			m.actionCursor--
		}
	case "down", "j":
		if m.stage == 0 && m.cursor < len(m.containers)-1 {
			m.cursor++
		}
		if m.stage == 1 && m.actionCursor < len(pickActions)-1 {
			m.actionCursor++
		}
	case "enter":
		if m.stage == 0 {
			m.selected = m.cursor
			if m.action != "" {
				return m, tea.Quit
			}
			m.stage = 1
			return m, nil
		}
		return m, tea.Quit
	}
	return m, nil
}

func (m pickModel) View() string {
	var builder strings.Builder
	if m.stage == 0 {
		builder.WriteString(pickTitleStyle.Render("Escolha um container"))
		builder.WriteString("\n\n")
		for index, container := range m.containers {
			line := fmt.Sprintf("%s  %s  %s", container.name, container.status, container.image)
			if index == m.cursor {
				builder.WriteString(pickSelectedStyle.Render("> " + line))
			} else {
				builder.WriteString("  " + line)
			}
			builder.WriteString("\n")
		}
	} else {
		builder.WriteString(pickTitleStyle.Render("Escolha a acao para " + m.containers[m.selected].name))
		builder.WriteString("\n\n")
		for index, action := range pickActions {
			if index == m.actionCursor {
				builder.WriteString(pickSelectedStyle.Render("> " + action))
			} else {
				builder.WriteString("  " + action)
			}
			builder.WriteString("\n")
		}
	}
	builder.WriteString("\n")
	builder.WriteString(pickMutedStyle.Render("setas/j/k navegam | enter seleciona | q sai"))
	builder.WriteString("\n")
	return builder.String()
}

func parseContainerOptions(output string) []containerOption {
	lines := strings.Split(strings.TrimSpace(output), "\n")
	containers := make([]containerOption, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "\t", 3)
		if len(parts) < 3 {
			continue
		}
		containers = append(containers, containerOption{
			name:   strings.TrimSpace(parts[0]),
			status: strings.TrimSpace(parts[1]),
			image:  strings.TrimSpace(parts[2]),
		})
	}
	return containers
}

func validPickAction(action string) bool {
	for _, candidate := range pickActions {
		if candidate == action {
			return true
		}
	}
	return false
}

func nonEmptyLines(output string) []string {
	lines := strings.Fields(output)
	if len(lines) == 0 {
		return nil
	}
	return lines
}

func findComposeFile(start string) (string, bool) {
	names := []string{"compose.yaml", "compose.yml", "docker-compose.yaml", "docker-compose.yml"}
	dir, err := filepath.Abs(start)
	if err != nil {
		return "", false
	}

	for {
		for _, name := range names {
			path := filepath.Join(dir, name)
			if info, err := os.Stat(path); err == nil && !info.IsDir() {
				return path, true
			}
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			return "", false
		}
		dir = parent
	}
}
