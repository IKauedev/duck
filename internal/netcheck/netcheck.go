package netcheck

import (
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/IKauedev/duck/internal/cli"
)

func CurlCommand() cli.Command {
	return cli.Command{
		Name:        "curl",
		Aliases:     []string{"http"},
		Description: "Testa URL/porta a partir da maquina local",
		Usage:       "curl <url> [--port porta] [--timeout segundos] [--insecure] [--method metodo]",
		Run:         curl,
		Examples: []string{
			"curl https://example.com",
			"curl api.local --port 8080",
			"curl https://api.local --insecure --timeout 5",
		},
	}
}

func PortCommand() cli.Command {
	return cli.Command{
		Name:        "port",
		Description: "Testa conectividade TCP local",
		Usage:       "port check <host> <port> [--timeout segundos]",
		Children: []cli.Command{
			{Name: "check", Description: "Testa host e porta via TCP", Usage: "port check <host> <port> [--timeout segundos]", Run: portCheck},
		},
		Examples: []string{
			"port check localhost 5432",
			"port check redis.local 6379 --timeout 3",
		},
	}
}

type curlOptions struct {
	url      string
	port     string
	timeout  time.Duration
	insecure bool
	method   string
}

func curl(_ cli.Context, args []string) error {
	opts, err := parseCurlArgs(args)
	if err != nil {
		return err
	}

	target, err := NormalizeURL(opts.url, opts.port)
	if err != nil {
		return err
	}

	client := http.Client{
		Timeout: opts.timeout,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: opts.insecure},
		},
	}

	start := time.Now()
	req, err := http.NewRequest(opts.method, target, nil)
	if err != nil {
		return err
	}
	resp, err := client.Do(req)
	elapsed := time.Since(start)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	fmt.Println("URL:", target)
	fmt.Println("Status:", resp.Status)
	fmt.Println("Tempo:", elapsed.Round(time.Millisecond))
	fmt.Println("Servidor:", resp.Header.Get("Server"))
	return nil
}

func parseCurlArgs(args []string) (curlOptions, error) {
	opts := curlOptions{timeout: 10 * time.Second, method: http.MethodGet}
	rest := make([]string, 0, len(args))

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--port":
			if i+1 >= len(args) {
				return opts, cli.UsageError("--port precisa de um valor")
			}
			opts.port = args[i+1]
			i++
		case "--timeout":
			if i+1 >= len(args) {
				return opts, cli.UsageError("--timeout precisa de um valor")
			}
			seconds, err := time.ParseDuration(args[i+1] + "s")
			if err != nil {
				return opts, cli.UsageError("--timeout precisa ser numero de segundos")
			}
			opts.timeout = seconds
			i++
		case "-k", "--insecure":
			opts.insecure = true
		case "-X", "--method":
			if i+1 >= len(args) {
				return opts, cli.UsageError(args[i] + " precisa de um valor")
			}
			opts.method = strings.ToUpper(args[i+1])
			i++
		default:
			rest = append(rest, args[i])
		}
	}
	if len(rest) != 1 {
		return opts, cli.UsageError("use: curl <url> [--port porta] [--timeout segundos] [--insecure]")
	}
	opts.url = rest[0]
	return opts, nil
}

func NormalizeURL(rawURL string, port string) (string, error) {
	if !strings.Contains(rawURL, "://") {
		rawURL = "http://" + rawURL
	}
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return "", err
	}
	if parsed.Host == "" {
		return "", fmt.Errorf("url invalida: %s", rawURL)
	}
	if port != "" && parsed.Port() == "" {
		parsed.Host = net.JoinHostPort(parsed.Hostname(), port)
	}
	return parsed.String(), nil
}

func portCheck(_ cli.Context, args []string) error {
	timeout := 5 * time.Second
	rest := make([]string, 0, len(args))
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--timeout":
			if i+1 >= len(args) {
				return cli.UsageError("--timeout precisa de um valor")
			}
			value, err := time.ParseDuration(args[i+1] + "s")
			if err != nil {
				return cli.UsageError("--timeout precisa ser numero de segundos")
			}
			timeout = value
			i++
		default:
			rest = append(rest, args[i])
		}
	}
	if len(rest) != 2 {
		return cli.UsageError("use: port check <host> <port> [--timeout segundos]")
	}

	address := net.JoinHostPort(rest[0], rest[1])
	start := time.Now()
	conn, err := net.DialTimeout("tcp", address, timeout)
	elapsed := time.Since(start).Round(time.Millisecond)
	if err != nil {
		return fmt.Errorf("%s indisponivel apos %s: %w", address, elapsed, err)
	}
	defer conn.Close()

	fmt.Println("TCP ok:", address)
	fmt.Println("Tempo:", elapsed)
	return nil
}
