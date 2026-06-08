package app

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode/utf16"

	"github.com/IKauedev/duck/internal/aws"
	"github.com/IKauedev/duck/internal/buildtools"
	"github.com/IKauedev/duck/internal/cli"
	"github.com/IKauedev/duck/internal/config"
	"github.com/IKauedev/duck/internal/docker"
	"github.com/IKauedev/duck/internal/envcheck"
	"github.com/IKauedev/duck/internal/gittools"
	"github.com/IKauedev/duck/internal/golang"
	"github.com/IKauedev/duck/internal/history"
	"github.com/IKauedev/duck/internal/install"
	"github.com/IKauedev/duck/internal/java"
	"github.com/IKauedev/duck/internal/kubernetes"
	"github.com/IKauedev/duck/internal/netcheck"
	"github.com/IKauedev/duck/internal/node"
	"github.com/IKauedev/duck/internal/ops"
	"github.com/IKauedev/duck/internal/project"
	"github.com/IKauedev/duck/internal/python"
	"github.com/IKauedev/duck/internal/runner"
	"github.com/IKauedev/duck/internal/selfupdate"
	"github.com/IKauedev/duck/internal/tui"
	"github.com/IKauedev/duck/internal/utils"
	"github.com/IKauedev/duck/internal/version"
	"github.com/IKauedev/duck/internal/wsl"
)

const Name = "duck"

func Run(args []string) int {
	cfg := config.Load()
	run := runner.New()
	if len(args) > 0 && args[0] == "--dry-run" {
		run = runner.NewDryRun()
		args = args[1:]
	}
	args = expandAliasArgs(args)
	commands := Commands(cfg, run)
	_ = history.Record(args)
	return cli.Run(Name, commands, args)
}

func Commands(cfg config.Config, run runner.Runner) []cli.Command {
	commands := []cli.Command{
		{
			Name:        "init",
			Description: "Configura o Duck no usuario atual",
			Usage:       "init",
			Run:         initDuck(cfg, run),
		},
		{
			Name:        "config",
			Description: "Mostra ou altera configuracoes do Duck",
			Usage:       "config <show|path|set|get|edit>",
			Run:         configCommand,
		},
		{Name: "profile", Description: "Gerencia perfis de ambiente", Usage: "profile <save|use|list|show|remove>", Run: profileCommand},
		{Name: "task", Description: "Gerencia tarefas customizadas", Usage: "task <add|run|list|remove>", Run: taskCommand},
		{Name: "aliases", Description: "Gerencia aliases customizados", Usage: "aliases <add|list|remove>", Run: aliasesCommand},
		{Name: "explain", Description: "Mostra expansao de comando Duck", Usage: "explain <comando...>", Run: explainCommand},
		{Name: "last", Description: "Repete o ultimo comando do historico", Usage: "last", Run: lastCommand},
		{Name: "recent", Description: "Mostra e executa comandos recentes", Usage: "recent [run N|top N]", Run: recentCommand},
		{Name: "favorites", Description: "Gerencia comandos favoritos", Usage: "favorites <add|run|list|remove>", Run: favoritesCommand},
		{Name: "palette", Aliases: []string{"command-palette"}, Description: "Busca e executa comandos por texto livre", Usage: "palette [termo]", Run: paletteCommand},
		{Name: "watch", Description: "Executa comando em loop", Usage: "watch [--interval segundos] <comando...>", Run: watchCommand},
		ops.DashboardCommand(cfg, run),
		ops.LogsCommand(cfg, run),
		ops.TroubleshootCommand(cfg, run),
		ops.DeployCommand(cfg, run),
		ops.MonitorCommand(cfg, run),
		ops.AlertsCommand(cfg, run),
		ops.TraceCommand(cfg, run),
		ops.LogsSearchCommand(cfg, run),
		utils.EncryptCommand(),
		utils.DecryptCommand(),
		utils.PasswordCommand(),
		utils.QRCommand(),
		utils.ServeCommand(),
		utils.ZipCommand(),
		utils.UnzipCommand(),
		utils.FindCommand(),
		utils.PerfCommand(),
		utils.LoadCommand(),
		utils.PortsCommand(),
		utils.KillPortCommand(),
		utils.OpenCommand(),
		utils.CIDRCommand(),
		utils.CalcCommand(),
		utils.JSONCommand(),
		utils.YAMLCommand(),
		gittools.Command(cfg, run),
		{
			Name:        "status",
			Description: "Mostra status das ferramentas usadas pelo Duck",
			Usage:       "status [--json]",
			Run:         status(cfg, run),
		},
		{
			Name:        "doctor",
			Description: "Diagnostica ferramentas e mostra sugestoes",
			Usage:       "doctor",
			Run:         doctor(cfg, run),
		},
		{
			Name:        "version",
			Description: "Mostra versao do Duck",
			Usage:       "version",
			Run:         showVersion,
		},
		{
			Name:        "update",
			Description: "Atualiza o Duck a partir do GitHub Releases",
			Usage:       "update",
			Run:         update(cfg, run),
		},
		{
			Name:        "completion",
			Aliases:     []string{"autocomplete"},
			Description: "Gera autocomplete para shell",
			Usage:       "completion [install] <bash|zsh|powershell>",
			Run:         completion,
		},
		{
			Name:        "history",
			Description: "Mostra comandos executados anteriormente",
			Usage:       "history [--limit N|--all|--clear|--path]",
			Run:         commandHistory,
		},
		{
			Name:        "terminal",
			Aliases:     []string{"console", "repl"},
			Description: "Abre um terminal interativo do Duck",
			Usage:       "terminal",
			Run:         terminal(cfg, run),
		},
		tui.Command(cfg, run),
		install.Command(),
		install.SetupCommand(),
		wsl.Command(cfg, run),
		docker.Command(cfg, run),
		golang.Command(cfg, run),
		java.Command(cfg, run),
		node.Command(cfg, run),
		python.Command(cfg, run),
		buildtools.Maven(run),
		buildtools.Gradle(run),
		buildtools.NPM(run),
		buildtools.PNPM(run),
		netcheck.CurlCommand(),
		netcheck.PortCommand(),
		kubernetes.Command(cfg, run),
		aws.Command(cfg, run),
		envcheck.Command(),
		project.Command(),
	}

	commands = append(commands, docker.LegacyCommands(cfg, run)...)
	return commands
}

