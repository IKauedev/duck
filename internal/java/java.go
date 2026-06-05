package java

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"

	"duck/internal/cli"
	"duck/internal/config"
	"duck/internal/runner"
)

type service struct {
	bin    string
	runner runner.Runner
}

func Command(cfg config.Config, run runner.Runner) cli.Command {
	svc := service{bin: cfg.JavaBin, runner: run}
	if settings, err := config.LoadSettings(); err == nil && settings["java.current"] != "" {
		svc.bin = filepath.Join(settings["java.current"], "bin", executable("java"))
	}
	return cli.Command{
		Name:        "java",
		Aliases:     []string{"j"},
		Description: "Gerencia versoes Java e JAVA_HOME",
		Usage:       "java <comando> [argumentos]",
		Children: []cli.Command{
			{Name: "current", Aliases: []string{"version"}, Description: "Mostra Java atual", Usage: "java current", Run: svc.current},
			{Name: "list", Aliases: []string{"ls"}, Description: "Lista instalacoes Java conhecidas", Usage: "java list", Run: svc.list},
			{Name: "add", Description: "Salva um alias para JAVA_HOME", Usage: "java add <alias> <java-home>", Run: svc.add},
			{Name: "use", Description: "Alterna Java no Duck ou persiste no usuario", Usage: "java use <alias|java-home> [--persist]", Run: svc.use},
			{Name: "path", Description: "Mostra comandos para configurar PATH", Usage: "java path <alias|java-home>", Run: svc.path},
			{Name: "home", Description: "Mostra JAVA_HOME configurado no Duck", Usage: "java home", Run: svc.home},
			{Name: "raw", Description: "Envia argumentos diretamente para java", Usage: "java raw <java args...>", Run: svc.raw},
		},
		Examples: []string{
			"java current",
			"java list",
			"java add 17 C:\\Program Files\\Java\\jdk-17",
			"java use 17 --persist",
			"java path 21",
		},
	}
}

func (s service) current(_ cli.Context, args []string) error {
	if len(args) > 0 {
		return cli.UsageError("current nao recebe argumentos")
	}

	if home := os.Getenv("JAVA_HOME"); home != "" {
		fmt.Println("JAVA_HOME:", home)
	}
	return s.runner.Run(s.bin, []string{"-version"}, runner.DefaultOptions())
}

func (s service) list(_ cli.Context, args []string) error {
	if len(args) > 0 {
		return cli.UsageError("list nao recebe argumentos")
	}

	homes := javaHomes()
	if len(homes) == 0 {
		fmt.Println("Nenhuma instalacao Java encontrada automaticamente.")
		fmt.Println("Use: duck java add <alias> <java-home>")
		return nil
	}

	for alias, home := range homes {
		fmt.Printf("%-16s %s\n", alias, home)
	}
	return nil
}

func (s service) add(_ cli.Context, args []string) error {
	if len(args) != 2 {
		return cli.UsageError("use: java add <alias> <java-home>")
	}

	home, err := filepath.Abs(args[1])
	if err != nil {
		return err
	}
	if err := validateJavaHome(home); err != nil {
		return err
	}
	if err := config.SetSetting("java.home."+args[0], home); err != nil {
		return err
	}
	fmt.Println("Java salvo:", args[0], "=>", home)
	return nil
}

func (s service) use(_ cli.Context, args []string) error {
	persist := false
	targets := make([]string, 0, len(args))
	for _, arg := range args {
		switch arg {
		case "--persist":
			persist = true
		default:
			targets = append(targets, arg)
		}
	}
	if len(targets) != 1 {
		return cli.UsageError("use: java use <alias|java-home> [--persist]")
	}

	home, err := resolveJavaHome(targets[0])
	if err != nil {
		return err
	}
	if err := validateJavaHome(home); err != nil {
		return err
	}
	if err := config.SetSetting("java.current", home); err != nil {
		return err
	}
	if persist {
		if err := persistJavaHome(home); err != nil {
			return err
		}
		fmt.Println("JAVA_HOME/PATH persistidos para o usuario.")
	} else {
		fmt.Println("JAVA_HOME salvo no Duck:", home)
		fmt.Println("Para persistir no sistema use: duck java use", targets[0], "--persist")
	}
	return nil
}

func (s service) path(_ cli.Context, args []string) error {
	if len(args) != 1 {
		return cli.UsageError("use: java path <alias|java-home>")
	}

	home, err := resolveJavaHome(args[0])
	if err != nil {
		return err
	}
	printPathInstructions(home)
	return nil
}

func (s service) home(_ cli.Context, args []string) error {
	if len(args) > 0 {
		return cli.UsageError("home nao recebe argumentos")
	}
	settings, err := config.LoadSettings()
	if err != nil {
		return err
	}
	if settings["java.current"] == "" {
		fmt.Println("Nenhum JAVA_HOME configurado no Duck.")
		return nil
	}
	fmt.Println(settings["java.current"])
	return nil
}

func (s service) raw(_ cli.Context, args []string) error {
	if len(args) == 0 {
		return cli.UsageError("informe argumentos para java")
	}
	return s.runner.Run(s.bin, args, runner.InteractiveOptions())
}

