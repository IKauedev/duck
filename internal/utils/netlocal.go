package utils

import (
	"bufio"
	"fmt"
	"math"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/IKauedev/duck/internal/cli"
)

func PerfCommand() cli.Command {
	return cli.Command{Name: "perf", Description: "Mede latencia HTTP, p95/p99 e throughput", Usage: "perf <url> [--requests N]", Run: perf}
}

func LoadCommand() cli.Command {
	return cli.Command{Name: "load", Description: "Executa teste de carga HTTP basico", Usage: "load <url> [--duration segundos] [--concurrency N]", Run: load}
}

func PortsCommand() cli.Command {
	return cli.Command{Name: "ports", Description: "Lista portas locais em uso e seus processos", Usage: "ports [--listen]", Run: ports}
}

func KillPortCommand() cli.Command {
	return cli.Command{Name: "kill-port", Description: "Mata processo usando uma porta local", Usage: "kill-port <porta>", Run: killPort}
}

func OpenCommand() cli.Command {
	return cli.Command{
		Name:        "open",
		Description: "Abre URLs e recursos uteis no navegador",
		Usage:       "open <url|local|swagger|github|aws-console|ingress> [args...]",
		Run:         openResource,
		Examples: []string{
			"open http://localhost:8080",
			"open local 3000",
			"open swagger 8080",
			"open github",
			"open aws-console ec2",
		},
	}
}

type perfOptions struct {
	url         string
	requests    int
	duration    time.Duration
	concurrency int
}

func perf(_ cli.Context, args []string) error {
	opts := perfOptions{requests: 20}
	rest := make([]string, 0, len(args))
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--requests", "-n":
			if i+1 >= len(args) {
				return cli.UsageError(args[i] + " precisa de um valor")
			}
			value, err := strconv.Atoi(args[i+1])
			if err != nil || value <= 0 {
				return cli.UsageError(args[i] + " precisa ser numero positivo")
			}
			opts.requests = value
			i++
		default:
			rest = append(rest, args[i])
		}
	}
	if len(rest) != 1 {
		return cli.UsageError("use: perf <url> [--requests N]")
	}
	opts.url = normalizeHTTPURL(rest[0])
	return runPerf(opts)
}

func load(_ cli.Context, args []string) error {
	opts := perfOptions{duration: 10 * time.Second, concurrency: 5}
	rest := make([]string, 0, len(args))
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--duration", "-d":
			if i+1 >= len(args) {
				return cli.UsageError(args[i] + " precisa de um valor")
			}
			seconds, err := strconv.Atoi(args[i+1])
			if err != nil || seconds <= 0 {
				return cli.UsageError(args[i] + " precisa ser numero positivo")
			}
			opts.duration = time.Duration(seconds) * time.Second
			i++
		case "--concurrency", "-c":
			if i+1 >= len(args) {
				return cli.UsageError(args[i] + " precisa de um valor")
			}
			value, err := strconv.Atoi(args[i+1])
			if err != nil || value <= 0 {
				return cli.UsageError(args[i] + " precisa ser numero positivo")
			}
			opts.concurrency = value
			i++
		default:
			rest = append(rest, args[i])
		}
	}
	if len(rest) != 1 {
		return cli.UsageError("use: load <url> [--duration segundos] [--concurrency N]")
	}
	opts.url = normalizeHTTPURL(rest[0])
	return runLoad(opts)
}

func runPerf(opts perfOptions) error {
	client := http.Client{Timeout: 30 * time.Second}
	latencies := make([]time.Duration, 0, opts.requests)
	var failures int
	start := time.Now()
	for i := 0; i < opts.requests; i++ {
		elapsed, err := requestOnce(client, opts.url)
		if err != nil {
			failures++
			continue
		}
		latencies = append(latencies, elapsed)
	}
	printHTTPStats(opts.url, time.Since(start), latencies, failures)
	return nil
}

func runLoad(opts perfOptions) error {
	client := http.Client{Timeout: 30 * time.Second}
	deadline := time.Now().Add(opts.duration)
	latencyCh := make(chan time.Duration, opts.concurrency*64)
	var failures int64
	var wg sync.WaitGroup
	start := time.Now()
	for i := 0; i < opts.concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for time.Now().Before(deadline) {
				elapsed, err := requestOnce(client, opts.url)
				if err != nil {
					atomic.AddInt64(&failures, 1)
					continue
				}
				latencyCh <- elapsed
			}
		}()
	}
	wg.Wait()
	close(latencyCh)
	latencies := make([]time.Duration, 0)
	for latency := range latencyCh {
		latencies = append(latencies, latency)
	}
	printHTTPStats(opts.url, time.Since(start), latencies, int(failures))
	return nil
}

