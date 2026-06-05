package envcheck

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/IKauedev/duck/internal/cli"
	"github.com/IKauedev/duck/internal/config"
)

func Command() cli.Command {
	return cli.Command{
		Name:        "env",
		Description: "Valida ambiente de desenvolvimento",
		Usage:       "env <comando>",
		Children: []cli.Command{
			{Name: "doctor", Description: "Valida PATH, JAVA_HOME e NODE_HOME", Usage: "env doctor", Run: doctor},
			{Name: "export", Description: "Exporta configuracao do Duck", Usage: "env export <arquivo>", Run: exportSettings},
			{Name: "import", Description: "Importa configuracao do Duck", Usage: "env import <arquivo>", Run: importSettings},
			{Name: "example", Description: "Gera .env.example removendo valores sensiveis", Usage: "env example [entrada] [saida]", Run: exampleEnv},
		},
	}
}

func doctor(_ cli.Context, args []string) error {
	if len(args) > 0 {
		return cli.UsageError("env doctor nao recebe argumentos")
	}

	checkPath()
	checkHome("JAVA_HOME", "java")
	checkHome("NODE_HOME", "node")
	checkBinary("docker")
	checkBinary("kubectl")
	checkBinary("aws")
	checkBinary("go")
	checkBinary("python")
	return nil
}

func checkPath() {
	if os.Getenv("PATH") == "" && os.Getenv("Path") == "" {
		fmt.Println("PATH         indisponivel")
		return
	}
	fmt.Println("PATH         ok")
}

func checkHome(name, binary string) {
	home := os.Getenv(name)
	if home == "" {
		fmt.Printf("%-12s nao definido\n", name)
		return
	}
	path := filepath.Join(home, "bin", executable(binary))
	if name == "NODE_HOME" {
		if _, err := os.Stat(filepath.Join(home, executable(binary))); err == nil {
			fmt.Printf("%-12s ok: %s\n", name, home)
			return
		}
	}
	if _, err := os.Stat(path); err != nil {
		fmt.Printf("%-12s invalido: %s\n", name, home)
		return
	}
	fmt.Printf("%-12s ok: %s\n", name, home)
}

func checkBinary(binary string) {
	if path, err := exec.LookPath(binary); err == nil {
		fmt.Printf("%-12s ok: %s\n", binary, path)
		return
	}
	fmt.Printf("%-12s nao encontrado no PATH\n", binary)
}

func executable(name string) string {
	if filepath.Separator == '\\' {
		return name + ".exe"
	}
	return name
}

func exportSettings(_ cli.Context, args []string) error {
	if len(args) != 1 {
		return cli.UsageError("use: env export <arquivo>")
	}
	settings, err := config.LoadSettings()
	if err != nil {
		return err
	}
	content, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Clean(args[0]), append(content, '\n'), 0644); err != nil {
		return err
	}
	fmt.Println("Configuracao exportada para:", args[0])
	return nil
}

func importSettings(_ cli.Context, args []string) error {
	if len(args) != 1 {
		return cli.UsageError("use: env import <arquivo>")
	}
	content, err := os.ReadFile(filepath.Clean(args[0]))
	if err != nil {
		return err
	}
	var settings config.Settings
	if err := json.Unmarshal(content, &settings); err != nil {
		return err
	}
	if settings == nil {
		settings = config.Settings{}
	}
	if err := config.SaveSettings(settings); err != nil {
		return err
	}
	fmt.Println("Configuracao importada.")
	return nil
}

func exampleEnv(_ cli.Context, args []string) error {
	input := ".env"
	output := ".env.example"
	switch len(args) {
	case 0:
	case 1:
		input = args[0]
	case 2:
		input = args[0]
		output = args[1]
	default:
		return cli.UsageError("use: env example [entrada] [saida]")
	}
	file, err := os.Open(filepath.Clean(input))
	if err != nil {
		return err
	}
	defer file.Close()

	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			lines = append(lines, line)
			continue
		}
		prefix := ""
		if strings.HasPrefix(trimmed, "export ") {
			prefix = "export "
			trimmed = strings.TrimSpace(strings.TrimPrefix(trimmed, "export "))
		}
		key, _, ok := strings.Cut(trimmed, "=")
		if !ok {
			lines = append(lines, line)
			continue
		}
		key = strings.TrimSpace(key)
		lines = append(lines, prefix+key+"="+placeholderForEnv(key))
	}
	if err := scanner.Err(); err != nil {
		return err
	}
	content := strings.Join(lines, "\n") + "\n"
	if err := os.WriteFile(filepath.Clean(output), []byte(content), 0644); err != nil {
		return err
	}
	fmt.Println("Gerado:", output)
	return nil
}

func placeholderForEnv(key string) string {
	upper := strings.ToUpper(key)
	if strings.Contains(upper, "PORT") {
		return "8080"
	}
	if strings.Contains(upper, "HOST") {
		return "localhost"
	}
	if strings.Contains(upper, "URL") {
		return "http://localhost:8080"
	}
	return ""
}
