package aws

import (
	"fmt"
	"strings"

	"duck/internal/cli"
	"duck/internal/config"
	"duck/internal/prompt"
	"duck/internal/runner"
)

type service struct {
	bin       string
	dockerBin string
	runner    runner.Runner
}

func Command(cfg config.Config, run runner.Runner) cli.Command {
	svc := service{bin: cfg.AWSBin, dockerBin: cfg.DockerBin, runner: run}
	return cli.Command{
		Name:        "aws",
		Aliases:     []string{"a"},
		Description: "Executa tarefas AWS com AWS CLI",
		Usage:       "aws <comando> [argumentos]",
		Children: []cli.Command{
			{Name: "status", Description: "Mostra AWS CLI, configuracao e identidade", Usage: "aws status", Run: svc.status},
			{Name: "profiles", Description: "Lista profiles configurados", Usage: "aws profiles", Run: svc.noExtra("configure", "list-profiles")},
			{Name: "configure", Description: "Executa aws configure", Usage: "aws configure [args...]", Run: svc.awsArgs("configure")},
			{Name: "whoami", Description: "Mostra a identidade AWS atual", Usage: "aws whoami [--profile p] [--region r]", Run: svc.whoami},
			{Name: "s3-ls", Description: "Lista buckets ou objetos S3", Usage: "aws s3-ls [s3://bucket/prefix] [args...]", Run: svc.s3With("ls")},
			{Name: "s3-cp", Description: "Copia arquivos de/para S3", Usage: "aws s3-cp <origem> <destino> [args...]", Run: svc.s3Transfer("cp")},
			{Name: "s3-sync", Description: "Sincroniza arquivos com S3", Usage: "aws s3-sync <origem> <destino> [args...]", Run: svc.s3Transfer("sync")},
			{Name: "s3-rm", Description: "Remove objetos S3 com confirmacao", Usage: "aws s3-rm <s3://...> [--recursive] [-f|--force]", Run: svc.s3Remove},
			{Name: "ec2-instances", Description: "Lista instancias EC2", Usage: "aws ec2-instances [--profile p] [--region r]", Run: svc.ec2Instances},
			{Name: "ec2-start", Description: "Inicia instancias EC2", Usage: "aws ec2-start <instance-id...> [--profile p] [--region r]", Run: svc.ec2InstanceAction("start-instances", false)},
			{Name: "ec2-stop", Description: "Para instancias EC2 com confirmacao", Usage: "aws ec2-stop <instance-id...> [--profile p] [--region r] [-f|--force]", Run: svc.ec2InstanceAction("stop-instances", true)},
			{Name: "ec2-reboot", Description: "Reinicia instancias EC2 com confirmacao", Usage: "aws ec2-reboot <instance-id...> [--profile p] [--region r] [-f|--force]", Run: svc.ec2InstanceAction("reboot-instances", true)},
			{Name: "eks-clusters", Description: "Lista clusters EKS", Usage: "aws eks-clusters [--profile p] [--region r]", Run: svc.eksClusters},
			{Name: "eks-describe", Description: "Descreve um cluster EKS", Usage: "aws eks-describe <cluster> [--profile p] [--region r]", Run: svc.eksDescribe},
			{Name: "eks-use", Description: "Atualiza kubeconfig para um cluster EKS", Usage: "aws eks-use <cluster> [--alias nome] [--profile p] [--region r]", Run: svc.eksUse},
			{Name: "eks-update-kubeconfig", Description: "Atualiza kubeconfig para um cluster EKS", Usage: "aws eks-update-kubeconfig <cluster> [--alias nome] [--profile p] [--region r]", Run: svc.eksUse},
			{Name: "ecr-login", Description: "Faz login Docker em um registry ECR", Usage: "aws ecr-login <registry> [--profile p] [--region r]", Run: svc.ecrLogin},
			{Name: "raw", Description: "Envia argumentos diretamente para AWS CLI", Usage: "aws raw <aws args...>", Run: svc.raw},
		},
		Examples: []string{
			"aws status",
			"a whoami --profile dev",
			"a s3-ls s3://meu-bucket",
			"a ec2-instances --region us-east-1",
			"a ec2-start i-0123456789abcdef0 --region us-east-1",
			"a eks-use meu-cluster --region us-east-1",
			"a raw cloudformation list-stacks",
		},
	}
}

