package tui

import (
	"fmt"
	"strings"

	"github.com/IKauedev/duck/internal/config"
)

const (
	favoriteDockerPrefix = "tui.fav.docker."
	favoriteKubePrefix   = "tui.fav.kube."
)

type favoriteItem struct {
	Key   string
	Label string
	Value string
}

func listFavorites(kind string) ([]favoriteItem, error) {
	settings, err := config.LoadSettings()
	if err != nil {
		return nil, err
	}
	prefix := favoriteDockerPrefix
	if kind == "kube" {
		prefix = favoriteKubePrefix
	}
	items := make([]favoriteItem, 0)
	for key, value := range settings {
		if !strings.HasPrefix(key, prefix) || strings.TrimSpace(value) == "" {
			continue
		}
		items = append(items, favoriteItem{
			Key:   strings.TrimPrefix(key, prefix),
			Label: strings.TrimPrefix(key, prefix),
			Value: value,
		})
	}
	return items, nil
}

func saveFavorite(kind, key, value string) error {
	prefix := favoriteDockerPrefix
	if kind == "kube" {
		prefix = favoriteKubePrefix
	}
	return config.SetSetting(prefix+sanitizeFavoriteKey(key), value)
}

func sanitizeFavoriteKey(key string) string {
	key = strings.TrimSpace(key)
	key = strings.ReplaceAll(key, " ", "-")
	key = strings.ReplaceAll(key, "/", "-")
	if key == "" {
		return "item"
	}
	return key
}

func (m *model) currentFavoriteValue() (kind, key, value string, ok bool) {
	switch m.activeView {
	case dockerView:
		row, found := m.selectedDockerRow()
		if !found {
			return "", "", "", false
		}
		return "docker", row.Name, row.Name, true
	case kubeView:
		row, found := m.selectedKubeRow()
		if !found {
			return "", "", "", false
		}
		value := fmt.Sprintf("%s|%s|%s", row.Resource, row.Namespace, row.Name)
		return "kube", row.Namespace + "/" + row.Name, value, true
	default:
		return "", "", "", false
	}
}

func (m *model) jumpToFavorite(item favoriteItem) {
	switch m.activeView {
	case dockerView:
		rows := m.dockerVisibleRows()
		for index, row := range rows {
			if row.Name == item.Value {
				m.dockerCursor = index
				return
			}
		}
	case kubeView:
		parts := strings.Split(item.Value, "|")
		if len(parts) < 2 {
			return
		}
		resource := ""
		namespace := ""
		name := ""
		switch len(parts) {
		case 2:
			namespace, name = parts[0], parts[1]
		default:
			resource, namespace, name = parts[0], parts[1], parts[2]
		}
		if resource != "" {
			m.kubeResource = kubeResourceFromName(resource)
		}
		rows := m.kubeVisibleRows()
		for index, row := range rows {
			if namespace == "" || row.Namespace == namespace {
				if row.Name == name {
					m.kubeCursor = index
					return
				}
			}
		}
	}
}

func (m model) renderFavorites() string {
	if len(m.menuItems) == 0 {
		return msgStyle.Render("Nenhum favorito salvo. Use ctrl+s no item desejado.")
	}
	var builder strings.Builder
	builder.WriteString(detailTitleStyle.Render("Favoritos"))
	builder.WriteString("\n\n")
	for index, item := range m.menuItems {
		line := fmt.Sprintf("%s -> %s", item.Name, item.Command)
		if index == m.menuCursor {
			builder.WriteString(selectedRowStyle.Render("› " + line))
		} else {
			builder.WriteString(rowStyle.Render("  " + line))
		}
		builder.WriteString("\n")
	}
	builder.WriteString("\n")
	builder.WriteString(helpStyle.Render("enter vai ao item | esc volta"))
	return builder.String()
}

func (m model) loadFavoriteItems() ([]favoriteItem, error) {
	switch m.activeView {
	case dockerView:
		return listFavorites("docker")
	case kubeView:
		return listFavorites("kube")
	default:
		return nil, nil
	}
}
