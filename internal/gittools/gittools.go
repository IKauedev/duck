package gittools

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/IKauedev/duck/internal/cli"
	"github.com/IKauedev/duck/internal/config"
	"github.com/IKauedev/duck/internal/prompt"
	"github.com/IKauedev/duck/internal/runner"
)

type service struct {
	cfg    config.Config
	runner runner.Runner
}

func Command(cfg config.Config, run runner.Runner) cli.Command {
	svc := service{cfg: cfg, runner: run}
	return cli.Command{
		Name:        "git",
		Aliases:     []string{"g"},
		Description: "Atalhos Git simplificados e compostos",
		Usage:       "git <comando>",
		Children: []cli.Command{
			{Name: "status", Aliases: []string{"s"}, Description: "Mostra branch e arquivos alterados", Usage: "git status", Run: svc.noExtra("status", "--short", "--branch")},
			{Name: "info", Description: "Mostra resumo do repositorio", Usage: "git info", Run: svc.info},
			{Name: "log", Aliases: []string{"l"}, Description: "Mostra commits recentes", Usage: "git log [-n N]", Run: svc.log},
			{Name: "diff", Description: "Mostra diff de trabalho ou staged", Usage: "git diff [--staged]", Run: svc.diff},
			{Name: "save", Description: "Adiciona tudo e cria commit", Usage: "git save <mensagem>", Run: svc.save},
			{Name: "wip", Description: "Cria commit WIP rapido", Usage: "git wip [mensagem]", Run: svc.wip},
			{Name: "sync", Description: "Faz pull --rebase e push", Usage: "git sync", Run: svc.sync},
			{Name: "publish", Description: "Publica branch atual no origin", Usage: "git publish", Run: svc.publish},
			{Name: "ship", Description: "Salva, sincroniza e envia em um comando", Usage: "git ship <mensagem>", Run: svc.ship},
			{Name: "branches", Aliases: []string{"branch"}, Description: "Lista branches locais e remotas", Usage: "git branches", Run: svc.noExtra("branch", "-a")},
			{Name: "new", Description: "Cria e troca para nova branch", Usage: "git new <branch>", Run: svc.withOne("checkout", "-b")},
			{Name: "switch", Aliases: []string{"co"}, Description: "Troca de branch", Usage: "git switch <branch>", Run: svc.withOne("switch")},
			{Name: "cleanup", Description: "Remove branches locais ja mergeadas", Usage: "git cleanup [-f|--force]", Run: svc.cleanup},
			{Name: "stash", Description: "Atalhos para stash", Usage: "git stash <save|pop|list> [mensagem]", Run: svc.stash},
			{Name: "tag", Description: "Cria tag anotada", Usage: "git tag <nome> [mensagem]", Run: svc.tag},
			{Name: "undo", Description: "Desfaz etapas comuns com seguranca", Usage: "git undo <unstage|last>", Run: svc.undo},
			{Name: "ignore", Description: "Adiciona padrao ao .gitignore", Usage: "git ignore <padrao>", Run: svc.ignore},
			{Name: "remote", Description: "Lista remotes", Usage: "git remote", Run: svc.noExtra("remote", "-v")},
			{Name: "root", Description: "Mostra raiz do repositorio", Usage: "git root", Run: svc.noExtra("rev-parse", "--show-toplevel")},
			{Name: "raw", Description: "Envia argumentos diretamente para git", Usage: "git raw <args...>", Run: svc.raw},
		},
		Examples: []string{
			"git status",
			"git save \"feat: add new command\"",
			"git sync",
			"git ship \"fix: adjust config\"",
			"git cleanup",
		},
	}
}

func (s service) noExtra(args ...string) func(cli.Context, []string) error {
	return func(_ cli.Context, rest []string) error {
		if len(rest) > 0 {
			return cli.UsageError("este comando nao recebe argumentos")
		}
		return s.run(args...)
	}
}

func (s service) withOne(prefix ...string) func(cli.Context, []string) error {
	return func(_ cli.Context, args []string) error {
		if len(args) != 1 {
			return cli.UsageError("informe exatamente um valor")
		}
		gitArgs := append([]string{}, prefix...)
		gitArgs = append(gitArgs, args[0])
		return s.run(gitArgs...)
	}
}

func (s service) info(_ cli.Context, args []string) error {
	if len(args) > 0 {
		return cli.UsageError("git info nao recebe argumentos")
	}
	steps := [][]string{
		{"rev-parse", "--show-toplevel"},
		{"branch", "--show-current"},
		{"remote", "-v"},
		{"log", "-1", "--oneline", "--decorate"},
		{"status", "--short", "--branch"},
	}
	for _, step := range steps {
		fmt.Println("==> git", strings.Join(step, " "))
		if err := s.run(step...); err != nil {
			return err
		}
	}
	return nil
}

func (s service) log(_ cli.Context, args []string) error {
	limit := 10
	if len(args) == 2 && (args[0] == "-n" || args[0] == "--limit") {
		value, err := strconv.Atoi(args[1])
		if err != nil || value <= 0 {
			return cli.UsageError(args[0] + " precisa ser numero positivo")
		}
		limit = value
	} else if len(args) > 0 {
		return cli.UsageError("use: git log [-n N]")
	}
	return s.run("log", "--oneline", "--decorate", "--graph", "-n", strconv.Itoa(limit))
}

func (s service) diff(_ cli.Context, args []string) error {
	gitArgs := []string{"diff"}
	if len(args) == 1 && args[0] == "--staged" {
		gitArgs = append(gitArgs, "--staged")
	} else if len(args) > 0 {
		return cli.UsageError("use: git diff [--staged]")
	}
	return s.run(gitArgs...)
}

