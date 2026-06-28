package shelltui

import (
	"github.com/IKauedev/duck/internal/cli"
	"github.com/IKauedev/duck/internal/config"
	"github.com/IKauedev/duck/internal/runner"
	tea "github.com/charmbracelet/bubbletea"
)

// Command retorna o comando CLI que registra o Duck Shell no app
func Command(cfg config.Config, run runner.Runner, commands func() []cli.Command) cli.Command {
	return cli.Command{
		Name:    "shell",
		Aliases: []string{"lab", "ui"},
		Description: "Terminal visual interativo com interface gráfica TUI — " +
			"viewport de output, tabs por tecnologia, sidebar de atalhos e histórico",
		Usage: "shell [--no-sidebar]",
		Run: func(_ cli.Context, args []string) error {
			return Run(cfg, run, commands, args)
		},
		Examples: []string{
			"shell",
			"shell --no-sidebar",
			"lab",
			"ui",
		},
	}
}

// Run inicia o Duck Shell TUI
func Run(cfg config.Config, run runner.Runner, commands func() []cli.Command, args []string) error {
	m := newModel(cfg, run, commands)

	// Opção --no-sidebar
	for _, arg := range args {
		if arg == "--no-sidebar" {
			m.showSidebar = false
		}
	}

	program := tea.NewProgram(
		m,
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
	)
	_, err := program.Run()
	return err
}
