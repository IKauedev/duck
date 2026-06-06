package ops

import (
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/IKauedev/duck/internal/cli"
	"github.com/IKauedev/duck/internal/config"
	"github.com/IKauedev/duck/internal/history"
	"github.com/IKauedev/duck/internal/runner"
	"github.com/IKauedev/duck/internal/tui"
)

type service struct {
	cfg    config.Config
	runner runner.Runner
}

func DashboardCommand(cfg config.Config, run runner.Runner) cli.Command {
	svc := service{cfg: cfg, runner: run}
	return cli.Command{
		Name:        "dashboard",
		Description: "Abre dashboard compacto do Duck (alias de duck tui --compact)",
		Usage:       "dashboard [--text]",
		Run:         svc.dashboard,
		Examples: []string{
			"dashboard",
			"dashboard --text",
		},
	}
}

func LogsCommand(cfg config.Config, run runner.Runner) cli.Command {
	svc := service{cfg: cfg, runner: run}
	return cli.Command{
		Name:        "logs",
		Description: "Mostra logs de Docker, Compose, Kubernetes ou ECS",
		Usage:       "logs <docker|compose|kube|ecs|auto> [args...]",
		Run:         svc.logs,
		Examples: []string{
			"logs compose -f",
			"logs docker api --tail 100",
			"logs kube pod/api -n default",
			"logs ecs cluster service",
		},
	}
}

func TroubleshootCommand(cfg config.Config, run runner.Runner) cli.Command {
	svc := service{cfg: cfg, runner: run}
	return cli.Command{Name: "troubleshoot", Description: "Diagnostico rapido do ambiente e projeto", Usage: "troubleshoot [host port]", Run: svc.troubleshoot}
}

func DeployCommand(cfg config.Config, run runner.Runner) cli.Command {
	svc := service{cfg: cfg, runner: run}
	return cli.Command{
		Name:        "deploy",
		Description: "Fluxos simples de deploy para Compose, Kubernetes, ECR e ECS",
		Usage:       "deploy <compose|kube|ecr|ecs>",
		Children: []cli.Command{
			{Name: "compose", Description: "Executa docker compose up -d", Usage: "deploy compose [--build]", Run: svc.deployCompose},
			{Name: "kube", Description: "Executa kubectl apply -f", Usage: "deploy kube [arquivo|diretorio]", Run: svc.deployKube},
			{Name: "ecr", Description: "Builda, loga e envia imagem para ECR", Usage: "deploy ecr <repo-uri> <tag> [contexto]", Run: svc.deployECR},
			{Name: "ecs", Description: "Forca novo deploy de servico ECS", Usage: "deploy ecs <cluster> <service>", Run: svc.deployECS},
		},
	}
}

func MonitorCommand(cfg config.Config, run runner.Runner) cli.Command {
	svc := service{cfg: cfg, runner: run}
	return cli.Command{Name: "monitor", Description: "Painel em loop com containers, pods, recursos e portas", Usage: "monitor [--interval segundos] [--once]", Run: svc.monitor}
}

func AlertsCommand(cfg config.Config, run runner.Runner) cli.Command {
	svc := service{cfg: cfg, runner: run}
	return cli.Command{Name: "alerts", Description: "Checa containers unhealthy, pods com erro, nodes e ECS", Usage: "alerts [--ecs <cluster> <service>]", Run: svc.alerts}
}

func TraceCommand(cfg config.Config, run runner.Runner) cli.Command {
	svc := service{cfg: cfg, runner: run}
	return cli.Command{Name: "trace", Description: "Diagnostica DNS, TCP, HTTP, pod temporario e ingress", Usage: "trace <host> [port] [url] [-n namespace]", Run: svc.trace}
}

func LogsSearchCommand(cfg config.Config, run runner.Runner) cli.Command {
	svc := service{cfg: cfg, runner: run}
	return cli.Command{
		Name:        "logs-search",
		Description: "Busca texto em logs Docker, Compose, Kubernetes ou ECS",
		Usage:       "logs-search <docker|compose|kube|ecs> <termo> [args...]",
		Examples: []string{
			"logs-search docker error api --tail 200",
			"logs-search compose timeout",
			"logs-search kube exception pod/api -n default",
			"logs-search ecs failed cluster service",
		},
		Run: svc.logsSearch,
	}
}

