package gittui

import (
	"github.com/IKauedev/duck/internal/cli"
	tea "github.com/charmbracelet/bubbletea"
)

// Command retorna o comando CLI que registra o Git TUI no duck
func Command() cli.Command {
	return cli.Command{
		Name:    "git-ui",
		Aliases: []string{"gitui", "gut"},
		Description: "Interface gráfica TUI para Git — navega commits, branches, " +
			"status e stash com teclado",
		Usage: "git-ui",
		Run: func(_ cli.Context, _ []string) error {
			return Run()
		},
		Examples: []string{
			"git-ui",
			"gitui",
			"gut",
		},
	}
}

// Run inicia o Git TUI
func Run() error {
	m := newModel()
	p := tea.NewProgram(
		m,
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
	)
	_, err := p.Run()
	return err
}
