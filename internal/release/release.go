package release

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

const (
	RepoOwner = "IKauedev"
	RepoName  = "duck"
)

type Info struct {
	TagName string
	HTMLURL string
	Assets  []Asset
}

type Asset struct {
	Name               string
	BrowserDownloadURL string
}

type DownloadOptions struct {
	Version string
	GOOS    string
	GOARCH  string
	DestDir string
}

type DownloadResult struct {
	BinaryPath string
	TagName    string
	HTMLURL    string
	AssetName  string
}

func FetchLatest() (Info, error) {
	return fetchRelease("https://api.github.com/repos/" + RepoOwner + "/" + RepoName + "/releases/latest")
}

func FetchVersion(version string) (Info, error) {
	version = strings.TrimPrefix(strings.TrimSpace(version), "v")
	if version == "" {
		return FetchLatest()
	}
	return fetchRelease("https://api.github.com/repos/" + RepoOwner + "/" + RepoName + "/releases/tags/v" + strings.TrimPrefix(version, "v"))
}

func fetchRelease(url string) (Info, error) {
	client := http.Client{Timeout: 30 * time.Second}
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return Info{}, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "duck-release")

	resp, err := client.Do(req)
	if err != nil {
		return Info{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return Info{}, fmt.Errorf("GitHub Releases retornou %s", resp.Status)
	}

	var payload struct {
		TagName string `json:"tag_name"`
		HTMLURL string `json:"html_url"`
		Assets  []struct {
			Name               string `json:"name"`
			BrowserDownloadURL string `json:"browser_download_url"`
		} `json:"assets"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return Info{}, err
	}
	if payload.TagName == "" {
		return Info{}, fmt.Errorf("release sem tag")
	}

	info := Info{TagName: payload.TagName, HTMLURL: payload.HTMLURL}
	for _, asset := range payload.Assets {
		info.Assets = append(info.Assets, Asset{
			Name:               asset.Name,
			BrowserDownloadURL: asset.BrowserDownloadURL,
		})
	}
	return info, nil
}

func AssetFileName(goos, goarch string) string {
	if goos == "" {
		goos = runtime.GOOS
	}
	if goarch == "" {
		goarch = runtime.GOARCH
	}
	if goos == "windows" {
		return fmt.Sprintf("duck_%s_%s.zip", goos, goarch)
	}
	return fmt.Sprintf("duck_%s_%s.tar.gz", goos, goarch)
}

func DirectAssetURL(version, goos, goarch string) string {
	fileName := AssetFileName(goos, goarch)
	if version == "" || version == "latest" {
		return "https://github.com/" + RepoOwner + "/" + RepoName + "/releases/latest/download/" + fileName
	}
	version = strings.TrimPrefix(version, "v")
	return "https://github.com/" + RepoOwner + "/" + RepoName + "/releases/download/v" + version + "/" + fileName
}

func DownloadBinary(opts DownloadOptions) (DownloadResult, error) {
	goos := opts.GOOS
	if goos == "" {
		goos = runtime.GOOS
	}
	goarch := opts.GOARCH
	if goarch == "" {
		goarch = runtime.GOARCH
	}
	destDir := opts.DestDir
	if destDir == "" {
		var err error
		destDir, err = os.MkdirTemp("", "duck-download-*")
		if err != nil {
			return DownloadResult{}, err
		}
	} else if err := os.MkdirAll(destDir, 0755); err != nil {
		return DownloadResult{}, err
	}

	var info Info
	var asset Asset
	var ok bool

	info, err := FetchVersion(opts.Version)
	if err == nil {
		asset, ok = SelectAsset(info.Assets, goos, goarch)
	}
	if err != nil || !ok {
		asset = Asset{
			Name:               AssetFileName(goos, goarch),
			BrowserDownloadURL: DirectAssetURL(opts.Version, goos, goarch),
		}
		if info.TagName == "" {
			info.TagName = normalizeVersionLabel(opts.Version)
			info.HTMLURL = "https://github.com/" + RepoOwner + "/" + RepoName + "/releases"
		}
	}

	tempDir, err := os.MkdirTemp("", "duck-extract-*")
	if err != nil {
		return DownloadResult{}, err
	}
	defer os.RemoveAll(tempDir)

	archivePath := filepath.Join(tempDir, asset.Name)
	if err := downloadFile(asset.BrowserDownloadURL, archivePath); err != nil {
		return DownloadResult{}, fmt.Errorf("falha ao baixar %s: %w", asset.BrowserDownloadURL, err)
	}

	binaryPath, err := extractBinary(archivePath, tempDir)
	if err != nil {
		return DownloadResult{}, err
	}

	finalPath := filepath.Join(destDir, ExecutableName(goos))
	if err := copyFile(binaryPath, finalPath); err != nil {
		return DownloadResult{}, err
	}
	if goos != "windows" {
		if err := os.Chmod(finalPath, 0755); err != nil {
			return DownloadResult{}, err
		}
	}

	return DownloadResult{
		BinaryPath: finalPath,
		TagName:    info.TagName,
		HTMLURL:    info.HTMLURL,
		AssetName:  asset.Name,
	}, nil
}

func SelectAsset(assets []Asset, goos, goarch string) (Asset, bool) {
	for _, candidate := range assets {
		name := strings.ToLower(candidate.Name)
		if strings.Contains(name, "checksum") {
			continue
		}
		if strings.Contains(name, goos) && strings.Contains(name, goarch) {
			return candidate, true
		}
	}
	return Asset{}, false
}

func ExecutableName(goos string) string {
	if goos == "" {
		goos = runtime.GOOS
	}
	if goos == "windows" {
		return "duck.exe"
	}
	return "duck"
}

func normalizeVersionLabel(version string) string {
	version = strings.TrimSpace(version)
	if version == "" || version == "latest" {
		return "latest"
	}
	if !strings.HasPrefix(version, "v") {
		return "v" + version
	}
	return version
}

func downloadFile(url string, destination string) error {
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
		target := filepath.Join(tempDir, ExecutableName(""))
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

		target := filepath.Join(tempDir, ExecutableName("windows"))
		dst, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, file.Mode())
		if err != nil {
			src.Close()
			return "", err
		}
		if _, err := io.Copy(dst, src); err != nil {
			src.Close()
			dst.Close()
			return "", err
		}
		src.Close()
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

		target := filepath.Join(tempDir, ExecutableName("linux"))
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
