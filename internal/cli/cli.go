package cli

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
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

type Context struct {
	AppName string
	Stdout  io.Writer
	Stderr  io.Writer
}

func Run(appName string, commands []Command, args []string) int {
	ctx := Context{
		AppName: appName,
		Stdout:  os.Stdout,
		Stderr:  os.Stderr,
	}

	if len(args) == 0 {
		PrintHelp(ctx, commands, nil)
		return 0
	}

	if args[0] == "help" || args[0] == "-h" || args[0] == "--help" {
		PrintHelp(ctx, commands, args[1:])
		return 0
	}

	if err := execute(ctx, commands, args, nil); err != nil {
		fmt.Fprintln(ctx.Stderr, "Erro:", err)
		return 1
	}

	return 0
}

func UsageError(message string) error {
	return errors.New(message)
}

func execute(ctx Context, commands []Command, args []string, path []string) error {
	if len(args) == 0 {
		PrintHelp(ctx, commands, path)
		return nil
	}

	selected, ok := Find(commands, args[0])
	if !ok {
		return fmt.Errorf("comando desconhecido: %s", args[0])
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
		return fmt.Errorf("%w. Use '%s help %s' para ver ajuda", err, ctx.AppName, strings.Join(nextPath, " "))
	}

	return nil
}

func PrintHelp(ctx Context, commands []Command, topic []string) {
	if len(topic) > 0 {
		printTopic(ctx, commands, topic, nil)
		return
	}

	fmt.Fprintln(ctx.Stdout, "Duck e um utilitario de terminal para Docker, Kubernetes e Go.")
	fmt.Fprintln(ctx.Stdout)
	fmt.Fprintln(ctx.Stdout, "Uso:")
	fmt.Fprintf(ctx.Stdout, "  %s <comando> [argumentos]\n\n", ctx.AppName)
	printGroupedCommandList(ctx, commands)
	fmt.Fprintln(ctx.Stdout)
	fmt.Fprintln(ctx.Stdout, "Exemplos:")
	fmt.Fprintf(ctx.Stdout, "  %s wsl status\n", ctx.AppName)
	fmt.Fprintf(ctx.Stdout, "  %s docker ps -a\n", ctx.AppName)
	fmt.Fprintf(ctx.Stdout, "  %s go check\n", ctx.AppName)
	fmt.Fprintf(ctx.Stdout, "  %s kube pods -n default\n", ctx.AppName)
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