func status(cfg config.Config, run runner.Runner) func(cli.Context, []string) error {
	return func(ctx cli.Context, args []string) error {
		jsonOutput := false
		for _, arg := range args {
			switch arg {
			case "--json":
				jsonOutput = true
			default:
				return cli.UsageError("opcao invalida para status: " + arg)
			}
		}

		statuses := toolStatuses(cfg, run)
		if jsonOutput {
			encoder := json.NewEncoder(ctx.Stdout)
			encoder.SetIndent("", "  ")
			return encoder.Encode(statuses)
		}

		printDuckVersion()
		for _, status := range statuses {
			printToolStatus(status)
		}
		return nil
	}
}

func printDuckVersion() {
	fmt.Printf("%-12s ok: %s (commit %s, build %s)\n", "Duck", version.Label(), version.Commit, version.Date)
}

func initDuck(cfg config.Config, run runner.Runner) func(cli.Context, []string) error {
	return func(_ cli.Context, args []string) error {
		if len(args) > 0 {
			return cli.UsageError("init nao recebe argumentos")
		}

		fmt.Println("Inicializando Duck...")
		if err := doctor(cfg, run)(cli.Context{}, nil); err != nil {
			return err
		}

		configPath, err := config.SettingsPath()
		if err != nil {
			return err
		}
		historyPath, err := history.Path()
		if err != nil {
			return err
		}
		fmt.Println("Config:", configPath)
		fmt.Println("Historico:", historyPath)
		fmt.Println("Dica: use 'duck completion install powershell|bash|zsh' para instalar autocomplete.")
		return nil
	}
}

func configCommand(_ cli.Context, args []string) error {
	if len(args) == 0 {
		args = []string{"show"}
	}

	switch args[0] {
	case "show":
		if len(args) != 1 {
			return cli.UsageError("use: config show")
		}
		settings, err := config.LoadSettings()
		if err != nil {
			return err
		}
		config.PrintSettings(settings)
	case "path":
		if len(args) != 1 {
			return cli.UsageError("use: config path")
		}
		path, err := config.SettingsPath()
		if err != nil {
			return err
		}
		fmt.Println(path)
	case "set":
		if len(args) != 3 {
			return cli.UsageError("use: config set <chave> <valor>")
		}
		if err := config.SetSetting(args[1], args[2]); err != nil {
			return err
		}
		fmt.Println("Configurado:", args[1])
	case "get":
		if len(args) != 2 {
			return cli.UsageError("use: config get <chave>")
		}
		settings, err := config.LoadSettings()
		if err != nil {
			return err
		}
		fmt.Println(settings[args[1]])
	case "edit":
		if len(args) != 1 {
			return cli.UsageError("use: config edit")
		}
		path, err := config.SettingsPath()
		if err != nil {
			return err
		}
		editor := os.Getenv("EDITOR")
		if editor == "" {
			if runtime.GOOS == "windows" {
				editor = "notepad"
			} else {
				editor = "vi"
			}
		}
		cmd := exec.Command(editor, path)
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		return cmd.Run()
	default:
		return cli.UsageError("subcomando invalido para config: " + args[0])
	}
	return nil
}

