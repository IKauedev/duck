package selfupdate

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

const latestReleaseURL = "https://api.github.com/repos/IKauedev/duck/releases/latest"

type release struct {
	TagName string  `json:"tag_name"`
	HTMLURL string  `json:"html_url"`
	Assets  []asset `json:"assets"`
}

type asset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
}

func Run(stdout io.Writer, currentVersion string) error {
	rel, err := latestRelease()
	if err != nil {
		return err
	}
	if isCurrent(currentVersion, rel.TagName) {
		fmt.Fprintf(stdout, "Duck ja esta atualizado: %s\n", currentVersion)
		return nil
	}

	selected, ok := selectAsset(rel.Assets)
	if !ok {
		return fmt.Errorf("release %s nao tem asset para %s/%s", rel.TagName, runtime.GOOS, runtime.GOARCH)
	}

	tempDir, err := os.MkdirTemp("", "duck-update-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tempDir)

	archivePath := filepath.Join(tempDir, selected.Name)
	if err := download(selected.BrowserDownloadURL, archivePath); err != nil {
		return err
	}

	binaryPath, err := extractBinary(archivePath, tempDir)
	if err != nil {
		return err
	}

	currentExe, err := os.Executable()
	if err != nil {
		return err
	}
	currentExe, err = filepath.EvalSymlinks(currentExe)
	if err != nil {
		return err
	}

	if runtime.GOOS == "windows" {
		return replaceWindows(stdout, binaryPath, currentExe, rel)
	}
	if err := replaceCurrent(binaryPath, currentExe); err != nil {
		return err
	}
	fmt.Fprintf(stdout, "Duck atualizado para %s\n%s\n", rel.TagName, rel.HTMLURL)
	return nil
}

func latestRelease() (release, error) {
	client := http.Client{Timeout: 30 * time.Second}
	req, err := http.NewRequest(http.MethodGet, latestReleaseURL, nil)
	if err != nil {
		return release{}, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "duck-update")

	resp, err := client.Do(req)
	if err != nil {
		return release{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return release{}, fmt.Errorf("GitHub Releases retornou %s", resp.Status)
	}

	var rel release
	if err := json.NewDecoder(resp.Body).Decode(&rel); err != nil {
		return release{}, err
	}
	if rel.TagName == "" {
		return release{}, fmt.Errorf("release sem tag")
	}
	return rel, nil
}

func isCurrent(currentVersion string, tag string) bool {
	if currentVersion == "" || currentVersion == "dev" {
		return false
	}
	tag = strings.TrimPrefix(tag, "v")
	currentVersion = strings.TrimPrefix(currentVersion, "v")
	return currentVersion == tag
}

func selectAsset(assets []asset) (asset, bool) {
	osName := runtime.GOOS
	arch := runtime.GOARCH
	for _, candidate := range assets {
		name := strings.ToLower(candidate.Name)
		if strings.Contains(name, "checksum") {
			continue
		}
		if strings.Contains(name, osName) && strings.Contains(name, arch) {
			return candidate, true
		}
	}
	return asset{}, false
}

func download(url string, destination string) error {
	client := http.Client{Timeout: 5 * time.Minute}
	resp, err := client.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download retornou %s", resp.Status)
	}

	file, err := os.Create(destination)
	if err != nil {
		return err
	}
	defer file.Close()
	_, err = io.Copy(file, resp.Body)
	return err
}

func extractBinary(archivePath string, tempDir string) (string, error) {
	lower := strings.ToLower(archivePath)
	switch {
	case strings.HasSuffix(lower, ".zip"):
		return extractZipBinary(archivePath, tempDir)
	case strings.HasSuffix(lower, ".tar.gz"), strings.HasSuffix(lower, ".tgz"):
		return extractTarGzBinary(archivePath, tempDir)
	default:
		target := filepath.Join(tempDir, executableName())
		if err := copyFile(archivePath, target); err != nil {
			return "", err
		}
		return target, os.Chmod(target, 0755)
	}
}

func extractZipBinary(archivePath string, tempDir string) (string, error) {
	reader, err := zip.OpenReader(archivePath)
	if err != nil {
		return "", err
	}
	defer reader.Close()

	for _, file := range reader.File {
		if !isDuckBinary(file.Name) {
			continue
		}
		src, err := file.Open()
		if err != nil {
			return "", err
		}
		defer src.Close()

		target := filepath.Join(tempDir, executableName())
		dst, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, file.Mode())
		if err != nil {
			return "", err
		}
		if _, err := io.Copy(dst, src); err != nil {
			dst.Close()
			return "", err
		}
		if err := dst.Close(); err != nil {
			return "", err
		}
		return target, os.Chmod(target, 0755)
	}
	return "", fmt.Errorf("binario duck nao encontrado no zip")
}

