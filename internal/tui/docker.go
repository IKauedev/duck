package tui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

type dockerRow struct {
	Name   string
	Status string
	Image  string
	Ports  string
	State  string
	Health string
}

type dockerLoadedMsg struct {
	rows    []dockerRow
	version string
	err     error
	errText string
}

func loadDocker(backend *toolBackend, showAll bool) tea.Cmd {
	return func() tea.Msg {
		cfg := backend.cfg
		version, err := backend.output(cfg.DockerBin, []string{
			"version", "--format", "Cliente: {{.Client.Version}} | Servidor: {{.Server.Version}}",
		})
		if err != nil {
			return dockerLoadedMsg{err: err, version: strings.TrimSpace(version), errText: formatCommandError(version, err)}
		}

		args := []string{"ps"}
		if showAll {
			args = append(args, "-a")
		}
		args = append(args, "--format", "{{.Names}}\t{{.Status}}\t{{.Image}}\t{{.Ports}}\t{{.State}}")

		output, err := backend.output(cfg.DockerBin, args)
		if err != nil {
			return dockerLoadedMsg{
				err:     err,
				version: strings.TrimSpace(version),
				errText: formatCommandError(output, err),
			}
		}

		return dockerLoadedMsg{
			rows:    parseDockerRows(output),
			version: strings.TrimSpace(version),
		}
	}
}

func parseDockerRows(output string) []dockerRow {
	lines := splitLines(output)
	rows := make([]dockerRow, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "\t", 6)
		if len(parts) < 4 {
			continue
		}
		row := dockerRow{
			Name:   strings.TrimSpace(parts[0]),
			Status: strings.TrimSpace(parts[1]),
			Image:  strings.TrimSpace(parts[2]),
			Ports:  strings.TrimSpace(parts[3]),
		}
		if len(parts) > 4 {
			row.State = strings.TrimSpace(parts[4])
		}
		row.Health = healthFromDockerStatus(row.Status)
		rows = append(rows, row)
	}
	return rows
}

func healthFromDockerStatus(status string) string {
	start := strings.LastIndex(status, "(")
	end := strings.LastIndex(status, ")")
	if start < 0 || end <= start {
		return ""
	}
	inner := strings.ToLower(strings.TrimSpace(status[start+1 : end]))
	switch {
	case inner == "healthy", inner == "unhealthy":
		return inner
	case strings.HasPrefix(inner, "health:"):
		return inner
	default:
		return ""
	}
}

func filterDockerRows(rows []dockerRow, filter string) []dockerRow {
	filter = strings.ToLower(strings.TrimSpace(filter))
	if filter == "" {
		return rows
	}
	filtered := make([]dockerRow, 0, len(rows))
	for _, row := range rows {
		haystack := strings.ToLower(row.Name + " " + row.Status + " " + row.Image + " " + row.Ports)
		if strings.Contains(haystack, filter) {
			filtered = append(filtered, row)
		}
	}
	return filtered
}

func (m *model) dockerVisibleRows() []dockerRow {
	return filterDockerRows(m.dockerRows, m.filter)
}

func (m *model) selectedDockerRow() (dockerRow, bool) {
	rows := m.dockerVisibleRows()
	if m.dockerCursor < 0 || m.dockerCursor >= len(rows) {
		return dockerRow{}, false
	}
	return rows[m.dockerCursor], true
}

func (m *model) renderDockerTable(width int) string {
	rows := m.dockerVisibleRows()
	if len(rows) == 0 {
		return rowStyle.Render("Nenhum container encontrado.")
	}

	nameW := 24
	statusW := 28
	imageW := 24
	portsW := 20
	remaining := width - nameW - statusW - imageW - portsW - 8
	if remaining > 0 {
		nameW += remaining / 2
		imageW += remaining / 2
	}

	var builder strings.Builder
	header := padRight("NAME", nameW) + "  " +
		padRight("STATUS", statusW) + "  " +
		padRight("IMAGE", imageW) + "  " +
		padRight("PORTS", portsW)
	builder.WriteString(tableHeaderStyle.Render(header))
	builder.WriteString("\n")

	for index, row := range rows {
		statusText := row.Status
		if row.Health != "" && !strings.Contains(strings.ToLower(row.Status), row.Health) {
			statusText += " (" + row.Health + ")"
		}
		statusColored := dockerStatusStyle(row.Status, row.Health, row.State).Render(padRight(statusText, statusW))
		line := padRight(row.Name, nameW) + "  " + statusColored + "  " +
			padRight(row.Image, imageW) + "  " + padRight(row.Ports, portsW)

		if index == m.dockerCursor {
			builder.WriteString(selectedRowStyle.Render("› " + line))
		} else {
			builder.WriteString(rowStyle.Render("  " + line))
		}
		builder.WriteString("\n")
	}
	return builder.String()
}

func (m *model) dockerHelpLine() string {
	if m.opts.Readonly {
		return "l logs | s shell | i inspect | a todos | / filtrar"
	}
	return "l logs | s shell | i inspect | S start | x stop | R restart | ctrl+d apagar | a todos | / filtrar"
}

func (m *model) runDockerAction(action string) tea.Cmd {
	row, ok := m.selectedDockerRow()
	if !ok {
		return nil
	}
	cfg := m.cfg
	backend := &m.dockerBackend
	name := row.Name

	switch action {
	case "logs":
		return m.runExternal(backend, cfg.DockerBin, []string{"logs", "--tail", "100", name}, false)
	case "shell":
		return m.runExternal(backend, cfg.DockerBin, dockerShellArgs(cfg, backend, name), true)
	case "inspect":
		return m.showDetail(func() (string, error) {
			return backend.output(cfg.DockerBin, []string{
				"inspect", "--format",
				"Nome: {{.Name}}\nImagem: {{.Config.Image}}\nStatus: {{.State.Status}}\nRunning: {{.State.Running}}\nHealth: {{if .State.Health}}{{.State.Health.Status}}{{else}}n/a{{end}}\nStarted: {{.State.StartedAt}}\nIP: {{range .NetworkSettings.Networks}}{{.IPAddress}} {{end}}\nPortas: {{json .NetworkSettings.Ports}}",
				name,
			})
		}, "Inspect: "+name)
	case "start", "stop", "restart":
		return m.runActionThenRefresh(backend, cfg.DockerBin, []string{action, name}, dockerView)
	case "delete":
		return m.runActionThenRefresh(backend, cfg.DockerBin, []string{"rm", "-f", name}, dockerView)
	default:
		return nil
	}
}