func profileCommand(_ cli.Context, args []string) error {
	if len(args) == 0 {
		args = []string{"list"}
	}
	switch args[0] {
	case "save":
		if len(args) != 2 {
			return cli.UsageError("use: profile save <nome>")
		}
		settings, err := config.LoadSettings()
		if err != nil {
			return err
		}
		for key, value := range settings {
			if isProfileKey(key) {
				settings["profile."+args[1]+"."+key] = value
			}
		}
		return config.SaveSettings(settings)
	case "use":
		if len(args) != 2 {
			return cli.UsageError("use: profile use <nome>")
		}
		return useProfile(args[1])
	case "list":
		settings, err := config.LoadSettings()
		if err != nil {
			return err
		}
		seen := map[string]bool{}
		for key := range settings {
			if strings.HasPrefix(key, "profile.") {
				parts := strings.Split(key, ".")
				if len(parts) >= 3 {
					seen[parts[1]] = true
				}
			}
		}
		if len(seen) == 0 {
			fmt.Println("Nenhum profile salvo.")
		}
		for name := range seen {
			fmt.Println(name)
		}
	case "show":
		if len(args) != 2 {
			return cli.UsageError("use: profile show <nome>")
		}
		return showPrefix("profile." + args[1] + ".")
	case "remove":
		if len(args) != 2 {
			return cli.UsageError("use: profile remove <nome>")
		}
		return removePrefix("profile." + args[1] + ".")
	default:
		return cli.UsageError("subcomando invalido para profile: " + args[0])
	}
	return nil
}

func taskCommand(_ cli.Context, args []string) error {
	if len(args) == 0 {
		args = []string{"list"}
	}
	switch args[0] {
	case "add":
		if len(args) < 3 {
			return cli.UsageError("use: task add <nome> <comando...>")
		}
		return config.SetSetting("task."+args[1], strings.Join(args[2:], " "))
	case "run":
		if len(args) != 2 {
			return cli.UsageError("use: task run <nome>")
		}
		settings, err := config.LoadSettings()
		if err != nil {
			return err
		}
		command := settings["task."+args[1]]
		if command == "" {
			return cli.UsageError("task nao encontrada: " + args[1])
		}
		parsed, err := parseTerminalLine(command)
		if err != nil {
			return err
		}
		return runInline(parsed)
	case "list":
		return showPrefix("task.")
	case "remove":
		if len(args) != 2 {
			return cli.UsageError("use: task remove <nome>")
		}
		return removeKey("task." + args[1])
	default:
		return cli.UsageError("subcomando invalido para task: " + args[0])
	}
}

func aliasesCommand(_ cli.Context, args []string) error {
	if len(args) == 0 {
		args = []string{"list"}
	}
	switch args[0] {
	case "add":
		if len(args) < 3 {
			return cli.UsageError("use: aliases add <nome> <comando...>")
		}
		return config.SetSetting("alias."+args[1], strings.Join(args[2:], " "))
	case "list":
		return showPrefix("alias.")
	case "remove":
		if len(args) != 2 {
			return cli.UsageError("use: aliases remove <nome>")
		}
		return removeKey("alias." + args[1])
	default:
		return cli.UsageError("subcomando invalido para aliases: " + args[0])
	}
}

func explainCommand(_ cli.Context, args []string) error {
	if len(args) == 0 {
		return cli.UsageError("informe um comando para explicar")
	}
	expanded := expandAliasArgs(args)
	fmt.Println("duck", strings.Join(expanded, " "))
	return nil
}

func lastCommand(_ cli.Context, args []string) error {
	if len(args) > 0 {
		return cli.UsageError("last nao recebe argumentos")
	}
	entries, err := history.List()
	if err != nil {
		return err
	}
	if len(entries) == 0 {
		return cli.UsageError("historico vazio")
	}
	parsed, err := parseTerminalLine(entries[len(entries)-1].Command)
	if err != nil {
		return err
	}
	return runInline(parsed)
}

