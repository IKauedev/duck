# Duck

`duck` e um utilitario de terminal feito em Go para simplificar tarefas comuns de Docker, Kubernetes, AWS e projetos Go no Windows, WSL e Linux.

Ele chama as ferramentas oficiais instaladas na maquina (`docker`, `kubectl`, `aws` e `go`), entao continua compativel com Docker Desktop, Docker Engine, Kubernetes local/remoto, AWS CLI e qualquer instalacao Go padrao.

## Requisitos

- Go 1.22 ou superior
- Docker instalado para comandos `duck docker`
- Docker Compose como plugin `docker compose` ou binario `docker-compose`
- Kubectl instalado para comandos `duck kube`
- AWS CLI instalado para comandos `duck aws`
- WSL instalado no Windows, caso voce queira usar Docker/Linux via WSL

Verifique:

```sh
go version
docker version
docker compose version
kubectl version --client
aws --version
```

## Build

Durante o desenvolvimento:

```sh
go run . help
go run ./cmd/duck help
```

Gerar binario no Windows:

```powershell
go build -o duck.exe .
```

Gerar binario no Linux ou WSL:

```sh
go build -o duck .
```

## Instalar e Configurar PATH

O Duck possui comandos para instalar o binario no usuario atual e configurar o `PATH`.

Instalacao recomendada:

```sh
duck install
```

Se voce ainda estiver na pasta do build, no Windows use:

```powershell
.\duck.exe install
```

No Linux ou WSL:

```sh
./duck install
```

O comando copia o executavel para a pasta `bin` do usuario e adiciona essa pasta ao `PATH`:

- Windows: `%USERPROFILE%\bin`, atualizando o `Path` do usuario.
- Linux/WSL: `$HOME/bin`, atualizando `.bashrc`, `.zshrc` ou `.profile` conforme o shell.

Depois abra um novo terminal e teste:

```sh
duck help
```

Outras opcoes:

```sh
duck install --force
duck install --dir /caminho/customizado
duck install --no-path
duck setup path
duck setup path --dir /caminho/customizado
```

`duck setup path` apenas adiciona uma pasta ao `PATH`; ele nao copia o executavel.

## Variaveis de Ambiente

Por padrao o Duck procura `docker`, `kubectl`, `aws` e `go` no `PATH`. Se quiser apontar para binarios especificos, use:

```sh
DUCK_DOCKER_BIN=/caminho/para/docker
DUCK_DOCKER_COMPOSE_BIN=/caminho/para/docker-compose
DUCK_KUBECTL_BIN=/caminho/para/kubectl
DUCK_AWS_BIN=/caminho/para/aws
DUCK_GO_BIN=/caminho/para/go
DUCK_WSL_BIN=/caminho/para/wsl
```

Variaveis nativas continuam funcionando normalmente, por exemplo:

```sh
DOCKER_HOST=tcp://localhost:2375
KUBECONFIG=/caminho/para/kubeconfig
AWS_PROFILE=dev
AWS_REGION=us-east-1
```

## Comandos Gerais

```sh
duck help
duck status
duck install
duck setup path
duck wsl status
duck docker help
duck go help
duck kube help
duck aws help
```

`duck status` mostra a disponibilidade de Go, Docker, Kubernetes e AWS sem impedir o uso dos demais grupos caso uma ferramenta esteja ausente.

## Docker

Grupo principal:

```sh
duck docker status
duck docker status <container...>
duck docker ps
duck docker ps -a
duck docker images
duck docker volumes
duck docker networks
duck docker logs <container>
duck docker logs <container> --tail 100
duck docker logs <container> --follow
duck docker shell <container>
duck docker shell <container> bash
duck docker exec <container> -- ls -la
duck docker start <container...>
duck docker stop <container...>
duck docker restart <container...>
duck docker rm [-f|--force] <container...>
duck docker rm-all [-f|--force]
duck docker clean-all [-f|--force]
duck docker rmi [-f|--force] <image...>
duck docker pull <image>
duck docker run <argumentos do docker run>
duck docker up [argumentos do docker compose up]
duck docker down [argumentos do docker compose down]
duck docker compose <argumentos do docker compose>
duck docker compose-status
duck docker compose-up [argumentos]
duck docker compose-ps
duck docker compose-logs
duck docker compose-stop [servico...]
duck docker compose-restart [servico...]
duck docker compose-down
duck docker compose-rm [-f|--force] [servico...]
duck docker prune [containers|images|volumes|networks|system] [-f|--force]
duck docker raw <argumentos diretos do docker>
```

No Linux/WSL, os comandos Compose tentam usar `docker compose` primeiro. Se o plugin nao existir, o Duck tenta usar o binario classico `docker-compose`.

Alias curto:

```sh
duck d ps -a
duck d status api
duck d rm-all --force
duck d clean-all --force
duck d up -d
duck d compose up -d
duck d compose-down
```

Atalhos legados tambem continuam funcionando:

