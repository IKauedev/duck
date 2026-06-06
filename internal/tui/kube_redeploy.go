package tui

import (
	"strings"

	"github.com/IKauedev/duck/internal/config"
	"github.com/IKauedev/duck/internal/runner"
	tea "github.com/charmbracelet/bubbletea"
)

func deploymentLabelSelector(backend *toolBackend, cfg config.Config, deployment, namespace string) string {
	output, err := backend.output(cfg.KubectlBin, []string{
		"get", "deployment", deployment, "-n", namespace,
		"-o", "jsonpath={range $k,$v := .spec.selector.matchLabels}{$k}={$v},{end}",
	})
	if err != nil {
		return ""
	}
	return strings.TrimSuffix(strings.TrimSpace(output), ",")
}

func (m *model) runKubeRedeploy(force bool) tea.Cmd {
	row, ok := m.selectedKubeRow()
	if !ok {
		return nil
	}
	cfg := m.cfg
	backend := &m.kubeBackend
	resource := m.kubeResource

	return func() tea.Msg {
		var lastErr error
		switch resource {
		case kubeResPods:
			args := []string{"delete", "pod", row.Name}
			args = appendNamespaceArg(args, row.Namespace)
			if force {
				args = append(args, "--grace-period=0", "--force")
			}
			lastErr = backend.runCommand(cfg.KubectlBin, args, runner.DefaultOptions())

		case kubeResDeployments:
			restart := []string{"rollout", "restart", "deployment/" + row.Name}
			restart = appendNamespaceArg(restart, row.Namespace)
			lastErr = backend.runCommand(cfg.KubectlBin, restart, runner.DefaultOptions())

			selector := deploymentLabelSelector(backend, cfg, row.Name, row.Namespace)
			if selector != "" {
				if force {
					pods := []string{"delete", "pods", "-n", row.Namespace, "-l", selector, "--grace-period=0", "--force", "--ignore-not-found=true"}
					if err := backend.runCommand(cfg.KubectlBin, pods, runner.DefaultOptions()); err != nil {
						lastErr = err
					}
				} else {
					cleanup := []string{
						"delete", "pods", "-n", row.Namespace, "-l", selector,
						"--field-selector", "status.phase=Failed",
						"--ignore-not-found=true",
					}
					if err := backend.runCommand(cfg.KubectlBin, cleanup, runner.DefaultOptions()); err != nil {
						lastErr = err
					}
					cleanupSucceeded := []string{
						"delete", "pods", "-n", row.Namespace, "-l", selector,
						"--field-selector", "status.phase=Succeeded",
						"--ignore-not-found=true",
					}
					_ = backend.runCommand(cfg.KubectlBin, cleanupSucceeded, runner.DefaultOptions())
				}
			}

			wait := []string{"rollout", "status", "deployment/" + row.Name, "--timeout=120s"}
			wait = appendNamespaceArg(wait, row.Namespace)
			_ = backend.runCommand(cfg.KubectlBin, wait, runner.DefaultOptions())

		default:
			return actionDoneMsg{view: kubeView, err: errUnsupportedRedeploy{}}
		}

		return actionDoneMsg{view: kubeView, err: lastErr}
	}
}

type errUnsupportedRedeploy struct{}

func (errUnsupportedRedeploy) Error() string {
	return "redeploy disponivel apenas para Pods e Deployments"
}
