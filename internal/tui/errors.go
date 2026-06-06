package tui

import (
	"strings"
)

func friendlyDockerError(output string) string {
	lower := strings.ToLower(output)
	switch {
	case strings.Contains(lower, "pipe/docker"),
		strings.Contains(lower, "dockerdesktop"),
		strings.Contains(lower, "cannot connect to the docker daemon"),
		strings.Contains(lower, "error during connect"),
		strings.Contains(lower, "is the docker daemon running"):
		return "Docker daemon nao esta rodando. Windows: abra o Docker Desktop. Linux/WSL: sudo systemctl start docker."
	case strings.Contains(lower, "docker.sock"),
		strings.Contains(lower, "permission denied") && strings.Contains(lower, "docker"):
		return "sem permissao para acessar o Docker. Linux/WSL: adicione seu usuario ao grupo docker."
	case strings.Contains(lower, "not recognized"),
		strings.Contains(lower, "no such file"),
		strings.Contains(lower, "executable file not found"):
		return "docker nao encontrado no PATH. Instale Docker ou configure DUCK_DOCKER_BIN."
	default:
		if text := firstErrorLine(output); text != "" {
			return text
		}
		return "nao foi possivel conectar ao Docker"
	}
}

func firstErrorLine(output string) string {
	output = normalizeCLIOutput(output)
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if idx := strings.Index(line, "err="); idx >= 0 {
			line = strings.Trim(line[idx+4:], `"`)
		}
		return line
	}
	return strings.TrimSpace(output)
}

func friendlyKubeError(output string) string {
	lower := strings.ToLower(output)
	switch {
	case strings.Contains(lower, "connection refused"),
		strings.Contains(lower, "actively refused"),
		strings.Contains(lower, "no connection could be made"),
		strings.Contains(lower, "connect: connection refused"):
		return "cluster parado ou inacessivel. Windows: habilite Kubernetes no Docker Desktop. Linux/WSL: kind start, minikube start ou kubectl cluster-info."
	case strings.Contains(lower, "unable to connect to the server"),
		strings.Contains(lower, "dial tcp"):
		return "sem conexao com a API do Kubernetes: " + firstErrorLine(output)
	case strings.Contains(lower, "the connection to the server"),
		strings.Contains(lower, "context was not found"):
		return "contexto kubectl invalido ou cluster removido: " + firstErrorLine(output)
	case strings.Contains(lower, "no resources found"):
		return ""
	default:
		if text := firstErrorLine(output); text != "" {
			return text
		}
		return "nao foi possivel listar pods no cluster"
	}
}

func friendlyAWSError(output string) string {
	lower := strings.ToLower(output)
	switch {
	case strings.Contains(lower, "signaturedoesnotmatch"):
		return "credenciais AWS invalidas ou expiradas. Rode `aws configure` ou verifique AWS_ACCESS_KEY_ID e AWS_SECRET_ACCESS_KEY."
	case strings.Contains(lower, "unable to locate credentials"),
		strings.Contains(lower, "no credentials"):
		return "credenciais AWS nao configuradas. Rode `aws configure` ou defina o profile ativo."
	case strings.Contains(lower, "expiredtoken"):
		return "sessao AWS expirada. Renove as credenciais ou faca login novamente."
	case strings.Contains(lower, "accessdenied"):
		return "acesso negado pela AWS: " + firstErrorLine(output)
	default:
		if text := firstErrorLine(output); text != "" {
			return text
		}
		return "nao foi possivel obter a identidade AWS"
	}
}

func formatCommandError(output string, err error) string {
	if text := strings.TrimSpace(normalizeCLIOutput(output)); text != "" {
		return text
	}
	if err != nil {
		return err.Error()
	}
	return ""
}

func actionableError(kind, message string) string {
	message = strings.TrimSpace(message)
	if message == "" {
		message = "erro desconhecido"
	}
	switch kind {
	case "docker":
		if friendly := friendlyDockerError(message); friendly != "" {
			message = friendly
		}
		return message + "\n\nProximo passo: " + dockerNextStep(message)
	case "kube":
		if friendly := friendlyKubeError(message); friendly != "" {
			message = friendly
		}
		return message + "\n\nProximo passo: " + kubeNextStep(message)
	case "aws":
		if friendly := friendlyAWSError(message); friendly != "" {
			message = friendly
		}
		return message + "\n\nProximo passo: " + awsNextStep(message)
	default:
		return message
	}
}

func dockerNextStep(message string) string {
	lower := strings.ToLower(message)
	switch {
	case strings.Contains(lower, "daemon"), strings.Contains(lower, "pipe"), strings.Contains(lower, "docker desktop"):
		return "abra o Docker Desktop no Windows ou rode `sudo systemctl start docker` no Linux/WSL."
	case strings.Contains(lower, "permissao"), strings.Contains(lower, "permission"):
		return "adicione seu usuario ao grupo docker e reinicie a sessao."
	case strings.Contains(lower, "nao encontrado"), strings.Contains(lower, "path"):
		return "instale o Docker ou defina DUCK_DOCKER_BIN."
	default:
		return "rode `duck docker status` ou `docker version` para validar a conexao."
	}
}

func kubeNextStep(message string) string {
	lower := strings.ToLower(message)
	switch {
	case strings.Contains(lower, "parado"), strings.Contains(lower, "inacessivel"), strings.Contains(lower, "connection refused"):
		return "inicie o cluster (Docker Desktop Kubernetes, kind, minikube) e confira com `kubectl cluster-info`."
	case strings.Contains(lower, "contexto"), strings.Contains(lower, "context"):
		return "liste contextos com `duck kube contexts` e troque com `duck kube use <contexto>`."
	default:
		return "rode `duck kube status` e `kubectl get pods -A` para validar o cluster."
	}
}

func awsNextStep(message string) string {
	lower := strings.ToLower(message)
	switch {
	case strings.Contains(lower, "signature"), strings.Contains(lower, "credencial"), strings.Contains(lower, "credentials"):
		return "gere novas chaves no IAM e rode `aws configure`, ou faca `aws sso login` se usar SSO."
	default:
		return "rode `aws sts get-caller-identity` e ajuste o profile com `aws configure list`."
	}
}
