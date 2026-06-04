package install

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	"duck/internal/cli"
	"duck/internal/prompt"
)

const appName = "duck"

func Command() cli.Command {
	return cli.Command{
		Name:        "install",
		Description: "Instala o Duck no usuario atual e configura o PATH",
		Usage:       "install [--dir <pasta>] [--force] [--no-path]",
		Run:         install,
		Examples: []string{
			"install",
			"install --force",
			"install --dir C:\\Users\\voce\\bin",
		},
	}
}

func SetupCommand() cli.Command {
	return cli.Command{
		Name:        "setup",
		Description: "Configura integracoes do Duck",
		Usage:       "setup <comando>",
		Children: []cli.Command{
			{
				Name:        "path",
				Description: "Adiciona a pasta do Duck ao PATH do usuario",
				Usage:       "setup path [--dir <pasta>]",
				Run:         setupPath,
				Examples: []string{
					"setup path",
					"setup path --dir C:\\Users\\voce\\bin",
				},
			},
		},
	}
}

func install(_ cli.Context, args []string) error {
	opts, err := parseOptions(args)
	if err != nil {
		return err
	}

	if opts.dir == "" {
		opts.dir, err = defaultInstallDir()
		if err != nil {
			return err
		}
	}

	source, err := os.Executable()
	if err != nil {
		return fmt.Errorf("nao foi possivel localizar o executavel atual: %w", err)
	}
	source, err = filepath.EvalSymlinks(source)
	if err != nil {
		return fmt.Errorf("nao foi possivel resolver o executavel atual: %w", err)
	}

	if err := os.MkdirAll(opts.dir, 0755); err != nil {
		return fmt.Errorf("nao foi possivel criar %s: %w", opts.dir, err)
	}

	target := filepath.Join(opts.dir, executableName())
	installed, err := copyExecutable(source, target, opts.force)
	if err != nil {
		return err
	}
	if !installed {
		return nil
	}

	fmt.Println("Duck instalado em:", target)
	if opts.noPath {
		return nil
	}

	if err := ensurePath(opts.dir); err != nil {
		return err
	}

	printPathNotice(opts.dir)
	return nil
}

func setupPath(_ cli.Context, args []string) error {
	opts, err := parseOptions(args)
	if err != nil {
		return err
	}
	if opts.force || opts.noPath {
		return cli.UsageError("setup path aceita apenas --dir")
	}

	if opts.dir == "" {
		exe, err := os.Executable()
		if err != nil {
			return fmt.Errorf("nao foi possivel localizar o executavel atual: %w", err)
		}
		opts.dir = filepath.Dir(exe)
	}

	if err := ensurePath(opts.dir); err != nil {
		return err
	}

	printPathNotice(opts.dir)
	return nil
}

type options struct {
	dir    string
	force  bool
	noPath bool
}

func parseOptions(args []string) (options, error) {
	var opts options

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--dir":
			if i+1 >= len(args) {
				return opts, cli.UsageError("--dir precisa de uma pasta")
			}
			opts.dir = args[i+1]
			i++
		case "--force", "-f":
			opts.force = true
		case "--no-path":
			opts.noPath = true
		default:
			return opts, cli.UsageError("opcao invalida: " + args[i])
		}
	}

	if opts.dir != "" {
		abs, err := filepath.Abs(opts.dir)
		if err != nil {
			return opts, err
		}
		opts.dir = abs
	}

	return opts, nil
}

func defaultInstallDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("nao foi possivel localizar a pasta do usuario: %w", err)
	}
	return filepath.Join(home, "bin"), nil
}

func executableName() string {
	if runtime.GOOS == "windows" {
		return appName + ".exe"
	}
	return appName
}

func copyExecutable(source, target string, force bool) (bool, error) {
	same, err := samePath(source, target)
	if err != nil {
		return false, err
	}
	if same {
		fmt.Println("O executavel ja esta na pasta de instalacao.")
		return true, nil
	}

	if _, err := os.Stat(target); err == nil && !force {
		ok, confirmErr := prompt.Confirm("O Duck ja existe no destino. Sobrescrever? [s/N] ")
		if confirmErr != nil {
			return false, confirmErr
		}
		if !ok {
			fmt.Println("Cancelado.")
			return false, nil
		}
	} else if err != nil && !errors.Is(err, os.ErrNotExist) {
		return false, err
	}

	input, err := os.Open(source)
	if err != nil {
		return false, fmt.Errorf("nao foi possivel abrir %s: %w", source, err)
	}
	defer input.Close()

	output, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0755)
	if err != nil {
		return false, fmt.Errorf("nao foi possivel escrever %s: %w", target, err)
	}
	defer output.Close()

	if _, err := io.Copy(output, input); err != nil {
		return false, fmt.Errorf("nao foi possivel copiar o executavel: %w", err)
	}

	if runtime.GOOS != "windows" {
		if err := os.Chmod(target, 0755); err != nil {
			return false, fmt.Errorf("nao foi possivel tornar %s executavel: %w", target, err)
		}
	}

	return true, nil
}

