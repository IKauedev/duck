package java

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"

	"github.com/IKauedev/duck/internal/certstore"
	"github.com/IKauedev/duck/internal/cli"
	"github.com/IKauedev/duck/internal/config"
	"github.com/IKauedev/duck/internal/runner"
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
			{Name: "cert", Description: "Importa certificado no truststore da JVM atual", Usage: "java cert <arquivo|url> [--alias nome] [--storepass senha] [--cacerts caminho] [--no-sudo]", Run: svc.cert},
			{Name: "raw", Description: "Envia argumentos diretamente para java", Usage: "java raw <java args...>", Run: svc.raw},
		},
		Examples: []string{
			"java current",
			"java list",
			"java add 17 C:\\Program Files\\Java\\jdk-17",
			"java use 17 --persist",
			"java path 21",
			"java cert C:\\certs\\empresa.crt --alias empresa",
			"java cert https://example.com/empresa.crt --alias empresa",
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

func (s service) cert(_ cli.Context, args []string) error {
	opts, err := parseCertArgs(args)
	if err != nil {
		return err
	}
	cert, err := certstore.Import(opts.source)
	if err != nil {
		return err
	}
	if opts.alias == "" {
		opts.alias = certificateAlias(cert.Name)
	}

	javaHome, err := currentJavaHome()
	if err != nil && opts.cacerts == "" {
		return err
	}
	keytool := "keytool"
	if javaHome != "" {
		keytool = filepath.Join(javaHome, "bin", executable("keytool"))
	}
	cacerts, err := resolveCACerts(javaHome, opts.cacerts)
	if err != nil {
		return err
	}

	fmt.Println("Certificado salvo em:", cert.Path)
	fmt.Println("Importando no truststore:", cacerts)
	keytoolArgs := []string{
		"-importcert",
		"-trustcacerts",
		"-noprompt",
		"-alias",
		opts.alias,
		"-file",
		cert.Path,
		"-keystore",
		cacerts,
		"-storepass",
		opts.storepass,
	}
	binary := keytool
	commandArgs := keytoolArgs
	options := runner.DefaultOptions()
	if runtime.GOOS != "windows" && !opts.noSudo && !canWriteFile(cacerts) {
		fmt.Println("Truststore sem permissao de escrita; usando sudo para executar keytool.")
		binary = "sudo"
		commandArgs = append([]string{keytool}, keytoolArgs...)
		options = runner.InteractiveOptions()
	}
	if err := s.runner.Run(binary, commandArgs, options); err != nil {
		return err
	}
	fmt.Println("Certificado importado na JVM com alias:", opts.alias)
	return nil
}

func (s service) raw(_ cli.Context, args []string) error {
	if len(args) == 0 {
		return cli.UsageError("informe argumentos para java")
	}
	return s.runner.Run(s.bin, args, runner.InteractiveOptions())
}

type certOptions struct {
	source    string
	alias     string
	storepass string
	cacerts   string
	noSudo    bool
}

func parseCertArgs(args []string) (certOptions, error) {
	opts := certOptions{storepass: "changeit"}
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--alias":
			if i+1 >= len(args) {
				return opts, cli.UsageError("--alias precisa de um valor")
			}
			opts.alias = args[i+1]
			i++
		case "--storepass":
			if i+1 >= len(args) {
				return opts, cli.UsageError("--storepass precisa de um valor")
			}
			opts.storepass = args[i+1]
			i++
		case "--cacerts":
			if i+1 >= len(args) {
				return opts, cli.UsageError("--cacerts precisa de um valor")
			}
			opts.cacerts = args[i+1]
			i++
		case "--no-sudo":
			opts.noSudo = true
		default:
			if opts.source != "" {
				return opts, cli.UsageError("use: java cert <arquivo|url> [--alias nome] [--storepass senha] [--cacerts caminho] [--no-sudo]")
			}
			opts.source = args[i]
		}
	}
	if opts.source == "" {
		return opts, cli.UsageError("use: java cert <arquivo|url> [--alias nome] [--storepass senha] [--cacerts caminho] [--no-sudo]")
	}
	if opts.alias != "" && strings.ContainsAny(opts.alias, " \t\r\n") {
		return opts, cli.UsageError("--alias nao pode conter espacos")
	}
	return opts, nil
}

func currentJavaHome() (string, error) {
	settings, err := config.LoadSettings()
	if err == nil && settings["java.current"] != "" {
		return settings["java.current"], nil
	}
	if home := os.Getenv("JAVA_HOME"); home != "" {
		return home, nil
	}
	return "", fmt.Errorf("JAVA_HOME nao encontrado. Use 'duck java use <alias|java-home>' ou informe --cacerts")
}

func resolveCACerts(javaHome string, override string) (string, error) {
	if override != "" {
		path, err := filepath.Abs(override)
		if err != nil {
			return "", err
		}
		if _, err := os.Stat(path); err != nil {
			return "", fmt.Errorf("truststore cacerts nao encontrado em %s", path)
		}
		return path, nil
	}
	candidates := []string{
		filepath.Join(javaHome, "lib", "security", "cacerts"),
		filepath.Join(javaHome, "jre", "lib", "security", "cacerts"),
	}
	for _, candidate := range candidates {
		if _, err := os.Stat(candidate); err == nil {
			return candidate, nil
		}
	}
	return "", fmt.Errorf("truststore cacerts nao encontrado em %s", strings.Join(candidates, " ou "))
}

func canWriteFile(path string) bool {
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_APPEND, 0)
	if err != nil {
		return false
	}
	_ = file.Close()
	return true
}

func certificateAlias(name string) string {
	alias := strings.TrimSuffix(name, filepath.Ext(name))
	alias = strings.TrimSpace(alias)
	alias = strings.NewReplacer(" ", "-", "_", "-", ".", "-", ":", "-").Replace(alias)
	if alias == "" {
		return "duck-certificate"
	}
	return alias
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
