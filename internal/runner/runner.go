package runner

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

type Runner struct {
	Stdin   io.Reader
	Stdout  io.Writer
	Stderr  io.Writer
	DryRun  bool
	Timeout time.Duration
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

	cmd, cancel := r.command(binary, args...)
	defer cancel()
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

	cmd, cancel := r.command(binary, args...)
	defer cancel()
	cmd.Env = os.Environ()
	output, err := cmd.CombinedOutput()
	return string(output), err
}

func (r Runner) command(binary string, args ...string) (*exec.Cmd, context.CancelFunc) {
	timeout := r.Timeout
	if timeout <= 0 {
		if value := os.Getenv("DUCK_TIMEOUT"); value != "" {
			if seconds, err := strconv.Atoi(value); err == nil && seconds > 0 {
				timeout = time.Duration(seconds) * time.Second
			}
		}
	}
	if timeout <= 0 {
		return exec.Command(binary, args...), func() {}
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	return exec.CommandContext(ctx, binary, args...), cancel
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
