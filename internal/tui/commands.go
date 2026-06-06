package tui

import (
	"github.com/IKauedev/duck/internal/config"
	"github.com/IKauedev/duck/internal/runner"
)

func dockerShellArgs(cfg config.Config, backend *toolBackend, container string) []string {
	shell := "sh"
	if _, err := backend.output(cfg.DockerBin, []string{"exec", container, "bash", "-lc", "true"}); err == nil {
		shell = "bash"
	}
	return []string{"exec", "-it", container, shell}
}

func kubeShellArgs(cfg config.Config, backend *toolBackend, pod, namespace string) []string {
	shell := "sh"
	testArgs := []string{"exec", pod, "-n", namespace, "--", "bash", "-lc", "true"}
	if _, err := backend.output(cfg.KubectlBin, testArgs); err == nil {
		shell = "bash"
	}
	return []string{"exec", "-it", pod, "-n", namespace, "--", shell}
}

func runPendingAction(p *pendingAction, run runner.Runner) error {
	if p == nil {
		return nil
	}
	opts := runner.DefaultOptions()
	if p.interactive {
		opts = runner.InteractiveOptions()
	}
	return run.Run(p.binary, p.args, opts)
}
