package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var (
	headerStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("229")).
			Background(lipgloss.Color("62")).
			Padding(0, 1)

	activeTabStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("229")).
			Background(lipgloss.Color("62")).
			Padding(0, 1)

	tabStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("245")).
			Padding(0, 1)

	helpStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	errorStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
	msgStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("214"))
	filterStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("229")).
			Background(lipgloss.Color("236")).
			Padding(0, 1)

	tableHeaderStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("117"))

	selectedRowStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("229")).
				Background(lipgloss.Color("62"))

	rowStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("252"))

	statusGood = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("82"))
	statusWarn = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("214"))
	statusBad  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("196"))
	statusInfo = lipgloss.NewStyle().Foreground(lipgloss.Color("39"))
	statusMuted = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))

	confirmStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("196")).
			Background(lipgloss.Color("236")).
			Padding(1, 2)

	detailTitleStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("117"))

	detailBodyStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
)

func dockerStatusStyle(status, health, state string) lipgloss.Style {
	lower := strings.ToLower(status)
	healthLower := strings.ToLower(health)
	stateLower := strings.ToLower(state)

	switch {
	case healthLower == "unhealthy":
		return statusBad
	case healthLower == "healthy":
		return statusGood
	case strings.Contains(lower, "up") && strings.Contains(lower, "running"):
		return statusGood
	case strings.Contains(lower, "up"):
		return statusGood
	case stateLower == "running":
		return statusGood
	case strings.Contains(lower, "restarting"):
		return statusWarn
	case strings.Contains(lower, "paused"):
		return statusWarn
	case strings.Contains(lower, "health: starting"):
		return statusWarn
	case strings.Contains(lower, "exited") || strings.Contains(lower, "dead"):
		return statusMuted
	case strings.Contains(lower, "created"):
		return statusInfo
	default:
		return statusMuted
	}
}

func kubeStatusStyle(phase, detail string) lipgloss.Style {
	phaseLower := strings.ToLower(phase)
	detailLower := strings.ToLower(detail)

	switch {
	case strings.Contains(detailLower, "crashloopbackoff"),
		strings.Contains(detailLower, "imagepullbackoff"),
		strings.Contains(detailLower, "errimagepull"),
		strings.Contains(detailLower, "oomkilled"),
		strings.Contains(detailLower, "error"):
		return statusBad
	case phaseLower == "running":
		return statusGood
	case phaseLower == "pending":
		return statusWarn
	case phaseLower == "failed":
		return statusBad
	case phaseLower == "succeeded", phaseLower == "completed":
		return statusInfo
	case strings.Contains(detailLower, "terminating"):
		return statusWarn
	default:
		return statusMuted
	}
}

func truncate(value string, max int) string {
	if max <= 0 {
		return ""
	}
	if len(value) <= max {
		return value
	}
	if max <= 3 {
		return value[:max]
	}
	return value[:max-3] + "..."
}

func padRight(value string, width int) string {
	if width <= 0 {
		return ""
	}
	if len(value) >= width {
		return truncate(value, width)
	}
	return value + strings.Repeat(" ", width-len(value))
}
