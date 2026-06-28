package golang

import (
	"fmt"
	"io/fs"
	"path/filepath"
	"strings"

	"github.com/IKauedev/duck/internal/cli"
	"github.com/IKauedev/duck/internal/config"
	"github.com/IKauedev/duck/internal/runner"
)

type service struct {
	bin    string
	runner runner.Runner
}

func Command(cfg config.Config, run runner.Runner) cli.Command {
	svc := service{bin: cfg.GoBin, runner: run}
	return cli.Command{
		Name:        "go",
		Description: "Executa tarefas comuns de projetos Go",
		Usage:       "go <comando> [argumentos]",
		Children: []cli.Command{
			{Name: "version", Description: "Mostra a versao do Go", Usage: "go version", Run: svc.noExtra("version")},
			{Name: "env", Description: "Mostra variaveis do Go", Usage: "go env [chaves...]", Run: svc.goArgs("env")},
			{Name: "tidy", Description: "Executa go mod tidy", Usage: "go tidy", Run: svc.noExtra("mod", "tidy")},
			{Name: "download", Description: "Executa go mod download", Usage: "go download", Run: svc.noExtra("mod", "download")},
			{Name: "test", Description: "Executa testes", Usage: "go test [flags...]", Run: svc.test},
			{Name: "build", Description: "Compila o projeto atual", Usage: "go build [flags...]", Run: svc.build},
			{Name: "run", Description: "Executa go run", Usage: "go run [--] [args...]", Run: svc.runGo},
			{Name: "fmt", Description: "Formata arquivos Go com gofmt", Usage: "go fmt", Run: svc.format},
			{Name: "vet", Description: "Executa go vet ./...", Usage: "go vet [flags...]", Run: svc.vet},
			{Name: "check", Description: "Executa tidy, fmt, vet e test", Usage: "go check", Run: svc.check},
			{Name: "raw", Description: "Envia argumentos diretamente para go", Usage: "go raw <go args...>", Run: svc.raw},
		},
		Examples: []string{
			"go check",
			"go test --race",
			"go build -o duck.exe ./cmd/duck",
		},
	}
}

func (s service) test(_ cli.Context, args []string) error {
	goArgs := []string{"test", "./..."}
	goArgs = append(goArgs, args...)
	return s.run(goArgs, runner.DefaultOptions())
}

func (s service) build(_ cli.Context, args []string) error {
	goArgs := []string{"build"}
	if len(args) == 0 {
		goArgs = append(goArgs, ".")
	} else {
		goArgs = append(goArgs, args...)
	}
	return s.run(goArgs, runner.DefaultOptions())
}

func (s service) runGo(_ cli.Context, args []string) error {
	goArgs := []string{"run"}
	if len(args) == 0 {
		goArgs = append(goArgs, ".")
	} else if args[0] == "--" {
		goArgs = append(goArgs, ".")
		goArgs = append(goArgs, args[1:]...)
	} else {
		goArgs = append(goArgs, args...)
	}
	return s.run(goArgs, runner.InteractiveOptions())
}

func (s service) format(_ cli.Context, args []string) error {
	if len(args) > 0 {
		return cli.UsageError("fmt nao recebe argumentos")
	}

	files, err := goFiles(".")
	if err != nil {
		return err
	}
	if len(files) == 0 {
		fmt.Println("Nenhum arquivo Go encontrado.")
		return nil
	}

	gofmtBin := "gofmt"
	if strings.HasSuffix(s.bin, "go") || strings.HasSuffix(s.bin, "go.exe") {
		gofmtBin = "gofmt"
	}

	goArgs := append([]string{"-w"}, files...)
	return s.runner.Run(gofmtBin, goArgs, runner.DefaultOptions())
}

func (s service) vet(_ cli.Context, args []string) error {
	goArgs := []string{"vet", "./..."}
	goArgs = append(goArgs, args...)
	return s.run(goArgs, runner.DefaultOptions())
}

func (s service) check(ctx cli.Context, args []string) error {
	if len(args) > 0 {
		return cli.UsageError("check nao recebe argumentos")
	}

	steps := []struct {
		name string
		run  func(cli.Context, []string) error
	}{
		{name: "go mod tidy", run: s.noExtra("mod", "tidy")},
		{name: "gofmt", run: s.format},
		{name: "go vet ./...", run: s.vet},
		{name: "go test ./...", run: s.test},
	}

	for _, step := range steps {
		fmt.Println("==>", step.name)
		if err := step.run(ctx, nil); err != nil {
			return err
		}
	}

	return nil
}

func (s service) raw(_ cli.Context, args []string) error {
	if len(args) == 0 {
		return cli.UsageError("informe argumentos para go")
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

func (s service) goArgs(prefix ...string) func(cli.Context, []string) error {
	return func(_ cli.Context, extra []string) error {
		goArgs := append([]string{}, prefix...)
		goArgs = append(goArgs, extra...)
		return s.run(goArgs, runner.DefaultOptions())
	}
}

func (s service) run(args []string, opts runner.Options) error {
	return s.runner.Run(s.bin, args, opts)
}

func goFiles(root string) ([]string, error) {
	var files []string
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			switch entry.Name() {
			case ".git", "vendor":
				return filepath.SkipDir
			}
			return nil
		}
		if strings.HasSuffix(entry.Name(), ".go") {
			files = append(files, path)
		}
		return nil
	})
	return files, err
}