func recentCommand(_ cli.Context, args []string) error {
	entries, err := history.List()
	if err != nil {
		return err
	}
	if len(entries) == 0 {
		fmt.Println("Nenhum comando no historico.")
		return nil
	}
	switch {
	case len(args) == 0:
		start := 0
		if len(entries) > 10 {
			start = len(entries) - 10
		}
		for i, entry := range entries[start:] {
			fmt.Printf("%4d  %s  duck %s\n", start+i+1, entry.Time, entry.Command)
		}
		return nil
	case len(args) == 2 && args[0] == "run":
		return historyRun(args[1])
	case len(args) == 2 && args[0] == "top":
		limit, err := strconv.Atoi(args[1])
		if err != nil || limit <= 0 {
			return cli.UsageError("use: recent top <N>")
		}
		counts := map[string]int{}
		for _, entry := range entries {
			counts[entry.Command]++
		}
		type item struct {
			command string
			count   int
		}
		items := make([]item, 0, len(counts))
		for command, count := range counts {
			items = append(items, item{command: command, count: count})
		}
		sort.Slice(items, func(i, j int) bool { return items[i].count > items[j].count })
		if limit > len(items) {
			limit = len(items)
		}
		for i := 0; i < limit; i++ {
			fmt.Printf("%4d  (%d) duck %s\n", i+1, items[i].count, items[i].command)
		}
		return nil
	default:
		return cli.UsageError("use: recent [run N|top N]")
	}
}

func favoritesCommand(_ cli.Context, args []string) error {
	if len(args) == 0 {
		args = []string{"list"}
	}
	switch args[0] {
	case "add":
		if len(args) < 3 {
			return cli.UsageError("use: favorites add <nome> <comando...>")
		}
		return config.SetSetting("favorite."+args[1], strings.Join(args[2:], " "))
	case "run":
		if len(args) != 2 {
			return cli.UsageError("use: favorites run <nome>")
		}
		settings, err := config.LoadSettings()
		if err != nil {
			return err
		}
		command := settings["favorite."+args[1]]
		if command == "" {
			return cli.UsageError("favorito nao encontrado: " + args[1])
		}
		parsed, err := parseTerminalLine(command)
		if err != nil {
			return err
		}
		return runInline(parsed)
	case "list":
		return showPrefix("favorite.")
	case "remove":
		if len(args) != 2 {
			return cli.UsageError("use: favorites remove <nome>")
		}
		return removeKey("favorite." + args[1])
	default:
		return cli.UsageError("subcomando invalido para favorites: " + args[0])
	}
}

func paletteCommand(_ cli.Context, args []string) error {
	query := strings.TrimSpace(strings.Join(args, " "))
	if query == "" {
		fmt.Print("Buscar comando: ")
		reader := bufio.NewReader(os.Stdin)
		line, err := reader.ReadString('\n')
		if err != nil {
			return err
		}
		query = strings.TrimSpace(line)
	}
	if query == "" {
		return cli.UsageError("informe um termo para buscar")
	}
	candidates := commandCatalog(Commands(config.Load(), runner.New()))
	type match struct {
		path  string
		score int
	}
	matches := make([]match, 0)
	lowerQuery := strings.ToLower(query)
	for _, candidate := range candidates {
		score := scoreMatch(lowerQuery, strings.ToLower(candidate))
		if score > 0 {
			matches = append(matches, match{path: candidate, score: score})
		}
	}
	if len(matches) == 0 {
		fmt.Println("Nenhum comando encontrado para:", query)
		return nil
	}
	sort.Slice(matches, func(i, j int) bool { return matches[i].score > matches[j].score })
	limit := 10
	if len(matches) < limit {
		limit = len(matches)
	}
	fmt.Println("Resultados:")
	for i := 0; i < limit; i++ {
		fmt.Printf("  %d) duck %s\n", i+1, matches[i].path)
	}
	fmt.Print("Escolha um numero para executar (enter cancela): ")
	reader := bufio.NewReader(os.Stdin)
	line, err := reader.ReadString('\n')
	if err != nil {
		return err
	}
	line = strings.TrimSpace(line)
	if line == "" {
		fmt.Println("Cancelado.")
		return nil
	}
	index, err := strconv.Atoi(line)
	if err != nil || index <= 0 || index > limit {
		return cli.UsageError("indice invalido")
	}
	parsed, err := parseTerminalLine(matches[index-1].path)
	if err != nil {
		return err
	}
	return runInline(parsed)
}

func commandCatalog(commands []cli.Command) []string {
	out := make([]string, 0)
	var walk func([]cli.Command, []string)
	walk = func(items []cli.Command, prefix []string) {
		for _, item := range items {
			current := append(prefix, item.Name)
			if len(item.Children) == 0 {
				out = append(out, strings.Join(current, " "))
				continue
			}
			walk(item.Children, current)
		}
	}
	walk(commands, nil)
	return out
}

