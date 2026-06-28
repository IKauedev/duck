package terminal

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"github.com/IKauedev/duck/internal/cli"
	"github.com/IKauedev/duck/internal/config"
	"github.com/IKauedev/duck/internal/history"
	"github.com/IKauedev/duck/internal/runner"
)

const appName = "duck"

type shellBackend struct {
	kind    string
	wslArgs []string
}

func Command(cfg config.Config, run runner.Runner, commands func() []cli.Command) cli.Command {
	return cli.Command{
		Name:        "terminal",
		Aliases:     []string{"console", "repl", "sh"},
		Description: "Abre um terminal interativo do Duck com shell nativa integrada",
		Usage:       "terminal [duck|cmd|powershell|bash|wsl] [args...]",
		Run: func(_ cli.Context, args []string) error {
			return runTerminal(cfg, run, commands, args)
		},
		Examples: []string{
			"terminal",
			"terminal cmd",
			"terminal powershell",
			"terminal bash",
			"terminal wsl Ubuntu-22.04",
		},
	}
}

func runTerminal(cfg config.Config, run runner.Runner, commands func() []cli.Command, args []string) error {
	if len(args) == 0 {
		return duckREPL(cfg, run, commands, shellBackend{})
	}

	shell := strings.ToLower(args[0])
	switch shell {
	case "duck", "repl":
		return duckREPL(cfg, run, commands, shellBackend{})
	case "cmd", "powershell", "ps", "pwsh", "bash", "wsl":
		return duckREPL(cfg, run, commands, shellBackend{kind: shell, wslArgs: args[1:]})
	default:
		return cli.UsageError("shell invalida: " + args[0] + ". Use duck, cmd, powershell, bash ou wsl")
	}
}

func duckREPL(cfg config.Config, run runner.Runner, commands func() []cli.Command, backend shellBackend) error {
	fmt.Println("Duck terminal interativo.")
	printShellMode(backend)
	printTerminalHelp()
	scanner := bufio.NewScanner(os.Stdin)
	for {
		fmt.Print(prompt(backend))
		if !scanner.Scan() {
			fmt.Println()
			return scanner.Err()
		}

		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		lower := strings.ToLower(line)
		switch lower {
		case "exit", "quit":
			return nil
		case "help", "?":
			printTerminalHelp()
			continue
		case "pwd":
			wd, err := os.Getwd()
			if err != nil {
				fmt.Println("Erro:", err)
			} else {
				fmt.Println(wd)
			}
			continue
		case "clear", "cls":
			fmt.Print("\033[H\033[2J")
			continue
		}

		if strings.HasPrefix(line, "cd ") {
			dir := strings.TrimSpace(strings.TrimPrefix(line, "cd "))
			if err := os.Chdir(dir); err != nil {
				fmt.Println("Erro:", err)
			}
			continue
		}
		if strings.HasPrefix(line, "!") {
			if err := historyRun(strings.TrimPrefix(line, "!"), cfg, commands); err != nil {
				fmt.Println("Erro:", err)
			}
			continue
		}
		if strings.HasPrefix(line, "$") || strings.HasPrefix(line, "@") {
			shellLine := strings.TrimSpace(line[1:])
			if shellLine == "" {
				continue
			}
			if err := runNativeShell(cfg, shellLine, backend); err != nil {
				fmt.Println("Erro:", err)
			}
			continue
		}
		if strings.HasPrefix(lower, "shell ") {
			parts, err := ParseLine(line)
			if err != nil {
				fmt.Println("Erro:", err)
				continue
			}
			if len(parts) < 2 {
				fmt.Println("Uso: shell <cmd|powershell|bash|wsl> [distro]")
				continue
			}
			if err := launchPureShell(cfg, strings.ToLower(parts[1]), parts[2:]); err != nil {
				fmt.Println("Erro:", err)
			}
			fmt.Println("Voltou ao Duck terminal.")
			continue
		}

		parsed, err := ParseLine(line)
		if err != nil {
			fmt.Println("Erro:", err)
			continue
		}
		if len(parsed) == 0 {
			continue
		}
		if parsed[0] == appName {
			parsed = parsed[1:]
		}
		if len(parsed) == 0 {
			continue
		}
		if parsed[0] == "terminal" || parsed[0] == "console" || parsed[0] == "repl" || parsed[0] == "sh" {
			fmt.Println("Voce ja esta no terminal Duck.")
			continue
		}

		parsed = expandAlias(parsed)
		cmds := commands()
		if IsDuckCommand(parsed, cmds) {
			_ = history.Record(parsed)
			code := cli.Run(appName, cmds, parsed)
			if code != 0 {
				fmt.Println("Comando finalizado com erro:", code)
			}
			continue
		}

		if err := runNativeShell(cfg, line, backend); err != nil {
			fmt.Println("Erro:", err)
		}
	}
}

