# Duck

`duck` e um utilitario de terminal feito em Go para simplificar tarefas comuns de Docker, Kubernetes, AWS, Java e projetos Go no Windows, WSL e Linux.

Ele chama as ferramentas oficiais instaladas na maquina (`docker`, `kubectl`, `aws`, `java` e `go`), entao continua compativel com Docker Desktop, Docker Engine, Kubernetes local/remoto, AWS CLI, JDKs instalados e qualquer instalacao Go padrao.

## Requisitos

- Go 1.22 ou superior
- Docker instalado para comandos `duck docker`
- Docker Compose como plugin `docker compose` ou binario `docker-compose`
- Kubectl instalado para comandos `duck kube`
- AWS CLI instalado para comandos `duck aws`
- Java/JDK instalado para comandos `duck java`
- Node.js instalado para comandos `duck node`
- Python instalado para comandos `duck python`
- WSL instalado no Windows, caso voce queira usar Docker/Linux via WSL

Verifique:

```sh
go version
docker version
docker compose version
kubectl version --client
aws --version
java -version
node --version
python --version
```

## Build

Durante o desenvolvimento:

```sh
go run . help
```

Gerar binario no Windows:

```powershell
go build -o duck.exe .
```

Gerar binario no Linux ou WSL:

```sh
go build -o duck .
```

Releases oficiais sao publicados no GitHub Releases com binarios para Windows, Linux e macOS. Para atualizar um Duck instalado por release:

```sh
duck update
```

## Instalar via Terminal

Nao e necessario ter Go instalado para usar o Duck. Baixe o binario do GitHub Releases e execute `duck install`.

Windows PowerShell:

```powershell
iwr https://github.com/IKauedev/duck/releases/latest/download/duck_windows_amd64.zip -OutFile duck.zip
Expand-Archive duck.zip -DestinationPath .
.\duck.exe install
```

Linux amd64:

```sh
curl -L https://github.com/IKauedev/duck/releases/latest/download/duck_linux_amd64.tar.gz -o duck.tar.gz
tar -xzf duck.tar.gz
./duck install
```

macOS Intel:

```sh
curl -L https://github.com/IKauedev/duck/releases/latest/download/duck_darwin_amd64.tar.gz -o duck.tar.gz
tar -xzf duck.tar.gz
./duck install
```

macOS Apple Silicon:

```sh
curl -L https://github.com/IKauedev/duck/releases/latest/download/duck_darwin_arm64.tar.gz -o duck.tar.gz
tar -xzf duck.tar.gz
./duck install
```

Depois abra um novo terminal e teste:

