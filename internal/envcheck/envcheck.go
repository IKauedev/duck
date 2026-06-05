package envcheck

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"duck/internal/cli"
	"duck/internal/config"
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