```sh
duck ps -a
duck images
duck rm-all --force
duck clean-all --force
duck compose up -d
duck up -d
duck down
duck compose-stop
```

## WSL

No Windows, use estes comandos para checar se o WSL esta instalado e quais distribuicoes estao disponiveis:

```powershell
duck wsl status
duck wsl list
duck wsl start
duck wsl start Ubuntu-22.04
duck wsl raw --version
```

Em Linux ou WSL, `duck wsl status` apenas informa que a checagem nao e necessaria, porque os comandos Docker/Kubernetes/Go podem ser executados diretamente no ambiente atual.

## Go

```sh
duck go version
duck go env
duck go tidy
duck go download
duck go test
duck go test --race
duck go build
duck go build -o duck.exe .
duck go run
duck go run -- arg1 arg2
duck go fmt
duck go vet
duck go check
duck go raw env GOPATH
```

`duck go check` executa uma rotina completa:

```text
go mod tidy
gofmt -w
go vet ./...
go test ./...
```

## Kubernetes

Grupo principal:

```sh
duck kube status
duck kube contexts
duck kube use <context>
duck kube ns
duck kube pods [-n namespace]
duck kube svc [-n namespace]
duck kube deploy [-n namespace]
duck kube logs <pod> [-n namespace]
duck kube logs <pod> -n apps --tail 100
duck kube logs <pod> -n apps --follow
duck kube exec <pod> -n apps -- sh
duck kube describe pod <pod> -n apps
duck kube apply -f deployment.yaml
duck kube delete -f deployment.yaml --force
duck kube raw get nodes
```

Alias curto:

```sh
duck k pods -n default
duck k logs api-123 -n apps --tail 100
```

## AWS

Grupo principal:

```sh
duck aws status
duck aws profiles
duck aws configure
duck aws whoami [--profile dev] [--region us-east-1]
duck aws s3-ls [s3://bucket/prefix]
duck aws s3-cp <origem> <destino> [args...]
duck aws s3-sync <origem> <destino> [args...]
duck aws s3-rm <s3://bucket/prefix> [--recursive] [-f|--force]
duck aws ec2-instances [--profile dev] [--region us-east-1]
duck aws ec2-start <instance-id...> [--profile dev] [--region us-east-1]
duck aws ec2-stop <instance-id...> [--profile dev] [--region us-east-1] [-f|--force]
duck aws ec2-reboot <instance-id...> [--profile dev] [--region us-east-1] [-f|--force]
duck aws eks-clusters [--profile dev] [--region us-east-1]
duck aws eks-describe <cluster> [--profile dev] [--region us-east-1]
duck aws eks-use <cluster> [--alias nome] [--profile dev] [--region us-east-1]
duck aws eks-update-kubeconfig <cluster> [--alias nome] [--profile dev] [--region us-east-1]
duck aws ecr-login <registry> [--profile dev] [--region us-east-1]
duck aws raw <argumentos diretos do aws>
```

Alias curto:

```sh
duck a whoami --profile dev
duck a s3-ls s3://meu-bucket
duck a ec2-instances --region us-east-1
duck a ec2-start i-0123456789abcdef0 --region us-east-1
duck a eks-use meu-cluster --region us-east-1
```

Os comandos AWS usam a configuracao normal da AWS CLI, incluindo `~/.aws/config`, `~/.aws/credentials`, `AWS_PROFILE`, `AWS_REGION` e flags `--profile`/`--region` quando aceitas pelo atalho.

## Acoes Destrutivas

Comandos destrutivos pedem confirmacao quando usados sem flags de confirmacao:

```sh
duck docker rm api
duck docker rm-all
duck docker clean-all
duck docker rmi minha-imagem:latest
duck docker prune system
duck kube delete deployment api -n apps
duck aws s3-rm s3://meu-bucket/prefix --recursive
duck aws ec2-stop i-0123456789abcdef0
duck aws ec2-reboot i-0123456789abcdef0
```

Para automacoes, use `--force`, `--yes` ou `-y` quando disponivel:

```sh
duck docker rm --force api
duck docker rm-all --force
duck docker clean-all --force
duck docker prune system --force
duck kube delete deployment api -n apps --yes
duck aws s3-rm s3://meu-bucket/prefix --recursive --force
duck aws ec2-stop i-0123456789abcdef0 --yes
```

## Observacoes Por Sistema

### Windows

Use com Docker Desktop aberto. Se `docker`, `kubectl`, `aws` e `go` funcionam no PowerShell, o `duck` tambem funcionara. Apos `duck install`, abra um novo terminal para carregar o novo `Path` do usuario.

### WSL

Funciona com Docker Desktop integrado ao WSL ou com Docker Engine instalado dentro da distribuicao. Instale o binario `duck` dentro do WSL para usar caminhos Linux naturalmente.

### Linux

Se Docker retornar erro de permissao, adicione seu usuario ao grupo `docker` ou execute conforme a politica da sua maquina.
