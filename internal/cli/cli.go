package cli

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/IKauedev/duck/internal/version"
)

type Command struct {
	Name        string
	Aliases     []string
	Description string
	Usage       string
	Examples    []string
	Children    []Command
	Category    string
	Run         func(Context, []string) error
}

type GlobalOptions struct {
	JSON    bool
	Quiet   bool
	NoColor bool
	Timeout time.Duration
	Force   bool
}

type Context struct {
	AppName string
	Stdout  io.Writer
	Stderr  io.Writer
	Options GlobalOptions
}

type usageError struct {
	message string
}

func (e usageError) Error() string {
	return e.message
}

func UsageError(message string) error {
	return usageError{message: message}
}

func ExtractGlobalOptions(args []string) ([]string, GlobalOptions, error) {
	opts := GlobalOptions{}
	filtered := make([]string, 0, len(args))
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--json":
			opts.JSON = true
		case "--quiet":
			opts.Quiet = true
		case "--no-color":
			opts.NoColor = true
		case "--timeout":
			if i+1 >= len(args) {
				return nil, opts, UsageError("--timeout precisa de um valor")
			}
			seconds, err := strconv.Atoi(args[i+1])
			if err != nil || seconds <= 0 {
				return nil, opts, UsageError("--timeout precisa ser numero positivo")
			}
			opts.Timeout = time.Duration(seconds) * time.Second
			i++
		case "--yes", "-y", "--force":
			opts.Force = true
		default:
			filtered = append(filtered, args[i])
		}
	}
	return filtered, opts, nil
}

func Run(appName string, commands []Command, args []string) int {
	filteredArgs, options, err := ExtractGlobalOptions(args)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Erro:", err)
		return 1
	}

	ctx := Context{
		AppName: appName,
		Stdout:  os.Stdout,
		Stderr:  os.Stderr,
		Options: options,
	}

	if options.NoColor {
		_ = os.Setenv("NO_COLOR", "1")
	}
	if options.Timeout > 0 {
		_ = os.Setenv("DUCK_TIMEOUT", strconv.Itoa(int(options.Timeout.Seconds())))
	}
	if options.Force {
		_ = os.Setenv("DUCK_FORCE", "1")
	}
	if options.Quiet {
		devNull, openErr := os.OpenFile(os.DevNull, os.O_WRONLY, 0644)
		if openErr == nil {
			defer devNull.Close()
			original := os.Stdout
			os.Stdout = devNull
			defer func() { os.Stdout = original }()
			ctx.Stdout = devNull
		}
	}

	if len(filteredArgs) == 0 {
		PrintHelp(ctx, commands, nil)
		return 0
	}

	if filteredArgs[0] == "help" || filteredArgs[0] == "-h" || filteredArgs[0] == "--help" {
		PrintHelp(ctx, commands, filteredArgs[1:])
		return 0
	}

	if len(filteredArgs) == 1 && (filteredArgs[0] == "--version" || filteredArgs[0] == "-V") {
		fmt.Fprintln(ctx.Stdout, version.Details())
		return 0
	}

	if err := execute(ctx, commands, filteredArgs, nil); err != nil {
		fmt.Fprintln(ctx.Stderr, "Erro:", err)
		return 1
	}

	return 0
}

// RunWithOutput executa um comando redirecionando stdout e stderr para w.
func RunWithOutput(appName string, commands []Command, args []string, w io.Writer) int {
	filteredArgs, options, err := ExtractGlobalOptions(args)
	if err != nil {
		fmt.Fprintln(w, "Erro:", err)
		return 1
	}

	ctx := Context{
		AppName: appName,
		Stdout:  w,
		Stderr:  w,
		Options: options,
	}

	if len(filteredArgs) == 0 {
		PrintHelp(ctx, commands, nil)
		return 0
	}

	if filteredArgs[0] == "help" || filteredArgs[0] == "-h" || filteredArgs[0] == "--help" {
		PrintHelp(ctx, commands, filteredArgs[1:])
		return 0
	}

	if len(filteredArgs) == 1 && (filteredArgs[0] == "--version" || filteredArgs[0] == "-V") {
		fmt.Fprintln(w, version.Details())
		return 0
	}

	if err := execute(ctx, commands, filteredArgs, nil); err != nil {
		fmt.Fprintln(w, "Erro:", err)
		return 1
	}

	return 0
}

