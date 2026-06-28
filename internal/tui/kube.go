package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

type kubeLoadedMsg struct {
	rows        []kubeRow
	context     string
	clusterInfo string
	err         error
	errText     string
}

func loadKube(backend *toolBackend, state kubeLoadState) tea.Cmd {
	return func() tea.Msg {
		cfg := backend.cfg
		context, err := backend.output(cfg.KubectlBin, []string{"config", "current-context"})
		if err != nil {
			return kubeLoadedMsg{
				err:     err,
				context: strings.TrimSpace(context),
				errText: formatCommandError(context, err),
			}
		}
		context = strings.TrimSpace(context)

		clusterInfo := loadClusterInfo(backend)

		output, err := fetchKubeResource(backend, state)
		if err != nil {
			return kubeLoadedMsg{
				err:         err,
				context:     context,
				clusterInfo: clusterInfo,
				errText:     formatCommandError(output, err),
			}
		}

		rows, parseErr := parseKubeRows(state.resource, output)
		if parseErr != nil {
			return kubeLoadedMsg{
				err:         parseErr,
				context:     context,
				clusterInfo: clusterInfo,
				errText:     parseErr.Error(),
			}
		}

		return kubeLoadedMsg{
			rows:        rows,
			context:     context,
			clusterInfo: clusterInfo,
		}
	}
}

type kubeLoadState struct {
	resource      kubeResourceKind
	allNamespaces bool
	namespace     string
}

func fetchKubeResource(backend *toolBackend, state kubeLoadState) (string, error) {
	cfg := backend.cfg
	switch state.resource {
	case kubeResContexts:
		return backend.output(cfg.KubectlBin, []string{"config", "view", "-o", "json"})
	case kubeResNodes:
		return backend.output(cfg.KubectlBin, []string{"get", "nodes", "-o", "json"})
	case kubeResNamespaces:
		return backend.output(cfg.KubectlBin, []string{"get", "namespaces", "-o", "json"})
	case kubeResEvents:
		args := []string{"get", "events", "-o", "json", "--sort-by=.metadata.creationTimestamp"}
		args = appendKubeScope(args, state)
		return backend.output(cfg.KubectlBin, args)
	default:
		resource := kubeResourceSingular(state.resource)
		if resource == "resource" {
			return "", fmt.Errorf("recurso invalido")
		}
		if resource == "ingress" {
			resource = "ingress"
		}
		args := []string{"get", resource, "-o", "json"}
		args = appendKubeScope(args, state)
		return backend.output(cfg.KubectlBin, args)
	}
}

func appendKubeScope(args []string, state kubeLoadState) []string {
	if state.resource == kubeResNamespaces || state.resource == kubeResNodes || state.resource == kubeResContexts {
		return args
	}
	if state.allNamespaces {
		return append(args, "-A")
	}
	if state.namespace != "" {
		return append(args, "-n", state.namespace)
	}
	return args
}

func loadClusterInfo(backend *toolBackend) string {
	cfg := backend.cfg
	nodes, err := backend.output(cfg.KubectlBin, []string{"get", "nodes", "-o", "jsonpath={.items[*].metadata.name}"})
	if err != nil {
		return "cluster indisponivel"
	}
	names := strings.Fields(nodes)
	version, _ := backend.output(cfg.KubectlBin, []string{"version", "--short"})
	version = strings.TrimSpace(strings.ReplaceAll(version, "\n", " | "))
	if version == "" {
		version = "kubectl ok"
	}
	return fmt.Sprintf("%d nodes | %s", len(names), truncate(version, 60))
}

func filterKubeRows(rows []kubeRow, filter string) []kubeRow {
	filter = strings.ToLower(strings.TrimSpace(filter))
	if filter == "" {
		return rows
	}
	filtered := make([]kubeRow, 0, len(rows))
	for _, row := range rows {
		haystack := strings.ToLower(strings.Join([]string{
			row.Resource, row.Namespace, row.Name, row.ColA, row.ColB, row.ColC, row.ColD,
			row.Status, row.Detail,
		}, " "))
		if strings.Contains(haystack, filter) {
			filtered = append(filtered, row)
		}
	}
	return filtered
}

