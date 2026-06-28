package selfupdate

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/IKauedev/duck/internal/release"
	"github.com/IKauedev/duck/internal/version"
)

type CheckResult struct {
	Current     string
	Latest      string
	UpdateReady bool
	HTMLURL     string
}

func Check(currentVersion string) (CheckResult, error) {
	info, err := release.FetchLatest()
	if err != nil {
		return CheckResult{}, err
	}
	result := CheckResult{
		Current: currentVersion,
		Latest:  info.TagName,
		HTMLURL: info.HTMLURL,
	}
	result.UpdateReady = !version.MatchesTag(info.TagName) && !isCurrent(currentVersion, info.TagName)
	return result, nil
}

func Run(stdout io.Writer, currentVersion string) error {
	return RunOptions(stdout, currentVersion, Options{})
}

type Options struct {
	CheckOnly bool
	Install   bool
}

func RunOptions(stdout io.Writer, currentVersion string, opts Options) error {
	check, err := Check(currentVersion)
	if err != nil {
		return err
	}
	if opts.CheckOnly {
		if check.UpdateReady {
			fmt.Fprintf(stdout, "Atualizacao disponivel: %s -> %s\n%s\n", check.Current, check.Latest, check.HTMLURL)
		} else {
			fmt.Fprintf(stdout, "Duck ja esta atualizado: %s\n", version.Label())
		}
		return nil
	}
	if !check.UpdateReady {
		fmt.Fprintf(stdout, "Duck ja esta atualizado: %s\n", version.Label())
		return nil
	}

	if opts.Install {
		return installLatest(stdout, check.Latest, check.HTMLURL)
	}

	tempDir, err := os.MkdirTemp("", "duck-update-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tempDir)

	result, err := release.DownloadBinary(release.DownloadOptions{
		Version: check.Latest,
		DestDir: tempDir,
	})
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
		return replaceWindows(stdout, result.BinaryPath, currentExe, check.Latest, check.HTMLURL)
	}
	if err := replaceCurrent(result.BinaryPath, currentExe); err != nil {
		return err
	}
	fmt.Fprintf(stdout, "Duck atualizado para %s\n%s\n", check.Latest, check.HTMLURL)
	return nil
}

func installLatest(stdout io.Writer, tagName, htmlURL string) error {
	installDir, err := defaultInstallDirForUpdate()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(installDir, 0755); err != nil {
		return err
	}

	result, err := release.DownloadBinary(release.DownloadOptions{
		Version: tagName,
		DestDir: installDir,
	})
	if err != nil {
		return err
	}

	fmt.Fprintf(stdout, "Duck %s instalado em %s\n%s\n", tagName, result.BinaryPath, htmlURL)
	fmt.Fprintln(stdout, "Execute: duck install --force --dir", installDir)
	return nil
}

func defaultInstallDirForUpdate() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, "bin"), nil
}

func isCurrent(currentVersion string, tag string) bool {
	if currentVersion == "" || currentVersion == "dev" {
		return false
	}
	tag = strings.TrimPrefix(tag, "v")
	currentVersion = strings.TrimPrefix(currentVersion, "v")
	return currentVersion == tag
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

func replaceWindows(stdout io.Writer, newBinary string, currentExe string, tagName string, htmlURL string) error {
	staged := currentExe + ".new"
	if err := copyFile(newBinary, staged); err != nil {
		return err
	}

	batchPath := staged + ".cmd"
	batch := fmt.Sprintf(`@echo off
:wait
tasklist /FI "PID eq %d" 2>nul | find "%d" >nul
if not errorlevel 1 (
  timeout /t 1 /nobreak >nul
  goto wait
)
if exist "%s.old" del /F /Q "%s.old"
move /Y "%s" "%s.old"
move /Y "%s" "%s"
if exist "%s.old" del /F /Q "%s.old"
del /F /Q "%%~f0"
`,
		os.Getpid(), os.Getpid(),
		currentExe, currentExe,
		currentExe, currentExe,
		staged, currentExe,
		currentExe, currentExe,
	)
	if err := os.WriteFile(batchPath, []byte(batch), 0644); err != nil {
		return replaceWindowsPowerShell(stdout, staged, currentExe, tagName, htmlURL)
	}

	cmd := exec.Command("cmd.exe", "/C", batchPath)
	if err := cmd.Start(); err != nil {
		return replaceWindowsPowerShell(stdout, staged, currentExe, tagName, htmlURL)
	}

	fmt.Fprintf(stdout, "Duck %s baixado. A troca sera concluida quando este processo encerrar.\n%s\n", tagName, htmlURL)
	return nil
}

func replaceWindowsPowerShell(stdout io.Writer, staged string, currentExe string, tagName string, htmlURL string) error {
	scriptPath := staged + ".ps1"
	script := fmt.Sprintf(`param([int]$PidToWait, [string]$Source, [string]$Target)
$Backup = "$Target.old"
Get-Process -Id $PidToWait -ErrorAction SilentlyContinue | Wait-Process
Remove-Item -LiteralPath $Backup -Force -ErrorAction SilentlyContinue
Move-Item -LiteralPath $Target -Destination $Backup -Force
Move-Item -LiteralPath $Source -Destination $Target -Force
Remove-Item -LiteralPath $Backup -Force -ErrorAction SilentlyContinue
Remove-Item -LiteralPath $MyInvocation.MyCommand.Path -Force -ErrorAction SilentlyContinue
`)
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
		return fmt.Errorf("nao foi possivel concluir a atualizacao no Windows: %w", err)
	}
	fmt.Fprintf(stdout, "Duck %s baixado via PowerShell. A troca sera concluida quando este processo encerrar.\n%s\n", tagName, htmlURL)
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
