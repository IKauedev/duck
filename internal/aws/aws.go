package aws

import (
	"fmt"
	"strconv"
	"strings"
	"time"

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
			{Name: "regions", Description: "Lista regioes AWS disponiveis", Usage: "aws regions [--profile p] [--region r]", Run: svc.regions},
			{Name: "switch-profile", Description: "Salva profile AWS padrao do Duck", Usage: "aws switch-profile <profile>", Run: svc.switchProfile},
			{Name: "sso-login", Description: "Executa login AWS SSO", Usage: "aws sso-login [--profile p]", Run: svc.ssoLogin},
			{Name: "s3-ls", Description: "Lista buckets ou objetos S3", Usage: "aws s3-ls [s3://bucket/prefix] [args...]", Run: svc.s3With("ls")},
			{Name: "s3-cp", Description: "Copia arquivos de/para S3", Usage: "aws s3-cp <origem> <destino> [args...]", Run: svc.s3Transfer("cp")},
			{Name: "s3-sync", Description: "Sincroniza arquivos com S3", Usage: "aws s3-sync <origem> <destino> [args...]", Run: svc.s3Transfer("sync")},
			{Name: "s3-rm", Description: "Remove objetos S3 com confirmacao", Usage: "aws s3-rm <s3://...> [--recursive] [-f|--force]", Run: svc.s3Remove},
			{Name: "ec2-instances", Description: "Lista instancias EC2", Usage: "aws ec2-instances [--profile p] [--region r]", Run: svc.ec2Instances},
			{Name: "ec2-ssh", Description: "Abre SSH em uma instancia EC2", Usage: "aws ec2-ssh <instance-id|host> [usuario] [--profile p] [--region r]", Run: svc.ec2SSH},
			{Name: "ec2-start", Description: "Inicia instancias EC2", Usage: "aws ec2-start <instance-id...> [--profile p] [--region r]", Run: svc.ec2InstanceAction("start-instances", false)},
			{Name: "ec2-stop", Description: "Para instancias EC2 com confirmacao", Usage: "aws ec2-stop <instance-id...> [--profile p] [--region r] [-f|--force]", Run: svc.ec2InstanceAction("stop-instances", true)},
			{Name: "ec2-reboot", Description: "Reinicia instancias EC2 com confirmacao", Usage: "aws ec2-reboot <instance-id...> [--profile p] [--region r] [-f|--force]", Run: svc.ec2InstanceAction("reboot-instances", true)},
			{Name: "eks-clusters", Description: "Lista clusters EKS", Usage: "aws eks-clusters [--profile p] [--region r]", Run: svc.eksClusters},
			{Name: "eks-nodegroups", Description: "Lista nodegroups EKS", Usage: "aws eks-nodegroups <cluster> [--profile p] [--region r]", Run: svc.eksNodegroups},
			{Name: "eks-scale", Description: "Escala nodegroup EKS", Usage: "aws eks-scale <cluster> <nodegroup> <min> <desired> <max> [--profile p] [--region r]", Run: svc.eksScale},
			{Name: "eks-contexts", Description: "Lista contextos kubectl de EKS", Usage: "aws eks-contexts", Run: svc.eksContexts},
			{Name: "eks-describe", Description: "Descreve um cluster EKS", Usage: "aws eks-describe <cluster> [--profile p] [--region r]", Run: svc.eksDescribe},
			{Name: "eks-use", Description: "Atualiza kubeconfig para um cluster EKS", Usage: "aws eks-use <cluster> [--alias nome] [--profile p] [--region r]", Run: svc.eksUse},
			{Name: "eks-update-kubeconfig", Description: "Atualiza kubeconfig para um cluster EKS", Usage: "aws eks-update-kubeconfig <cluster> [--alias nome] [--profile p] [--region r]", Run: svc.eksUse},
			{Name: "logs", Description: "Acompanha logs do CloudWatch", Usage: "aws logs <log-group> [--follow] [--since 10m] [--profile p] [--region r]", Run: svc.logs},
			{Name: "logs-search", Description: "Busca termo em logs do CloudWatch", Usage: "aws logs-search <log-group> <termo> [--profile p] [--region r]", Run: svc.logsSearch},
			{Name: "costs", Description: "Mostra custo AWS recente", Usage: "aws costs [--days N] [--profile p] [--region r]", Run: svc.costs},
			{Name: "ecs-services", Description: "Lista servicos ECS", Usage: "aws ecs-services <cluster> [--profile p] [--region r]", Run: svc.ecsServices},
			{Name: "ecs-restart", Description: "Forca novo deploy de servico ECS", Usage: "aws ecs-restart <cluster> <service> [--profile p] [--region r]", Run: svc.ecsRestart},
			{Name: "rds-list", Description: "Lista instancias RDS", Usage: "aws rds-list [--profile p] [--region r]", Run: svc.rdsList},
			{Name: "rds-connect-info", Description: "Mostra endpoint e porta de RDS", Usage: "aws rds-connect-info <db> [--profile p] [--region r]", Run: svc.rdsConnectInfo},
			{Name: "sg-open", Description: "Abre porta em Security Group com confirmacao", Usage: "aws sg-open <sg> <port> <cidr> [--profile p] [--region r] [-f|--force]", Run: svc.sgOpen},
			{Name: "iam-who-can", Description: "Simula acao IAM para usuario/role", Usage: "aws iam-who-can <principal-arn> <action> [resource] [--profile p] [--region r]", Run: svc.iamWhoCan},
			{Name: "secrets", Description: "Le segredo do Secrets Manager", Usage: "aws secrets <nome> [--profile p] [--region r]", Run: svc.secrets},
			{Name: "params", Description: "Lista parametros do SSM por prefixo", Usage: "aws params <prefixo> [--profile p] [--region r]", Run: svc.params},
			{Name: "deploy-ecr", Description: "Build/tag/push de imagem para ECR", Usage: "aws deploy-ecr <repo-uri> <tag> [contexto] [--profile p] [--region r]", Run: svc.deployECR},
			{Name: "ecr-images", Description: "Lista imagens de um repositorio ECR", Usage: "aws ecr-images <repo> [--profile p] [--region r]", Run: svc.ecrImages},
			{Name: "ecr-login", Description: "Faz login Docker em um registry ECR", Usage: "aws ecr-login <registry> [--profile p] [--region r]", Run: svc.ecrLogin},
			{Name: "raw", Description: "Envia argumentos diretamente para AWS CLI", Usage: "aws raw <aws args...>", Run: svc.raw},
		},
		Examples: []string{
			"aws status",
			"a whoami --profile dev",
			"a s3-ls s3://meu-bucket",
			"a ec2-instances --region us-east-1",
			"a sso-login --profile dev",
			"a ec2-start i-0123456789abcdef0 --region us-east-1",
			"a eks-use meu-cluster --region us-east-1",
			"a logs /aws/lambda/minha-funcao --follow",
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

func (s service) regions(_ cli.Context, args []string) error {
	awsArgs, err := appendProfileRegion([]string{"ec2", "describe-regions", "--query", "Regions[].RegionName", "--output", "table"}, args)
	if err != nil {
		return err
	}
	return s.run(awsArgs, runner.DefaultOptions())
}

func (s service) ssoLogin(_ cli.Context, args []string) error {
	awsArgs := []string{"sso", "login"}
	hasProfile := false
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--profile":
			if i+1 >= len(args) {
				return cli.UsageError("--profile precisa de um valor")
			}
			awsArgs = append(awsArgs, args[i], args[i+1])
			hasProfile = true
			i++
		default:
			return cli.UsageError("opcao invalida: " + args[i])
		}
	}
	awsArgs = appendDefaultProfile(awsArgs, hasProfile)
	return s.run(awsArgs, runner.InteractiveOptions())
}

