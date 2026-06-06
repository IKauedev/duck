package tui

import (
	"github.com/IKauedev/duck/internal/cli"
	"github.com/IKauedev/duck/internal/config"
	"github.com/IKauedev/duck/internal/runner"
)

func Command(cfg config.Config, run runner.Runner) cli.Command {
	return cli.Command{
		Name:        "tui",
		Description: "Interface interativa estilo k9s para Docker, Kubernetes e AWS",
		Usage:       "tui [--compact] [--readonly] [--confirm <modo>] [--help]",
		Run: func(_ cli.Context, args []string) error {
			opts, err := ParseOptions(args)
			if err != nil {
				if IsHelpRequested(err) {
					PrintHelp()
					return nil
				}
				return err
			}
			return Run(cfg, run, opts)
		},
		Examples: []string{
			"tui",
			"tui --compact",
			"tui --readonly",
			"tui --confirm never",
			"tui --help",
		},
	}
}
