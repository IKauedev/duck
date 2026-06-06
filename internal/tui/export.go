package tui

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func (m model) exportCurrent(format string) (string, error) {
	var (
		payload any
		name    string
	)
	switch m.activeView {
	case dockerView:
		payload = m.dockerVisibleRows()
		name = "docker"
	case kubeView:
		payload = m.kubeVisibleRows()
		name = "kube"
	default:
		return "", fmt.Errorf("exportacao disponivel apenas nas abas Docker e Kubernetes")
	}

	timestamp := time.Now().Format("20060102-150405")
	filename := fmt.Sprintf("duck-tui-%s-%s.%s", name, timestamp, format)
	path, err := filepath.Abs(filename)
	if err != nil {
		return "", err
	}

	switch format {
	case "json":
		content, err := json.MarshalIndent(payload, "", "  ")
		if err != nil {
			return "", err
		}
		if err := os.WriteFile(path, append(content, '\n'), 0644); err != nil {
			return "", err
		}
	case "csv":
		if err := writeCSV(path, m.activeView, payload); err != nil {
			return "", err
		}
	default:
		return "", fmt.Errorf("formato invalido: %s", format)
	}
	return path, nil
}

func writeCSV(path string, view viewKind, payload any) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	switch view {
	case dockerView:
		rows := payload.([]dockerRow)
		_ = writer.Write([]string{"name", "status", "image", "ports", "state", "health"})
		for _, row := range rows {
			_ = writer.Write([]string{row.Name, row.Status, row.Image, row.Ports, row.State, row.Health})
		}
	case kubeView:
		rows := payload.([]kubeRow)
		_ = writer.Write([]string{"resource", "namespace", "name", "col_a", "col_b", "col_c", "status", "detail", "age"})
		for _, row := range rows {
			_ = writer.Write([]string{
				row.Resource,
				row.Namespace,
				row.Name,
				row.ColA,
				row.ColB,
				row.ColC,
				row.Status,
				row.Detail,
				row.Age,
			})
		}
	}
	return writer.Error()
}

func (m model) renderExportMessage(path string) string {
	return msgStyle.Render("Exportado: " + strings.TrimSpace(path))
}