func launchPureShell(cfg config.Config, shell string, args []string) error {
	printShellBanner(shell)
	switch shell {
	case "cmd":
		if runtime.GOOS != "windows" {
			return fmt.Errorf("cmd disponivel apenas no Windows")
		}
		cmd := exec.Command("cmd.exe", "/K", "prompt duck-cmd$G")
		return attachIO(cmd)
	case "powershell", "ps", "pwsh":
		binary, err := findPowerShell()
		if err != nil {
			return err
		}
		cmd := exec.Command(binary, "-NoLogo")
		return attachIO(cmd)
	case "bash":
		binary, err := exec.LookPath("bash")
		if err != nil {
			return fmt.Errorf("bash nao encontrado no PATH")
		}
		cmd := exec.Command(binary, "-l")
		return attachIO(cmd)
	case "wsl":
		if runtime.GOOS != "windows" {
			return fmt.Errorf("wsl disponivel apenas no Windows")
		}
		wslArgs := make([]string, 0, 4)
		if len(args) > 0 {
			wslArgs = append(wslArgs, "-d", args[0])
		}
		cmd := exec.Command(cfg.WSLBin, wslArgs...)
		return attachIO(cmd)
	default:
		return fmt.Errorf("shell invalida: %s", shell)
	}
}

func printShellMode(backend shellBackend) {
	switch backend.kind {
	case "cmd":
		fmt.Println("Shell nativa: CMD (comandos Duck e CMD no mesmo prompt)")
	case "bash":
		fmt.Println("Shell nativa: Bash (comandos Duck e shell no mesmo prompt)")
	case "powershell", "ps", "pwsh":
		fmt.Println("Shell nativa: PowerShell (comandos Duck e PowerShell no mesmo prompt)")
	case "wsl":
		distro := "padrao"
		if len(backend.wslArgs) > 0 {
			distro = backend.wslArgs[0]
		}
		fmt.Printf("Shell nativa: WSL (%s) (comandos Duck e Ubuntu no mesmo prompt)\n", distro)
	default:
		if runtime.GOOS == "windows" {
			fmt.Println("Shell nativa: CMD (comandos Duck e CMD no mesmo prompt)")
		} else {
			fmt.Println("Shell nativa: Bash (comandos Duck e shell no mesmo prompt)")
		}
	}
}

func printShellBanner(shell string) {
	fmt.Println()
	fmt.Println("Abrindo shell nativa:", shell)
	fmt.Println("Dica: digite 'exit' para voltar ao Duck terminal.")
	fmt.Println()
}

func printTerminalHelp() {
	fmt.Println()
	fmt.Println("Comandos do terminal Duck:")
	fmt.Println("  help              mostra esta ajuda")
	fmt.Println("  exit / quit       sai do terminal")
	fmt.Println("  pwd / cd <pasta>  navega entre pastas")
	fmt.Println("  clear / cls       limpa a tela")
	fmt.Println("  !N                repete comando do historico duck")
	fmt.Println("  $ <comando>       forca execucao na shell nativa")
	fmt.Println("  shell wsl         abre Ubuntu/WSL interativo e volta depois")
	fmt.Println("  shell cmd         abre CMD interativo (Windows)")
	fmt.Println("  shell bash        abre Bash interativo")
	fmt.Println("  docker ps         comandos Duck funcionam sem prefixo")
	fmt.Println("  dir / ls          comandos nativos funcionam sem prefixo")
	fmt.Println()
	fmt.Println("Abrir com shell nativa preferida:")
	fmt.Println("  duck terminal cmd")
	fmt.Println("  duck terminal powershell")
	fmt.Println("  duck terminal bash")
	fmt.Println("  duck terminal wsl Ubuntu-22.04")
	fmt.Println()
}

func prompt(backend shellBackend) string {
	settings, _ := config.LoadSettings()
	awsProfile := settings["aws.profile"]
	kubeNamespace := settings["kube.namespace"]
	parts := make([]string, 0, 2)
	if awsProfile != "" {
		parts = append(parts, "aws:"+awsProfile)
	}
	if kubeNamespace != "" {
		parts = append(parts, "ns:"+kubeNamespace)
	}

	shellTag := shellPromptTag(backend)
	if len(parts) == 0 {
		return "duck" + shellTag + "> "
	}
	return "duck[" + strings.Join(parts, " ") + "]" + shellTag + "> "
}