func extractTarGzBinary(archivePath string, tempDir string) (string, error) {
	file, err := os.Open(archivePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	gzipReader, err := gzip.NewReader(file)
	if err != nil {
		return "", err
	}
	defer gzipReader.Close()

	tarReader := tar.NewReader(gzipReader)
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", err
		}
		if header.Typeflag != tar.TypeReg || !isDuckBinary(header.Name) {
			continue
		}

		target := filepath.Join(tempDir, executableName())
		dst, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.FileMode(header.Mode))
		if err != nil {
			return "", err
		}
		if _, err := io.Copy(dst, tarReader); err != nil {
			dst.Close()
			return "", err
		}
		if err := dst.Close(); err != nil {
			return "", err
		}
		return target, os.Chmod(target, 0755)
	}
	return "", fmt.Errorf("binario duck nao encontrado no tar.gz")
}

func isDuckBinary(name string) bool {
	base := filepath.Base(name)
	return base == "duck" || base == "duck.exe"
}

func executableName() string {
	if runtime.GOOS == "windows" {
		return "duck.exe"
	}
	return "duck"
}

func replaceCurrent(newBinary string, currentExe string) error {
	if err := os.Chmod(newBinary, 0755); err != nil {
		return err
	}
	backup := currentExe + ".old"
	_ = os.Remove(backup)
	if err := os.Rename(currentExe, backup); err != nil {
		return err
	}
	if err := os.Rename(newBinary, currentExe); err != nil {
		_ = os.Rename(backup, currentExe)
		return err
	}
	_ = os.Remove(backup)
	return nil
}

func replaceWindows(stdout io.Writer, newBinary string, currentExe string, rel release) error {
	staged := currentExe + ".new"
	if err := copyFile(newBinary, staged); err != nil {
		return err
	}

	scriptPath := staged + ".ps1"
	script := `param([int]$PidToWait, [string]$Source, [string]$Target)
$Backup = "$Target.old"
Get-Process -Id $PidToWait -ErrorAction SilentlyContinue | Wait-Process
Remove-Item -LiteralPath $Backup -Force -ErrorAction SilentlyContinue
Move-Item -LiteralPath $Target -Destination $Backup -Force
Move-Item -LiteralPath $Source -Destination $Target -Force
Remove-Item -LiteralPath $Backup -Force -ErrorAction SilentlyContinue
Remove-Item -LiteralPath $MyInvocation.MyCommand.Path -Force -ErrorAction SilentlyContinue
`
	if err := os.WriteFile(scriptPath, []byte(script), 0644); err != nil {
		return err
	}

	cmd := exec.Command(
		"powershell",
		"-NoProfile",
		"-ExecutionPolicy",
		"Bypass",
		"-File",
		scriptPath,
		"-PidToWait",
		fmt.Sprint(os.Getpid()),
		"-Source",
		staged,
		"-Target",
		currentExe,
	)
	if err := cmd.Start(); err != nil {
		return err
	}
	fmt.Fprintf(stdout, "Duck %s baixado. A troca sera concluida quando este processo encerrar.\n%s\n", rel.TagName, rel.HTMLURL)
	return nil
}

func copyFile(source string, destination string) error {
	src, err := os.Open(source)
	if err != nil {
		return err
	}
	defer src.Close()

	dst, err := os.OpenFile(destination, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0755)
	if err != nil {
		return err
	}
	defer dst.Close()
	_, err = io.Copy(dst, src)
	return err
}