func (s service) switchProfile(_ cli.Context, args []string) error {
	if len(args) != 1 {
		return cli.UsageError("informe exatamente um profile")
	}
	if err := config.SetSetting("aws.profile", args[0]); err != nil {
		return err
	}
	fmt.Println("Profile AWS padrao do Duck:", args[0])
	return nil
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

func (s service) ec2SSH(_ cli.Context, args []string) error {
	awsFlags, rest, err := splitProfileRegion(args)
	if err != nil {
		return err
	}
	if len(rest) == 0 || len(rest) > 2 {
		return cli.UsageError("use: ec2-ssh <instance-id|host> [usuario] [--profile p] [--region r]")
	}

	host := rest[0]
	if strings.HasPrefix(host, "i-") {
		host, err = s.ec2InstanceHost(host, awsFlags)
		if err != nil {
			return err
		}
	}

	user := "ec2-user"
	if len(rest) == 2 {
		user = rest[1]
	}
	return s.runner.Run("ssh", []string{user + "@" + host}, runner.InteractiveOptions())
}

func (s service) ec2InstanceHost(instanceID string, awsFlags []string) (string, error) {
	awsArgs := []string{
		"ec2",
		"describe-instances",
		"--instance-ids",
		instanceID,
		"--query",
		"Reservations[0].Instances[0].PublicDnsName",
		"--output",
		"text",
	}
	awsArgs = append(awsArgs, awsFlags...)
	output, err := s.runner.Output(s.bin, awsArgs)
	if err != nil {
		return "", err
	}
	host := strings.TrimSpace(output)
	if host == "" || host == "None" {
		return "", fmt.Errorf("instancia sem PublicDnsName")
	}
	return host, nil
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

func (s service) eksNodegroups(_ cli.Context, args []string) error {
	awsFlags, rest, err := splitProfileRegion(args)
	if err != nil {
		return err
	}
	if len(rest) != 1 {
		return cli.UsageError("use: eks-nodegroups <cluster>")
	}
	awsArgs := []string{"eks", "list-nodegroups", "--cluster-name", rest[0], "--output", "table"}
	awsArgs = append(awsArgs, awsFlags...)
	return s.run(awsArgs, runner.DefaultOptions())
}

func (s service) eksScale(_ cli.Context, args []string) error {
	awsFlags, rest, err := splitProfileRegion(args)
	if err != nil {
		return err
	}
	if len(rest) != 5 {
		return cli.UsageError("use: eks-scale <cluster> <nodegroup> <min> <desired> <max>")
	}
	scaling := fmt.Sprintf("minSize=%s,desiredSize=%s,maxSize=%s", rest[2], rest[3], rest[4])
	awsArgs := []string{"eks", "update-nodegroup-config", "--cluster-name", rest[0], "--nodegroup-name", rest[1], "--scaling-config", scaling}
	awsArgs = append(awsArgs, awsFlags...)
	return s.run(awsArgs, runner.DefaultOptions())
}

func (s service) eksContexts(_ cli.Context, args []string) error {
	if len(args) > 0 {
		return cli.UsageError("eks-contexts nao recebe argumentos")
	}
	output, err := s.runner.Output("kubectl", []string{"config", "get-contexts", "-o", "name"})
	if err != nil {
		return err
	}
	for _, context := range strings.Fields(output) {
		if strings.Contains(context, "eks") || strings.Contains(context, "amazonaws.com") {
			fmt.Println(context)
		}
	}
	return nil
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

func (s service) logs(_ cli.Context, args []string) error {
	awsFlags, rest, err := splitLogsArgs(args)
	if err != nil {
		return err
	}
	if len(rest) != 1 {
		return cli.UsageError("informe exatamente um log group")
	}

	awsArgs := []string{"logs", "tail", rest[0]}
	awsArgs = append(awsArgs, awsFlags...)
	return s.run(awsArgs, runner.InteractiveOptions())
}

func (s service) logsSearch(_ cli.Context, args []string) error {
	awsFlags, rest, err := splitProfileRegion(args)
	if err != nil {
		return err
	}
	if len(rest) != 2 {
		return cli.UsageError("use: logs-search <log-group> <termo>")
	}

	awsArgs := []string{"logs", "filter-log-events", "--log-group-name", rest[0], "--filter-pattern", rest[1], "--output", "table"}
	awsArgs = append(awsArgs, awsFlags...)
	return s.run(awsArgs, runner.DefaultOptions())
}

func (s service) costs(_ cli.Context, args []string) error {
	days, awsFlags, err := parseCostArgs(args)
	if err != nil {
		return err
	}
	end := time.Now().UTC()
	start := end.AddDate(0, 0, -days)
	awsArgs := []string{
		"ce",
		"get-cost-and-usage",
		"--time-period",
		"Start=" + start.Format("2006-01-02") + ",End=" + end.Format("2006-01-02"),
		"--granularity",
		"MONTHLY",
		"--metrics",
		"UnblendedCost",
		"--output",
		"table",
	}
	awsArgs = append(awsArgs, awsFlags...)
	return s.run(awsArgs, runner.DefaultOptions())
}

func (s service) ecsServices(_ cli.Context, args []string) error {
	awsFlags, rest, err := splitProfileRegion(args)
	if err != nil {
		return err
	}
	if len(rest) != 1 {
		return cli.UsageError("use: ecs-services <cluster>")
	}
	awsArgs := []string{"ecs", "list-services", "--cluster", rest[0], "--output", "table"}
	awsArgs = append(awsArgs, awsFlags...)
	return s.run(awsArgs, runner.DefaultOptions())
}

func (s service) ecsRestart(_ cli.Context, args []string) error {
	awsFlags, rest, err := splitProfileRegion(args)
	if err != nil {
		return err
	}
	if len(rest) != 2 {
		return cli.UsageError("use: ecs-restart <cluster> <service>")
	}
	awsArgs := []string{"ecs", "update-service", "--cluster", rest[0], "--service", rest[1], "--force-new-deployment"}
	awsArgs = append(awsArgs, awsFlags...)
	return s.run(awsArgs, runner.DefaultOptions())
}

func (s service) rdsList(_ cli.Context, args []string) error {
	awsArgs, err := appendProfileRegion([]string{"rds", "describe-db-instances", "--query", "DBInstances[].{ID:DBInstanceIdentifier,Engine:Engine,Status:DBInstanceStatus,Endpoint:Endpoint.Address,Port:Endpoint.Port}", "--output", "table"}, args)
	if err != nil {
		return err
	}
	return s.run(awsArgs, runner.DefaultOptions())
}

func (s service) rdsConnectInfo(_ cli.Context, args []string) error {
	awsFlags, rest, err := splitProfileRegion(args)
	if err != nil {
		return err
	}
	if len(rest) != 1 {
		return cli.UsageError("use: rds-connect-info <db>")
	}
	awsArgs := []string{"rds", "describe-db-instances", "--db-instance-identifier", rest[0], "--query", "DBInstances[0].Endpoint", "--output", "table"}
	awsArgs = append(awsArgs, awsFlags...)
	return s.run(awsArgs, runner.DefaultOptions())
}

func (s service) sgOpen(_ cli.Context, args []string) error {
	force, args := stripForce(args)
	awsFlags, rest, err := splitProfileRegion(args)
	if err != nil {
		return err
	}
	if len(rest) != 3 {
		return cli.UsageError("use: sg-open <sg> <port> <cidr>")
	}
	if !force {
		ok, err := prompt.Confirm("Abrir porta no Security Group informado? [s/N] ")
		if err != nil {
			return err
		}
		if !ok {
			fmt.Println("Cancelado.")
			return nil
		}
	}
	awsArgs := []string{"ec2", "authorize-security-group-ingress", "--group-id", rest[0], "--protocol", "tcp", "--port", rest[1], "--cidr", rest[2]}
	awsArgs = append(awsArgs, awsFlags...)
	return s.run(awsArgs, runner.DefaultOptions())
}

func (s service) iamWhoCan(_ cli.Context, args []string) error {
	awsFlags, rest, err := splitProfileRegion(args)
	if err != nil {
		return err
	}
	if len(rest) < 2 || len(rest) > 3 {
		return cli.UsageError("use: iam-who-can <principal-arn> <action> [resource]")
	}
	awsArgs := []string{"iam", "simulate-principal-policy", "--policy-source-arn", rest[0], "--action-names", rest[1]}
	if len(rest) == 3 {
		awsArgs = append(awsArgs, "--resource-arns", rest[2])
	}
	awsArgs = append(awsArgs, "--output", "table")
	awsArgs = append(awsArgs, awsFlags...)
	return s.run(awsArgs, runner.DefaultOptions())
}

func (s service) secrets(_ cli.Context, args []string) error {
	awsFlags, rest, err := splitProfileRegion(args)
	if err != nil {
		return err
	}
	if len(rest) != 1 {
		return cli.UsageError("informe exatamente um segredo")
	}

	awsArgs := []string{"secretsmanager", "get-secret-value", "--secret-id", rest[0], "--query", "SecretString", "--output", "text"}
	awsArgs = append(awsArgs, awsFlags...)
	return s.run(awsArgs, runner.DefaultOptions())
}

func (s service) params(_ cli.Context, args []string) error {
	awsFlags, rest, err := splitProfileRegion(args)
	if err != nil {
		return err
	}
	if len(rest) != 1 {
		return cli.UsageError("informe exatamente um prefixo")
	}

	awsArgs := []string{"ssm", "get-parameters-by-path", "--path", rest[0], "--recursive", "--with-decryption", "--output", "table"}
	awsArgs = append(awsArgs, awsFlags...)
	return s.run(awsArgs, runner.DefaultOptions())
}

func (s service) deployECR(_ cli.Context, args []string) error {
	awsFlags, rest, err := splitProfileRegion(args)
	if err != nil {
		return err
	}
	if len(rest) < 2 || len(rest) > 3 {
		return cli.UsageError("use: deploy-ecr <repo-uri> <tag> [contexto]")
	}

	repo := rest[0]
	tag := rest[1]
	context := "."
	if len(rest) == 3 {
		context = rest[2]
	}
	image := repo + ":" + tag
	registry := strings.Split(repo, "/")[0]

	if err := s.runner.Run(s.dockerBin, []string{"build", "-t", image, context}, runner.DefaultOptions()); err != nil {
		return err
	}
	password, err := s.runner.Output(s.bin, append([]string{"ecr", "get-login-password"}, awsFlags...))
	if err != nil {
		return err
	}
	if err := s.runner.Run(s.dockerBin, []string{"login", "--username", "AWS", "--password-stdin", registry}, runner.Options{Stdin: strings.NewReader(password)}); err != nil {
		return err
	}
	return s.runner.Run(s.dockerBin, []string{"push", image}, runner.DefaultOptions())
}

func (s service) ecrImages(_ cli.Context, args []string) error {
	awsFlags, rest, err := splitProfileRegion(args)
	if err != nil {
		return err
	}
	if len(rest) != 1 {
		return cli.UsageError("informe exatamente um repositorio ECR")
	}

	awsArgs := []string{
		"ecr",
		"describe-images",
		"--repository-name",
		rest[0],
		"--query",
		"imageDetails[].{Tags:imageTags,Size:imageSizeInBytes,Pushed:imagePushedAt,Digest:imageDigest}",
		"--output",
		"table",
	}
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
	hasProfile := false
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--profile", "--region":
			if i+1 >= len(args) {
				return nil, cli.UsageError(args[i] + " precisa de um valor")
			}
			if args[i] == "--profile" {
				hasProfile = true
			}
			awsArgs = append(awsArgs, args[i], args[i+1])
			i++
		default:
			return nil, cli.UsageError("opcao invalida: " + args[i])
		}
	}
	return appendDefaultProfile(awsArgs, hasProfile), nil
}

func splitProfileRegion(args []string) ([]string, []string, error) {
	return splitAWSFlags(args, false)
}

func splitProfileRegionAlias(args []string) ([]string, []string, error) {
	return splitAWSFlags(args, true)
}

func splitLogsArgs(args []string) ([]string, []string, error) {
	flags := make([]string, 0, len(args))
	rest := make([]string, 0, len(args))
	hasProfile := false

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--profile", "--region", "--since":
			if i+1 >= len(args) {
				return nil, nil, cli.UsageError(args[i] + " precisa de um valor")
			}
			if args[i] == "--profile" {
				hasProfile = true
			}
			flags = append(flags, args[i], args[i+1])
			i++
		case "--follow":
			flags = append(flags, args[i])
		default:
			rest = append(rest, args[i])
		}
	}

	return appendDefaultProfile(flags, hasProfile), rest, nil
}

