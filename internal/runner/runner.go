package runner

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
)

type Runner struct {
	Stdin  io.Reader
	Stdout io.Writer
	Stderr io.Writer
	DryRun bool
}

type Options struct {
	Stdin       io.Reader
	Stdout      io.Writer
	Stderr      io.Writer
	Interactive bool
}

func New() Runner {
	return Runner{
		Stdin:  os.Stdin,
		Stdout: os.Stdout,
		Stderr: os.Stderr,
	}
}

func NewDryRun() Runner {
	run := New()
	run.DryRun = true
	return run
}

func (r Runner) Run(binary string, args []string, opts Options) error {
	if binary == "" {
		return fmt.Errorf("binario nao informado")
	}

	if _, err := exec.LookPath(binary); err != nil {
		return fmt.Errorf("%s nao encontrado no PATH; configure o PATH ou a variavel DUCK_*_BIN correspondente", binary)
	}

	if r.DryRun {
		fmt.Fprintln(firstWriter(opts.Stdout, r.Stdout), "dry-run:", shellCommand(binary, args))
		return nil
	}

	cmd := exec.Command(binary, args...)
	cmd.Stdin = firstReader(opts.Stdin, r.Stdin)
	cmd.Stdout = firstWriter(opts.Stdout, r.Stdout)
	cmd.Stderr = firstWriter(opts.Stderr, r.Stderr)
	cmd.Env = os.Environ()

	return cmd.Run()
}

func (r Runner) Output(binary string, args []string) (string, error) {
	if binary == "" {
		return "", fmt.Errorf("binario nao informado")
	}

	if _, err := exec.LookPath(binary); err != nil {
		return "", fmt.Errorf("%s nao encontrado no PATH; configure o PATH ou a variavel DUCK_*_BIN correspondente", binary)
	}

	if r.DryRun {
		return "dry-run: " + shellCommand(binary, args), nil
	}

	cmd := exec.Command(binary, args...)
	cmd.Env = os.Environ()
	output, err := cmd.CombinedOutput()
	return string(output), err
}

func DefaultOptions() Options {
	return Options{}
}

func InteractiveOptions() Options {
	return Options{Interactive: true}
}

func firstReader(primary io.Reader, fallback io.Reader) io.Reader {
	if primary != nil {
		return primary
	}
	return fallback
}

func firstWriter(primary io.Writer, fallback io.Writer) io.Writer {
	if primary != nil {
		return primary
	}
	return fallback
}

func shellCommand(binary string, args []string) string {
	parts := append([]string{binary}, args...)
	for index, part := range parts {
		if strings.ContainsAny(part, " \t\"'") {
			parts[index] = fmt.Sprintf("%q", part)
		}
	}
	return strings.Join(parts, " ")
}