func (s service) dashboard(_ cli.Context, args []string) error {
	textMode := false
	for _, arg := range args {
		switch arg {
		case "--text":
			textMode = true
		default:
			return cli.UsageError("opcao invalida para dashboard: " + arg)
		}
	}
	if !textMode {
		opts := tui.DefaultOptions()
		opts.Compact = true
		return tui.Run(s.cfg, s.runner, opts)
	}
	fmt.Println("Duck Dashboard")
	fmt.Println()
	printSection("Projeto")
	printDetected("Go", "go.mod")
	printDetected("Node.js", "package.json")
	printDetected("Python", "pyproject.toml", "requirements.txt")
	printDetected("Maven", "pom.xml")
	printDetected("Gradle", "build.gradle", "build.gradle.kts")
	printDetected("Docker Compose", "compose.yaml", "compose.yml", "docker-compose.yml", "docker-compose.yaml")
	printDetected("Kubernetes", "kustomization.yaml", "Chart.yaml")

	printSection("Configuracao")
	settings, err := config.LoadSettings()
	if err != nil {
		return err
	}
	for _, key := range []string{"profile.current", "aws.profile", "kube.namespace", "java.current", "node.current", "python.venv"} {
		if value := settings[key]; value != "" {
			fmt.Printf("%-16s %s\n", key, value)
		}
	}

	printSection("Ferramentas")
	s.printOutput("Docker", s.cfg.DockerBin, []string{"version", "--format", "{{.Client.Version}}"})
	s.printOutput("Kube context", s.cfg.KubectlBin, []string{"config", "current-context"})
	s.printOutput("AWS", s.cfg.AWSBin, []string{"--version"})

	printSection("Docker")
	s.printOutput("Containers", s.cfg.DockerBin, []string{"ps", "-a", "--format", "table {{.Names}}\t{{.Status}}\t{{.Ports}}"})

	printSection("Historico")
	entries, err := history.List()
	if err == nil && len(entries) > 0 {
		last := entries[len(entries)-1]
		fmt.Println(last.Time, "duck", last.Command)
	} else {
		fmt.Println("Nenhum comando recente.")
	}
	return nil
}

func (s service) logs(_ cli.Context, args []string) error {
	if len(args) == 0 || args[0] == "auto" {
		if hasCompose() {
			return s.runner.Run(s.cfg.DockerBin, []string{"compose", "logs"}, runner.InteractiveOptions())
		}
		return cli.UsageError("informe docker, compose, kube ou ecs")
	}
	mode, rest := args[0], args[1:]
	switch mode {
	case "docker":
		if len(rest) == 0 {
			return cli.UsageError("use: logs docker <container> [args...]")
		}
		return s.runner.Run(s.cfg.DockerBin, append([]string{"logs"}, rest...), runner.InteractiveOptions())
	case "compose":
		return s.runner.Run(s.cfg.DockerBin, append([]string{"compose", "logs"}, rest...), runner.InteractiveOptions())
	case "kube", "kubernetes":
		if len(rest) == 0 {
			return cli.UsageError("use: logs kube <pod|resource> [args...]")
		}
		return s.runner.Run(s.cfg.KubectlBin, append([]string{"logs"}, rest...), runner.InteractiveOptions())
	case "ecs":
		if len(rest) != 2 {
			return cli.UsageError("use: logs ecs <cluster> <service>")
		}
		return s.runner.Run(s.cfg.AWSBin, []string{"ecs", "describe-services", "--cluster", rest[0], "--services", rest[1], "--query", "services[0].events[0:10].[createdAt,message]", "--output", "table"}, runner.DefaultOptions())
	default:
		return cli.UsageError("backend invalido para logs: " + mode)
	}
}