func execute(ctx Context, commands []Command, args []string, path []string) error {
	if len(args) == 0 {
		PrintHelp(ctx, commands, path)
		return nil
	}

	selected, ok := Find(commands, args[0])
	if !ok {
		suggestion := suggestedCommand(args[0], commands)
		if suggestion != "" {
			return fmt.Errorf("comando desconhecido: %s. Voce quis dizer: %s", args[0], suggestion)
		}
		return fmt.Errorf("comando desconhecido: %s. Dica: use '%s help search %s'", args[0], ctx.AppName, args[0])
	}

	nextPath := append(path, selected.Name)
	if len(selected.Children) > 0 {
		if len(args) == 1 || args[1] == "help" || args[1] == "-h" || args[1] == "--help" {
			printCommandHelp(ctx, selected, nextPath)
			return nil
		}
		return execute(ctx, selected.Children, args[1:], nextPath)
	}

	if selected.Run == nil {
		PrintHelp(ctx, commands, path)
		return nil
	}

	if err := selected.Run(ctx, args[1:]); err != nil {
		var ue usageError
		if errors.As(err, &ue) {
			return fmt.Errorf("%w. Proximo passo: %s help %s", err, ctx.AppName, strings.Join(nextPath, " "))
		}
		return fmt.Errorf("%w. Use '%s help %s' para ver ajuda", err, ctx.AppName, strings.Join(nextPath, " "))
	}

	return nil
}

func PrintHelp(ctx Context, commands []Command, topic []string) {
	if len(topic) > 0 {
		if topic[0] == "search" {
			helpSearch(ctx, commands, topic[1:])
			return
		}
		printTopic(ctx, commands, topic, nil)
		return
	}

	fmt.Fprintln(ctx.Stdout, "Duck e um utilitario de terminal para Docker, Kubernetes, AWS, Java e Go.")
	fmt.Fprintln(ctx.Stdout)
	fmt.Fprintln(ctx.Stdout, "Uso:")
	fmt.Fprintf(ctx.Stdout, "  %s <comando> [argumentos]\n\n", ctx.AppName)
	printGroupedCommandList(ctx, commands)
	fmt.Fprintln(ctx.Stdout)
	fmt.Fprintln(ctx.Stdout, "Exemplos:")
	fmt.Fprintf(ctx.Stdout, "  %s doctor\n", ctx.AppName)
	fmt.Fprintf(ctx.Stdout, "  %s status --json\n", ctx.AppName)
	fmt.Fprintf(ctx.Stdout, "  %s terminal\n", ctx.AppName)
	fmt.Fprintf(ctx.Stdout, "  %s tui\n", ctx.AppName)
	fmt.Fprintf(ctx.Stdout, "  %s history\n", ctx.AppName)
	fmt.Fprintf(ctx.Stdout, "  %s help search git\n", ctx.AppName)
}

func helpSearch(ctx Context, commands []Command, args []string) {
	if len(args) != 1 {
		fmt.Fprintf(ctx.Stderr, "Uso: %s help search <termo>\n", ctx.AppName)
		return
	}
	term := strings.ToLower(args[0])
	results := searchCommands(commands, nil, term)
	if len(results) == 0 {
		fmt.Fprintln(ctx.Stdout, "Nenhum comando encontrado para:", args[0])
		return
	}
	fmt.Fprintln(ctx.Stdout, "Comandos encontrados:")
	for _, result := range results {
		fmt.Fprintln(ctx.Stdout, " ", result)
	}
}

func searchCommands(commands []Command, path []string, term string) []string {
	results := make([]string, 0)
	for _, command := range commands {
		currentPath := append(path, command.Name)
		full := strings.Join(currentPath, " ")
		haystack := strings.ToLower(full + " " + command.Description + " " + command.Usage + " " + strings.Join(command.Aliases, " "))
		if strings.Contains(haystack, term) {
			results = append(results, full+" - "+command.Description)
		}
		if len(command.Children) > 0 {
			results = append(results, searchCommands(command.Children, currentPath, term)...)
		}
	}
	return results
}

