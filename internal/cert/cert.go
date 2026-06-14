package cert

import (
	"fmt"
	"strings"
	"time"

	"github.com/IKauedev/duck/internal/certstore"
	"github.com/IKauedev/duck/internal/cli"
)

func Command() cli.Command {
	return cli.Command{
		Name:        "cert",
		Description: "Importa e busca certificados SSL/TLS",
		Usage:       "cert <fetch|import|dir>",
		Category:    "Utilitarios",
		Children: []cli.Command{
			{
				Name:        "fetch",
				Description: "Busca o certificado apresentado por um servidor HTTPS/TLS",
				Usage:       "cert fetch <url|host> [--port porta] [--chain] [--timeout segundos] [--dir pasta]",
				Run:         fetch,
				Examples: []string{
					"cert fetch https://example.com",
					"cert fetch api.local --port 8443",
					"cert fetch example.com --chain",
				},
			},
			{
				Name:        "import",
				Description: "Copia ou baixa um arquivo de certificado para a pasta do Duck",
				Usage:       "cert import <arquivo|url>",
				Run:         importCert,
				Examples: []string{
					"cert import C:\\certs\\empresa.crt",
					"cert import https://example.com/ca.crt",
				},
			},
			{
				Name:        "dir",
				Description: "Mostra a pasta onde os certificados sao salvos",
				Usage:       "cert dir",
				Run:         showDir,
			},
		},
		Examples: []string{
			"cert fetch https://example.com",
			"cert import https://example.com/ca.crt",
			"cert dir",
		},
	}
}

func fetch(_ cli.Context, args []string) error {
	opts, target, err := parseFetchArgs(args)
	if err != nil {
		return err
	}

	result, err := certstore.FetchFromTLS(target, opts)
	if err != nil {
		return err
	}

	fmt.Printf("Certificado obtido de %s:%s\n", result.Host, result.Port)
	for _, file := range result.Files {
		fmt.Println("Salvo em:", file.Path)
	}
	fmt.Println()
	fmt.Println("Proximos passos:")
	fmt.Println("  duck java cert", result.Files[0].Path, "--alias", suggestAlias(result.Host))
	fmt.Println("  duck node cert", result.Files[0].Path)
	return nil
}

func importCert(_ cli.Context, args []string) error {
	if len(args) != 1 {
		return cli.UsageError("use: cert import <arquivo|url>")
	}

	cert, err := certstore.Import(args[0])
	if err != nil {
		return err
	}

	fmt.Println("Certificado salvo em:", cert.Path)
	return nil
}

func showDir(_ cli.Context, args []string) error {
	if len(args) != 0 {
		return cli.UsageError("cert dir nao recebe argumentos")
	}

	dir, err := certstore.Directory()
	if err != nil {
		return err
	}
	fmt.Println(dir)
	return nil
}

func parseFetchArgs(args []string) (certstore.FetchOptions, string, error) {
	opts := certstore.FetchOptions{Timeout: 15 * time.Second}
	rest := make([]string, 0, len(args))

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--port":
			if i+1 >= len(args) {
				return opts, "", cli.UsageError("--port precisa de um valor")
			}
			opts.Port = args[i+1]
			i++
		case "--timeout":
			if i+1 >= len(args) {
				return opts, "", cli.UsageError("--timeout precisa de um valor")
			}
			value, err := time.ParseDuration(args[i+1] + "s")
			if err != nil {
				return opts, "", cli.UsageError("--timeout precisa ser numero de segundos")
			}
			opts.Timeout = value
			i++
		case "--dir":
			if i+1 >= len(args) {
				return opts, "", cli.UsageError("--dir precisa de um valor")
			}
			opts.Dir = args[i+1]
			i++
		case "--chain":
			opts.Chain = true
		default:
			rest = append(rest, args[i])
		}
	}

	if len(rest) != 1 {
		return opts, "", cli.UsageError("use: cert fetch <url|host> [--port porta] [--chain] [--timeout segundos] [--dir pasta]")
	}

	return opts, rest[0], nil
}

func suggestAlias(host string) string {
	alias := strings.NewReplacer(".", "-", ":", "-").Replace(strings.ToLower(host))
	if alias == "" {
		return "duck-cert"
	}
	return alias
}
