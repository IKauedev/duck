package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func (m model) renderCompactSummary() string {
	var lines []string
	lines = append(lines, detailTitleStyle.Render("Duck Dashboard"))
	lines = append(lines, "")

	dockerLine := fmt.Sprintf("Docker: %s", m.dockerCountLabel())
	if m.alerts.dockerAlert() {
		dockerLine += fmt.Sprintf(" | %d unhealthy", m.alerts.DockerUnhealthy)
	}
	lines = append(lines, rowStyle.Render(dockerLine))

	kubeLine := fmt.Sprintf("Kubernetes: %s", m.kubeCountLabel())
	if m.kubeContext != "" {
		kubeLine += " | " + m.kubeContext
	}
	if m.alerts.kubeAlert() {
		kubeLine += fmt.Sprintf(" | %d com problema", m.alerts.KubeFailed)
	}
	lines = append(lines, rowStyle.Render(kubeLine))

	awsLine := "AWS: " + m.awsCountLabel()
	if m.alerts.awsAlert() {
		awsLine += " | credencial com problema"
	}
	lines = append(lines, rowStyle.Render(awsLine))

	if m.opts.Readonly {
		lines = append(lines, msgStyle.Render("modo somente leitura"))
	}
	lines = append(lines, helpStyle.Render(fmt.Sprintf("atualiza a cada %s | duck tui para modo completo", m.opts.Refresh)))
	return strings.Join(lines, "\n")
}

func (m model) renderCompactBody() string {
	width := m.width
	if width <= 0 {
		width = 100
	}
	limit := 8
	switch m.activeView {
	case dockerView:
		if m.dockerErr != nil {
			return m.renderActionableError("docker", m.dockerErrorText())
		}
		return truncateTable(m.renderDockerTable(width), limit)
	case kubeView:
		if m.kubeErr != nil {
			return m.renderKubeError()
		}
		return truncateTable(m.renderKubeTable(width), limit)
	case awsView:
		return m.renderAWS()
	default:
		return ""
	}
}

func truncateTable(table string, limit int) string {
	lines := strings.Split(table, "\n")
	if len(lines) <= limit+1 {
		return table
	}
	trimmed := append(lines[:limit+1], helpStyle.Render(fmt.Sprintf("... e mais %d itens (duck tui para lista completa)", len(lines)-limit-1)))
	return strings.Join(trimmed, "\n")
}

func (m model) renderActionableError(kind, message string) string {
	body := actionableError(kind, message)
	return errorStyle.Render(body)
}

func alertTabStyle(alert bool, active bool, label string) string {
	if !alert {
		if active {
			return activeTabStyle.Render(label)
		}
		return tabStyle.Render(label)
	}
	style := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("196"))
	if active {
		style = style.Background(lipgloss.Color("52"))
	}
	return style.Render(label)
}