func (s service) save(_ cli.Context, args []string) error {
	if len(args) == 0 {
		return cli.UsageError("use: git save <mensagem>")
	}
	return s.commitAll(strings.Join(args, " "))
}

func (s service) wip(_ cli.Context, args []string) error {
	message := "wip"
	if len(args) > 0 {
		message = "wip: " + strings.Join(args, " ")
	}
	return s.commitAll(message)
}

func (s service) ship(_ cli.Context, args []string) error {
	if len(args) == 0 {
		return cli.UsageError("use: git ship <mensagem>")
	}
	if err := s.commitAll(strings.Join(args, " ")); err != nil {
		return err
	}
	return s.sync(cli.Context{}, nil)
}

func (s service) commitAll(message string) error {
	changed, err := s.hasChanges()
	if err != nil {
		return err
	}
	if !changed {
		fmt.Println("Nenhuma alteracao para commitar.")
		return nil
	}
	if err := s.run("add", "-A"); err != nil {
		return err
	}
	return s.run("commit", "-m", message)
}

func (s service) sync(_ cli.Context, args []string) error {
	if len(args) > 0 {
		return cli.UsageError("git sync nao recebe argumentos")
	}
	if err := s.run("pull", "--rebase"); err != nil {
		return err
	}
	return s.run("push")
}

func (s service) publish(_ cli.Context, args []string) error {
	if len(args) > 0 {
		return cli.UsageError("git publish nao recebe argumentos")
	}
	return s.run("push", "-u", "origin", "HEAD")
}

func (s service) cleanup(_ cli.Context, args []string) error {
	force := false
	for _, arg := range args {
		switch arg {
		case "-f", "--force":
			force = true
		default:
			return cli.UsageError("use: git cleanup [-f|--force]")
		}
	}
	current, err := s.output("branch", "--show-current")
	if err != nil {
		return err
	}
	output, err := s.output("branch", "--merged")
	if err != nil {
		return err
	}
	var branches []string
	for _, line := range strings.Split(output, "\n") {
		branch := strings.TrimSpace(strings.TrimPrefix(line, "*"))
		if branch == "" || branch == current || protectedBranch(branch) {
			continue
		}
		branches = append(branches, branch)
	}
	if len(branches) == 0 {
		fmt.Println("Nenhuma branch mergeada para remover.")
		return nil
	}
	fmt.Println("Branches que serao removidas:")
	for _, branch := range branches {
		fmt.Println(" ", branch)
	}
	if !force {
		ok, err := prompt.Confirm("Remover branches mergeadas? [s/N] ")
		if err != nil {
			return err
		}
		if !ok {
			fmt.Println("Cancelado.")
			return nil
		}
	}
	for _, branch := range branches {
		if err := s.run("branch", "-d", branch); err != nil {
			return err
		}
	}
	return nil
}

func (s service) stash(_ cli.Context, args []string) error {
	if len(args) == 0 {
		args = []string{"list"}
	}
	switch args[0] {
	case "list":
		if len(args) != 1 {
			return cli.UsageError("use: git stash list")
		}
		return s.run("stash", "list")
	case "pop":
		if len(args) != 1 {
			return cli.UsageError("use: git stash pop")
		}
		return s.run("stash", "pop")
	case "save":
		message := "duck stash"
		if len(args) > 1 {
			message = strings.Join(args[1:], " ")
		}
		return s.run("stash", "push", "-u", "-m", message)
	default:
		return cli.UsageError("use: git stash <save|pop|list> [mensagem]")
	}
}

func (s service) tag(_ cli.Context, args []string) error {
	if len(args) < 1 {
		return cli.UsageError("use: git tag <nome> [mensagem]")
	}
	message := args[0]
	if len(args) > 1 {
		message = strings.Join(args[1:], " ")
	}
	return s.run("tag", "-a", args[0], "-m", message)
}

func (s service) undo(_ cli.Context, args []string) error {
	if len(args) != 1 {
		return cli.UsageError("use: git undo <unstage|last>")
	}
	switch args[0] {
	case "unstage":
		return s.run("restore", "--staged", ".")
	case "last":
		return s.run("reset", "--soft", "HEAD~1")
	default:
		return cli.UsageError("use: git undo <unstage|last>")
	}
}

func (s service) ignore(_ cli.Context, args []string) error {
	if len(args) != 1 {
		return cli.UsageError("use: git ignore <padrao>")
	}
	path := filepath.Clean(".gitignore")
	file, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer file.Close()
	if _, err := fmt.Fprintln(file, args[0]); err != nil {
		return err
	}
	fmt.Println("Adicionado ao .gitignore:", args[0])
	return nil
}

func (s service) raw(_ cli.Context, args []string) error {
	if len(args) == 0 {
		return cli.UsageError("use: git raw <args...>")
	}
	return s.run(args...)
}

func (s service) hasChanges() (bool, error) {
	output, err := s.output("status", "--porcelain")
	if err != nil {
		return false, err
	}
	return strings.TrimSpace(output) != "", nil
}

func (s service) run(args ...string) error {
	return s.runner.Run(s.cfg.GitBin, args, runner.InteractiveOptions())
}

func (s service) output(args ...string) (string, error) {
	return s.runner.Output(s.cfg.GitBin, args)
}

func protectedBranch(branch string) bool {
	switch branch {
	case "main", "master", "develop", "dev", "staging", "production":
		return true
	default:
		return false
	}
}
