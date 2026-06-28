package appinstall

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/IKauedev/duck/internal/cli"
)

// Command retorna o comando `duck app` com subcomandos de instalação
func Command() cli.Command {
	return cli.Command{
		Name:        "app",
		Description: "Instala o Duck Shell como aplicativo nativo (atalho no Desktop/Launcher)",
		Usage:       "app <install|uninstall|build>",
		Children: []cli.Command{
			{
				Name:        "install",
				Description: "Cria atalho no Desktop e no Menu Iniciar/Launcher para abrir o Duck Shell",
				Usage:       "app install [--dir caminho]",
				Run:         installCmd,
				Examples: []string{
					"app install",
					"app install --dir ~/bin",
				},
			},
			{
				Name:        "uninstall",
				Description: "Remove o atalho do Desktop e do Menu Iniciar/Launcher",
				Usage:       "app uninstall",
				Run:         uninstallCmd,
			},
			{
				Name:        "build",
				Description: "Compila o duckapp.exe/duckapp launcher (GUI sem janela de console)",
				Usage:       "app build [--out caminho]",
				Run:         buildCmd,
				Examples: []string{
					"app build",
					"app build --out C:\\Users\\user\\bin\\duckapp.exe",
				},
			},
		},
	}
}

// ─────────────────────────── install ───────────────────────────

func installCmd(ctx cli.Context, args []string) error {
	outDir := ""
	for i, arg := range args {
		if arg == "--dir" && i+1 < len(args) {
			outDir = args[i+1]
		}
	}

	duckExe, err := findDuckExe()
	if err != nil {
		return fmt.Errorf("duck não encontrado: %w\nDica: execute duck install primeiro", err)
	}

	switch runtime.GOOS {
	case "windows":
		return installWindows(ctx, duckExe, outDir)
	case "darwin":
		return installMacOS(ctx, duckExe)
	default:
		return installLinux(ctx, duckExe)
	}
}

func installWindows(ctx cli.Context, duckExe, outDir string) error {
	// 1. Criar duckapp.exe (launcher sem console) se não existir
	launcherPath := filepath.Join(filepath.Dir(duckExe), "duckapp.exe")
	if _, err := os.Stat(launcherPath); os.IsNotExist(err) {
		fmt.Fprintln(ctx.Stdout, "Compilando launcher Duck Shell...")
		if err := compileLauncher(launcherPath); err != nil {
			// Fallback: usar .bat como launcher
			launcherPath = ""
		}
	}

	// 2. Obter caminho real do Desktop (funciona com OneDrive redirecionado)
	desktopDir := windowsDesktopDir()

	shortcutPath := filepath.Join(desktopDir, "Duck Shell.lnk")
	target := duckExe
	if launcherPath != "" {
		target = launcherPath
	}

	if err := createWindowsShortcut(shortcutPath, target, "Duck Shell — Terminal Interativo", duckExe); err != nil {
		// Fallback: criar .bat no Desktop
		batPath := filepath.Join(desktopDir, "Duck Shell.bat")
		batContent := fmt.Sprintf("@echo off\nstart \"Duck Shell\" \"%s\" shell\n", duckExe)
		if werr := os.WriteFile(batPath, []byte(batContent), 0644); werr != nil {
			return fmt.Errorf("não foi possível criar atalho: %w", err)
		}
		fmt.Fprintf(ctx.Stdout, "✓ Atalho criado em: %s\n", batPath)
	} else {
		fmt.Fprintf(ctx.Stdout, "✓ Atalho criado em: %s\n", shortcutPath)
	}

	// 3. Menu Iniciar
	appDataDir := os.Getenv("APPDATA")
	if appDataDir != "" {
		menuDir := filepath.Join(appDataDir, "Microsoft", "Windows", "Start Menu", "Programs")
		menuShortcut := filepath.Join(menuDir, "Duck Shell.lnk")
		_ = createWindowsShortcut(menuShortcut, target, "Duck Shell", duckExe)
		fmt.Fprintf(ctx.Stdout, "✓ Menu Iniciar: %s\n", menuShortcut)
	}

	fmt.Fprintln(ctx.Stdout, "\nDuck Shell instalado! Procure 'Duck Shell' no Menu Iniciar ou clique no atalho do Desktop.")
	fmt.Fprintln(ctx.Stdout, "Para abrir pelo terminal: duck shell")
	return nil
}