func scoreMatch(query, candidate string) int {
	if query == candidate {
		return 100
	}
	if strings.HasPrefix(candidate, query) {
		return 50
	}
	if strings.Contains(candidate, query) {
		return 20
	}
	score := 0
	cursor := 0
	for _, char := range query {
		pos := strings.IndexRune(candidate[cursor:], char)
		if pos < 0 {
			continue
		}
		score++
		cursor += pos + 1
		if cursor >= len(candidate) {
			break
		}
	}
	return score
}

func watchCommand(_ cli.Context, args []string) error {
	interval := 5 * time.Second
	commandStart := 0
	if len(args) >= 2 && args[0] == "--interval" {
		seconds, err := strconv.Atoi(args[1])
		if err != nil || seconds <= 0 {
			return cli.UsageError("--interval precisa ser numero positivo")
		}
		interval = time.Duration(seconds) * time.Second
		commandStart = 2
	}
	if commandStart >= len(args) {
		return cli.UsageError("use: watch [--interval segundos] <comando...>")
	}
	for {
		fmt.Println("==>", time.Now().Format(time.RFC3339), "duck", strings.Join(args[commandStart:], " "))
		_ = runInline(args[commandStart:])
		time.Sleep(interval)
	}
}

func expandAliasArgs(args []string) []string {
	if len(args) == 0 {
		return args
	}
	settings, err := config.LoadSettings()
	if err != nil {
		return args
	}
	alias := settings["alias."+args[0]]
	if alias == "" {
		return args
	}
	parsed, err := parseTerminalLine(alias)
	if err != nil {
		return args
	}
	return append(parsed, args[1:]...)
}

func useProfile(name string) error {
	settings, err := config.LoadSettings()
	if err != nil {
		return err
	}
	prefix := "profile." + name + "."
	found := false
	for key, value := range settings {
		if strings.HasPrefix(key, prefix) {
			settings[strings.TrimPrefix(key, prefix)] = value
			found = true
		}
	}
	if !found {
		return cli.UsageError("profile nao encontrado: " + name)
	}
	settings["profile.current"] = name
	return config.SaveSettings(settings)
}

func isProfileKey(key string) bool {
	return key == "aws.profile" || key == "kube.namespace" || key == "java.current" || key == "node.current" || key == "python.venv"
}

func showPrefix(prefix string) error {
	settings, err := config.LoadSettings()
	if err != nil {
		return err
	}
	found := false
	for key, value := range settings {
		if strings.HasPrefix(key, prefix) {
			fmt.Printf("%s=%s\n", strings.TrimPrefix(key, prefix), value)
			found = true
		}
	}
	if !found {
		fmt.Println("Nenhum item encontrado.")
	}
	return nil
}

func removePrefix(prefix string) error {
	settings, err := config.LoadSettings()
	if err != nil {
		return err
	}
	for key := range settings {
		if strings.HasPrefix(key, prefix) {
			delete(settings, key)
		}
	}
	return config.SaveSettings(settings)
}

func removeKey(key string) error {
	settings, err := config.LoadSettings()
	if err != nil {
		return err
	}
	delete(settings, key)
	return config.SaveSettings(settings)
}

func doctor(cfg config.Config, run runner.Runner) func(cli.Context, []string) error {
	return func(_ cli.Context, args []string) error {
		if len(args) > 0 {
			return cli.UsageError("doctor nao recebe argumentos")
		}

		printDuckVersion()
		fmt.Println()
		statuses := toolStatuses(cfg, run)
		allOK := true
		for _, status := range statuses {
			printToolStatus(status)
			if !status.OK {
				allOK = false
				fmt.Println("  Sugestao:", status.Suggestion)
			}
		}
		if allOK {
			fmt.Println("Tudo certo.")
		}
		return nil
	}
}

func showVersion(_ cli.Context, args []string) error {
	if len(args) > 0 {
		return cli.UsageError("version nao recebe argumentos")
	}
	fmt.Println(version.Details())
	return nil
}

func update(cfg config.Config, run runner.Runner) func(cli.Context, []string) error {
	return func(ctx cli.Context, args []string) error {
		if len(args) > 0 {
			return cli.UsageError("update nao recebe argumentos")
		}
		return selfupdate.Run(ctx.Stdout, version.Label())
	}
}

