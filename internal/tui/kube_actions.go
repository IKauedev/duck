package tui

import (
	"strconv"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

func (m *model) runKubeAction(action string) tea.Cmd {
	row, ok := m.selectedKubeRow()
	if !ok {
		return nil
	}
	cfg := m.cfg
	backend := &m.kubeBackend

	switch action {
	case "logs":
		if m.kubeResource != kubeResPods {
			return nil
		}
		return m.runExternal(backend, cfg.KubectlBin, []string{"logs", "--tail", "100", row.Name, "-n", row.Namespace}, false)
	case "shell":
		if m.kubeResource != kubeResPods {
			return nil
		}
		return m.runExternal(backend, cfg.KubectlBin, kubeShellArgs(cfg, backend, row.Name, row.Namespace), true)
	case "describe":
		if m.kubeResource == kubeResEvents || m.kubeResource == kubeResContexts {
			return nil
		}
		kind := kubeResourceSingular(m.kubeResource)
		args := []string{"describe", kind, row.Name}
		args = appendNamespaceArg(args, row.Namespace)
		return m.showDetail(func() (string, error) {
			return backend.output(cfg.KubectlBin, args)
		}, "Describe: "+m.kubeResourceRef(row))
	case "yaml":
		if !m.kubeSupportsYAML() {
			return nil
		}
		return m.showDetail(func() (string, error) {
			return backend.output(cfg.KubectlBin, m.kubeYAMLArgs(row))
		}, "YAML: "+m.kubeResourceRef(row))
	case "export-yaml":
		if !m.kubeSupportsYAML() {
			return nil
		}
		return func() tea.Msg {
			content, err := backend.output(cfg.KubectlBin, m.kubeYAMLArgs(row))
			if err != nil {
				return actionDoneMsg{view: kubeView, err: err, output: content}
			}
			path, err := m.exportKubeYAMLToFile(row, content)
			if err != nil {
				return actionDoneMsg{view: kubeView, err: err}
			}
			return actionDoneMsg{view: kubeView, output: "YAML exportado: " + path}
		}
	case "redeploy", "force-redeploy":
		if m.kubeResource != kubeResPods && m.kubeResource != kubeResDeployments {
			return nil
		}
		return m.runKubeRedeploy(action == "force-redeploy")
	case "delete":
		if m.kubeResource == kubeResEvents || m.kubeResource == kubeResContexts || m.kubeResource == kubeResNodes {
			return nil
		}
		kind := kubeResourceSingular(m.kubeResource)
		args := []string{"delete", kind, row.Name}
		args = appendNamespaceArg(args, row.Namespace)
		return m.runActionThenRefresh(backend, cfg.KubectlBin, args, kubeView)
	case "restart":
		if m.kubeResource != kubeResDeployments {
			return nil
		}
		args := []string{"rollout", "restart", "deployment/" + row.Name}
		args = appendNamespaceArg(args, row.Namespace)
		return m.runActionThenRefresh(backend, cfg.KubectlBin, args, kubeView)
	case "scale-up", "scale-down":
		if m.kubeResource != kubeResDeployments {
			return nil
		}
		return m.runKubeScale(row, action == "scale-up")
	case "set-image":
		if m.kubeResource != kubeResDeployments || strings.TrimSpace(m.kubeEditInput) == "" {
			return nil
		}
		args := []string{"set", "image", "deployment/" + row.Name, strings.TrimSpace(m.kubeEditInput)}
		args = appendNamespaceArg(args, row.Namespace)
		m.kubeEditInput = ""
		m.mode = "list"
		return m.runActionThenRefresh(backend, cfg.KubectlBin, args, kubeView)
	case "port-forward":
		if strings.TrimSpace(m.kubeEditInput) == "" {
			return nil
		}
		resource := row.Name
		if m.kubeResource == kubeResServices {
			resource = "service/" + row.Name
		}
		args := []string{"port-forward", resource, strings.TrimSpace(m.kubeEditInput)}
		args = appendNamespaceArg(args, row.Namespace)
		m.kubeEditInput = ""
		m.mode = "list"
		return m.runExternal(backend, cfg.KubectlBin, args, true)
	case "edit":
		if m.kubeResource == kubeResEvents || m.kubeResource == kubeResContexts || m.kubeResource == kubeResNodes {
			return nil
		}
		kind := kubeResourceSingular(m.kubeResource)
		args := []string{"edit", kind, row.Name}
		args = appendNamespaceArg(args, row.Namespace)
		return m.runExternal(backend, cfg.KubectlBin, args, true)
	case "use-namespace":
		return func() tea.Msg {
			return kubeNamespaceSelectedMsg{namespace: row.Name}
		}
	case "use-context":
		args := []string{"config", "use-context", row.Name}
		return m.runActionThenRefresh(backend, cfg.KubectlBin, args, kubeView)
	default:
		return nil
	}
}

func (m *model) runKubeScale(row kubeRow, up bool) tea.Cmd {
	parts := strings.Split(row.ColA, "/")
	if len(parts) != 2 {
		return nil
	}
	current, err := strconv.Atoi(strings.TrimSpace(parts[1]))
	if err != nil {
		return nil
	}
	if up {
		current++
	} else if current > 0 {
		current--
	}
	cfg := m.cfg
	backend := &m.kubeBackend
	args := []string{"scale", "deployment/" + row.Name, "--replicas", strconv.Itoa(current)}
	args = appendNamespaceArg(args, row.Namespace)
	return m.runActionThenRefresh(backend, cfg.KubectlBin, args, kubeView)
}

func appendNamespaceArg(args []string, namespace string) []string {
	if namespace == "" {
		return args
	}
	return append(args, "-n", namespace)
}

func (m model) kubeResourceRef(row kubeRow) string {
	if row.Namespace == "" {
		return row.Name
	}
	return row.Namespace + "/" + row.Name
}

func (m model) beginKubeInput(title, placeholder string) model {
	m.mode = "kube-input"
	m.kubeEditTitle = title
	m.kubeEditInput = placeholder
	return m
}

func (m model) renderKubeInputBar() string {
	return filterStyle.Render(m.kubeEditTitle + ": " + m.kubeEditInput + "█")
}

func (m model) kubeSupportsYAML() bool {
	switch m.kubeResource {
	case kubeResPods, kubeResDeployments, kubeResServices, kubeResIngress, kubeResNamespaces, kubeResNodes:
		return true
	default:
		return false
	}
}
