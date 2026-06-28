package tui

import (
	"fmt"
	"sort"
	"strings"

	"github.com/IKauedev/duck/internal/config"
)

type menuItem struct {
	Kind    string
	Name    string
	Command string
}

func listTasksAndAliases() ([]menuItem, error) {
	settings, err := config.LoadSettings()
	if err != nil {
		return nil, err
	}
	items := make([]menuItem, 0)
	for key, value := range settings {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		switch {
		case strings.HasPrefix(key, "task."):
			items = append(items, menuItem{
				Kind:    "task",
				Name:    strings.TrimPrefix(key, "task."),
				Command: value,
			})
		case strings.HasPrefix(key, "alias."):
			items = append(items, menuItem{
				Kind:    "alias",
				Name:    strings.TrimPrefix(key, "alias."),
				Command: value,
			})
		}
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].Kind == items[j].Kind {
			return items[i].Name < items[j].Name
		}
		return items[i].Kind < items[j].Kind
	})
	return items, nil
}

func (m model) renderTasksMenu() string {
	if len(m.menuItems) == 0 {
		return msgStyle.Render("Nenhuma task ou alias configurada. Use duck task add e duck aliases add.")
	}
	var builder strings.Builder
	builder.WriteString(detailTitleStyle.Render("Tasks e Aliases"))
	builder.WriteString("\n\n")
	for index, item := range m.menuItems {
		line := fmt.Sprintf("[%s] %s -> %s", item.Kind, item.Name, item.Command)
		if index == m.menuCursor {
			builder.WriteString(selectedRowStyle.Render("› " + line))
		} else {
			builder.WriteString(rowStyle.Render("  " + line))
		}
		builder.WriteString("\n")
	}
	builder.WriteString("\n")
	builder.WriteString(helpStyle.Render("enter executa | esc volta"))
	return builder.String()
}

func (m model) selectedTaskDuckArgs() []string {
	if m.menuCursor < 0 || m.menuCursor >= len(m.menuItems) {
		return nil
	}
	item := m.menuItems[m.menuCursor]
	switch item.Kind {
	case "task":
		return []string{"task", "run", item.Name}
	case "alias":
		parts := strings.Fields(item.Command)
		if len(parts) == 0 {
			return nil
		}
		return append([]string{parts[0]}, parts[1:]...)
	default:
		return nil
	}
}
