//go:build windows

package install

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"golang.org/x/sys/windows/registry"
)

func ensureWindowsPath(dir string) error {
	if err := ensureWindowsPathRegistry(dir); err == nil {
		prependProcessPath(dir)
		return nil
	}

	if err := ensureWindowsPathSetx(dir); err == nil {
		prependProcessPath(dir)
		return nil
	}

	return fmt.Errorf("nao foi possivel atualizar o PATH do usuario sem PowerShell; adicione manualmente: %s", dir)
}

func ensureWindowsPathRegistry(dir string) error {
	key, err := registry.OpenKey(registry.CURRENT_USER, `Environment`, registry.SET_VALUE|registry.QUERY_VALUE)
	if err != nil {
		return err
	}
	defer key.Close()

	current, _, err := key.GetStringValue("Path")
	if err != nil && err != registry.ErrNotExist {
		return err
	}

	target := strings.TrimRight(filepathClean(dir), `\`)
	for _, entry := range splitPath(current) {
		if strings.EqualFold(strings.TrimRight(entry, `\`), target) {
			fmt.Println("PATH ja contem:", dir)
			return nil
		}
	}

	newPath := dir
	if strings.TrimSpace(current) != "" {
		newPath = current + ";" + dir
	}
	if err := key.SetStringValue("Path", newPath); err != nil {
		return err
	}
	return nil
}

func ensureWindowsPathSetx(dir string) error {
	current := os.Getenv("Path")
	if current == "" {
		current = os.Getenv("PATH")
	}
	for _, entry := range splitPath(current) {
		if strings.EqualFold(strings.TrimRight(filepathClean(entry), `\`), strings.TrimRight(filepathClean(dir), `\`)) {
			fmt.Println("PATH ja contem:", dir)
			return nil
		}
	}

	newPath := dir
	if strings.TrimSpace(current) != "" {
		newPath = current + ";" + dir
	}

	cmd := exec.Command("setx", "Path", newPath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s: %s", err, strings.TrimSpace(string(output)))
	}
	return nil
}

func splitPath(value string) []string {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return strings.Split(value, ";")
}

func filepathClean(path string) string {
	return strings.ReplaceAll(path, "/", `\`)
}
