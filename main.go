package main

import (
	"os"

	"github.com/IKauedev/duck/internal/app"
)

func main() {
	args := os.Args[1:]

	// Sem argumentos: abre o Duck Shell.
	// No Windows, se não há console (lançado via Win+R, atalho etc.),
	// reabre em uma nova janela de terminal.
	// Se já há console (terminal aberto), executa o shell inline.
	if len(args) == 0 {
		if !hasConsole() {
			launchInTerminal()
			return
		}
		args = []string{"shell"}
	}

	os.Exit(app.Run(args))
}