func (m *model) kubeVisibleRows() []kubeRow {
	return filterKubeRows(m.kubeRows, m.filter)
}

func (m *model) selectedKubeRow() (kubeRow, bool) {
	rows := m.kubeVisibleRows()
	if m.kubeCursor < 0 || m.kubeCursor >= len(rows) {
		return kubeRow{}, false
	}
	return rows[m.kubeCursor], true
}

func (m *model) kubeLoadState() kubeLoadState {
	return kubeLoadState{
		resource:      m.kubeResource,
		allNamespaces: m.kubeAllNamespaces,
		namespace:     m.kubeNamespace,
	}
}

func (m *model) renderKubeTable(width int) string {
	rows := m.kubeVisibleRows()
	if len(rows) == 0 {
		return rowStyle.Render("Nenhum " + strings.ToLower(kubeResourceLabel(m.kubeResource)) + " encontrado.")
	}

	var builder strings.Builder
	builder.WriteString(m.renderKubeResourceBar())
	builder.WriteString("\n")
	builder.WriteString(m.renderKubeHeader(width))
	builder.WriteString("\n")

	for index, row := range rows {
		line := m.renderKubeDataRow(row, width)
		if index == m.kubeCursor {
			builder.WriteString(selectedRowStyle.Render("› " + line))
		} else {
			builder.WriteString(rowStyle.Render("  " + line))
		}
		builder.WriteString("\n")
	}
	return builder.String()
}

func (m model) renderKubeResourceBar() string {
	nsLabel := "todos"
	if !m.kubeAllNamespaces {
		if m.kubeNamespace != "" {
			nsLabel = m.kubeNamespace
		} else {
			nsLabel = "default"
		}
	}
	parts := []string{
		"recurso: " + kubeResourceLabel(m.kubeResource),
		"ns: " + nsLabel,
	}
	if m.kubeClusterInfo != "" {
		parts = append([]string{m.kubeClusterInfo}, parts...)
	}
	return helpStyle.Render(strings.Join(parts, " | "))
}

func (m model) renderKubeHeader(width int) string {
	switch m.kubeResource {
	case kubeResPods:
		return m.kubeHeader(width, "NAMESPACE", "NAME", "READY", "STATUS", "RST", "AGE")
	case kubeResDeployments:
		return m.kubeHeader(width, "NAMESPACE", "NAME", "READY", "UPDATED", "AVAILABLE", "AGE")
	case kubeResServices:
		return m.kubeHeader(width, "NAMESPACE", "NAME", "TYPE", "CLUSTER-IP", "PORTS", "AGE")
	case kubeResIngress:
		return m.kubeHeader(width, "NAMESPACE", "NAME", "CLASS", "HOSTS", "ADDRESS", "AGE")
	case kubeResNamespaces:
		return m.kubeHeader(width, "NAME", "STATUS", "", "", "", "AGE")
	case kubeResNodes:
		return m.kubeHeader(width, "NAME", "STATUS", "ROLE", "VERSION", "", "AGE")
	case kubeResEvents:
		return m.kubeHeader(width, "NAMESPACE", "OBJECT", "TYPE", "REASON", "MESSAGE", "AGE")
	case kubeResContexts:
		return m.kubeHeader(width, "CONTEXT", "CLUSTER", "", "", "", "")
	default:
		return ""
	}
}

func (m model) kubeHeader(width int, c1, c2, c3, c4, c5, c6 string) string {
	w := kubeColumnWidths(width, c3 != "" && c4 != "", c5 != "")
	cols := []string{c1, c2, c3, c4, c5, c6}
	parts := make([]string, 0, 6)
	for i, col := range cols {
		if col == "" {
			continue
		}
		if i < len(w) {
			parts = append(parts, padRight(col, w[i]))
		}
	}
	return tableHeaderStyle.Render(strings.Join(parts, "  "))
}

func kubeColumnWidths(width int, hasExtra, hasMore bool) []int {
	if width <= 0 {
		width = 100
	}
	if !hasExtra {
		return []int{28, 16, 12}
	}
	if !hasMore {
		return []int{14, 24, 14, 14, 12}
	}
	return []int{12, 22, 10, 14, 14, 10, 8}
}