func requestOnce(client http.Client, url string) (time.Duration, error) {
	start := time.Now()
	resp, err := client.Get(url)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 500 {
		return 0, fmt.Errorf("status %s", resp.Status)
	}
	return time.Since(start), nil
}

func printHTTPStats(url string, elapsed time.Duration, latencies []time.Duration, failures int) {
	sort.Slice(latencies, func(i, j int) bool { return latencies[i] < latencies[j] })
	total := len(latencies) + failures
	fmt.Println("URL:", url)
	fmt.Println("Requests:", total)
	fmt.Println("Sucesso:", len(latencies))
	fmt.Println("Falhas:", failures)
	fmt.Println("Tempo total:", elapsed.Round(time.Millisecond))
	if elapsed > 0 {
		fmt.Printf("Throughput: %.2f req/s\n", float64(len(latencies))/elapsed.Seconds())
	}
	if len(latencies) == 0 {
		return
	}
	fmt.Println("Min:", latencies[0].Round(time.Millisecond))
	fmt.Println("Media:", averageDuration(latencies).Round(time.Millisecond))
	fmt.Println("P95:", percentile(latencies, 95).Round(time.Millisecond))
	fmt.Println("P99:", percentile(latencies, 99).Round(time.Millisecond))
	fmt.Println("Max:", latencies[len(latencies)-1].Round(time.Millisecond))
}

func averageDuration(values []time.Duration) time.Duration {
	var total time.Duration
	for _, value := range values {
		total += value
	}
	return total / time.Duration(len(values))
}

func percentile(values []time.Duration, p float64) time.Duration {
	if len(values) == 0 {
		return 0
	}
	index := int(math.Ceil((p/100)*float64(len(values)))) - 1
	if index < 0 {
		index = 0
	}
	if index >= len(values) {
		index = len(values) - 1
	}
	return values[index]
}

func ports(_ cli.Context, args []string) error {
	listenOnly := false
	for _, arg := range args {
		switch arg {
		case "--listen":
			listenOnly = true
		default:
			return cli.UsageError("opcao invalida para ports: " + arg)
		}
	}
	entries, err := listPorts(listenOnly)
	if err != nil {
		return err
	}
	if len(entries) == 0 {
		fmt.Println("Nenhuma porta encontrada.")
		return nil
	}
	fmt.Printf("%-8s %-8s %-24s %s\n", "PROTO", "PORTA", "STATUS", "PID")
	for _, entry := range entries {
		fmt.Printf("%-8s %-8s %-24s %s\n", entry.proto, entry.port, entry.status, entry.pid)
	}
	return nil
}

func killPort(_ cli.Context, args []string) error {
	if len(args) != 1 {
		return cli.UsageError("use: kill-port <porta>")
	}
	entries, err := listPorts(false)
	if err != nil {
		return err
	}
	pids := map[string]bool{}
	for _, entry := range entries {
		if entry.port == args[0] && entry.pid != "" && entry.pid != "0" && entry.pid != "-" {
			pids[entry.pid] = true
		}
	}
	if len(pids) == 0 {
		return fmt.Errorf("nenhum processo encontrado na porta %s", args[0])
	}
	for pid := range pids {
		value, err := strconv.Atoi(pid)
		if err != nil {
			return err
		}
		process, err := os.FindProcess(value)
		if err != nil {
			return err
		}
		if err := process.Kill(); err != nil {
			return err
		}
		fmt.Println("Processo finalizado:", pid)
	}
	return nil
}

type portEntry struct {
	proto  string
	port   string
	status string
	pid    string
}

func listPorts(listenOnly bool) ([]portEntry, error) {
	if runtime.GOOS == "windows" {
		output, err := exec.Command("netstat", "-ano", "-p", "tcp").CombinedOutput()
		if err != nil {
			return nil, fmt.Errorf("%w: %s", err, strings.TrimSpace(string(output)))
		}
		return parseNetstatWindows(string(output), listenOnly), nil
	}
	if path, err := exec.LookPath("lsof"); err == nil {
		args := []string{"-nP", "-iTCP"}
		if listenOnly {
			args = append(args, "-sTCP:LISTEN")
		}
		output, err := exec.Command(path, args...).CombinedOutput()
		if err == nil {
			return parseLsof(string(output)), nil
		}
	}
	output, err := exec.Command("netstat", "-tulpn").CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("%w: %s", err, strings.TrimSpace(string(output)))
	}
	return parseNetstatUnix(string(output), listenOnly), nil
}

func parseNetstatWindows(output string, listenOnly bool) []portEntry {
	var entries []portEntry
	scanner := bufio.NewScanner(strings.NewReader(output))
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) < 5 || strings.ToUpper(fields[0]) != "TCP" {
			continue
		}
		status := fields[3]
		if listenOnly && status != "LISTENING" {
			continue
		}
		entries = append(entries, portEntry{proto: "tcp", port: portFromAddress(fields[1]), status: status, pid: fields[4]})
	}
	return entries
}

