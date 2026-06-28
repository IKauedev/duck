# Debug do Duck

Este guia descreve como depurar o Duck no **Cursor** ou **VS Code**, no **Windows**, **WSL** e **Linux**.

O projeto ja inclui configuracoes prontas em [`.vscode/`](../.vscode/):

| Arquivo | Funcao |
|---------|--------|
| `launch.json` | Perfis de debug (CLI, TUI, testes, attach) |
| `tasks.json` | Build, testes, instalar Delve, checagem `gofmt` |
| `settings.json` | Preferencias Go/Delve e format on save |
| `extensions.json` | Recomenda a extensao **Go** (`golang.go`) |

## Pre-requisitos

1. **Go 1.22+** instalado (`go version`)
2. Extensao **Go** no Cursor/VS Code
3. **Delve** (`dlv`) — depurador usado pela extensao Go

Instalar o Delve:

```sh
go install github.com/go-delve/delve/cmd/dlv@latest
```

No Windows, confirme que `%USERPROFILE%\go\bin` esta no `PATH`. No Cursor, voce tambem pode rodar a task **go: install delve** (`Terminal` → `Run Task...`).

## Inicio rapido

1. Abra a pasta do projeto no Cursor
2. Pressione `Ctrl+Shift+D` (Run and Debug)
3. Escolha um perfil na lista
4. Pressione **F5**

O ponto de entrada do binario e a raiz do modulo (`main.go`), que chama `internal/app.Run`.

## Perfis de debug

### CLI

| Perfil | Uso |
|--------|-----|
| **Duck: comando (escolher)** | Menu com `version`, `doctor`, `tui`, `dashboard` |
| **Duck: custom (edite args)** | Edite o array `args` em `launch.json` antes de iniciar |
| **Duck: doctor** | Diagnostico de ferramentas |
| **Duck: docker ps** | Lista containers |
| **Duck: kube pods** | Lista pods do cluster atual |

Exemplo de `args` customizados:

```json
"args": ["kube", "pods", "-n", "default"]
```

```json
"args": ["cert", "fetch", "https://example.com"]
```

### TUI (Bubble Tea)

Interfaces de terminal precisam do **terminal integrado**. Os perfis abaixo ja usam `"console": "integratedTerminal"`:

| Perfil | Descricao |
|--------|-----------|
| **Duck: tui** | Interface completa |
| **Duck: tui (readonly)** | Modo seguro, sem acoes destrutivas |
| **Duck: dashboard** | Dashboard compacto (`duck dashboard`) |

Variaveis de ambiente uteis durante o debug do TUI:

| Variavel | Efeito |
|----------|--------|
| `DUCK_TUI_REFRESH=30s` | Atualizacao mais lenta (menos ruido no debugger) |
| `DUCK_TUI_READONLY=true` | Bloqueia acoes mutaveis |
| `DUCK_TUI_CONFIRM=never` | Pula confirmacoes destrutivas |

Arquivos comuns para breakpoints no TUI:

- `internal/tui/app.go` — loop principal, teclas, render
- `internal/tui/docker.go` — carga e acoes Docker
- `internal/tui/kube.go` / `kube_actions.go` — Kubernetes
- `internal/tui/aws.go` — aba AWS
- `internal/tui/errors.go` — mensagens de erro amigaveis

### Testes

| Perfil | Uso |
|--------|-----|
| **Test: arquivo atual** | Depura o teste no arquivo aberto |
| **Test: internal/tui** | Todos os testes do pacote TUI |
| **Test: todos os pacotes** | `go test ./...` com debugger |

### Attach

**Attach: processo local** — anexa a um `duck` ou `duck.exe` ja em execucao. Util quando o problema so aparece fora do debugger.

## Tasks

Atalho **Ctrl+Shift+B** executa a task padrao **go: build duck** e gera `bin/duck.exe` (Windows) ou equivalente.

| Task | Comando |
|------|---------|
| **go: build duck** | `go build -o bin/duck.exe .` |
| **go: test** | `go test ./...` |
| **go: install delve** | Instala/atualiza o Delve |
| **go: fmt check** | Lista arquivos fora do padrao `gofmt` |

## Debug pela linha de comando (Delve)

Sem o editor, use o Delve diretamente:

```sh
# Depurar um comando
dlv debug . -- tui --readonly

# Depurar testes de um pacote
dlv test ./internal/tui -- -test.v

# Anexar a processo em execucao (Linux/macOS/WSL)
dlv attach <pid>
```

No Windows, `dlv attach` funciona para processos locais com permissoes adequadas.

Comandos uteis dentro do `dlv`:

```text
break internal/tui/app.go:200
continue
next
step
print m.dockerRows
quit
```

## Versao durante o debug

Builds locais pelo debugger mostram `dev` em `duck version`, a menos que voce passe `ldflags`. A task **go: build duck** injeta metadados basicos; releases oficiais usam o GoReleaser.

Para simular uma versao de release no build manual:

```sh
go build -ldflags "-X github.com/IKauedev/duck/internal/version.Version=0.1.2 -X github.com/IKauedev/duck/internal/version.Commit=$(git rev-parse --short HEAD) -X github.com/IKauedev/duck/internal/version.Date=$(date -u +%Y-%m-%dT%H:%M:%SZ)" -o bin/duck.exe .
```

## Problemas comuns

### `dlv` nao encontrado

Instale o Delve e adicione `~/go/bin` (ou `%USERPROFILE%\go\bin`) ao `PATH`. Reinicie o Cursor.

### TUI nao renderiza no debug

Use um perfil com `"console": "integratedTerminal"`. O painel **Debug Console** nao suporta aplicacoes fullscreen do Bubble Tea.

### Breakpoint nao para no TUI

O loop do Bubble Tea roda em goroutines. Coloque breakpoints em `Update`, handlers de mensagens (`dockerLoadedMsg`, etc.) ou em funcoes chamadas por `tea.Cmd`.

### Docker/Kube/AWS falham no debug

O debugger usa o mesmo ambiente do terminal integrado. Valide com `duck doctor` fora do debug. No Windows, ferramentas no WSL podem exigir que o PATH do WSL esteja configurado — o TUI tenta fallback via `wsl -e` automaticamente.

### `gofmt` falha no CI

Rode localmente:

```sh
gofmt -w .
```

Ou a task **go: fmt check** para listar arquivos pendentes.

## Referencia rapida de atalhos

| Atalho | Acao |
|--------|------|
| F5 | Iniciar / continuar |
| F9 | Toggle breakpoint |
| F10 | Step over |
| F11 | Step into |
| Shift+F11 | Step out |
| Shift+F5 | Parar debug |

## Leitura adicional

- [Delve documentation](https://github.com/go-delve/delve/tree/master/Documentation)
- [VS Code Go debugging](https://github.com/golang/vscode-go/wiki/debugging)
- [Bubble Tea](https://github.com/charmbracelet/bubbletea)
