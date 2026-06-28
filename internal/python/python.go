package python

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/IKauedev/duck/internal/cli"
	"github.com/IKauedev/duck/internal/config"
	"github.com/IKauedev/duck/internal/runner"
)

type service struct {
	bin    string
	runner runner.Runner
}

func Command(cfg config.Config, run runner.Runner) cli.Command {
	svc := service{bin: cfg.PythonBin, runner: run}
	return cli.Command{
		Name:        "python",
		Aliases:     []string{"py"},
		Description: "Gerencia Python e ambientes virtuais",
		Usage:       "python <comando> [argumentos]",
		Children: []cli.Command{
			{Name: "version", Description: "Mostra Python atual", Usage: "python version", Run: svc.noExtra("--version")},
			{Name: "venv", Description: "Cria virtualenv", Usage: "python venv [path]", Run: svc.venv},
			{Name: "create", Description: "Cria virtualenv", Usage: "python create [path]", Run: svc.venv},
			{Name: "use", Description: "Mostra comando de ativacao e salva venv atual", Usage: "python use <path>", Run: svc.use},
			{Name: "test", Description: "Executa pytest", Usage: "python test [args...]", Run: svc.pythonModule("pytest")},
			{Name: "lint", Description: "Executa ruff check", Usage: "python lint [args...]", Run: svc.pythonModule("ruff", "check")},
			{Name: "format", Description: "Executa ruff format", Usage: "python format [args...]", Run: svc.pythonModule("ruff", "format")},
			{Name: "pip-install", Description: "Instala pacote com pip", Usage: "python pip-install <pkg...>", Run: svc.pythonModule("pip", "install")},
			{Name: "raw", Description: "Envia argumentos diretamente para python", Usage: "python raw <python args...>", Run: svc.raw},
		},
		Examples: []string{
			"python venv .venv",
			"python use .venv",
			"python raw -m pip install -r requirements.txt",
		},
	}
}

func (s service) venv(_ cli.Context, args []string) error {
	path := ".venv"
	if len(args) > 1 {
		return cli.UsageError("venv aceita no maximo um path")
	}
	if len(args) == 1 {
		path = args[0]
	}
	if err := s.runner.Run(s.bin, []string{"-m", "venv", path}, runner.DefaultOptions()); err != nil {
		return err
	}
	abs, err := filepath.Abs(path)
	if err == nil {
		_ = config.SetSetting("python.venv", abs)
	}
	return nil
}

func (s service) use(_ cli.Context, args []string) error {
	if len(args) != 1 {
		return cli.UsageError("use: python use <path>")
	}
	abs, err := filepath.Abs(args[0])
	if err != nil {
		return err
	}
	if _, err := os.Stat(abs); err != nil {
		return fmt.Errorf("venv nao encontrado: %s", abs)
	}
	if err := config.SetSetting("python.venv", abs); err != nil {
		return err
	}
	printActivation(abs)
	return nil
}

func (s service) raw(_ cli.Context, args []string) error {
	if len(args) == 0 {
		return cli.UsageError("informe argumentos para python")
	}
	return s.runner.Run(s.bin, args, runner.InteractiveOptions())
}

func (s service) pythonModule(module string, prefix ...string) func(cli.Context, []string) error {
	return func(_ cli.Context, args []string) error {
		pythonArgs := []string{"-m", module}
		pythonArgs = append(pythonArgs, prefix...)
		pythonArgs = append(pythonArgs, args...)
		return s.runner.Run(s.bin, pythonArgs, runner.InteractiveOptions())
	}
}

func (s service) noExtra(args ...string) func(cli.Context, []string) error {
	return func(_ cli.Context, extra []string) error {
		if len(extra) > 0 {
			return cli.UsageError("este comando nao recebe argumentos extras")
		}
		return s.runner.Run(s.bin, args, runner.DefaultOptions())
	}
}

func printActivation(path string) {
	fmt.Println("Venv salvo no Duck:", path)
	if runtime.GOOS == "windows" {
		fmt.Println("PowerShell:")
		fmt.Println(filepath.Join(path, "Scripts", "Activate.ps1"))
		fmt.Println("cmd:")
		fmt.Println(filepath.Join(path, "Scripts", "activate.bat"))
		return
	}
	fmt.Println("Shell:")
	fmt.Println("source " + filepath.ToSlash(filepath.Join(path, "bin", "activate")))
}