func (s service) status(_ cli.Context, args []string) error {
	if len(args) > 0 {
		return cli.UsageError("status nao recebe argumentos")
	}

	version, err := s.runner.Output(s.bin, []string{"--version"})
	if err != nil {
		return fmt.Errorf("aws cli nao encontrado ou indisponivel: %w", err)
	}
	printBlock("AWS CLI:", version)

	fmt.Println()
	fmt.Println("Configuracao:")
	if err := s.run([]string{"configure", "list"}, runner.DefaultOptions()); err != nil {
		return err
	}

	fmt.Println()
	fmt.Println("Identidade:")
	identity, err := s.runner.Output(s.bin, []string{"sts", "get-caller-identity", "--output", "table"})
	if err != nil {
		message := strings.TrimSpace(identity)
		if message == "" {
			message = err.Error()
		}
		fmt.Println("Nao foi possivel validar credenciais:", message)
		return nil
	}
	printBlock("", identity)
	return nil
}

func (s service) whoami(_ cli.Context, args []string) error {
	awsArgs, err := appendProfileRegion([]string{"sts", "get-caller-identity", "--output", "table"}, args)
	if err != nil {
		return err
	}
	return s.run(awsArgs, runner.DefaultOptions())
}

func (s service) s3With(command string) func(cli.Context, []string) error {
	return func(_ cli.Context, args []string) error {
		awsArgs := append([]string{"s3", command}, args...)
		return s.run(awsArgs, runner.DefaultOptions())
	}
}

func (s service) s3Transfer(command string) func(cli.Context, []string) error {
	return func(_ cli.Context, args []string) error {
		if len(args) < 2 {
			return cli.UsageError("informe origem e destino")
		}
		awsArgs := append([]string{"s3", command}, args...)
		return s.run(awsArgs, runner.DefaultOptions())
	}
}

func (s service) s3Remove(_ cli.Context, args []string) error {
	force, awsArgs := stripForce(args)
	if len(awsArgs) == 0 {
		return cli.UsageError("informe o caminho S3 para remover")
	}

	if !force {
		ok, err := prompt.Confirm("Remover objetos S3 informados? [s/N] ")
		if err != nil {
			return err
		}
		if !ok {
			fmt.Println("Cancelado.")
			return nil
		}
	}

	return s.run(append([]string{"s3", "rm"}, awsArgs...), runner.DefaultOptions())
}

func (s service) ec2Instances(_ cli.Context, args []string) error {
	awsArgs, err := appendProfileRegion([]string{
		"ec2",
		"describe-instances",
		"--output",
		"table",
		"--query",
		"Reservations[].Instances[].{ID:InstanceId,State:State.Name,Type:InstanceType,AZ:Placement.AvailabilityZone,Name:Tags[?Key=='Name']|[0].Value}",
	}, args)
	if err != nil {
		return err
	}
	return s.run(awsArgs, runner.DefaultOptions())
}

func (s service) ec2InstanceAction(action string, confirm bool) func(cli.Context, []string) error {
	return func(_ cli.Context, args []string) error {
		force, args := stripForce(args)
		awsFlags, instances, err := splitProfileRegion(args)
		if err != nil {
			return err
		}
		if len(instances) == 0 {
			return cli.UsageError("informe ao menos uma instancia EC2")
		}

		if confirm && !force {
			ok, err := prompt.Confirm("Executar acao nas instancias EC2 informadas? [s/N] ")
			if err != nil {
				return err
			}
			if !ok {
				fmt.Println("Cancelado.")
				return nil
			}
		}

		awsArgs := []string{"ec2", action, "--instance-ids"}
		awsArgs = append(awsArgs, instances...)
		awsArgs = append(awsArgs, awsFlags...)
		return s.run(awsArgs, runner.DefaultOptions())
	}
}

func (s service) eksClusters(_ cli.Context, args []string) error {
	awsArgs, err := appendProfileRegion([]string{"eks", "list-clusters", "--output", "table"}, args)
	if err != nil {
		return err
	}
	return s.run(awsArgs, runner.DefaultOptions())
}