func (s service) troubleshoot(_ cli.Context, args []string) error {
	if len(args) != 0 && len(args) != 2 {
		return cli.UsageError("use: troubleshoot [host port]")
	}
	printSection("Ferramentas")
	s.printOutput("Docker", s.cfg.DockerBin, []string{"version", "--format", "{{.Client.Version}}"})
	s.printOutput("Kube context", s.cfg.KubectlBin, []string{"config", "current-context"})
	s.printOutput("AWS identity", s.cfg.AWSBin, []string{"sts", "get-caller-identity", "--query", "Arn", "--output", "text"})

	printSection("Projeto")
	if hasCompose() {
		fmt.Println("Docker Compose detectado.")
		s.printOutput("Compose ps", s.cfg.DockerBin, []string{"compose", "ps"})
	} else {
		fmt.Println("Docker Compose nao detectado.")
	}
	if exists("kustomization.yaml") || exists("Chart.yaml") {
		fmt.Println("Kubernetes/Helm detectado.")
		s.printOutput("Pods com falha", s.cfg.KubectlBin, []string{"get", "pods", "--field-selector=status.phase!=Running,status.phase!=Succeeded"})
	}
	if len(args) == 2 {
		printSection("Rede")
		address := args[0] + ":" + args[1]
		fmt.Println("Alvo:", address)
		conn, err := net.DialTimeout("tcp", net.JoinHostPort(args[0], args[1]), 5*time.Second)
		if err != nil {
			fmt.Println("TCP local     indisponivel:", err)
		} else {
			_ = conn.Close()
			fmt.Println("TCP local     ok")
		}
	}
	return nil
}

func (s service) deployCompose(_ cli.Context, args []string) error {
	composeArgs := []string{"compose", "up", "-d"}
	for _, arg := range args {
		switch arg {
		case "--build":
			composeArgs = append(composeArgs, "--build")
		default:
			return cli.UsageError("opcao invalida para deploy compose: " + arg)
		}
	}
	return s.runner.Run(s.cfg.DockerBin, composeArgs, runner.InteractiveOptions())
}

func (s service) deployKube(_ cli.Context, args []string) error {
	if len(args) > 1 {
		return cli.UsageError("use: deploy kube [arquivo|diretorio]")
	}
	target := "."
	if len(args) == 1 {
		target = args[0]
	}
	return s.runner.Run(s.cfg.KubectlBin, []string{"apply", "-f", target}, runner.InteractiveOptions())
}

func (s service) deployECR(_ cli.Context, args []string) error {
	if len(args) < 2 || len(args) > 3 {
		return cli.UsageError("use: deploy ecr <repo-uri> <tag> [contexto]")
	}
	repo, tag := args[0], args[1]
	context := "."
	if len(args) == 3 {
		context = args[2]
	}
	image := repo + ":" + tag
	registry := strings.Split(repo, "/")[0]
	if registry == repo {
		return cli.UsageError("repo-uri deve incluir registry ECR")
	}
	password, err := s.runner.Output(s.cfg.AWSBin, []string{"ecr", "get-login-password"})
	if err != nil {
		return err
	}
	if err := s.runner.Run(s.cfg.DockerBin, []string{"login", "--username", "AWS", "--password-stdin", registry}, runner.Options{Stdin: strings.NewReader(password)}); err != nil {
		return err
	}
	if err := s.runner.Run(s.cfg.DockerBin, []string{"build", "-t", image, context}, runner.InteractiveOptions()); err != nil {
		return err
	}
	return s.runner.Run(s.cfg.DockerBin, []string{"push", image}, runner.InteractiveOptions())
}

func (s service) deployECS(_ cli.Context, args []string) error {
	if len(args) != 2 {
		return cli.UsageError("use: deploy ecs <cluster> <service>")
	}
	return s.runner.Run(s.cfg.AWSBin, []string{"ecs", "update-service", "--cluster", args[0], "--service", args[1], "--force-new-deployment"}, runner.DefaultOptions())
}

func (s service) monitor(_ cli.Context, args []string) error {
	interval := 5 * time.Second
	once := false
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--once":
			once = true
		case "--interval":
			if i+1 >= len(args) {
				return cli.UsageError("--interval precisa de um valor")
			}
			seconds, err := strconv.Atoi(args[i+1])
			if err != nil || seconds <= 0 {
				return cli.UsageError("--interval precisa ser numero positivo")
			}
			interval = time.Duration(seconds) * time.Second
			i++
		default:
			return cli.UsageError("opcao invalida para monitor: " + args[i])
		}
	}
	for {
		fmt.Print("\033[H\033[2J")
		fmt.Println("Duck Monitor", time.Now().Format(time.RFC3339))
		printSection("Docker containers")
		s.printRaw(s.cfg.DockerBin, []string{"ps", "--format", "table {{.Names}}\t{{.Status}}\t{{.Ports}}"})
		printSection("Docker stats")
		s.printRaw(s.cfg.DockerBin, []string{"stats", "--no-stream", "--format", "table {{.Name}}\t{{.CPUPerc}}\t{{.MemUsage}}\t{{.NetIO}}"})
		printSection("Kubernetes pods")
		s.printRaw(s.cfg.KubectlBin, []string{"get", "pods", "-A", "-o", "wide"})
		printSection("Kubernetes top pods")
		s.printRaw(s.cfg.KubectlBin, []string{"top", "pods", "-A"})
		if once {
			return nil
		}
		time.Sleep(interval)
	}
}