func parseCostArgs(args []string) (int, []string, error) {
	days := 30
	flags := make([]string, 0, len(args))
	hasProfile := false

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--days":
			if i+1 >= len(args) {
				return 0, nil, cli.UsageError("--days precisa de um valor")
			}
			value, err := strconv.Atoi(args[i+1])
			if err != nil || value <= 0 {
				return 0, nil, cli.UsageError("--days precisa ser um numero positivo")
			}
			days = value
			i++
		case "--profile", "--region":
			if i+1 >= len(args) {
				return 0, nil, cli.UsageError(args[i] + " precisa de um valor")
			}
			if args[i] == "--profile" {
				hasProfile = true
			}
			flags = append(flags, args[i], args[i+1])
			i++
		default:
			return 0, nil, cli.UsageError("opcao invalida: " + args[i])
		}
	}

	return days, appendDefaultProfile(flags, hasProfile), nil
}

func splitAWSFlags(args []string, allowAlias bool) ([]string, []string, error) {
	flags := make([]string, 0, len(args))
	rest := make([]string, 0, len(args))
	hasProfile := false

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--profile", "--region":
			if i+1 >= len(args) {
				return nil, nil, cli.UsageError(args[i] + " precisa de um valor")
			}
			if args[i] == "--profile" {
				hasProfile = true
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

	return appendDefaultProfile(flags, hasProfile), rest, nil
}

func appendDefaultProfile(args []string, hasProfile bool) []string {
	if hasProfile {
		return args
	}
	settings, err := config.LoadSettings()
	if err != nil || settings["aws.profile"] == "" {
		return args
	}
	return append(args, "--profile", settings["aws.profile"])
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
