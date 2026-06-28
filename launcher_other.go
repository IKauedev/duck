//go:build !windows

package main

// hasConsole sempre retorna true em sistemas não-Windows
// (assume-se que há um terminal disponível).
func hasConsole() bool {
	return true
}

// launchInTerminal não tem uso em não-Windows, mas precisa existir para compilar.
func launchInTerminal() {}
