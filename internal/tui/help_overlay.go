package tui

import "strings"

func (m model) renderHelpOverlay() string {
	var builder strings.Builder
	builder.WriteString(detailTitleStyle.Render("Ajuda rapida"))
	builder.WriteString("\n\n")
	sections := []struct {
		title string
		rows  []struct{ key, desc string }
	}{
		{
			title: "Geral",
			rows: []struct{ key, desc string }{
				{"tab / 1-3", "abas"},
				{"j/k", "navegar"},
				{"/", "filtrar"},
				{"r", "atualizar"},
				{"?", "esta ajuda"},
				{"e / E", "exportar JSON / CSV"},
				{"F", "favoritos"},
				{"T", "tasks e aliases"},
				{"ctrl+s", "salvar favorito"},
				{"q", "sair"},
			},
		},
		{
			title: "Docker",
			rows: []struct{ key, desc string }{
				{"l", "logs"}, {"s", "shell"}, {"i", "inspect"},
				{"S", "start"}, {"x", "stop"}, {"R", "restart"},
				{"ctrl+d", "apagar"}, {"a", "todos"},
			},
		},
		{
			title: "Kubernetes",
			rows: []struct{ key, desc string }{
				{"[ / ]", "trocar recurso"}, {"n", "namespaces"}, {"c", "contexts"},
				{"y", "ver yaml"}, {"Y", "exportar yaml"}, {"e", "exportar no painel yaml"},
				{"U", "redeploy"}, {"shift+U", "redeploy forçado"},
				{"l", "logs"}, {"s", "shell"}, {"d", "describe"},
				{"R", "restart deploy"}, {"+/-", "scale"}, {"I", "imagem"},
				{"f", "port-forward"}, {"E", "kubectl edit"}, {"ctrl+d", "apagar"},
				{"a", "escopo ns"}, {"enter", "usar ns/contexto"},
			},
		},
	}
	for _, section := range sections {
		builder.WriteString(tableHeaderStyle.Render(section.title))
		builder.WriteString("\n")
		for _, row := range section.rows {
			builder.WriteString(rowStyle.Render("  " + padRight(row.key, 12) + row.desc))
			builder.WriteString("\n")
		}
		builder.WriteString("\n")
	}
	if m.opts.Readonly {
		builder.WriteString(msgStyle.Render("modo somente leitura ativo"))
		builder.WriteString("\n")
	}
	if m.opts.Compact {
		builder.WriteString(helpStyle.Render("modo compacto: use `duck tui` para visao completa"))
		builder.WriteString("\n")
	}
	builder.WriteString(helpStyle.Render("esc fecha"))
	return builder.String()
}

func (m model) renderHelpCommand() string {
	var builder strings.Builder
	builder.WriteString(detailTitleStyle.Render("duck tui --help"))
	builder.WriteString("\n\n")
	builder.WriteString(detailBodyStyle.Render(strings.ReplaceAll(strings.TrimSpace(`
Modos:
  duck tui              interface completa
  duck tui --compact    dashboard compacto
  duck dashboard        alias do modo compacto

Env:
  DUCK_TUI_REFRESH=2s
  DUCK_TUI_CONFIRM=destructive|always|never
  DUCK_TUI_READONLY=true
`), "\n", "\n")))
	builder.WriteString("\n\n")
	builder.WriteString(m.renderHelpOverlay())
	return builder.String()
}
