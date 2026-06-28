package buildtools

import (
	"github.com/IKauedev/duck/internal/cli"
	"github.com/IKauedev/duck/internal/runner"
)

func Maven(run runner.Runner) cli.Command {
	return tool("maven", []string{"mvn"}, run, []cli.Command{
		{Name: "test", Description: "Executa mvn test", Usage: "maven test [args...]", Run: runWith("mvn", run, "test")},
		{Name: "package", Description: "Executa mvn package", Usage: "maven package [args...]", Run: runWith("mvn", run, "package")},
		{Name: "run", Description: "Executa spring-boot:run", Usage: "maven run [args...]", Run: runWith("mvn", run, "spring-boot:run")},
	})
}

func Gradle(run runner.Runner) cli.Command {
	return tool("gradle", []string{"g"}, run, []cli.Command{
		{Name: "test", Description: "Executa gradle test", Usage: "gradle test [args...]", Run: runWith("gradle", run, "test")},
		{Name: "build", Description: "Executa gradle build", Usage: "gradle build [args...]", Run: runWith("gradle", run, "build")},
		{Name: "run", Description: "Executa gradle run", Usage: "gradle run [args...]", Run: runWith("gradle", run, "run")},
	})
}

func NPM(run runner.Runner) cli.Command {
	return tool("npm", nil, run, []cli.Command{
		{Name: "install", Description: "Executa npm install", Usage: "npm install [args...]", Run: runWith("npm", run, "install")},
		{Name: "test", Description: "Executa npm test", Usage: "npm test [args...]", Run: runWith("npm", run, "test")},
		{Name: "build", Description: "Executa npm run build", Usage: "npm build [args...]", Run: runWith("npm", run, "run", "build")},
		{Name: "dev", Description: "Executa npm run dev", Usage: "npm dev [args...]", Run: runWith("npm", run, "run", "dev")},
	})
}

func PNPM(run runner.Runner) cli.Command {
	return tool("pnpm", nil, run, []cli.Command{
		{Name: "install", Description: "Executa pnpm install", Usage: "pnpm install [args...]", Run: runWith("pnpm", run, "install")},
		{Name: "test", Description: "Executa pnpm test", Usage: "pnpm test [args...]", Run: runWith("pnpm", run, "test")},
		{Name: "build", Description: "Executa pnpm build", Usage: "pnpm build [args...]", Run: runWith("pnpm", run, "build")},
		{Name: "dev", Description: "Executa pnpm dev", Usage: "pnpm dev [args...]", Run: runWith("pnpm", run, "dev")},
	})
}

func tool(name string, aliases []string, run runner.Runner, children []cli.Command) cli.Command {
	return cli.Command{Name: name, Aliases: aliases, Description: "Executa atalhos de " + name, Usage: name + " <comando>", Children: children}
}

func runWith(binary string, run runner.Runner, prefix ...string) func(cli.Context, []string) error {
	return func(_ cli.Context, args []string) error {
		commandArgs := append([]string{}, prefix...)
		commandArgs = append(commandArgs, args...)
		return run.Run(binary, commandArgs, runner.InteractiveOptions())
	}
}