func commandHistory(_ cli.Context, args []string) error {
	limit := 50
	showAll := false
	jsonOutput := false
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--all":
			showAll = true
		case "--json":
			jsonOutput = true
		case "search":
			if i+1 >= len(args) {
				return cli.UsageError("use: history search <termo>")
			}
			return historySearch(args[i+1])
		case "run":
			if i+1 >= len(args) {
				return cli.UsageError("use: history run <numero>")
			}
			return historyRun(args[i+1])
		case "--clear":
			if len(args) > 1 {
				return cli.UsageError("--clear nao aceita outras opcoes")
			}
			if err := history.Clear(); err != nil {
				return err
			}
			fmt.Println("Historico limpo.")
			return nil
		case "--path":
			if len(args) > 1 {
				return cli.UsageError("--path nao aceita outras opcoes")
			}
			path, err := history.Path()
			if err != nil {
				return err
			}
			fmt.Println(path)
			return nil
		case "--limit", "-n":
			if i+1 >= len(args) {
				return cli.UsageError(args[i] + " precisa de um valor")
			}
			value, err := strconv.Atoi(args[i+1])
			if err != nil || value <= 0 {
				return cli.UsageError(args[i] + " precisa ser um numero positivo")
			}
			limit = value
			i++
		default:
			return cli.UsageError("opcao invalida para history: " + args[i])
		}
	}

	entries, err := history.List()
	if err != nil {
		return err
	}
	if len(entries) == 0 {
		fmt.Println("Nenhum comando no historico.")
		return nil
	}
	if jsonOutput {
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")
		return encoder.Encode(entries)
	}

	start := 0
	if !showAll && len(entries) > limit {
		start = len(entries) - limit
	}
	for index, entry := range entries[start:] {
		fmt.Printf("%4d  %s  duck %s\n", start+index+1, entry.Time, entry.Command)
	}
	return nil
}

func historySearch(term string) error {
	entries, err := history.List()
	if err != nil {
		return err
	}
	term = strings.ToLower(term)
	for index, entry := range entries {
		if strings.Contains(strings.ToLower(entry.Command), term) {
			fmt.Printf("%4d  %s  duck %s\n", index+1, entry.Time, entry.Command)
		}
	}
	return nil
}

func historyRun(value string) error {
	index, err := strconv.Atoi(value)
	if err != nil || index <= 0 {
		return cli.UsageError("numero de historico invalido")
	}
	entries, err := history.List()
	if err != nil {
		return err
	}
	if index > len(entries) {
		return cli.UsageError("entrada de historico nao encontrada")
	}
	args, err := parseTerminalLine(entries[index-1].Command)
	if err != nil {
		return err
	}
	return runInline(args)
}

func terminal(cfg config.Config, run runner.Runner) func(cli.Context, []string) error {
	return func(_ cli.Context, args []string) error {
		if len(args) > 0 {
			return cli.UsageError("terminal nao recebe argumentos")
		}

		fmt.Println("Duck terminal. Digite 'help' para ajuda ou 'exit' para sair.")
		scanner := bufio.NewScanner(os.Stdin)
		for {
			fmt.Print(terminalPrompt())
			if !scanner.Scan() {
				fmt.Println()
				return scanner.Err()
			}

			line := strings.TrimSpace(scanner.Text())
			if line == "" {
				continue
			}
			switch strings.ToLower(line) {
			case "exit", "quit":
				return nil
			case "pwd":
				wd, err := os.Getwd()
				if err != nil {
					fmt.Println("Erro:", err)
				} else {
					fmt.Println(wd)
				}
				continue
			case "clear", "cls":
				fmt.Print("\033[H\033[2J")
				continue
			}
			if strings.HasPrefix(line, "cd ") {
				dir := strings.TrimSpace(strings.TrimPrefix(line, "cd "))
				if err := os.Chdir(dir); err != nil {
					fmt.Println("Erro:", err)
				}
				continue
			}
			if strings.HasPrefix(line, "!") {
				if err := historyRun(strings.TrimPrefix(line, "!")); err != nil {
					fmt.Println("Erro:", err)
				}
				continue
			}

			parsed, err := parseTerminalLine(line)
			if err != nil {
				fmt.Println("Erro:", err)
				continue
			}
			if len(parsed) == 0 {
				continue
			}
			if parsed[0] == Name {
				parsed = parsed[1:]
			}
			if len(parsed) == 0 {
				continue
			}
			if parsed[0] == "terminal" || parsed[0] == "console" || parsed[0] == "repl" {
				fmt.Println("Voce ja esta no terminal Duck.")
				continue
			}

			_ = history.Record(parsed)
			code := cli.Run(Name, Commands(cfg, run), parsed)
			if code != 0 {
				fmt.Println("Comando finalizado com erro:", code)
			}
		}
	}
}

func terminalPrompt() string {
	settings, _ := config.LoadSettings()
	awsProfile := settings["aws.profile"]
	kubeNamespace := settings["kube.namespace"]
	parts := make([]string, 0, 2)
	if awsProfile != "" {
		parts = append(parts, "aws:"+awsProfile)
	}
	if kubeNamespace != "" {
		parts = append(parts, "ns:"+kubeNamespace)
	}
	if len(parts) == 0 {
		return "duck> "
	}
	return "duck[" + strings.Join(parts, " ") + "]> "
}

