package node

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"duck/internal/cli"
	"duck/internal/config"
	"duck/internal/runner"
)

type service struct {
	bin    string
	runner runner.Runner
}

func Command(cfg config.Config, run runner.Runner) cli.Command {
	svc := service{bin: cfg.NodeBin, runner: run}
	if settings, err := config.LoadSettings(); err == nil && settings["node.current"] != "" {
		svc.bin = filepath.Join(settings["node.current"], "bin", executable("node"))
		if runtime.GOOS == "windows" {
			svc.bin = filepath.Join(settings["node.current"], executable("node"))
		}
	}
	return cli.Command{
		Name:        "node",
		Aliases:     []string{"n"},
		Description: "Gerencia versoes Node.js",
		Usage:       "node <comando> [argumentos]",
		Children: []cli.Command{
			{Name: "current", Aliases: []string{"version"}, Description: "Mostra Node atual", Usage: "node current", Run: svc.current},
			{Name: "list", Aliases: []string{"ls"}, Description: "Lista instalacoes Node conhecidas", Usage: "node list", Run: svc.list},
			{Name: "add", Description: "Salva um alias para NODE_HOME", Usage: "node add <versao> <node-home>", Run: svc.add},
			{Name: "use", Description: "Alterna Node no Duck ou persiste PATH", Usage: "node use <versao|node-home> [--persist]", Run: svc.use},
			{Name: "home", Description: "Mostra NODE_HOME configurado", Usage: "node home", Run: svc.home},
			{Name: "raw", Description: "Envia argumentos diretamente para node", Usage: "node raw <node args...>", Run: svc.raw},
		},
		Examples: []string{
			"node current",
			"node add 20 C:\\tools\\node-v20",
			"node use 20 --persist",
		},
	}
}

func (s service) current(_ cli.Context, args []string) error {
	if len(args) > 0 {
		return cli.UsageError("current nao recebe argumentos")
	}
	if home := os.Getenv("NODE_HOME"); home != "" {
		fmt.Println("NODE_HOME:", home)
	}
	return s.runner.Run(s.bin, []string{"--version"}, runner.DefaultOptions())
}

func (s service) list(_ cli.Context, args []string) error {
	if len(args) > 0 {
		return cli.UsageError("list nao recebe argumentos")
	}
	homes := nodeHomes()
	if len(homes) == 0 {
		fmt.Println("Nenhuma instalacao Node conhecida. Use: duck node add <versao> <node-home>")
		return nil
	}
	for alias, home := range homes {
		fmt.Printf("%-16s %s\n", alias, home)
	}
	return nil
}

func (s service) add(_ cli.Context, args []string) error {
	if len(args) != 2 {
		return cli.UsageError("use: node add <versao> <node-home>")
	}
	home, err := filepath.Abs(args[1])
	if err != nil {
		return err
	}
	if err := validateNodeHome(home); err != nil {
		return err
	}
	if err := config.SetSetting("node.home."+args[0], home); err != nil {
		return err
	}
	fmt.Println("Node salvo:", args[0], "=>", home)
	return nil
}

func (s service) use(_ cli.Context, args []string) error {
	persist := false
	targets := make([]string, 0, len(args))
	for _, arg := range args {
		if arg == "--persist" {
			persist = true
		} else {
			targets = append(targets, arg)
		}
	}
	if len(targets) != 1 {
		return cli.UsageError("use: node use <versao|node-home> [--persist]")
	}
	home, err := resolveNodeHome(targets[0])
	if err != nil {
		return err
	}
	if err := validateNodeHome(home); err != nil {
		return err
	}
	if err := config.SetSetting("node.current", home); err != nil {
		return err
	}
	if persist {
		return persistNodeHome(home)
	}
	fmt.Println("NODE_HOME salvo no Duck:", home)
	return nil
}

func (s service) home(_ cli.Context, args []string) error {
	if len(args) > 0 {
		return cli.UsageError("home nao recebe argumentos")
	}
	settings, err := config.LoadSettings()
	if err != nil {
		return err
	}
	fmt.Println(settings["node.current"])
	return nil
}

func (s service) raw(_ cli.Context, args []string) error {
	if len(args) == 0 {
		return cli.UsageError("informe argumentos para node")
	}
	return s.runner.Run(s.bin, args, runner.InteractiveOptions())
}

func nodeHomes() map[string]string {
	homes := map[string]string{}
	if home := os.Getenv("NODE_HOME"); home != "" {
		homes["env"] = home
	}
	settings, err := config.LoadSettings()
	if err == nil {
		for key, value := range settings {
			if strings.HasPrefix(key, "node.home.") {
				homes[strings.TrimPrefix(key, "node.home.")] = value
			}
		}
		if current := settings["node.current"]; current != "" {
			homes["current"] = current
		}
	}
	return homes
}

func resolveNodeHome(value string) (string, error) {
	if home, ok := nodeHomes()[value]; ok {
		return home, nil
	}
	if filepath.IsAbs(value) || strings.Contains(value, string(os.PathSeparator)) {
		return filepath.Abs(value)
	}
	return "", fmt.Errorf("node nao encontrado: %s. Use 'duck node add'", value)
}

func validateNodeHome(home string) error {
	if _, err := os.Stat(filepath.Join(home, executable("node"))); err == nil && runtime.GOOS == "windows" {
		return nil
	}
	if _, err := os.Stat(filepath.Join(home, "bin", executable("node"))); err != nil {
		return fmt.Errorf("NODE_HOME invalido, node nao encontrado em %s", home)
	}
	return nil
}

func persistNodeHome(home string) error {
	if err := config.SetSetting("node.current", home); err != nil {
		return err
	}
	fmt.Println("NODE_HOME salvo no Duck:", home)
	fmt.Println("Use 'duck node home' ou 'duck node path' para consultar. Para PATH global, configure manualmente conforme seu gerenciador Node.")
	return nil
}

func executable(name string) string {
	if runtime.GOOS == "windows" {
		return name + ".exe"
	}
	return name
}
