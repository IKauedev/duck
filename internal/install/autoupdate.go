package install

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	"github.com/IKauedev/duck/internal/cli"
)

const autoUpdateTaskName = "DuckAutoUpdate"

func setupAutoupdate(_ cli.Context, args []string) error {
	if len(args) == 0 {
		args = []string{"status"}
	}

	switch args[0] {
	case "enable", "on":
		return enableAutoupdate(args[1:])
	case "disable", "off":
		return disableAutoupdate()
	case "status":
		return autoupdateStatus()
	default:
		return cli.UsageError("use: setup autoupdate <enable|disable|status> [--time HH:MM]")
	}
}

func enableAutoupdate(args []string) error {
	at := "09:00"
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--time":
			if i+1 >= len(args) {
				return cli.UsageError("--time precisa de HH:MM")
			}
			at = args[i+1]
			i++
		default:
			return cli.UsageError("opcao invalida: " + args[i])
		}
	}

	duckBin, err := currentDuckExecutable()
	if err != nil {
		return err
	}

	switch runtime.GOOS {
	case "windows":
		return enableWindowsAutoupdate(duckBin, at)
	case "linux", "darwin":
		return enableUnixAutoupdate(duckBin, at)
	default:
		return fmt.Errorf("autoupdate nao suportado em %s", runtime.GOOS)
	}
}

func disableAutoupdate() error {
	switch runtime.GOOS {
	case "windows":
		cmd := exec.Command("schtasks", "/Delete", "/TN", autoUpdateTaskName, "/F")
		output, err := cmd.CombinedOutput()
		if err != nil {
			text := strings.TrimSpace(string(output))
			if strings.Contains(text, "ERROR: The system cannot find") {
				fmt.Println("Autoupdate ja estava desativado.")
				return nil
			}
			return fmt.Errorf("%s: %s", err, text)
		}
		fmt.Println("Autoupdate desativado (tarefa agendada removida).")
		return nil
	case "linux", "darwin":
		return disableUnixAutoupdate()
	default:
		return fmt.Errorf("autoupdate nao suportado em %s", runtime.GOOS)
	}
}

func autoupdateStatus() error {
	switch runtime.GOOS {
	case "windows":
		cmd := exec.Command("schtasks", "/Query", "/TN", autoUpdateTaskName, "/FO", "LIST")
		output, err := cmd.CombinedOutput()
		if err != nil {
			fmt.Println("Autoupdate: desativado")
			fmt.Println("Ative com: duck setup autoupdate enable")
			return nil
		}
		fmt.Println("Autoupdate: ativo (Windows Task Scheduler)")
		fmt.Print(string(output))
		return nil
	case "linux", "darwin":
		cmd := exec.Command("crontab", "-l")
		output, err := cmd.Output()
		if err != nil || !strings.Contains(string(output), entryMarker) {
			fmt.Println("Autoupdate: desativado")
			fmt.Println("Ative com: duck setup autoupdate enable")
			return nil
		}
		fmt.Println("Autoupdate: ativo (crontab)")
		for _, line := range strings.Split(string(output), "\n") {
			if strings.Contains(line, entryMarker) || strings.Contains(line, "duck update") {
				fmt.Println(" ", strings.TrimSpace(line))
			}
		}
		return nil
	default:
		return fmt.Errorf("autoupdate nao suportado em %s", runtime.GOOS)
	}
}

func enableWindowsAutoupdate(duckBin, at string) error {
	parts := strings.Split(at, ":")
	if len(parts) != 2 {
		return cli.UsageError("--time precisa usar HH:MM")
	}
	hour, minute := parts[0], parts[1]

	cmd := exec.Command(
		"schtasks",
		"/Create",
		"/TN", autoUpdateTaskName,
		"/TR", fmt.Sprintf(`"%s" update --yes`, duckBin),
		"/SC", "DAILY",
		"/ST", fmt.Sprintf("%s:%s", hour, minute),
		"/F",
	)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("nao foi possivel criar tarefa agendada: %s: %s", err, strings.TrimSpace(string(output)))
	}

	fmt.Println("Autoupdate ativado via schtasks (sem PowerShell).")
	fmt.Printf("Horario: %s:%s\n", hour, minute)
	fmt.Println("Comando:", duckBin, "update --yes")
	fmt.Println("Desative com: duck setup autoupdate disable")
	return nil
}

const entryMarker = "# duck-autoupdate"

func enableUnixAutoupdate(duckBin, at string) error {
	parts := strings.Split(at, ":")
	if len(parts) != 2 {
		return cli.UsageError("--time precisa usar HH:MM")
	}
	entry := fmt.Sprintf("%s %s * * * %s update --yes >/dev/null 2>&1", parts[1], parts[0], shellQuote(duckBin))

	current, _ := exec.Command("crontab", "-l").Output()
	lines := strings.Split(strings.TrimRight(string(current), "\n"), "\n")
	filtered := make([]string, 0, len(lines)+2)
	for _, line := range lines {
		if strings.Contains(line, entryMarker) || strings.Contains(line, "duck update --yes") {
			continue
		}
		if strings.TrimSpace(line) != "" {
			filtered = append(filtered, line)
		}
	}
	filtered = append(filtered, entryMarker, entry)

	cmd := exec.Command("crontab", "-")
	cmd.Stdin = strings.NewReader(strings.Join(filtered, "\n") + "\n")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("nao foi possivel atualizar crontab: %s: %s", err, strings.TrimSpace(string(output)))
	}

	fmt.Println("Autoupdate ativado via crontab.")
	fmt.Println("Entrada:", entry)
	fmt.Println("Desative com: duck setup autoupdate disable")
	return nil
}

func disableUnixAutoupdate() error {
	current, err := exec.Command("crontab", "-l").Output()
	if err != nil {
		fmt.Println("Autoupdate ja estava desativado.")
		return nil
	}
	lines := strings.Split(strings.TrimRight(string(current), "\n"), "\n")
	filtered := make([]string, 0, len(lines))
	removed := false
	for _, line := range lines {
		if strings.Contains(line, entryMarker) || strings.Contains(line, "duck update --yes") {
			removed = true
			continue
		}
		if strings.TrimSpace(line) != "" {
			filtered = append(filtered, line)
		}
	}
	if !removed {
		fmt.Println("Autoupdate ja estava desativado.")
		return nil
	}

	cmd := exec.Command("crontab", "-")
	if len(filtered) == 0 {
		cmd.Stdin = strings.NewReader("")
	} else {
		cmd.Stdin = strings.NewReader(strings.Join(filtered, "\n") + "\n")
	}
	if err := cmd.Run(); err != nil {
		return err
	}
	fmt.Println("Autoupdate desativado (crontab atualizado).")
	return nil
}

func shellQuote(value string) string {
	if strings.ContainsAny(value, " \t") {
		return strconv.Quote(value)
	}
	return value
}

func currentDuckExecutable() (string, error) {
	path, err := os.Executable()
	if err != nil {
		return "", err
	}
	path, err = filepath.EvalSymlinks(path)
	if err != nil {
		return "", err
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	return abs, nil
}