func installLinux(ctx cli.Context, duckExe string) error {
	// Criar .desktop file para freedesktop (GNOME, KDE, XFCE, etc.)
	desktopDir := filepath.Join(os.Getenv("HOME"), ".local", "share", "applications")
	_ = os.MkdirAll(desktopDir, 0755)

	// Detectar terminal disponível
	terminalExec := detectLinuxTerminalExec(duckExe)

	desktopContent := fmt.Sprintf(`[Desktop Entry]
Version=1.0
Type=Application
Name=Duck Shell
Comment=Terminal interativo com interface gráfica TUI
Exec=%s
Icon=utilities-terminal
Terminal=false
Categories=System;TerminalEmulator;
Keywords=duck;shell;terminal;docker;kubernetes;aws;
StartupNotify=true
`, terminalExec)

	desktopFile := filepath.Join(desktopDir, "duck-shell.desktop")
	if err := os.WriteFile(desktopFile, []byte(desktopContent), 0644); err != nil {
		return fmt.Errorf("erro ao criar .desktop: %w", err)
	}

	// Tornar executável
	_ = os.Chmod(desktopFile, 0755)

	// Atualizar banco de dados de aplicativos
	_ = exec.Command("update-desktop-database", desktopDir).Run()
	_ = exec.Command("xdg-desktop-menu", "forceupdate").Run()

	// Desktop (opcional)
	desktopHomeDir := filepath.Join(os.Getenv("HOME"), "Desktop")
	if _, err := os.Stat(desktopHomeDir); err == nil {
		destFile := filepath.Join(desktopHomeDir, "Duck Shell.desktop")
		_ = copyFile(desktopFile, destFile)
		_ = os.Chmod(destFile, 0755)
		fmt.Fprintf(ctx.Stdout, "✓ Atalho no Desktop: %s\n", destFile)
	}

	fmt.Fprintf(ctx.Stdout, "✓ Launcher criado em: %s\n", desktopFile)
	fmt.Fprintln(ctx.Stdout, "\nDuck Shell instalado! Procure 'Duck Shell' no launcher de aplicativos.")
	fmt.Fprintln(ctx.Stdout, "Para abrir pelo terminal: duck shell")
	return nil
}

func installMacOS(ctx cli.Context, duckExe string) error {
	// Criar um script AppleScript como .app bundle simples
	appDir := filepath.Join(os.Getenv("HOME"), "Applications", "Duck Shell.app")
	contentsDir := filepath.Join(appDir, "Contents", "MacOS")
	_ = os.MkdirAll(contentsDir, 0755)

	// Info.plist
	plist := `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>CFBundleName</key><string>Duck Shell</string>
	<key>CFBundleIdentifier</key><string>dev.duck.shell</string>
	<key>CFBundleVersion</key><string>1.0</string>
	<key>CFBundleExecutable</key><string>duck-shell</string>
</dict>
</plist>`
	_ = os.WriteFile(filepath.Join(appDir, "Contents", "Info.plist"), []byte(plist), 0644)

	// Script executável
	script := fmt.Sprintf(`#!/bin/bash
osascript -e 'tell application "Terminal" to do script "%s shell"'
`, duckExe)
	scriptPath := filepath.Join(contentsDir, "duck-shell")
	if err := os.WriteFile(scriptPath, []byte(script), 0755); err != nil {
		return fmt.Errorf("erro ao criar .app: %w", err)
	}

	fmt.Fprintf(ctx.Stdout, "✓ App criado em: %s\n", appDir)
	fmt.Fprintln(ctx.Stdout, "\nDuck Shell instalado em ~/Applications!")
	fmt.Fprintln(ctx.Stdout, "Para abrir pelo terminal: duck shell")
	return nil
}

// ─────────────────────────── uninstall ───────────────────────────

func uninstallCmd(ctx cli.Context, _ []string) error {
	removed := 0

	switch runtime.GOOS {
	case "windows":
		desktop := filepath.Join(os.Getenv("USERPROFILE"), "Desktop", "Duck Shell.lnk")
		batDesktop := filepath.Join(os.Getenv("USERPROFILE"), "Desktop", "Duck Shell.bat")
		menu := filepath.Join(os.Getenv("APPDATA"), "Microsoft", "Windows", "Start Menu", "Programs", "Duck Shell.lnk")
		for _, f := range []string{desktop, batDesktop, menu} {
			if err := os.Remove(f); err == nil {
				fmt.Fprintf(ctx.Stdout, "✓ Removido: %s\n", f)
				removed++
			}
		}
	case "darwin":
		appDir := filepath.Join(os.Getenv("HOME"), "Applications", "Duck Shell.app")
		if err := os.RemoveAll(appDir); err == nil {
			fmt.Fprintf(ctx.Stdout, "✓ Removido: %s\n", appDir)
			removed++
		}
	default:
		desktopFile := filepath.Join(os.Getenv("HOME"), ".local", "share", "applications", "duck-shell.desktop")
		desktopLink := filepath.Join(os.Getenv("HOME"), "Desktop", "Duck Shell.desktop")
		for _, f := range []string{desktopFile, desktopLink} {
			if err := os.Remove(f); err == nil {
				fmt.Fprintf(ctx.Stdout, "✓ Removido: %s\n", f)
				removed++
			}
		}
		_ = exec.Command("update-desktop-database").Run()
	}

	if removed == 0 {
		fmt.Fprintln(ctx.Stdout, "Nenhum atalho Duck Shell encontrado.")
	} else {
		fmt.Fprintln(ctx.Stdout, "Duck Shell desinstalado do launcher.")
	}
	return nil
}

// ─────────────────────────── build ───────────────────────────

