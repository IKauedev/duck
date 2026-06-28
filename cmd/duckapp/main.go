// Duck Shell App Launcher
//
// Abre o Duck Shell (duck shell) em uma janela de terminal dedicada,
// como se fosse um aplicativo nativo. No Windows usa Windows Terminal
// ou cmd; no Linux abre o emulador de terminal padrão.
//
// Build no Windows (sem janela de console):
//
//	go build -ldflags "-H windowsgui -X main.version=1.0" -o duckapp.exe ./cmd/duckapp/
//
// Build no Linux:
//
//	go build -o duckapp ./cmd/duckapp/
package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

var version = "dev"

func main() {
	duckExe := findDuck()

	switch runtime.GOOS {
	case "windows":
		launchWindows(duckExe)
	case "darwin":
		launchMacOS(duckExe)
	default:
		launchLinux(duckExe)
	}
}

// findDuck localiza o duck.exe/duck no PATH ou na mesma pasta do launcher
func findDuck() string {
	// 1. Mesmo diretório do launcher
	self, err := os.Executable()
	if err == nil {
		dir := filepath.Dir(self)
		candidates := []string{
			filepath.Join(dir, "duck.exe"),
			filepath.Join(dir, "duck"),
		}
		for _, c := range candidates {
			if _, err := os.Stat(c); err == nil {
				return c
			}
		}
	}

	// 2. PATH
	if p, err := exec.LookPath("duck"); err == nil {
		return p
	}
	if p, err := exec.LookPath("duck.exe"); err == nil {
		return p
	}

	// Fallback — espera que esteja no PATH
	if runtime.GOOS == "windows" {
		return "duck.exe"
	}
	return "duck"
}

// launchWindows abre duck shell no Windows Terminal, ou cai para cmd.exe
func launchWindows(duckExe string) {
	title := "Duck Shell"
	shellCmd := `"` + duckExe + `" shell`

	// Tenta Windows Terminal (wt.exe)
	if wt, err := exec.LookPath("wt.exe"); err == nil {
		// wt -p "Command Prompt" --title "Duck Shell" cmd /k duck shell
		cmd := exec.Command(wt,
			"--title", title,
			"--",
			"cmd.exe", "/k", shellCmd,
		)
		if err := cmd.Start(); err == nil {
			return
		}
	}

	// Tenta PowerShell 7 (pwsh.exe)
	if pwsh, err := exec.LookPath("pwsh.exe"); err == nil {
		cmd := exec.Command(pwsh,
			"-NoExit",
			"-Command",
			shellCmd,
		)
		cmd.SysProcAttr = windowsNewWindow()
		if err := cmd.Start(); err == nil {
			return
		}
	}

	// Fallback: cmd.exe clássico
	cmd := exec.Command("cmd.exe", "/k", shellCmd)
	cmd.SysProcAttr = windowsNewWindow()
	_ = cmd.Start()
}

// launchMacOS abre duck shell no Terminal.app ou iTerm2
func launchMacOS(duckExe string) {
	shellCmd := duckExe + " shell"

	// Tenta iTerm2
	itermScript := `tell application "iTerm2"
		create window with default profile
		tell current session of current window
			write text "` + shellCmd + `"
		end tell
	end tell`
	if _, err := exec.LookPath("osascript"); err == nil {
		cmd := exec.Command("osascript", "-e", itermScript)
		if err := cmd.Start(); err == nil {
			return
		}
	}

	// Fallback: Terminal.app
	termScript := `tell application "Terminal"
		activate
		do script "` + shellCmd + `"
	end tell`
	cmd := exec.Command("osascript", "-e", termScript)
	_ = cmd.Start()
}

// launchLinux tenta vários emuladores de terminal comuns
func launchLinux(duckExe string) {
	shellCmd := duckExe + " shell"

	type terminal struct {
		bin  string
		args []string
	}

	// Detecta emulador pelo $TERM_PROGRAM ou $XDG_CURRENT_DESKTOP
	preferred := detectLinuxTerminal()

	terminals := []terminal{}
	if preferred != "" {
		terminals = append(terminals, terminal{bin: preferred, args: []string{"-e", shellCmd}})
	}

	terminals = append(terminals,
		terminal{"gnome-terminal", []string{"--", "bash", "-c", shellCmd + "; bash"}},
		terminal{"konsole", []string{"-e", shellCmd}},
		terminal{"xfce4-terminal", []string{"-e", shellCmd}},
		terminal{"tilix", []string{"-e", shellCmd}},
		terminal{"alacritty", []string{"-e", "bash", "-c", shellCmd}},
		terminal{"kitty", []string{shellCmd}},
		terminal{"wezterm", []string{"start", "--", "bash", "-c", shellCmd}},
		terminal{"xterm", []string{"-e", shellCmd}},
		terminal{"lxterminal", []string{"-e", shellCmd}},
		terminal{"mate-terminal", []string{"-e", shellCmd}},
	)

	for _, t := range terminals {
		if path, err := exec.LookPath(t.bin); err == nil {
			cmd := exec.Command(path, t.args...)
			if err := cmd.Start(); err == nil {
				return
			}
		}
	}

	// Último fallback: rodar no mesmo processo (sem janela nova)
	cmd := exec.Command(duckExe, "shell")
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	_ = cmd.Run()
}

func detectLinuxTerminal() string {
	// Verifica variáveis de ambiente comuns
	for _, env := range []string{"TERM_PROGRAM", "COLORTERM"} {
		v := strings.ToLower(os.Getenv(env))
		switch v {
		case "alacritty", "kitty", "wezterm", "xterm", "gnome-terminal":
			return v
		}
	}
	return ""
}
