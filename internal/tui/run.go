package tui

import (
	"os"
	"os/exec"

	"github.com/IKauedev/duck/internal/config"
	"github.com/IKauedev/duck/internal/runner"
	tea "github.com/charmbracelet/bubbletea"
)

func Run(cfg config.Config, run runner.Runner, opts Options) error {
	program := tea.NewProgram(newModel(cfg, run, opts), tea.WithAltScreen())
	result, err := program.Run()
	if err != nil {
		return err
	}
	final, ok := result.(model)
	if !ok {
		return nil
	}
	return runPending(final.pending, run)
}

func runPending(pending *pendingAction, run runner.Runner) error {
	if pending == nil {
		return nil
	}
	if len(pending.duckArgs) > 0 {
		executable, err := os.Executable()
		if err != nil {
			return err
		}
		cmd := exec.Command(executable, pending.duckArgs...)
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		return cmd.Run()
	}
	return runPendingAction(pending, run)
}