```sh
duck help
duck update
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

Por padrao o Duck procura `docker`, `kubectl`, `aws`, `java`, `node`, `python` e `go` no `PATH`. Se quiser apontar para binarios especificos, use:

```sh
DUCK_DOCKER_BIN=/caminho/para/docker
DUCK_DOCKER_COMPOSE_BIN=/caminho/para/docker-compose
DUCK_KUBECTL_BIN=/caminho/para/kubectl
DUCK_AWS_BIN=/caminho/para/aws
DUCK_GO_BIN=/caminho/para/go
DUCK_GIT_BIN=/caminho/para/git
DUCK_JAVA_BIN=/caminho/para/java
DUCK_NODE_BIN=/caminho/para/node
DUCK_PYTHON_BIN=/caminho/para/python
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
duck init
duck config show
duck config set aws.profile dev
duck config edit
duck profile save dev
duck profile use dev
duck task add start "docker compose-up -d"
duck task run start
duck aliases add dps "docker ps -a"
duck help search git
duck palette docker
duck recent
duck recent top 15
duck favorites add deploy "deploy compose --build"
duck favorites list
duck favorites run deploy
duck explain docker ps
duck last
duck watch --interval 5 status
duck dashboard
duck logs compose -f
duck logs-search compose error
duck troubleshoot
duck troubleshoot api.local 443
duck monitor --once
duck alerts
duck trace api.local 443 https://api.local
duck perf https://api.local --requests 50
duck load https://api.local --duration 10 --concurrency 5
duck ports --listen
duck kill-port 3000
duck env example .env .env.example
duck open local 3000
duck open swagger 8080
duck open github
duck open aws-console ec2
duck git status
duck git save "feat: adiciona comando"
duck git sync
duck git ship "fix: ajusta README"
duck password --length 32
duck password --token 32
duck encrypt .env.local .env.local.duck --pass minha-senha
duck decrypt .env.local.duck .env.local --pass minha-senha
duck qr http://localhost:8080
duck serve . --port 8080
duck zip backup.zip ./logs .env
duck unzip backup.zip ./restore
duck find --ext pdf relatorio
duck find --size +100MB
duck cidr aws 10.0.1.0/24
duck cidr overlap 10.0.0.0/16 10.0.1.0/24
duck calc ip 172.31.0.0/20
duck json format package.json
duck yaml validate docker-compose.yml
duck deploy compose --build
duck deploy kube k8s/
duck deploy ecr 123456789012.dkr.ecr.us-east-1.amazonaws.com/app v1 .
duck deploy ecs my-cluster my-service
duck status
duck status --json
duck --dry-run docker clean-all
duck doctor
duck version
duck update
duck completion <bash|zsh|powershell>
duck completion install <bash|zsh|powershell>
duck autocomplete <bash|zsh|powershell>
duck history [--limit N|--all|--clear|--path]
duck history search <termo>
duck history run <numero>
duck terminal
duck tui
duck install
duck setup path
duck setup tools docker
duck setup tools compose
duck setup tools kubectl
duck setup tools curl
duck setup tools all
duck wsl status
duck docker help
duck docker pick
duck go help
duck java help
duck node help
duck python help
duck env doctor
duck env export duck-config.json
duck env import duck-config.json
duck env example .env .env.example
duck project detect
duck curl <url> [--port porta] [--timeout segundos] [--insecure]
duck port check <host> <port>
duck kube help
duck aws help
```

Flags globais disponiveis em qualquer comando:

```sh
duck --json <comando>
duck --quiet <comando>
duck --no-color <comando>
duck --timeout 30 <comando>
duck --yes <comando>
```

`duck status` mostra a disponibilidade de Go, Docker, Kubernetes, AWS, Java, Node e Python sem impedir o uso dos demais grupos caso uma ferramenta esteja ausente.

`duck setup tools` tenta instalar ferramentas externas:

```sh
duck setup tools docker
duck setup tools compose
duck setup tools kubectl
duck setup tools curl
duck setup tools all
```

No Windows, usa `winget` quando disponivel. No Linux, tenta usar gerenciadores comuns como `apt-get`, `dnf`, `yum`, `pacman` ou `apk`.

`duck curl` testa uma URL a partir da maquina local sem depender do binario `curl`, pois usa HTTP nativo do Go:

```sh
duck curl https://example.com
duck curl api.local --port 8080 --timeout 5
duck curl https://api.local --insecure
duck port check localhost 5432
```

`duck profile`, `duck task`, `duck aliases` e `duck favorites` usam o arquivo de configuracao do Duck para salvar preferencias e atalhos. `duck recent` mostra comandos recentes/mais usados e `duck palette` permite buscar e executar comandos por texto livre.

## Git

```sh
duck git status
duck git info
duck git log -n 20
duck git diff
duck git diff --staged
duck git save "feat: adiciona recurso"
duck git wip "ajustes locais"
duck git sync
duck git publish
duck git ship "fix: corrige fluxo"
duck git branches
duck git new feature/minha-branch
duck git switch main
duck git cleanup
duck git stash save "pausando trabalho"
duck git stash list
duck git stash pop
duck git tag v1.2.3 "Release v1.2.3"
duck git undo unstage
duck git undo last
duck git ignore ".env.local"
duck git remote
duck git root
duck git raw status
```

`duck git save` faz `add -A` e commit em um comando. `duck git sync` faz `pull --rebase` e `push`. `duck git ship` combina commit, rebase e push para fluxos simples. `duck git cleanup` remove apenas branches locais ja mergeadas e preserva branches comuns como `main`, `master`, `develop`, `staging` e `production`.

## Utilitarios Locais

```sh
duck password [--length N]
duck password --token <bytes>
duck encrypt <arquivo> [saida] --pass <senha>
duck decrypt <arquivo> [saida] --pass <senha>
duck qr <texto|url>
duck serve [pasta] [--port porta] [--host host]
duck perf <url> [--requests N]
duck load <url> [--duration segundos] [--concurrency N]
duck ports [--listen]
duck kill-port <porta>
duck open <url>
duck open local [porta]
duck open swagger [porta]
duck open github
duck open aws-console [servico]
duck open ingress [host]
duck zip <saida.zip> <arquivo|pasta...>
duck unzip <arquivo.zip> [destino]
duck find [--path pasta] [--ext extensao] [--size +100MB|-10MB] [termo]
duck search [--path pasta] [--ext extensao] [--size +100MB|-10MB] [termo]
duck cidr calc <cidr>
duck cidr aws <cidr>
duck cidr overlap <cidr1> <cidr2>
duck calc ip <cidr>
duck json format [arquivo]
duck json validate [arquivo]
duck json get <arquivo> <path>
duck yaml format [arquivo]
duck yaml validate [arquivo]
```

`duck perf` mede latencia HTTP com min/media/p95/p99 e throughput. `duck load` executa carga HTTP basica com concorrencia. `duck ports` lista portas locais e PIDs; `duck kill-port` finaliza o processo preso em uma porta. `duck open` abre recursos uteis no navegador. `duck zip` e `duck unzip` compactam e descompactam arquivos/pastas diretamente em Go, sem depender de ferramentas externas do sistema. `duck find` busca arquivos por nome, extensao e tamanho; `duck search` e um alias do mesmo comando. `duck cidr aws` calcula subnets IPv4 pensando em AWS: mostra se o range e publico ou privado, total de IPs, IPs utilizaveis e os cinco IPs reservados pela AWS em cada subnet. `duck json` e `duck yaml` aceitam arquivo ou stdin nos comandos de formatacao/validacao.

## Operacoes Inteligentes

```sh
duck dashboard
duck logs auto
duck logs docker <container> --tail 100
duck logs compose -f
duck logs kube <pod> -n default
duck logs ecs <cluster> <service>
duck logs-search docker <termo> <container> --tail 200
duck logs-search compose <termo>
duck logs-search kube <termo> <pod> -n default
duck logs-search ecs <termo> <cluster> <service>
duck troubleshoot
duck troubleshoot <host> <port>
duck monitor [--interval segundos] [--once]
duck alerts [--ecs <cluster> <service>]
duck trace <host> [port] [url] [-n namespace]
duck deploy compose [--build]
duck deploy kube [arquivo|diretorio]
duck deploy ecr <repo-uri> <tag> [contexto]
duck deploy ecs <cluster> <service>
```

`duck dashboard` resume projeto, configuracao, ferramentas, containers e ultimo comando. `duck logs` centraliza logs de Docker, Compose, Kubernetes e ECS. `duck logs-search` busca termos nesses mesmos backends. `duck monitor` mostra containers, pods, CPU/memoria e portas em loop. `duck alerts` procura containers unhealthy, pods com erro, pressao em nodes e status ECS opcional. `duck trace` testa DNS/TCP/HTTP local, DNS/TCP a partir do cluster e ingress. `duck troubleshoot` faz um diagnostico rapido sem alterar recursos. `duck deploy` oferece fluxos simples para Compose, Kubernetes, ECR e ECS.

`duck history` mostra os comandos executados anteriormente pelo Duck. O historico fica no diretorio de configuracao do usuario e pode ser localizado com:

```sh
duck history --path
```

`duck terminal` abre um modo interativo para usar o Duck como um terminal personalizado em Windows, Linux ou WSL:

```text
duck> status
duck> docker ps -a
duck> aws whoami
duck> exit
```

`duck tui` abre uma interface terminal full-screen com abas para Docker, Kubernetes e AWS:

```sh
duck tui
```

Use `tab`/`shift+tab` para trocar de aba, `r` para atualizar e `q` para sair.

Para habilitar autocomplete no shell atual, gere o script ou instale diretamente no perfil do usuario:

```sh
duck completion powershell
duck completion install powershell
duck completion install bash
duck completion install zsh
```

Veja a pagina dedicada em [`docs/completion.md`](docs/completion.md). O caminho recomendado e `duck completion install <bash|zsh|powershell>`.

## Docker

Grupo principal:

```sh
duck docker status
duck docker status <container...>
duck docker ps
duck docker ps -a
duck docker pick
duck docker pick logs
duck docker pick shell
duck docker find <termo>
duck docker stats [container...]
duck docker ports <container>
duck docker inspect <container>
duck docker health [container...]
duck docker wait-healthy <container> [--timeout segundos]
duck docker cp-from <container> <origem> <destino>
duck docker cp-to <container> <origem> <destino>
duck docker size
duck docker open <container>
duck docker env <container>
duck docker top <container>
duck docker backup-volume <volume> <arquivo.tar.gz>
duck docker restore-volume <volume> <arquivo.tar.gz>
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
duck docker clean-images [-f|--force]
duck docker clean-volumes [-f|--force]
duck docker rmi [-f|--force] <image...>
duck docker pull <image>
duck docker run <argumentos do docker run>
duck docker up [argumentos do docker compose up]
duck docker down [argumentos do docker compose down]
duck docker compose <argumentos do docker compose>
duck docker compose-find
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