func shellPromptTag(backend shellBackend) string {
	switch backend.kind {
	case "cmd":
		return "-cmd"
	case "bash":
		return "-bash"
	case "powershell", "ps", "pwsh":
		return "-ps"
	case "wsl":
		return "-wsl"
	default:
		if runtime.GOOS == "windows" {
			return "-cmd"
		}
		return "-bash"
	}
}

func runNativeShell(cfg config.Config, line string, backend shellBackend) error {
	kind := backend.kind
	if kind == "" {
		if runtime.GOOS == "windows" {
			kind = "cmd"
		} else {
			kind = "bash"
		}
	}

	switch kind {
	case "wsl":
		wslArgs := make([]string, 0, 6)
		if len(backend.wslArgs) > 0 {
			wslArgs = append(wslArgs, "-d", backend.wslArgs[0])
		}
		wslArgs = append(wslArgs, "--", "sh", "-c", line)
		cmd := exec.Command(cfg.WSLBin, wslArgs...)
		return attachIO(cmd)
	case "bash":
		binary, err := exec.LookPath("bash")
		if err != nil {
			return fmt.Errorf("bash nao encontrado no PATH")
		}
		cmd := exec.Command(binary, "-lc", line)
		return attachIO(cmd)
	case "powershell", "ps", "pwsh":
		binary, err := findPowerShell()
		if err != nil {
			return err
		}
		cmd := exec.Command(binary, "-NoLogo", "-NoProfile", "-Command", line)
		return attachIO(cmd)
	case "cmd":
		cmd := exec.Command("cmd.exe", "/C", line)
		return attachIO(cmd)
	default:
		cmd := exec.Command("sh", "-c", line)
		return attachIO(cmd)
	}
}

func attachIO(cmd *exec.Cmd) error {
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func findPowerShell() (string, error) {
	for _, candidate := range []string{"pwsh", "powershell", "powershell.exe", "pwsh.exe"} {
		if path, err := exec.LookPath(candidate); err == nil {
			return path, nil
		}
	}
	return "", fmt.Errorf("powershell/pwsh nao encontrado no PATH")
}

func historyRun(index string, cfg config.Config, commands func() []cli.Command) error {
	entries, err := history.List()
	if err != nil {
		return err
	}
	if len(entries) == 0 {
		return fmt.Errorf("historico vazio")
	}
	var target int
	if _, err := fmt.Sscanf(index, "%d", &target); err != nil || target < 1 || target > len(entries) {
		return fmt.Errorf("numero invalido de historico: %s", index)
	}
	args, err := ParseLine(entries[target-1].Command)
	if err != nil {
		return err
	}
	code := cli.Run(appName, commands(), args)
	if code != 0 {
		return fmt.Errorf("comando finalizado com erro: %d", code)
	}
	return nil
}

func expandAlias(args []string) []string {
	if len(args) == 0 {
		return args
	}
	settings, err := config.LoadSettings()
	if err != nil {
		return args
	}
	alias := settings["alias."+args[0]]
	if alias == "" {
		return args
	}
	parsed, err := ParseLine(alias)
	if err != nil {
		return args
	}
	return append(parsed, args[1:]...)
}

// IsDuckCommand reports whether args match a Duck command path.
func IsDuckCommand(args []string, commands []cli.Command) bool {
	if len(args) == 0 {
		return false
	}
	switch args[0] {
	case "help", "-h", "--help":
		return true
	case "--version", "-V":
		return len(args) == 1
	}
	return matchCommandPath(commands, args)
}

func matchCommandPath(commands []cli.Command, args []string) bool {
	if len(args) == 0 {
		return false
	}
	selected, ok := cli.Find(commands, args[0])
	if !ok {
		return false
	}
	if len(selected.Children) == 0 {
		return true
	}
	if len(args) == 1 {
		return true
	}
	return matchCommandPath(selected.Children, args[1:])
}

// ParseLine divide uma linha de terminal respeitando aspas.
func ParseLine(line string) ([]string, error) {
	var args []string
	var current strings.Builder
	var quote rune
	escaped := false

	for _, char := range line {
		if escaped {
			current.WriteRune(char)
			escaped = false
			continue
		}
		if char == '\\' {
			escaped = true
			continue
		}
		if quote != 0 {
			if char == quote {
				quote = 0
				continue
			}
			current.WriteRune(char)
			continue
		}
		switch char {
		case '\'', '"':
			quote = char
		case ' ', '\t':
			if current.Len() > 0 {
				args = append(args, current.String())
				current.Reset()
			}
		default:
			current.WriteRune(char)
		}
	}
	if escaped {
		current.WriteRune('\\')
	}
	if quote != 0 {
		return nil, fmt.Errorf("aspas nao fechadas")
	}
	if current.Len() > 0 {
		args = append(args, current.String())
	}
	return args, nil
}
