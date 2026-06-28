package shelltui

import (
	"bytes"
	"io"
	"os"
	"os/exec"
	"runtime"
	"strings"
)

// runShellCapture executa um comando nativo do SO capturando stdout/stderr
func runShellCapture(line string, out io.Writer) error {
	var cmd *exec.Cmd

	if strings.HasPrefix(line, "$ ") {
		line = strings.TrimPrefix(line, "$ ")
	}

	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("cmd.exe", "/C", line)
	default:
		shell := os.Getenv("SHELL")
		if shell == "" {
			shell = "/bin/sh"
		}
		cmd = exec.Command(shell, "-c", line)
	}

	cmd.Stdout = out
	cmd.Stderr = out
	return cmd.Run()
}

// captureOutput executa uma função que escreve em os.Stdout/Stderr e captura o resultado
func captureOutput(fn func()) string {
	r, w, _ := os.Pipe()
	origStdout := os.Stdout
	origStderr := os.Stderr
	os.Stdout = w
	os.Stderr = w

	fn()

	w.Close()
	os.Stdout = origStdout
	os.Stderr = origStderr

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	r.Close()
	return buf.String()
}