`duck docker pick` lista containers em uma interface interativa para escolher o alvo e executar uma acao comum:

```sh
duck docker pick
duck docker pick logs
duck docker pick shell
duck docker pick restart
```

Alias curto:

```sh
duck d ps -a
duck d status api
duck d find api
duck d stats
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

## Java

Use estes comandos para ver e alternar JDKs pelo Duck:

```sh
duck java current
duck java list
duck java add 17 /caminho/para/jdk-17
duck java use 17
duck java use 17 --persist
duck java path 17
duck java home
duck java cert /caminho/para/certificado.crt --alias empresa
duck java cert https://example.com/certificado.crt --alias empresa
duck java raw -version
```

No Windows, `duck java use <alias|JAVA_HOME> --persist` usa `setx` para persistir `JAVA_HOME` e colocar `%JAVA_HOME%\bin` no `PATH` do usuario. Abra um novo terminal depois.

No Linux/WSL, o mesmo comando adiciona `JAVA_HOME` e `PATH` ao perfil do shell (`.bashrc`, `.zshrc` ou `.profile`).

`duck java cert` copia ou baixa o certificado para a pasta de configuracao do Duck e importa no truststore `cacerts` da JVM atual usando `keytool`. Por padrao a senha do truststore e `changeit`; use `--storepass` ou `--cacerts` quando sua JVM usar outro caminho/senha. No Linux/Unix, se o `cacerts` nao for gravavel pelo usuario, o Duck tenta executar `sudo keytool`; use `--no-sudo` para desativar esse comportamento.

Para salvar um JDK detectado manualmente:

```powershell
duck java add 21 "C:\Program Files\Java\jdk-21"
duck java use 21 --persist
```

## Node.js

```sh
duck node current
duck node list
duck node add 20 /caminho/para/node-v20
duck node use 20
duck node use 20 --persist
duck node home
duck node cert /caminho/para/certificado.pem
duck node cert https://example.com/certificado.pem
duck node raw --version
```

`duck node use <version>` salva a versao atual do Node no config do Duck. Use `duck node add` para mapear uma versao para uma pasta instalada.

`duck node cert` copia ou baixa o certificado para a pasta de configuracao do Duck e configura `NODE_EXTRA_CA_CERTS` para o usuario. No Windows usa `setx`; no Linux/Unix adiciona a variavel em `.bashrc`, `.zshrc` ou `.profile`. Abra um novo terminal para o Node carregar a variavel persistida.

## Python

```sh
duck python version
duck python venv .venv
duck python create .venv
duck python use .venv
duck python test
duck python lint
duck python format
duck python pip-install requests
duck python raw -m pip install -r requirements.txt
```

`duck python use .venv` salva o virtualenv atual e imprime o comando de ativacao correto para Windows ou Linux.

## Ambiente E Projeto

```sh
duck env doctor
duck env export duck-config.json
duck env import duck-config.json
duck env example .env .env.example
duck project detect
duck project doctor
duck project up
duck project down
```

`duck env doctor` valida `PATH`, `JAVA_HOME`, `NODE_HOME` e binarios comuns. `duck env export/import` facilita levar perfis, tasks, aliases e preferencias do Duck para outra maquina. `duck env example` gera `.env.example` preservando chaves e removendo valores reais. `duck project detect` identifica stacks pelo projeto atual, como Go, Node.js, Python, Java, Docker, Compose, Kubernetes, Helm e Terraform.

## Build Tools

```sh
duck maven test
duck maven package
duck maven run
duck gradle test
duck gradle build
duck gradle run
duck npm install
duck npm test
duck npm build
duck npm dev
duck pnpm install
duck pnpm test
duck pnpm build
duck pnpm dev
```

## Kubernetes

Grupo principal:

```sh
duck kube status
duck kube contexts
duck kube ctx
duck kube use <context>
duck kube ns
duck kube pods [-n namespace]
duck kube svc [-n namespace]
duck kube deploy [-n namespace]
duck kube events [-n namespace]
duck kube logs <pod> [-n namespace]
duck kube logs <pod> -n apps --tail 100
duck kube logs <pod> -n apps --follow
duck kube exec <pod> -n apps -- sh
duck kube shell <pod> [-n namespace] [shell]
duck kube debug <pod> [-n namespace]
duck kube restart <deployment> [-n namespace]
duck kube scale <deployment> <replicas> [-n namespace]
duck kube image <deployment> <container=image> [-n namespace]
duck kube wait <deployment> [-n namespace]
duck kube port-forward <recurso> <porta> [-n namespace]
duck kube curl <url> [--port porta] [-n namespace] [--timeout segundos] [--insecure]
duck kube curl-many <arquivo> [-n namespace]
duck kube dns <host> [-n namespace]
duck kube tcp <host> <port> [-n namespace]
duck kube ingress [-n namespace]
duck kube resources [-n namespace]
duck kube failed [-n namespace]
duck kube clean-failed [-n namespace] [-f|--force]
duck kube top-pods [-n namespace]
duck kube top-nodes
duck kube describe pod <pod> -n apps
duck kube apply -f deployment.yaml
duck kube delete -f deployment.yaml --force
duck kube raw get nodes
```

Alias curto:

```sh
duck k pods -n default
duck k logs api-123 -n apps --tail 100
duck k curl api.interno.local --port 8080 -n apps
```

`duck kube curl` cria um pod temporario no cluster com `curlimages/curl` e testa a URL a partir da rede do Kubernetes/EKS. Use isso para validar se o cluster consegue acessar uma URL e porta externas ou internas.

Resumo:

- `duck curl ...`: testa a partir da sua maquina.
- `duck kube curl ...`: testa de dentro do cluster Kubernetes/EKS.
- `duck setup tools curl`: instala o binario `curl` externo, se voce tambem quiser usar `curl` fora do Duck.

## AWS

Grupo principal:

```sh
duck aws status
duck aws profiles
duck aws configure
duck aws whoami [--profile dev] [--region us-east-1]
duck aws regions [--profile dev] [--region us-east-1]
duck aws switch-profile <profile>
duck aws sso-login [--profile dev]
duck aws s3-ls [s3://bucket/prefix]
duck aws s3-cp <origem> <destino> [args...]
duck aws s3-sync <origem> <destino> [args...]
duck aws s3-rm <s3://bucket/prefix> [--recursive] [-f|--force]
duck aws ec2-instances [--profile dev] [--region us-east-1]
duck aws ec2-ssh <instance-id|host> [usuario] [--profile dev] [--region us-east-1]
duck aws ec2-start <instance-id...> [--profile dev] [--region us-east-1]
duck aws ec2-stop <instance-id...> [--profile dev] [--region us-east-1] [-f|--force]
duck aws ec2-reboot <instance-id...> [--profile dev] [--region us-east-1] [-f|--force]
duck aws eks-clusters [--profile dev] [--region us-east-1]
duck aws eks-nodegroups <cluster> [--profile dev] [--region us-east-1]
duck aws eks-scale <cluster> <nodegroup> <min> <desired> <max> [--profile dev] [--region us-east-1]
duck aws eks-contexts
duck aws eks-describe <cluster> [--profile dev] [--region us-east-1]
duck aws eks-use <cluster> [--alias nome] [--profile dev] [--region us-east-1]
duck aws eks-update-kubeconfig <cluster> [--alias nome] [--profile dev] [--region us-east-1]
duck aws logs <log-group> [--follow] [--since 10m] [--profile dev] [--region us-east-1]
duck aws logs-search <log-group> <termo> [--profile dev] [--region us-east-1]
duck aws costs [--days 30] [--profile dev] [--region us-east-1]
duck aws ecs-services <cluster> [--profile dev] [--region us-east-1]
duck aws ecs-restart <cluster> <service> [--profile dev] [--region us-east-1]
duck aws rds-list [--profile dev] [--region us-east-1]
duck aws rds-connect-info <db> [--profile dev] [--region us-east-1]
duck aws sg-open <sg> <port> <cidr> [--profile dev] [--region us-east-1] [-f|--force]
duck aws iam-who-can <principal-arn> <action> [resource] [--profile dev] [--region us-east-1]
duck aws secrets <nome> [--profile dev] [--region us-east-1]
duck aws params <prefixo> [--profile dev] [--region us-east-1]
duck aws deploy-ecr <repo-uri> <tag> [contexto] [--profile dev] [--region us-east-1]
duck aws ecr-images <repo> [--profile dev] [--region us-east-1]
duck aws ecr-login <registry> [--profile dev] [--region us-east-1]
duck aws raw <argumentos diretos do aws>
```

Alias curto:

```sh
duck a whoami --profile dev
duck a sso-login --profile dev
duck a s3-ls s3://meu-bucket
duck a ec2-instances --region us-east-1
duck a ec2-ssh i-0123456789abcdef0 ubuntu --region us-east-1
duck a ec2-start i-0123456789abcdef0 --region us-east-1
duck a eks-use meu-cluster --region us-east-1
duck a logs /aws/lambda/minha-funcao --follow
```

Os comandos AWS usam a configuracao normal da AWS CLI, incluindo `~/.aws/config`, `~/.aws/credentials`, `AWS_PROFILE`, `AWS_REGION` e flags `--profile`/`--region` quando aceitas pelo atalho.

## Acoes Destrutivas

Comandos destrutivos pedem confirmacao quando usados sem flags de confirmacao:

```sh
duck docker rm api
duck docker rm-all
duck docker clean-all
duck docker clean-images
duck docker clean-volumes
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
duck docker clean-images --force
duck docker clean-volumes --force
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