func runInline(args []string) error {
	if len(args) == 0 {
		return nil
	}
	cfg := config.Load()
	run := runner.New()
	_ = history.Record(args)
	code := cli.Run(Name, Commands(cfg, run), args)
	if code != 0 {
		return fmt.Errorf("comando finalizado com erro: %d", code)
	}
	return nil
}

func completion(_ cli.Context, args []string) error {
	install := false
	if len(args) == 2 && args[0] == "install" {
		install = true
		args = args[1:]
	}
	if len(args) == 1 && args[0] == "words" {
		fmt.Println(completionWords())
		return nil
	}
	if len(args) != 1 {
		return cli.UsageError("informe bash, zsh ou powershell")
	}

	if install {
		return installCompletion(args[0])
	}

	fmt.Print(completionScript(args[0]))
	return nil
}

func completionScript(shell string) string {
	commands := completionWords()
	switch shell {
	case "bash":
		return fmt.Sprintf(`_duck_complete() {
  COMPREPLY=($(compgen -W "%s" -- "${COMP_WORDS[COMP_CWORD]}"))
}
complete -F _duck_complete duck
`, commands)
	case "zsh":
		return fmt.Sprintf(`#compdef duck
_arguments "1: :(%s)"
`, commands)
	case "powershell":
		return fmt.Sprintf(`Register-ArgumentCompleter -Native -CommandName duck -ScriptBlock {
  param($wordToComplete)
  "%s".Split(" ") | Where-Object { $_ -like "$wordToComplete*" } | ForEach-Object {
    [System.Management.Automation.CompletionResult]::new($_, $_, "ParameterValue", $_)
  }
}
`, commands)
	default:
		return ""
	}
}

func completionWords() string {
	return "init config status doctor version update completion autocomplete help history terminal console repl tui profile task aliases explain last recent favorites palette command-palette watch dashboard logs troubleshoot deploy monitor alerts trace logs-search perf load ports kill-port open encrypt decrypt password qr serve zip unzip find search cidr calc json yaml git g install setup tools wsl docker d go java j node n python py maven gradle npm pnpm env project port curl http kube k aws a check export import example format validate get aws overlap ip ps pick stats ports inspect health wait-healthy cp-from cp-to size ext dir open top backup-volume restore-volume images volumes networks start stop restart logs shell exec rm rm-all clean-all clean-images clean-volumes rmi pull run up down compose compose-find compose-status compose-up compose-ps compose-logs compose-stop compose-restart compose-down compose-rm prune raw current version list ls add path home use cert alias storepass cacerts no-sudo no-persist venv create detect doctor ingress resources failed clean-failed dns tcp curl-many contexts ctx ns pods svc deploy events port-forward top-pods top-nodes scale image wait debug profiles configure whoami regions switch-profile sso-login s3-ls s3-cp s3-sync s3-rm ec2-instances ec2-ssh ec2-start ec2-stop ec2-reboot eks-clusters eks-nodegroups eks-scale eks-contexts eks-describe eks-use eks-update-kubeconfig ecs-services ecs-restart rds-list rds-connect-info sg-open iam-who-can costs secrets params deploy-ecr ecr-images ecr-login test package build dev install lint format pip-install auto once interval namespace length token pass host duration concurrency listen requests local swagger github aws-console save wip sync publish ship branches branch new switch co cleanup stash tag undo ignore remote root staged unstage run top search yes force timeout quiet no-color json"
}

func installCompletion(shell string) error {
	script := completionScript(shell)
	if script == "" {
		return cli.UsageError("shell invalido: " + shell)
	}

	profile, err := completionProfile(shell)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(profile), 0755); err != nil {
		return err
	}
	file, err := os.OpenFile(profile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	if _, err := fmt.Fprintf(file, "\n# duck autocomplete\n%s\n", script); err != nil {
		return err
	}
	fmt.Println("Autocomplete instalado em:", profile)
	fmt.Println("Abra um novo terminal ou recarregue o perfil do shell.")
	return nil
}

func completionProfile(shell string) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	switch shell {
	case "bash":
		return filepath.Join(home, ".bashrc"), nil
	case "zsh":
		return filepath.Join(home, ".zshrc"), nil
	case "powershell":
		if runtime.GOOS == "windows" {
			return filepath.Join(home, "Documents", "PowerShell", "Microsoft.PowerShell_profile.ps1"), nil
		}
		return filepath.Join(home, ".config", "powershell", "Microsoft.PowerShell_profile.ps1"), nil
	default:
		return "", cli.UsageError("shell invalido: " + shell)
	}
}

