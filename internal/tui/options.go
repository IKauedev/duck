package tui

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/IKauedev/duck/internal/cli"
)

const (
	confirmDestructive = "destructive"
	confirmAlways      = "always"
	confirmNever       = "never"

	defaultRefresh = 5 * time.Second
	minRefresh     = 1 * time.Second
	maxRefresh     = 5 * time.Minute
)

type Options struct {
	Compact  bool
	Readonly bool
	Confirm  string
	Refresh  time.Duration
}

func DefaultOptions() Options {
	return Options{
		Confirm: confirmDestructive,
		Refresh: defaultRefresh,
	}
}

func ParseOptions(args []string) (Options, error) {
	opts := DefaultOptions()
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch arg {
		case "--compact", "-c":
			opts.Compact = true
		case "--readonly", "-R":
			opts.Readonly = true
		case "--help", "-h":
			return opts, errHelpRequested{}
		case "--confirm":
			if i+1 >= len(args) {
				return opts, cli.UsageError("--confirm precisa de always, destructive ou never")
			}
			i++
			opts.Confirm = normalizeConfirmMode(args[i])
			if opts.Confirm == "" {
				return opts, cli.UsageError("modo de confirmacao invalido: " + args[i])
			}
		default:
			return opts, cli.UsageError("opcao invalida para tui: " + arg)
		}
	}
	opts = mergeEnvOptions(opts)
	return opts, nil
}

func mergeEnvOptions(opts Options) Options {
	if value := strings.TrimSpace(os.Getenv("DUCK_TUI_READONLY")); value != "" {
		if parsed, err := strconv.ParseBool(value); err == nil {
			opts.Readonly = parsed
		}
	}
	if value := strings.TrimSpace(os.Getenv("DUCK_TUI_CONFIRM")); value != "" {
		if mode := normalizeConfirmMode(value); mode != "" {
			opts.Confirm = mode
		}
	}
	if value := strings.TrimSpace(os.Getenv("DUCK_TUI_REFRESH")); value != "" {
		if parsed, err := parseRefreshDuration(value); err == nil {
			opts.Refresh = parsed
		}
	}
	return opts
}

func parseRefreshDuration(value string) (time.Duration, error) {
	if duration, err := time.ParseDuration(value); err == nil {
		return clampRefresh(duration), nil
	}
	seconds, err := strconv.Atoi(strings.TrimSuffix(value, "s"))
	if err != nil {
		return 0, err
	}
	return clampRefresh(time.Duration(seconds) * time.Second), nil
}

func clampRefresh(duration time.Duration) time.Duration {
	if duration < minRefresh {
		return minRefresh
	}
	if duration > maxRefresh {
		return maxRefresh
	}
	return duration
}

func normalizeConfirmMode(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case confirmDestructive, confirmAlways, confirmNever:
		return strings.ToLower(strings.TrimSpace(value))
	default:
		return ""
	}
}

func (o Options) needsConfirm(action string) bool {
	switch o.Confirm {
	case confirmNever:
		return false
	case confirmAlways:
		return isMutatingAction(action)
	default:
		return action == "delete"
	}
}

func isMutatingAction(action string) bool {
	switch action {
	case "delete", "start", "stop", "restart":
		return true
	default:
		return false
	}
}

type errHelpRequested struct{}

func (errHelpRequested) Error() string { return "help requested" }

func IsHelpRequested(err error) bool {
	_, ok := err.(errHelpRequested)
	return ok
}

func PrintHelp() {
	fmt.Println("duck tui - interface interativa estilo k9s")
	fmt.Println()
	fmt.Println("Uso:")
	fmt.Println("  duck tui [--compact] [--readonly] [--confirm <modo>]")
	fmt.Println("  duck dashboard            # alias para duck tui --compact")
	fmt.Println()
	fmt.Println("Flags:")
	fmt.Println("  --compact, -c     modo dashboard compacto (resumo + lista curta)")
	fmt.Println("  --readonly, -R    somente leitura, sem acoes destrutivas")
	fmt.Println("  --confirm <modo>  always | destructive | never")
	fmt.Println("  --help, -h        mostra esta ajuda")
	fmt.Println()
	fmt.Println("Variaveis de ambiente:")
	fmt.Println("  DUCK_TUI_REFRESH=2s       intervalo de atualizacao automatica")
	fmt.Println("  DUCK_TUI_CONFIRM=never    confirmação de acoes")
	fmt.Println("  DUCK_TUI_READONLY=true    modo somente leitura")
	fmt.Println()
	printShortcutsTable()
}

func printShortcutsTable() {
	fmt.Println("Atalhos gerais:")
	rows := []struct{ key, desc string }{
		{"tab / 1-3", "alternar abas Docker, Kubernetes, AWS"},
		{"j/k ou setas", "navegar na lista"},
		{"/", "filtrar"},
		{"r", "atualizar"},
		{"?", "ajuda de atalhos"},
		{"e", "exportar JSON"},
		{"E", "exportar CSV"},
		{"F", "favoritos salvos"},
		{"T", "tasks e aliases do duck"},
		{"ctrl+s", "salvar item atual nos favoritos"},
		{"q / esc", "sair"},
	}
	for _, row := range rows {
		fmt.Printf("  %-14s %s\n", row.key, row.desc)
	}
	fmt.Println()
	fmt.Println("Docker:")
	for _, row := range []struct{ key, desc string }{
		{"l", "logs"}, {"s", "shell"}, {"i", "inspect"}, {"S", "start"}, {"x", "stop"}, {"R", "restart"}, {"ctrl+d", "apagar"}, {"a", "todos os containers"},
	} {
		fmt.Printf("  %-14s %s\n", row.key, row.desc)
	}
	fmt.Println()
	fmt.Println("Kubernetes:")
	for _, row := range []struct{ key, desc string }{
		{"l", "logs"}, {"s", "shell"}, {"d", "describe"}, {"ctrl+d", "apagar pod"}, {"a", "todos os namespaces"},
	} {
		fmt.Printf("  %-14s %s\n", row.key, row.desc)
	}
}