func (s service) alerts(_ cli.Context, args []string) error {
	ecsCluster := ""
	ecsService := ""
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--ecs":
			if i+2 >= len(args) {
				return cli.UsageError("use: alerts [--ecs <cluster> <service>]")
			}
			ecsCluster = args[i+1]
			ecsService = args[i+2]
			i += 2
		default:
			return cli.UsageError("opcao invalida para alerts: " + args[i])
		}
	}
	printSection("Docker unhealthy")
	s.printRaw(s.cfg.DockerBin, []string{"ps", "--filter", "health=unhealthy", "--format", "table {{.Names}}\t{{.Status}}\t{{.Image}}"})
	printSection("Kubernetes pods com erro")
	s.printRaw(s.cfg.KubectlBin, []string{"get", "pods", "-A", "--field-selector=status.phase!=Running,status.phase!=Succeeded"})
	printSection("Kubernetes status/restarts")
	s.printRaw(s.cfg.KubectlBin, []string{"get", "pods", "-A", "-o", "custom-columns=NAMESPACE:.metadata.namespace,NAME:.metadata.name,PHASE:.status.phase,READY:.status.containerStatuses[*].ready,RESTARTS:.status.containerStatuses[*].restartCount,REASON:.status.containerStatuses[*].state.waiting.reason"})
	printSection("Kubernetes nodes")
	s.printRaw(s.cfg.KubectlBin, []string{"get", "nodes", "-o", "custom-columns=NAME:.metadata.name,READY:.status.conditions[?(@.type==\"Ready\")].status,PRESSURE:.status.conditions[?(@.status==\"True\")].type"})
	if ecsCluster != "" {
		printSection("ECS service")
		s.printRaw(s.cfg.AWSBin, []string{"ecs", "describe-services", "--cluster", ecsCluster, "--services", ecsService, "--query", "services[0].{Status:status,Running:runningCount,Desired:desiredCount,Pending:pendingCount,Events:events[0:5].[createdAt,message]}", "--output", "table"})
	}
	return nil
}

func (s service) trace(_ cli.Context, args []string) error {
	opts, err := parseTraceArgs(args)
	if err != nil {
		return err
	}
	printSection("DNS local")
	addresses, err := net.LookupHost(opts.host)
	if err != nil {
		fmt.Println("Falha:", err)
	} else {
		fmt.Println(strings.Join(addresses, ", "))
	}
	if opts.port != "" {
		printSection("TCP local")
		conn, err := net.DialTimeout("tcp", net.JoinHostPort(opts.host, opts.port), 5*time.Second)
		if err != nil {
			fmt.Println("Falha:", err)
		} else {
			_ = conn.Close()
			fmt.Println("ok")
		}
	}
	if opts.url != "" {
		printSection("HTTP local")
		client := http.Client{Timeout: 10 * time.Second}
		resp, err := client.Get(opts.url)
		if err != nil {
			fmt.Println("Falha:", err)
		} else {
			defer resp.Body.Close()
			fmt.Println(resp.Status)
		}
	}
	printSection("DNS no cluster")
	dnsArgs := appendNamespace([]string{"run", "duck-trace-dns", "--rm", "-i", "--restart=Never", "--image=busybox:1.36"}, opts.namespace)
	dnsArgs = append(dnsArgs, "--", "nslookup", opts.host)
	s.printRaw(s.cfg.KubectlBin, dnsArgs)
	if opts.port != "" {
		printSection("TCP no cluster")
		tcpArgs := appendNamespace([]string{"run", "duck-trace-tcp", "--rm", "-i", "--restart=Never", "--image=busybox:1.36"}, opts.namespace)
		tcpArgs = append(tcpArgs, "--", "sh", "-c", "nc -vz "+opts.host+" "+opts.port)
		s.printRaw(s.cfg.KubectlBin, tcpArgs)
	}
	printSection("Ingress")
	output, err := s.runner.Output(s.cfg.KubectlBin, []string{"get", "ingress", "-A"})
	if err != nil {
		fmt.Println(strings.TrimSpace(output))
	} else {
		printMatchingLines(output, opts.host)
	}
	return nil
}

