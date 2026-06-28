package wsl

import (
	"fmt"
	"runtime"
	"strings"
	"unicode/utf16"

	"github.com/IKauedev/duck/internal/cli"
	"github.com/IKauedev/duck/internal/config"
	"github.com/IKauedev/duck/internal/runner"
)

type service struct {
	bin    string
	runner runner.Runner
}

func Command(cfg config.Config, run runner.Runner) cli.Command {
	svc := service{bin: cfg.WSLBin, runner: run}
	return cli.Command{
		Name:        "wsl",
		Description: "Checa a instalacao do WSL no Windows",
		Usage:       "wsl <comando>",
		Children: []cli.Command{
			{Name: "status", Aliases: []string{"check"}, Description: "Mostra status do WSL", Usage: "wsl status", Run: svc.status},
			{Name: "list", Aliases: []string{"ls"}, Description: "Lista distribuicoes WSL", Usage: "wsl list", Run: svc.list},
			{Name: "start", Description: "Inicia a distribuicao WSL padrao ou informada", Usage: "wsl start [distro]", Run: svc.start},
			{Name: "raw", Description: "Envia argumentos diretamente para wsl", Usage: "wsl raw <args...>", Run: svc.raw},
		},
		Examples: []string{
			"wsl status",
			"wsl list",
			"wsl start Ubuntu-22.04",
			"wsl raw --version",
		},
	}
}

func Check(cfg config.Config, run runner.Runner) {
	if runtime.GOOS != "windows" {
		fmt.Println("WSL          nao necessario neste sistema")
		return
	}

	output, err := run.Output(cfg.WSLBin, []string{"--status"})
	output = normalizeOutput(output)
	if err != nil {
		if output == "" {
			output = err.Error()
		}
		fmt.Printf("WSL          indisponivel: %s\n", output)
		return
	}

	fmt.Printf("WSL          ok: %s", output)
}

func (s service) status(_ cli.Context, args []string) error {
	if len(args) > 0 {
		return cli.UsageError("status nao recebe argumentos")
	}
	if runtime.GOOS != "windows" {
		fmt.Println("WSL e necessario apenas no Windows. Neste sistema, execute os comandos Duck diretamente.")
		return nil
	}

	if err := s.runText([]string{"--status"}); err != nil {
		return fmt.Errorf("wsl nao esta disponivel ou nao esta configurado: %w", err)
	}

	fmt.Println()
	fmt.Println("Distribuicoes:")
	return s.runText([]string{"-l", "-v"})
}

func (s service) list(_ cli.Context, args []string) error {
	if len(args) > 0 {
		return cli.UsageError("list nao recebe argumentos")
	}
	if runtime.GOOS != "windows" {
		fmt.Println("WSL e necessario apenas no Windows.")
		return nil
	}
	return s.runText([]string{"-l", "-v"})
}

func (s service) start(_ cli.Context, args []string) error {
	if len(args) > 1 {
		return cli.UsageError("informe no maximo uma distribuicao")
	}
	if runtime.GOOS != "windows" {
		fmt.Println("WSL e necessario apenas no Windows.")
		return nil
	}

	wslArgs := []string{"--exec", "true"}
	if len(args) == 1 {
		wslArgs = []string{"-d", args[0], "--exec", "true"}
	}

	if err := s.runText(wslArgs); err != nil {
		return fmt.Errorf("nao foi possivel iniciar a distribuicao WSL: %w", err)
	}

	fmt.Println("Distribuicao WSL iniciada.")
	fmt.Println()
	fmt.Println("Distribuicoes:")
	return s.runText([]string{"-l", "-v"})
}

func (s service) raw(_ cli.Context, args []string) error {
	if len(args) == 0 {
		return cli.UsageError("informe argumentos para wsl")
	}
	return s.runText(args)
}

func (s service) runText(args []string) error {
	output, err := s.runner.Output(s.bin, args)
	output = normalizeOutput(output)
	if output != "" {
		fmt.Print(output)
		if !strings.HasSuffix(output, "\n") {
			fmt.Println()
		}
	}
	return err
}

func normalizeOutput(output string) string {
	bytes := []byte(output)
	if len(bytes) < 2 || !looksUTF16LE(bytes) {
		return output
	}

	u16 := make([]uint16, 0, len(bytes)/2)
	for i := 0; i+1 < len(bytes); i += 2 {
		value := uint16(bytes[i]) | uint16(bytes[i+1])<<8
		if value == 0xfeff {
			continue
		}
		u16 = append(u16, value)
	}

	return string(utf16.Decode(u16))
}

func looksUTF16LE(bytes []byte) bool {
	limit := len(bytes)
	if limit > 80 {
		limit = 80
	}

	zeros := 0
	for i := 1; i < limit; i += 2 {
		if bytes[i] == 0 {
			zeros++
		}
	}

	return zeros >= limit/4
}