func buildCmd(ctx cli.Context, args []string) error {
	outPath := "duckapp.exe"
	if runtime.GOOS != "windows" {
		outPath = "duckapp"
	}
	for i, arg := range args {
		if arg == "--out" && i+1 < len(args) {
			outPath = args[i+1]
		}
	}

	fmt.Fprintf(ctx.Stdout, "Compilando Duck Shell launcher → %s\n", outPath)
	if err := compileLauncher(outPath); err != nil {
		return fmt.Errorf("build falhou: %w\nDica: execute na raiz do repositório duck", err)
	}
	fmt.Fprintf(ctx.Stdout, "✓ Launcher compilado: %s\n", outPath)
	return nil
}

// ─────────────────────────── helpers ───────────────────────────

func findDuckExe() (string, error) {
	self, err := os.Executable()
	if err == nil {
		return self, nil
	}
	p, err := exec.LookPath("duck")
	if err == nil {
		return p, nil
	}
	return exec.LookPath("duck.exe")
}

func compileLauncher(outPath string) error {
	ldflags := "-H windowsgui"
	if runtime.GOOS != "windows" {
		ldflags = ""
	}
	args := []string{"build"}
	if ldflags != "" {
		args = append(args, "-ldflags", ldflags)
	}
	args = append(args, "-o", outPath, "./cmd/duckapp/")

	cmd := exec.Command("go", args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%w\n%s", err, string(out))
	}
	return nil
}

// windowsDesktopDir retorna o caminho real do Desktop, incluindo quando está
// redirecionado pelo OneDrive. Usa PowerShell [Environment]::GetFolderPath.
func windowsDesktopDir() string {
	ps := findPowerShell()
	if ps != "" {
		out, err := exec.Command(ps, "-NoProfile", "-NonInteractive", "-Command",
			`[Environment]::GetFolderPath('Desktop')`).Output()
		if err == nil {
			p := strings.TrimSpace(string(out))
			if p != "" {
				return p
			}
		}
	}
	// fallback
	home, _ := os.UserHomeDir()
	return filepath.Join(home, "Desktop")
}

func findPowerShell() string {
	for _, name := range []string{"powershell.exe", "pwsh.exe"} {
		if p, err := exec.LookPath(name); err == nil {
			return p
		}
	}
	return ""
}

func createWindowsShortcut(shortcutPath, target, description, workDir string) error {
	ps := findPowerShell()
	if ps == "" {
		return fmt.Errorf("PowerShell não encontrado")
	}

	// Escreve script em arquivo temporário UTF-8 para evitar problemas de
	// encoding com paths que contenham caracteres não-ASCII (ex: "Área de Trabalho").
	script := fmt.Sprintf(
		"$ws = New-Object -ComObject WScript.Shell\r\n"+
			"$sc = $ws.CreateShortcut(\"%s\")\r\n"+
			"$sc.TargetPath = \"%s\"\r\n"+
			"$sc.Arguments = \"shell\"\r\n"+
			"$sc.Description = \"%s\"\r\n"+
			"$sc.WorkingDirectory = \"%s\"\r\n"+
			"$sc.Save()\r\n",
		escapePS(shortcutPath),
		escapePS(target),
		escapePS(description),
		escapePS(filepath.Dir(workDir)),
	)

	tmp, err := os.CreateTemp("", "duck-shortcut-*.ps1")
	if err != nil {
		return fmt.Errorf("erro ao criar script temporário: %w", err)
	}
	defer os.Remove(tmp.Name())

	// Escreve BOM UTF-8 + conteúdo para que PowerShell leia corretamente
	if _, err := tmp.Write([]byte("\xef\xbb\xbf" + script)); err != nil {
		tmp.Close()
		return err
	}
	tmp.Close()

	cmd := exec.Command(ps, "-NoProfile", "-NonInteractive", "-ExecutionPolicy", "Bypass", "-File", tmp.Name())
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%w: %s", err, string(out))
	}
	return nil
}

// escapePS escapa aspas duplas para uso em strings PowerShell entre aspas duplas
func escapePS(s string) string {
	return strings.ReplaceAll(s, `"`, "`\"")
}

func detectLinuxTerminalExec(duckExe string) string {
	type term struct {
		bin    string
		format string
	}

	terminals := []term{
		{"gnome-terminal", `gnome-terminal -- bash -c "%s shell; bash"`},
		{"konsole", `konsole -e %s shell`},
		{"xfce4-terminal", `xfce4-terminal -e "%s shell"`},
		{"alacritty", `alacritty -e %s shell`},
		{"kitty", `kitty %s shell`},
		{"wezterm", `wezterm start -- %s shell`},
		{"tilix", `tilix -e "%s shell"`},
		{"xterm", `xterm -e %s shell`},
		{"lxterminal", `lxterminal -e "%s shell"`},
	}

	for _, t := range terminals {
		if _, err := exec.LookPath(t.bin); err == nil {
			return fmt.Sprintf(t.format, duckExe)
		}
	}

	// Fallback genérico
	return fmt.Sprintf(`x-terminal-emulator -e %s shell`, duckExe)
}

func copyFile(src, dst string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, data, 0644)
}

func init() {
	// Suprimir warning de import não usado de strings em compilações antigas
	_ = strings.TrimSpace
}