func parseTerminalLine(line string) ([]string, error) {
	var args []string
	var current strings.Builder
	var quote rune
	escaped := false

	for _, char := range line {
		if escaped {
			current.WriteRune(char)
			escaped = false
			continue
		}
		if char == '\\' {
			escaped = true
			continue
		}
		if quote != 0 {
			if char == quote {
				quote = 0
				continue
			}
			current.WriteRune(char)
			continue
		}
		switch char {
		case '\'', '"':
			quote = char
		case ' ', '\t':
			if current.Len() > 0 {
				args = append(args, current.String())
				current.Reset()
			}
		default:
			current.WriteRune(char)
		}
	}
	if escaped {
		current.WriteRune('\\')
	}
	if quote != 0 {
		return nil, fmt.Errorf("aspas nao fechadas")
	}
	if current.Len() > 0 {
		args = append(args, current.String())
	}
	return args, nil
}

type toolStatus struct {
	Name       string `json:"name"`
	OK         bool   `json:"ok"`
	Message    string `json:"message"`
	Suggestion string `json:"suggestion,omitempty"`
}

func toolStatuses(cfg config.Config, run runner.Runner) []toolStatus {
	statuses := []toolStatus{
		checkToolStatus("Go", cfg.GoBin, []string{"version"}, "Instale Go 1.22+ ou ajuste DUCK_GO_BIN.", run),
		checkToolStatus("Git", cfg.GitBin, []string{"--version"}, "Instale Git ou ajuste DUCK_GIT_BIN.", run),
		checkToolStatus("Docker", cfg.DockerBin, []string{"version", "--format", "{{.Client.Version}}"}, "Instale/abra Docker ou ajuste DUCK_DOCKER_BIN.", run),
		checkToolStatus("Kubernetes", cfg.KubectlBin, []string{"config", "current-context"}, "Instale kubectl ou configure KUBECONFIG/DUCK_KUBECTL_BIN.", run),
		checkToolStatus("AWS", cfg.AWSBin, []string{"--version"}, "Instale AWS CLI v2 ou ajuste DUCK_AWS_BIN.", run),
		checkToolStatus("Java", cfg.JavaBin, []string{"-version"}, "Instale Java/JDK ou ajuste DUCK_JAVA_BIN.", run),
		checkToolStatus("Node", cfg.NodeBin, []string{"--version"}, "Instale Node.js ou ajuste DUCK_NODE_BIN.", run),
		checkToolStatus("Python", cfg.PythonBin, []string{"--version"}, "Instale Python ou ajuste DUCK_PYTHON_BIN.", run),
	}

	wslStatus := checkToolStatus("WSL", cfg.WSLBin, []string{"--status"}, "Instale/configure WSL se estiver no Windows.", run)
	if runtime.GOOS != "windows" {
		wslStatus = toolStatus{Name: "WSL", OK: true, Message: "nao necessario neste sistema"}
	}
	statuses = append(statuses, wslStatus)
	return statuses
}

func checkToolStatus(label, bin string, args []string, suggestion string, run runner.Runner) toolStatus {
	output, err := run.Output(bin, args)
	output = normalizeOutput(output)
	message := strings.TrimSpace(output)
	if err != nil {
		if message == "" {
			message = err.Error()
		}
		return toolStatus{Name: label, OK: false, Message: message, Suggestion: suggestion}
	}
	return toolStatus{Name: label, OK: true, Message: message}
}

func printToolStatus(status toolStatus) {
	if !status.OK {
		fmt.Printf("%-12s indisponivel: %s\n", status.Name, status.Message)
		return
	}
	fmt.Printf("%-12s ok: %s\n", status.Name, status.Message)
}

func normalizeOutput(output string) string {
	bytes := []byte(output)
	if len(bytes) < 2 || !looksUTF16LE(bytes) {
		return output
	}

	u16 := make([]uint16, 0, len(bytes)/2)
	for i := 0; i+1 < len(bytes); i += 2 {
		value := uint16(bytes[i]) | uint16(bytes[i+1])<<8
		if value == 0xfeff {
			continue
		}
		u16 = append(u16, value)
	}

	return string(utf16.Decode(u16))
}

func looksUTF16LE(bytes []byte) bool {
	limit := len(bytes)
	if limit > 80 {
		limit = 80
	}

	zeros := 0
	for i := 1; i < limit; i += 2 {
		if bytes[i] == 0 {
			zeros++
		}
	}

	return zeros >= limit/4
}
