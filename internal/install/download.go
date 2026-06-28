package install

import (
	"fmt"
	"path/filepath"
	"runtime"

	"github.com/IKauedev/duck/internal/cli"
	"github.com/IKauedev/duck/internal/release"
)

func DownloadCommand() cli.Command {
	return cli.Command{
		Name:        "download",
		Description: "Baixa o Duck do GitHub Releases sem compilar",
		Usage:       "download [--dir <pasta>] [--install] [--version <tag>] [--os <os>] [--arch <arch>] [--urls]",
		Run:         downloadRelease,
		Examples: []string{
			"download",
			"download --install",
			"download --dir . --install",
			"download --version v1.0.0 --install",
			"download --urls",
		},
	}
}

func downloadRelease(_ cli.Context, args []string) error {
	opts, err := parseDownloadOptions(args)
	if err != nil {
		return err
	}

	if opts.showURLs {
		return printDownloadURLs(opts)
	}

	destDir := opts.dir
	if destDir == "" {
		if opts.install {
			destDir, err = defaultInstallDir()
			if err != nil {
				return err
			}
		} else {
			destDir = "."
		}
	}

	fmt.Println("Baixando Duck...")
	result, err := release.DownloadBinary(release.DownloadOptions{
		Version: opts.version,
		GOOS:    opts.goos,
		GOARCH:  opts.goarch,
		DestDir: destDir,
	})
	if err != nil {
		return err
	}

	fmt.Println("Arquivo:", result.AssetName)
	if result.TagName != "" {
		fmt.Println("Versao:", result.TagName)
	}
	fmt.Println("Binario:", result.BinaryPath)
	if result.HTMLURL != "" {
		fmt.Println("Release:", result.HTMLURL)
	}

	if !opts.install {
		fmt.Println()
		fmt.Println("Proximo passo:")
		fmt.Println(" ", filepath.Join(destDir, executableName()), "install")
		return nil
	}

	fmt.Println("Duck instalado em:", result.BinaryPath)
	if opts.noPath {
		return nil
	}
	if err := ensurePath(destDir); err != nil {
		return err
	}
	printPathNotice(destDir)
	return nil
}

type downloadOptions struct {
	dir      string
	version  string
	goos     string
	goarch   string
	install  bool
	force    bool
	noPath   bool
	showURLs bool
}

func parseDownloadOptions(args []string) (downloadOptions, error) {
	var opts downloadOptions
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--dir":
			if i+1 >= len(args) {
				return opts, cli.UsageError("--dir precisa de uma pasta")
			}
			opts.dir = args[i+1]
			i++
		case "--version":
			if i+1 >= len(args) {
				return opts, cli.UsageError("--version precisa de uma tag")
			}
			opts.version = args[i+1]
			i++
		case "--os":
			if i+1 >= len(args) {
				return opts, cli.UsageError("--os precisa de um valor")
			}
			opts.goos = args[i+1]
			i++
		case "--arch":
			if i+1 >= len(args) {
				return opts, cli.UsageError("--arch precisa de um valor")
			}
			opts.goarch = args[i+1]
			i++
		case "--install":
			opts.install = true
		case "--force", "-f":
			opts.force = true
		case "--no-path":
			opts.noPath = true
		case "--urls":
			opts.showURLs = true
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

func printDownloadURLs(opts downloadOptions) error {
	goos := opts.goos
	if goos == "" {
		goos = runtime.GOOS
	}
	goarch := opts.goarch
	if goarch == "" {
		goarch = runtime.GOARCH
	}

	fmt.Println("URLs diretas de download (funcionam sem API do GitHub):")
	fmt.Println()
	fmt.Println("  Windows amd64:", release.DirectAssetURL(opts.version, "windows", "amd64"))
	fmt.Println("  Windows arm64:", release.DirectAssetURL(opts.version, "windows", "arm64"))
	fmt.Println("  Linux amd64:  ", release.DirectAssetURL(opts.version, "linux", "amd64"))
	fmt.Println("  Linux arm64:  ", release.DirectAssetURL(opts.version, "linux", "arm64"))
	fmt.Println("  macOS amd64:  ", release.DirectAssetURL(opts.version, "darwin", "amd64"))
	fmt.Println("  macOS arm64:  ", release.DirectAssetURL(opts.version, "darwin", "arm64"))
	fmt.Println()
	fmt.Println("Sistema atual:", goos+"/"+goarch)
	fmt.Println("  URL:", release.DirectAssetURL(opts.version, goos, goarch))
	fmt.Println()
	fmt.Println("Scripts sem PowerShell:")
	fmt.Println("  scripts\\install-windows.cmd")
	fmt.Println("  scripts/install-linux.sh")
	return nil
}