func (m model) renderKubeDataRow(row kubeRow, width int) string {
	w := kubeColumnWidths(width, true, true)
	switch m.kubeResource {
	case kubeResPods:
		statusText := row.Status
		if row.Detail != "" && !strings.EqualFold(row.Detail, row.Status) {
			statusText = row.Detail
		}
		statusColored := kubeStatusStyle(row.Status, row.Detail).Render(padRight(statusText, w[3]))
		rstStyle := rowStyle
		if row.Restarts > 0 {
			rstStyle = statusWarn
		}
		return joinCols(w, padRight(row.Namespace, w[0]), padRight(row.Name, w[1]), padRight(row.ColA, w[2]),
			statusColored, rstStyle.Render(padRight(fmt.Sprintf("%d", row.Restarts), w[4])), padRight(row.Age, w[5]))
	case kubeResDeployments:
		readyColored := kubeStatusStyle(row.Status, row.Detail).Render(padRight(row.ColA, w[2]))
		return joinCols(w, padRight(row.Namespace, w[0]), padRight(row.Name, w[1]), readyColored,
			padRight(row.ColB, w[3]), padRight(row.ColC, w[4]), padRight(row.Age, w[5]))
	case kubeResServices:
		return joinCols(w, padRight(row.Namespace, w[0]), padRight(row.Name, w[1]), padRight(row.ColA, w[2]),
			padRight(row.ColB, w[3]), padRight(row.ColC, w[4]), padRight(row.Age, w[5]))
	case kubeResIngress:
		addressColored := kubeStatusStyle(row.Status, row.Detail).Render(padRight(row.ColC, w[4]))
		return joinCols(w, padRight(row.Namespace, w[0]), padRight(row.Name, w[1]), padRight(row.ColA, w[2]),
			padRight(truncate(row.ColB, w[3]), w[3]), addressColored, padRight(row.Age, w[5]))
	case kubeResNamespaces:
		return joinCols(w, padRight(row.Name, w[0]), kubeStatusStyle(row.Status, "").Render(padRight(row.Status, w[1])), padRight(row.Age, w[2]))
	case kubeResNodes:
		statusColored := kubeStatusStyle(row.Status, "").Render(padRight(row.Status, w[1]))
		return joinCols(w, padRight(row.Name, w[0]), statusColored, padRight(row.ColA, w[2]), padRight(row.ColB, w[3]), padRight(row.Age, w[5]))
	case kubeResEvents:
		typeStyle := rowStyle
		if strings.EqualFold(row.ColA, "Warning") {
			typeStyle = statusWarn
		}
		if strings.EqualFold(row.ColA, "Error") {
			typeStyle = statusBad
		}
		return joinCols(w, padRight(row.Namespace, w[0]), padRight(row.Name, w[1]), typeStyle.Render(padRight(row.ColA, w[2])),
			padRight(row.ColB, w[3]), padRight(row.ColC, w[4]), padRight(row.Age, w[5]))
	case kubeResContexts:
		name := row.Name
		if row.Current {
			name = "* " + name
		}
		return joinCols(w, padRight(name, w[0]), padRight(strings.TrimPrefix(row.ColA, "* "), w[1]))
	default:
		return row.Name
	}
}

func joinCols(_ []int, parts ...string) string {
	return strings.Join(parts, "  ")
}

func (m *model) kubeHelpLine() string {
	base := "[ ] recurso | n ns | c ctx | d describe | y yaml | Y exportar yaml"
	if m.opts.Readonly {
		return base + " | l logs | s shell | a escopo | / filtrar"
	}
	switch m.kubeResource {
	case kubeResPods:
		return base + " | U redeploy | U! force | l logs | s shell | f pf | ctrl+d apagar"
	case kubeResDeployments:
		return base + " | U redeploy | U! force+pods | R restart | +/- scale | I imagem | E edit"
	case kubeResServices:
		return base + " | f port-forward | E edit | ctrl+d apagar"
	case kubeResIngress:
		return base + " | y yaml | Y exportar | E edit | ctrl+d apagar"
	case kubeResNamespaces:
		return base + " | enter filtrar ns"
	case kubeResNodes:
		return base + " | d describe"
	case kubeResEvents:
		return base + " | d describe"
	case kubeResContexts:
		return base + " | enter trocar contexto"
	default:
		return base
	}
}