func javaHomes() map[string]string {
	homes := map[string]string{}
	if home := os.Getenv("JAVA_HOME"); home != "" {
		homes["env"] = home
	}

	settings, err := config.LoadSettings()
	if err == nil {
		for key, value := range settings {
			if strings.HasPrefix(key, "java.home.") {
				homes[strings.TrimPrefix(key, "java.home.")] = value
			}
		}
		if current := settings["java.current"]; current != "" {
			homes["current"] = current
		}
	}

	for _, root := range javaSearchRoots() {
		entries, err := os.ReadDir(root)
		if err != nil {
			continue
		}
		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}
			name := entry.Name()
			home := filepath.Join(root, name)
			if validateJavaHome(home) == nil {
				homes[name] = home
			}
		}
	}
	return sortedMap(homes)
}

func javaSearchRoots() []string {
	if runtime.GOOS == "windows" {
		return []string{
			`C:\Program Files\Java`,
			`C:\Program Files\Eclipse Adoptium`,
			`C:\Program Files\Microsoft`,
			`C:\Program Files\Amazon Corretto`,
			`C:\Program Files\Zulu`,
		}
	}

	home, _ := os.UserHomeDir()
	return []string{
		"/usr/lib/jvm",
		"/Library/Java/JavaVirtualMachines",
		filepath.Join(home, ".sdkman", "candidates", "java"),
		filepath.Join(home, ".jenv", "versions"),
	}
}

func resolveJavaHome(value string) (string, error) {
	if home, ok := javaHomes()[value]; ok {
		return home, nil
	}
	if filepath.IsAbs(value) || strings.Contains(value, string(os.PathSeparator)) {
		return filepath.Abs(value)
	}
	return "", fmt.Errorf("java nao encontrado: %s. Use 'duck java list' ou 'duck java add'", value)
}

func validateJavaHome(home string) error {
	bin := filepath.Join(home, "bin", executable("java"))
	if _, err := os.Stat(bin); err != nil {
		return fmt.Errorf("JAVA_HOME invalido, java nao encontrado em %s", bin)
	}
	return nil
}

func persistJavaHome(home string) error {
	if runtime.GOOS == "windows" {
		return persistWindowsJava(home)
	}
	return persistUnixJava(home)
}

func persistWindowsJava(home string) error {
	cmd := exec.Command("setx", "JAVA_HOME", home)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("falha ao persistir JAVA_HOME: %s", strings.TrimSpace(string(output)))
	}
	script := `
$javaBin = ` + powershellString(filepath.Join(home, "bin")) + `
$userPath = [Environment]::GetEnvironmentVariable('Path', 'User')
if ([string]::IsNullOrWhiteSpace($userPath)) {
  $parts = @()
} else {
  $parts = $userPath -split ';' | Where-Object { -not [string]::IsNullOrWhiteSpace($_) }
}
$exists = $false
foreach ($part in $parts) {
  if ($part.TrimEnd('\') -ieq $javaBin.TrimEnd('\')) {
    $exists = $true
  }
}
if (-not $exists) {
  if ([string]::IsNullOrWhiteSpace($userPath)) {
    $newPath = $javaBin
  } else {
    $newPath = "$javaBin;$userPath"
  }
  [Environment]::SetEnvironmentVariable('Path', $newPath, 'User')
}
`
	pathCmd := exec.Command("powershell", "-NoProfile", "-NonInteractive", "-ExecutionPolicy", "Bypass", "-Command", script)
	if output, err := pathCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("falha ao atualizar PATH: %s", strings.TrimSpace(string(output)))
	}
	return nil
}

func persistUnixJava(home string) error {
	profile := filepath.Join(os.Getenv("HOME"), ".profile")
	if shell := filepath.Base(os.Getenv("SHELL")); shell == "bash" {
		profile = filepath.Join(os.Getenv("HOME"), ".bashrc")
	} else if shell == "zsh" {
		profile = filepath.Join(os.Getenv("HOME"), ".zshrc")
	}

	line := "export JAVA_HOME=" + shellQuote(home) + "\nexport PATH=\"$JAVA_HOME/bin:$PATH\""
	file, err := os.OpenFile(profile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return err
	}
	defer file.Close()
	_, err = fmt.Fprintf(file, "\n# duck java\n%s\n", line)
	return err
}

func printPathInstructions(home string) {
	if runtime.GOOS == "windows" {
		fmt.Println("PowerShell temporario:")
		fmt.Printf("$env:JAVA_HOME = %q\n", home)
		fmt.Println("$env:Path = \"$env:JAVA_HOME\\bin;$env:Path\"")
		fmt.Println()
		fmt.Println("Persistente:")
		fmt.Printf("duck java use %q --persist\n", home)
		return
	}

	fmt.Println("Shell temporario:")
	fmt.Println("export JAVA_HOME=" + shellQuote(home))
	fmt.Println("export PATH=\"$JAVA_HOME/bin:$PATH\"")
	fmt.Println()
	fmt.Println("Persistente:")
	fmt.Printf("duck java use %q --persist\n", home)
}

func executable(name string) string {
	if runtime.GOOS == "windows" {
		return name + ".exe"
	}
	return name
}

func shellQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "'\"'\"'") + "'"
}

func powershellString(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "''") + "'"
}

func sortedMap(input map[string]string) map[string]string {
	keys := make([]string, 0, len(input))
	for key := range input {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	output := make(map[string]string, len(input))
	for _, key := range keys {
		output[key] = input[key]
	}
	return output
}
