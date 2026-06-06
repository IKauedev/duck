package tui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

type alertsState struct {
	DockerUnhealthy int
	KubeFailed      int
	AWSCredential   bool
}

type alertsLoadedMsg struct {
	state alertsState
}

func loadAlerts(dockerBackend, kubeBackend, awsBackend *toolBackend) tea.Cmd {
	return func() tea.Msg {
		state := alertsState{}
		cfg := dockerBackend.cfg

		if output, err := dockerBackend.output(cfg.DockerBin, []string{
			"ps", "--filter", "health=unhealthy", "--format", "{{.Names}}",
		}); err == nil {
			state.DockerUnhealthy = len(nonEmptyLines(output))
		}

		if output, err := kubeBackend.output(cfg.KubectlBin, []string{
			"get", "pods", "-A", "--field-selector=status.phase!=Running,status.phase!=Succeeded",
			"-o", "jsonpath={range .items[*]}{.metadata.name}{\"\\n\"}{end}",
		}); err == nil {
			state.KubeFailed = len(nonEmptyLines(output))
		}

		if _, err := awsBackend.output(cfg.AWSBin, []string{"sts", "get-caller-identity", "--output", "json"}); err != nil {
			state.AWSCredential = true
		}

		return alertsLoadedMsg{state: state}
	}
}

func nonEmptyLines(output string) []string {
	lines := splitLines(output)
	result := make([]string, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			result = append(result, line)
		}
	}
	return result
}

func (s alertsState) dockerAlert() bool {
	return s.DockerUnhealthy > 0
}

func (s alertsState) kubeAlert() bool {
	return s.KubeFailed > 0
}

func (s alertsState) awsAlert() bool {
	return s.AWSCredential
}
