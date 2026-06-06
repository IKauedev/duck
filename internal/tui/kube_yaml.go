package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func (m *model) kubeYAMLArgs(row kubeRow) []string {
	kind := kubeResourceSingular(m.kubeResource)
	args := []string{"get", kind, row.Name, "-o", "yaml"}
	return appendNamespaceArg(args, row.Namespace)
}

func (m model) exportKubeYAMLToFile(row kubeRow, content string) (string, error) {
	if strings.TrimSpace(content) == "" {
		cfg := m.cfg
		backend := &m.kubeBackend
		var err error
		content, err = backend.output(cfg.KubectlBin, m.kubeYAMLArgs(row))
		if err != nil {
			return "", err
		}
	}

	kind := kubeResourceSingular(m.kubeResource)
	ns := row.Namespace
	if ns == "" {
		ns = "cluster"
	}
	timestamp := time.Now().Format("20060102-150405")
	filename := fmt.Sprintf("duck-kube-%s-%s-%s-%s.yaml", kind, ns, row.Name, timestamp)
	path, err := filepath.Abs(filename)
	if err != nil {
		return "", err
	}
	if err := os.WriteFile(path, []byte(normalizeCLIOutput(content)), 0644); err != nil {
		return "", err
	}
	return path, nil
}

func exportYAMLContent(title, body string) (string, error) {
	body = normalizeCLIOutput(body)
	if strings.TrimSpace(body) == "" {
		return "", fmt.Errorf("yaml vazio")
	}
	slug := strings.ToLower(strings.TrimPrefix(title, "YAML:"))
	slug = strings.ReplaceAll(slug, "/", "-")
	slug = strings.ReplaceAll(slug, " ", "-")
	timestamp := time.Now().Format("20060102-150405")
	filename := fmt.Sprintf("duck-kube-%s-%s.yaml", slug, timestamp)
	path, err := filepath.Abs(filename)
	if err != nil {
		return "", err
	}
	if err := os.WriteFile(path, []byte(body), 0644); err != nil {
		return "", err
	}
	return path, nil
}

func isYAMLDetail(title string) bool {
	return strings.HasPrefix(title, "YAML:")
}