func samePath(source, target string) (bool, error) {
	sourceAbs, err := filepath.Abs(source)
	if err != nil {
		return false, err
	}
	targetAbs, err := filepath.Abs(target)
	if err != nil {
		return false, err
	}

	if runtime.GOOS == "windows" {
		return strings.EqualFold(filepath.Clean(sourceAbs), filepath.Clean(targetAbs)), nil
	}
	return filepath.Clean(sourceAbs) == filepath.Clean(targetAbs), nil
}

func ensurePath(dir string) error {
	if dirInCurrentPath(dir) {
		fmt.Println("PATH ja contem:", dir)
		return nil
	}

	if runtime.GOOS == "windows" {
		return ensureWindowsPath(dir)
	}
	return ensureUnixPath(dir)
}

func dirInCurrentPath(dir string) bool {
	current := os.Getenv("PATH")
	if current == "" {
		current = os.Getenv("Path")
	}

	target := filepath.Clean(dir)
	for _, entry := range filepath.SplitList(current) {
		cleanEntry := filepath.Clean(entry)
		if runtime.GOOS == "windows" {
			if strings.EqualFold(cleanEntry, target) {
				return true
			}
			continue
		}
		if cleanEntry == target {
			return true
		}
	}
	return false
}

func ensureWindowsPath(dir string) error {
	script := `
$installDir = ` + powershellString(dir) + `
$userPath = [Environment]::GetEnvironmentVariable('Path', 'User')
if ([string]::IsNullOrWhiteSpace($userPath)) {
  $parts = @()
} else {
  $parts = $userPath -split ';' | Where-Object { -not [string]::IsNullOrWhiteSpace($_) }
}
$exists = $false
foreach ($part in $parts) {
  if ($part.TrimEnd('\') -ieq $installDir.TrimEnd('\')) {
    $exists = $true
  }
}
if (-not $exists) {
  if ([string]::IsNullOrWhiteSpace($userPath)) {
    $newPath = $installDir
  } else {
    $newPath = "$userPath;$installDir"
  }
  [Environment]::SetEnvironmentVariable('Path', $newPath, 'User')
}
`
	cmd := exec.Command("powershell", "-NoProfile", "-NonInteractive", "-ExecutionPolicy", "Bypass", "-Command", script)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("nao foi possivel atualizar o PATH do usuario: %s", strings.TrimSpace(string(output)))
	}

	prependProcessPath(dir)
	return nil
}

func powershellString(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "''") + "'"
}

func ensureUnixPath(dir string) error {
	profile, err := shellProfile()
	if err != nil {
		return err
	}

	content, err := os.ReadFile(profile)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}

	line := "export PATH=" + shellPathEntry(dir) + ":$PATH"
	if strings.Contains(string(content), line) {
		fmt.Println("Perfil de shell ja contem PATH do Duck:", profile)
		prependProcessPath(dir)
		return nil
	}

	file, err := os.OpenFile(profile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return fmt.Errorf("nao foi possivel atualizar %s: %w", profile, err)
	}
	defer file.Close()

	if _, err := fmt.Fprintf(file, "\n# duck CLI PATH\n%s\n", line); err != nil {
		return err
	}

	fmt.Println("PATH do Duck adicionado em:", profile)
	prependProcessPath(dir)
	return nil
}

func shellProfile() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	switch filepath.Base(os.Getenv("SHELL")) {
	case "zsh":
		return filepath.Join(home, ".zshrc"), nil
	case "bash":
		return filepath.Join(home, ".bashrc"), nil
	default:
		return filepath.Join(home, ".profile"), nil
	}
}

func shellPathEntry(dir string) string {
	home, err := os.UserHomeDir()
	if err == nil {
		rel, relErr := filepath.Rel(home, dir)
		if relErr == nil && rel != "." && !strings.HasPrefix(rel, "..") {
			return strconv.Quote("$HOME/" + filepath.ToSlash(rel))
		}
		if relErr == nil && rel == "." {
			return strconv.Quote("$HOME")
		}
	}
	return strconv.Quote(filepath.ToSlash(dir))
}

func prependProcessPath(dir string) {
	current := os.Getenv("PATH")
	if current == "" {
		current = os.Getenv("Path")
	}
	_ = os.Setenv("PATH", dir+string(os.PathListSeparator)+current)
}

func printPathNotice(dir string) {
	fmt.Println("PATH configurado com:", dir)
	if runtime.GOOS == "windows" {
		fmt.Println("Abra um novo terminal para usar 'duck' de qualquer pasta.")
		return
	}
	fmt.Println("Abra um novo terminal ou recarregue seu perfil de shell para usar 'duck' de qualquer pasta.")
}