func printTopic(ctx Context, commands []Command, topic []string, path []string) {
	selected, ok := Find(commands, topic[0])
	if !ok {
		fmt.Fprintf(ctx.Stderr, "Comando de ajuda nao encontrado: %s\n", strings.Join(topic, " "))
		return
	}

	nextPath := append(path, selected.Name)
	if len(topic) > 1 && len(selected.Children) > 0 {
		printTopic(ctx, selected.Children, topic[1:], nextPath)
		return
	}

	printCommandHelp(ctx, selected, nextPath)
}

func printCommandHelp(ctx Context, selected Command, path []string) {
	usage := selected.Usage
	if usage == "" {
		usage = strings.Join(path, " ")
	}

	fmt.Fprintf(ctx.Stdout, "%s %s\n\n", ctx.AppName, usage)
	fmt.Fprintln(ctx.Stdout, selected.Description)
	if len(selected.Aliases) > 0 {
		fmt.Fprintln(ctx.Stdout, "Aliases:", strings.Join(selected.Aliases, ", "))
	}
	if len(selected.Children) > 0 {
		fmt.Fprintln(ctx.Stdout)
		fmt.Fprintln(ctx.Stdout, "Subcomandos:")
		printCommandList(ctx, selected.Children)
	}
	if len(selected.Examples) > 0 {
		fmt.Fprintln(ctx.Stdout)
		fmt.Fprintln(ctx.Stdout, "Exemplos:")
		for _, example := range selected.Examples {
			fmt.Fprintf(ctx.Stdout, "  %s %s\n", ctx.AppName, example)
		}
	}
}

func printCommandList(ctx Context, commands []Command) {
	width := commandNameWidth(commands)
	for _, command := range commands {
		aliasText := ""
		if len(command.Aliases) > 0 {
			aliasText = " (" + strings.Join(command.Aliases, ", ") + ")"
		}
		fmt.Fprintf(ctx.Stdout, "  %-*s %s%s\n", width, command.Name, command.Description, aliasText)
	}
}

func printGroupedCommandList(ctx Context, commands []Command) {
	groups := make(map[string][]Command)
	order := make([]string, 0)

	for _, command := range commands {
		category := command.Category
		if category == "" {
			category = "Comandos principais"
		}
		if _, exists := groups[category]; !exists {
			order = append(order, category)
		}
		groups[category] = append(groups[category], command)
	}

	for index, category := range order {
		if index > 0 {
			fmt.Fprintln(ctx.Stdout)
		}
		fmt.Fprintln(ctx.Stdout, category+":")
		printCommandList(ctx, groups[category])
	}
}

func commandNameWidth(commands []Command) int {
	width := 14
	for _, command := range commands {
		if len(command.Name) > width {
			width = len(command.Name)
		}
	}
	return width + 2
}

func Find(commands []Command, name string) (Command, bool) {
	for _, command := range commands {
		if command.Name == name {
			return command, true
		}
		for _, alias := range command.Aliases {
			if alias == name {
				return command, true
			}
		}
	}
	return Command{}, false
}

func suggestedCommand(input string, commands []Command) string {
	input = strings.ToLower(input)
	best := ""
	bestScore := 0
	for _, candidate := range flattenCommandNames(commands) {
		score := fuzzyScore(input, strings.ToLower(candidate))
		if score > bestScore {
			bestScore = score
			best = candidate
		}
	}
	if bestScore < 2 {
		return ""
	}
	return best
}

func flattenCommandNames(commands []Command) []string {
	out := make([]string, 0)
	var walk func([]Command, []string)
	walk = func(items []Command, prefix []string) {
		for _, item := range items {
			path := append(prefix, item.Name)
			full := strings.Join(path, " ")
			out = append(out, full)
			for _, alias := range item.Aliases {
				out = append(out, strings.Join(append(prefix, alias), " "))
			}
			if len(item.Children) > 0 {
				walk(item.Children, path)
			}
		}
	}
	walk(commands, nil)
	return out
}

func fuzzyScore(input, candidate string) int {
	if input == candidate {
		return 100
	}
	if strings.HasPrefix(candidate, input) {
		return 50
	}
	if strings.Contains(candidate, input) {
		return 20
	}
	score := 0
	index := 0
	for _, char := range input {
		position := strings.IndexRune(candidate[index:], char)
		if position == -1 {
			continue
		}
		score++
		index += position + 1
		if index >= len(candidate) {
			break
		}
	}
	return score
}