func parseLsof(output string) []portEntry {
	var entries []portEntry
	scanner := bufio.NewScanner(strings.NewReader(output))
	first := true
	for scanner.Scan() {
		if first {
			first = false
			continue
		}
		fields := strings.Fields(scanner.Text())
		if len(fields) < 9 {
			continue
		}
		pid := fields[1]
		name := fields[8]
		status := ""
		if len(fields) > 9 {
			status = strings.Trim(fields[9], "()")
		}
		entries = append(entries, portEntry{proto: "tcp", port: portFromAddress(name), status: status, pid: pid})
	}
	return entries
}

func parseNetstatUnix(output string, listenOnly bool) []portEntry {
	var entries []portEntry
	scanner := bufio.NewScanner(strings.NewReader(output))
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) < 7 || !strings.HasPrefix(fields[0], "tcp") {
			continue
		}
		status := fields[5]
		if listenOnly && status != "LISTEN" {
			continue
		}
		pid := strings.Split(fields[6], "/")[0]
		entries = append(entries, portEntry{proto: "tcp", port: portFromAddress(fields[3]), status: status, pid: pid})
	}
	return entries
}

func portFromAddress(address string) string {
	address = strings.TrimSuffix(address, "->")
	index := strings.LastIndex(address, ":")
	if index == -1 {
		return address
	}
	return strings.Trim(address[index+1:], "*")
}

func openResource(_ cli.Context, args []string) error {
	if len(args) == 0 {
		return cli.UsageError("use: open <url|local|swagger|github|aws-console|ingress> [args...]")
	}
	target := args[0]
	if strings.HasPrefix(target, "http://") || strings.HasPrefix(target, "https://") {
		return openURL(target)
	}
	switch target {
	case "local":
		port := "8080"
		if len(args) > 1 {
			port = args[1]
		}
		return openURL("http://localhost:" + port)
	case "swagger":
		port := "8080"
		if len(args) > 1 {
			port = args[1]
		}
		return openURL("http://localhost:" + port + "/swagger")
	case "github":
		url, err := githubRemoteURL()
		if err != nil {
			return err
		}
		return openURL(url)
	case "aws-console":
		service := ""
		if len(args) > 1 {
			service = args[1]
		}
		return openURL(awsConsoleURL(service))
	case "ingress":
		if len(args) > 1 {
			return openURL(ensureHTTP(args[1]))
		}
		host, err := firstIngressHost()
		if err != nil {
			return err
		}
		return openURL(ensureHTTP(host))
	default:
		return openURL(ensureHTTP(target))
	}
}

func openURL(url string) error {
	fmt.Println("Abrindo:", url)
	switch runtime.GOOS {
	case "windows":
		return exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
	case "darwin":
		return exec.Command("open", url).Start()
	default:
		return exec.Command("xdg-open", url).Start()
	}
}

func githubRemoteURL() (string, error) {
	output, err := exec.Command("git", "config", "--get", "remote.origin.url").CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("nao foi possivel ler remote.origin.url")
	}
	remote := strings.TrimSpace(string(output))
	if strings.HasPrefix(remote, "git@github.com:") {
		remote = strings.TrimSuffix(strings.TrimPrefix(remote, "git@github.com:"), ".git")
		return "https://github.com/" + remote, nil
	}
	if strings.HasPrefix(remote, "https://") {
		return strings.TrimSuffix(remote, ".git"), nil
	}
	return remote, nil
}

func awsConsoleURL(service string) string {
	region := os.Getenv("AWS_REGION")
	if region == "" {
		region = os.Getenv("AWS_DEFAULT_REGION")
	}
	if region == "" {
		region = "us-east-1"
	}
	if service == "" {
		return "https://console.aws.amazon.com/console/home?region=" + region
	}
	return "https://" + service + ".console.aws.amazon.com/" + service + "/home?region=" + region
}

func firstIngressHost() (string, error) {
	output, err := exec.Command("kubectl", "get", "ingress", "-A", "-o", "jsonpath={.items[0].spec.rules[0].host}").CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("%w: %s", err, strings.TrimSpace(string(output)))
	}
	host := strings.TrimSpace(string(output))
	if host == "" {
		return "", fmt.Errorf("nenhum ingress com host encontrado")
	}
	return host, nil
}

func normalizeHTTPURL(raw string) string {
	if strings.Contains(raw, "://") {
		return raw
	}
	return "http://" + raw
}

func ensureHTTP(raw string) string {
	return normalizeHTTPURL(raw)
}