func (s service) eksDescribe(_ cli.Context, args []string) error {
	awsFlags, rest, err := splitProfileRegion(args)
	if err != nil {
		return err
	}
	if len(rest) != 1 {
		return cli.UsageError("informe exatamente um cluster")
	}

	awsArgs := []string{"eks", "describe-cluster", "--name", rest[0], "--output", "table"}
	awsArgs = append(awsArgs, awsFlags...)
	return s.run(awsArgs, runner.DefaultOptions())
}

func (s service) eksUse(_ cli.Context, args []string) error {
	awsFlags, rest, err := splitProfileRegionAlias(args)
	if err != nil {
		return err
	}
	if len(rest) != 1 {
		return cli.UsageError("informe exatamente um cluster")
	}

	awsArgs := []string{"eks", "update-kubeconfig", "--name", rest[0]}
	awsArgs = append(awsArgs, awsFlags...)
	return s.run(awsArgs, runner.DefaultOptions())
}

func (s service) ecrLogin(_ cli.Context, args []string) error {
	if len(args) == 0 {
		return cli.UsageError("informe o registry ECR")
	}

	registry := args[0]
	awsArgs, err := appendProfileRegion([]string{"ecr", "get-login-password"}, args[1:])
	if err != nil {
		return err
	}

	password, err := s.runner.Output(s.bin, awsArgs)
	if err != nil {
		return err
	}

	return s.runner.Run(s.dockerBin, []string{"login", "--username", "AWS", "--password-stdin", registry}, runner.Options{
		Stdin: strings.NewReader(password),
	})
}

func (s service) raw(_ cli.Context, args []string) error {
	if len(args) == 0 {
		return cli.UsageError("informe argumentos para aws")
	}
	return s.run(args, runner.InteractiveOptions())
}

func (s service) noExtra(args ...string) func(cli.Context, []string) error {
	return func(_ cli.Context, extra []string) error {
		if len(extra) > 0 {
			return cli.UsageError("este comando nao recebe argumentos extras")
		}
		return s.run(args, runner.DefaultOptions())
	}
}

func (s service) awsArgs(prefix ...string) func(cli.Context, []string) error {
	return func(_ cli.Context, extra []string) error {
		awsArgs := append([]string{}, prefix...)
		awsArgs = append(awsArgs, extra...)
		return s.run(awsArgs, runner.InteractiveOptions())
	}
}

func (s service) run(args []string, opts runner.Options) error {
	return s.runner.Run(s.bin, args, opts)
}

func appendProfileRegion(prefix []string, args []string) ([]string, error) {
	awsArgs := append([]string{}, prefix...)
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--profile", "--region":
			if i+1 >= len(args) {
				return nil, cli.UsageError(args[i] + " precisa de um valor")
			}
			awsArgs = append(awsArgs, args[i], args[i+1])
			i++
		default:
			return nil, cli.UsageError("opcao invalida: " + args[i])
		}
	}
	return awsArgs, nil
}

func splitProfileRegion(args []string) ([]string, []string, error) {
	return splitAWSFlags(args, false)
}

func splitProfileRegionAlias(args []string) ([]string, []string, error) {
	return splitAWSFlags(args, true)
}

func splitAWSFlags(args []string, allowAlias bool) ([]string, []string, error) {
	flags := make([]string, 0, len(args))
	rest := make([]string, 0, len(args))

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--profile", "--region":
			if i+1 >= len(args) {
				return nil, nil, cli.UsageError(args[i] + " precisa de um valor")
			}
			flags = append(flags, args[i], args[i+1])
			i++
		case "--alias":
			if !allowAlias {
				return nil, nil, cli.UsageError("opcao invalida: " + args[i])
			}
			if i+1 >= len(args) {
				return nil, nil, cli.UsageError(args[i] + " precisa de um valor")
			}
			flags = append(flags, args[i], args[i+1])
			i++
		default:
			rest = append(rest, args[i])
		}
	}

	return flags, rest, nil
}

func stripForce(args []string) (bool, []string) {
	force := false
	rest := make([]string, 0, len(args))

	for _, arg := range args {
		switch arg {
		case "-f", "--force", "-y", "--yes":
			force = true
		default:
			rest = append(rest, arg)
		}
	}

	return force, rest
}

func printBlock(label string, output string) {
	output = strings.TrimSpace(output)
	if label != "" {
		fmt.Println(label)
	}
	if output == "" {
		return
	}
	fmt.Println(output)
}
