package certstore

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/IKauedev/duck/internal/config"
)

type Certificate struct {
	Path string
	Name string
}

func Import(source string) (Certificate, error) {
	if source == "" {
		return Certificate{}, fmt.Errorf("informe o caminho ou URL do certificado")
	}

	dir, err := Directory()
	if err != nil {
		return Certificate{}, err
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		return Certificate{}, err
	}

	if isURL(source) {
		return download(source, dir)
	}
	return copyLocal(source, dir)
}

func Directory() (string, error) {
	settingsPath, err := config.SettingsPath()
	if err != nil {
		return "", err
	}
	return filepath.Join(filepath.Dir(settingsPath), "certificates"), nil
}

func isURL(source string) bool {
	parsed, err := url.Parse(source)
	return err == nil && (parsed.Scheme == "http" || parsed.Scheme == "https") && parsed.Host != ""
}

func download(source string, dir string) (Certificate, error) {
	parsed, err := url.Parse(source)
	if err != nil {
		return Certificate{}, err
	}
	name := path.Base(parsed.Path)
	if name == "." || name == "/" || name == "" {
		name = parsed.Host + ".crt"
	}
	name = safeName(name)

	client := http.Client{Timeout: 60 * time.Second}
	resp, err := client.Get(source)
	if err != nil {
		return Certificate{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return Certificate{}, fmt.Errorf("download do certificado retornou %s", resp.Status)
	}

	target := filepath.Join(dir, name)
	file, err := os.Create(target)
	if err != nil {
		return Certificate{}, err
	}
	defer file.Close()
	if _, err := io.Copy(file, resp.Body); err != nil {
		return Certificate{}, err
	}
	return Certificate{Path: target, Name: name}, nil
}

func copyLocal(source string, dir string) (Certificate, error) {
	abs, err := filepath.Abs(source)
	if err != nil {
		return Certificate{}, err
	}
	info, err := os.Stat(abs)
	if err != nil {
		return Certificate{}, err
	}
	if info.IsDir() {
		return Certificate{}, fmt.Errorf("o caminho do certificado precisa apontar para um arquivo")
	}

	name := safeName(filepath.Base(abs))
	target := filepath.Join(dir, name)
	if samePath(abs, target) {
		return Certificate{Path: target, Name: name}, nil
	}
	if err := copyFile(abs, target); err != nil {
		return Certificate{}, err
	}
	return Certificate{Path: target, Name: name}, nil
}

func copyFile(source string, target string) error {
	src, err := os.Open(source)
	if err != nil {
		return err
	}
	defer src.Close()

	dst, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	defer dst.Close()
	_, err = io.Copy(dst, src)
	return err
}

func samePath(left string, right string) bool {
	leftAbs, leftErr := filepath.Abs(left)
	rightAbs, rightErr := filepath.Abs(right)
	if leftErr != nil || rightErr != nil {
		return false
	}
	leftClean := filepath.Clean(leftAbs)
	rightClean := filepath.Clean(rightAbs)
	if runtime.GOOS == "windows" {
		return strings.EqualFold(leftClean, rightClean)
	}
	return leftClean == rightClean
}

func safeName(name string) string {
	name = filepath.Base(strings.TrimSpace(name))
	if name == "." || name == string(filepath.Separator) || name == "" {
		return "certificate.crt"
	}
	replacer := strings.NewReplacer("/", "_", "\\", "_", ":", "_", "*", "_", "?", "_", "\"", "_", "<", "_", ">", "_", "|", "_")
	return replacer.Replace(name)
}
