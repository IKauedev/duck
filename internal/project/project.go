package project

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"duck/internal/cli"
)

func Command() cli.Command {
	return cli.Command{
		Name:        "project",
		Description: "Detecta stack do projeto atual",
		Usage:       "project <comando>",
		Children: []cli.Command{
			{Name: "detect", Description: "Detecta stack do projeto atual", Usage: "project detect", Run: detect},
			{Name: "doctor", Description: "Valida o projeto atual conforme stack detectada", Usage: "project doctor", Run: doctor},
			{Name: "up", Description: "Sobe ambiente do projeto detectado", Usage: "project up", Run: up},
			{Name: "down", Description: "Derruba ambiente do projeto detectado", Usage: "project down", Run: down},
		},
	}
}

func detect(_ cli.Context, args []string) error {
	if len(args) > 0 {
		return cli.UsageError("project detect nao recebe argumentos")
	}

	checks := []struct {
		file  string
		stack string
	}{
		{"go.mod", "Go"},
		{"package.json", "Node.js"},
		{"pnpm-lock.yaml", "PNPM"},
		{"yarn.lock", "Yarn"},
		{"requirements.txt", "Python"},
		{"pyproject.toml", "Python"},
		{"pom.xml", "Java/Maven"},
		{"build.gradle", "Java/Gradle"},
		{"Dockerfile", "Docker"},
		{"compose.yaml", "Docker Compose"},
		{"compose.yml", "Docker Compose"},
		{"docker-compose.yml", "Docker Compose"},
		{"kustomization.yaml", "Kubernetes/Kustomize"},
		{"Chart.yaml", "Helm"},
		{".terraform", "Terraform"},
	}

	found := false
	for _, check := range checks {
		if exists(check.file) {
			fmt.Printf("%-20s %s\n", check.stack, check.file)
			found = true
		}
	}
	if !found {
		fmt.Println("Nenhuma stack conhecida detectada.")
	}
	return nil
}

func exists(path string) bool {
	_, err := os.Stat(filepath.Clean(path))
	return err == nil
}

func doctor(_ cli.Context, args []string) error {
	if len(args) > 0 {
		return cli.UsageError("project doctor nao recebe argumentos")
	}
	steps := [][]string{}
	if exists("go.mod") {
		steps = append(steps, []string{"go", "test", "./..."})
	}
	if exists("package.json") {
		steps = append(steps, []string{"npm", "test"})
	}
	if exists("requirements.txt") || exists("pyproject.toml") {
		steps = append(steps, []string{"python", "--version"})
	}
	if exists("Dockerfile") {
		steps = append(steps, []string{"docker", "build", "-t", "duck-project-check", "."})
	}
	if len(steps) == 0 {
		fmt.Println("Nenhuma validacao conhecida para este projeto.")
		return nil
	}
	for _, step := range steps {
		fmt.Println("==>", step)
		if err := run(step...); err != nil {
			return err
		}
	}
	return nil
}

func up(_ cli.Context, args []string) error {
	if len(args) > 0 {
		return cli.UsageError("project up nao recebe argumentos")
	}
	if hasCompose() {
		return run("docker", "compose", "up", "-d")
	}
	if exists("kustomization.yaml") {
		return run("kubectl", "apply", "-k", ".")
	}
	if exists("Chart.yaml") {
		dir, _ := os.Getwd()
		return run("helm", "upgrade", "--install", filepath.Base(dir), ".")
	}
	return fmt.Errorf("nenhum ambiente conhecido para subir")
}

func down(_ cli.Context, args []string) error {
	if len(args) > 0 {
		return cli.UsageError("project down nao recebe argumentos")
	}
	if hasCompose() {
		return run("docker", "compose", "down")
	}
	if exists("kustomization.yaml") {
		return run("kubectl", "delete", "-k", ".")
	}
	if exists("Chart.yaml") {
		dir, _ := os.Getwd()
		return run("helm", "uninstall", filepath.Base(dir))
	}
	return fmt.Errorf("nenhum ambiente conhecido para derrubar")
}

func hasCompose() bool {
	return exists("compose.yaml") || exists("compose.yml") || exists("docker-compose.yml") || exists("docker-compose.yaml")
}

func run(args ...string) error {
	cmd := exec.Command(args[0], args[1:]...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