func (s service) logsSearch(_ cli.Context, args []string) error {
	if len(args) < 2 {
		return cli.UsageError("use: logs-search <docker|compose|kube|ecs> <termo> [args...]")
	}
	mode, term, rest := args[0], args[1], args[2:]
	var output string
	var err error
	switch mode {
	case "docker":
		if len(rest) == 0 {
			return cli.UsageError("use: logs-search docker <termo> <container> [args...]")
		}
		output, err = s.runner.Output(s.cfg.DockerBin, append([]string{"logs"}, rest...))
	case "compose":
		output, err = s.runner.Output(s.cfg.DockerBin, append([]string{"compose", "logs"}, rest...))
	case "kube", "kubernetes":
		if len(rest) == 0 {
			return cli.UsageError("use: logs-search kube <termo> <pod|resource> [args...]")
		}
		output, err = s.runner.Output(s.cfg.KubectlBin, append([]string{"logs"}, rest...))
	case "ecs":
		if len(rest) != 2 {
			return cli.UsageError("use: logs-search ecs <termo> <cluster> <service>")
		}
		output, err = s.runner.Output(s.cfg.AWSBin, []string{"ecs", "describe-services", "--cluster", rest[0], "--services", rest[1], "--query", "services[0].events[].message", "--output", "text"})
	default:
		return cli.UsageError("backend invalido para logs-search: " + mode)
	}
	if err != nil && strings.TrimSpace(output) == "" {
		return err
	}
	printMatchingLines(output, term)
	return nil
}

func (s service) printOutput(label, binary string, args []string) {
	output, err := s.runner.Output(binary, args)
	output = strings.TrimSpace(output)
	if err != nil {
		if output == "" {
			output = err.Error()
		}
		fmt.Printf("%-14s indisponivel: %s\n", label, output)
		return
	}
	if output == "" {
		output = "ok"
	}
	fmt.Printf("%-14s %s\n", label, output)
}

func (s service) printRaw(binary string, args []string) {
	output, err := s.runner.Output(binary, args)
	output = strings.TrimSpace(output)
	if err != nil {
		if output == "" {
			output = err.Error()
		}
		fmt.Println("indisponivel:", output)
		return
	}
	if output == "" {
		fmt.Println("sem resultados")
		return
	}
	fmt.Println(output)
}

type traceOptions struct {
	host      string
	port      string
	url       string
	namespace string
}

func parseTraceArgs(args []string) (traceOptions, error) {
	opts := traceOptions{}
	rest := make([]string, 0, len(args))
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "-n", "--namespace":
			if i+1 >= len(args) {
				return opts, cli.UsageError(args[i] + " precisa de um valor")
			}
			opts.namespace = args[i+1]
			i++
		default:
			rest = append(rest, args[i])
		}
	}
	if len(rest) < 1 || len(rest) > 3 {
		return opts, cli.UsageError("use: trace <host> [port] [url] [-n namespace]")
	}
	opts.host = rest[0]
	if len(rest) >= 2 {
		opts.port = rest[1]
	}
	if len(rest) == 3 {
		opts.url = rest[2]
	} else if opts.port != "" {
		opts.url = "http://" + net.JoinHostPort(opts.host, opts.port)
	}
	return opts, nil
}

func appendNamespace(args []string, namespace string) []string {
	if namespace == "" {
		return args
	}
	return append(args, "-n", namespace)
}

func printMatchingLines(output, term string) {
	termLower := strings.ToLower(term)
	found := false
	for _, line := range strings.Split(output, "\n") {
		if strings.Contains(strings.ToLower(line), termLower) {
			fmt.Println(line)
			found = true
		}
	}
	if !found {
		fmt.Println("Nenhuma linha encontrada para:", term)
	}
}

func printSection(title string) {
	fmt.Println()
	fmt.Println(title + ":")
}

func printDetected(name string, files ...string) {
	for _, file := range files {
		if exists(file) {
			fmt.Printf("%-16s %s\n", name, file)
			return
		}
	}
}

func hasCompose() bool {
	return exists("compose.yaml") || exists("compose.yml") || exists("docker-compose.yml") || exists("docker-compose.yaml")
}

func exists(path string) bool {
	_, err := os.Stat(filepath.Clean(path))
	return err == nil
}
