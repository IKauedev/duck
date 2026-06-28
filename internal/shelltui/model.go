package shelltui

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"strings"
	"time"

	"github.com/IKauedev/duck/internal/cli"
	"github.com/IKauedev/duck/internal/config"
	"github.com/IKauedev/duck/internal/runner"
	"github.com/IKauedev/duck/internal/terminal"
	"github.com/IKauedev/duck/internal/version"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// tabKind representa a aba ativa
type tabKind int

const (
	tabShell tabKind = iota
	tabDocker
	tabKubernetes
	tabAWS
	tabGit
	tabTerraform
)

var tabNames = []string{"Shell", "Docker", "Kubernetes", "AWS", "Git", "Terraform"}
var tabKeys = []string{"1", "2", "3", "4", "5", "6"}

// focusArea controla qual área está com foco
type focusArea int

const (
	focusInput focusArea = iota
	focusViewport
)

// outputLine representa uma linha de output com tipo para colorização
type outputLine struct {
	text string
	kind string // "cmd", "out", "err", "info", "success", "warn", "system"
	ts   time.Time
}

// model é o estado completo do terminal TUI
type model struct {
	cfg      config.Config
	run      runner.Runner
	commands func() []cli.Command

	width  int
	height int

	activeTab tabKind
	focus     focusArea

	input    textinput.Model
	viewport viewport.Model

	history     []string
	historyIdx  int
	historyTemp string

	outputLines []outputLine
	cwd         string
	username    string
	hostname    string
	gitBranch   string

	showHelp    bool
	showSidebar bool

	lastError  error
	lastCmd    string
	lastStatus string // "ok", "error", "running"
	cmdCount   int
}

// cmdResultMsg carrega o resultado de um comando executado em goroutine
type cmdResultMsg struct {
	output string
	err    error
	cmd    string
	cwd    string
}

// gitBranchMsg carrega o branch git atual
type gitBranchMsg struct {
	branch string
}

// tabContentMsg carrega conteúdo de uma aba especial (docker, k8s, etc)
type tabContentMsg struct {
	tab     tabKind
	content string
	err     error
}

func newModel(cfg config.Config, run runner.Runner, commands func() []cli.Command) model {
	ti := textinput.New()
	ti.Placeholder = "comando duck ou shell..."
	ti.Focus()
	ti.CharLimit = 512

	cwd, _ := os.Getwd()

	username := resolveUsername()
	hostname := resolveHostname()

	vp := viewport.New(80, 20)
	vp.SetContent(welcomeMessage(username, hostname))

	return model{
		cfg:         cfg,
		run:         run,
		commands:    commands,
		input:       ti,
		viewport:    vp,
		activeTab:   tabShell,
		focus:       focusInput,
		cwd:         cwd,
		username:    username,
		hostname:    hostname,
		showSidebar: true,
		lastStatus:  "ok",
		outputLines: []outputLine{
			{text: welcomeMessage(username, hostname), kind: "system", ts: time.Now()},
		},
	}
}

func resolveUsername() string {
	if u, err := user.Current(); err == nil && u.Username != "" {
		// No Windows retorna DOMAIN\user; pegar só o user
		parts := strings.Split(u.Username, "\\")
		return parts[len(parts)-1]
	}
	if v := os.Getenv("USER"); v != "" {
		return v
	}
	if v := os.Getenv("USERNAME"); v != "" {
		return v
	}
	return "user"
}

func resolveHostname() string {
	if h, err := os.Hostname(); err == nil && h != "" {
		// Pegar só o primeiro segmento do FQDN
		return strings.SplitN(h, ".", 2)[0]
	}
	return "localhost"
}

func fetchGitBranch() tea.Cmd {
	return func() tea.Msg {
		out, err := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD").Output()
		if err != nil {
			return gitBranchMsg{branch: ""}
		}
		return gitBranchMsg{branch: strings.TrimSpace(string(out))}
	}
}

func (m model) Init() tea.Cmd {
	return tea.Batch(textinput.Blink, fetchGitBranch())
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.recalcSizes()
		return m, nil

	case gitBranchMsg:
		m.gitBranch = msg.branch
		return m, nil

	case cmdResultMsg:
		m.lastCmd = msg.cmd
		if msg.cwd != "" {
			m.cwd = msg.cwd
		}
		if msg.err != nil {
			m.lastError = msg.err
			m.lastStatus = "error"
			m.addOutput(styleError.Render("✗ "+msg.err.Error()), "err")
		} else {
			m.lastError = nil
			m.lastStatus = "ok"
		}
		if msg.output != "" {
			m.addOutput(msg.output, "out")
		}
		m.addOutput("", "system") // linha em branco após output
		m.syncViewport()
		m.cmdCount++
		return m, fetchGitBranch()

	case tabContentMsg:
		if msg.err != nil {
			m.addOutput(styleError.Render("Erro: "+msg.err.Error()), "err")
		} else {
			m.addOutput(msg.content, "out")
		}
		m.syncViewport()
		return m, nil

	case tea.KeyMsg:
		// Atalhos globais (sempre ativos)
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit

		case "ctrl+h", "f1":
			m.showHelp = !m.showHelp
			return m, nil

		case "ctrl+b":
			m.showSidebar = !m.showSidebar
			m.recalcSizes()
			return m, nil

		case "ctrl+l":
			m.outputLines = nil
			m.syncViewport()
			return m, nil

		case "esc":
			if m.showHelp {
				m.showHelp = false
				return m, nil
			}
			m.focus = focusInput
			m.input.Focus()
			return m, nil

		case "tab":
			if m.focus == focusInput {
				m.focus = focusViewport
				m.input.Blur()
			} else {
				m.focus = focusInput
				m.input.Focus()
			}
			return m, nil

		// Troca de abas
		case "alt+1", "f2":
			return m.switchTab(tabShell)
		case "alt+2", "f3":
			return m.switchTab(tabDocker)
		case "alt+3", "f4":
			return m.switchTab(tabKubernetes)
		case "alt+4", "f5":
			return m.switchTab(tabAWS)
		case "alt+5", "f6":
			return m.switchTab(tabGit)
		case "alt+6", "f7":
			return m.switchTab(tabTerraform)

		// Navegação histórico no input
		case "up":
			if m.focus == focusInput && len(m.history) > 0 {
				if m.historyIdx == 0 {
					m.historyTemp = m.input.Value()
				}
				if m.historyIdx < len(m.history) {
					m.historyIdx++
					m.input.SetValue(m.history[len(m.history)-m.historyIdx])
				}
				return m, nil
			}

		case "down":
			if m.focus == focusInput && m.historyIdx > 0 {
				m.historyIdx--
				if m.historyIdx == 0 {
					m.input.SetValue(m.historyTemp)
				} else {
					m.input.SetValue(m.history[len(m.history)-m.historyIdx])
				}
				return m, nil
			}

		case "enter":
			if m.focus == focusInput {
				line := strings.TrimSpace(m.input.Value())
				if line == "" {
					return m, nil
				}
				m.input.SetValue("")
				m.historyIdx = 0
				m.historyTemp = ""
				m.history = append(m.history, line)
				return m, m.executeCommand(line)
			}
		}
	}

	// Delegar atualização para o componente com foco
	if m.focus == focusViewport && !m.showHelp {
		var vpCmd tea.Cmd
		m.viewport, vpCmd = m.viewport.Update(msg)
		cmds = append(cmds, vpCmd)
	} else if m.focus == focusInput {
		var tiCmd tea.Cmd
		m.input, tiCmd = m.input.Update(msg)
		cmds = append(cmds, tiCmd)
	}

	return m, tea.Batch(cmds...)
}

func (m *model) switchTab(tab tabKind) (model, tea.Cmd) {
	m.activeTab = tab
	m.focus = focusInput
	m.input.Focus()
	var cmd tea.Cmd
	switch tab {
	case tabDocker:
		cmd = m.loadTabContent(tabDocker, "duck docker ps --all")
	case tabKubernetes:
		cmd = m.loadTabContent(tabKubernetes, "duck kubectl get pods --all-namespaces")
	case tabAWS:
		cmd = m.loadTabContent(tabAWS, "duck aws whoami")
	case tabGit:
		cmd = m.loadTabContent(tabGit, "duck git status")
	case tabTerraform:
		cmd = m.loadTabContent(tabTerraform, "duck terraform status")
	}
	m.addOutput(styleWarn.Render(fmt.Sprintf("─── Aba: %s ───", tabNames[tab])), "system")
	m.syncViewport()
	return *m, cmd
}

func (m *model) recalcSizes() {
	if m.width == 0 || m.height == 0 {
		return
	}
	// Altura do viewport = total - tabs(1) - input(2) - borders(2)
	vpHeight := m.height - 1 - 2 - 2
	if vpHeight < 5 {
		vpHeight = 5
	}
	vpWidth := m.width - 4
	if vpWidth < 20 {
		vpWidth = 20
	}
	m.viewport.Width = vpWidth
	m.viewport.Height = vpHeight
}

func (m *model) addOutput(text, kind string) {
	m.outputLines = append(m.outputLines, outputLine{
		text: text,
		kind: kind,
		ts:   time.Now(),
	})
}

func (m *model) syncViewport() {
	var sb strings.Builder
	for _, line := range m.outputLines {
		sb.WriteString(m.renderOutputLine(line))
		sb.WriteString("\n")
	}
	m.viewport.SetContent(sb.String())
	m.viewport.GotoBottom()
}

func (m model) renderOutputLine(line outputLine) string {
	switch line.kind {
	case "cmd":
		return line.text // já formatado com prompt Ubuntu
	case "err":
		return line.text
	case "success":
		return styleSuccess.Render(line.text)
	case "warn":
		return styleWarn.Render(line.text)
	case "system":
		return styleMuted.Render(line.text)
	default:
		return line.text
	}
}

func (m model) executeCommand(line string) tea.Cmd {
	lower := strings.ToLower(strings.TrimSpace(line))

	// Feedback imediato da linha de comando no viewport
	cwd, _ := os.Getwd()
	m.cwd = cwd
	m.lastStatus = "running"
	// Prompt estilo Ubuntu: user@host:path (branch)$
	m.addOutput(m.ubuntuPromptLine(cwd)+" "+styleBold.Render(line), "cmd")
	m.syncViewport()

	// Comandos internos do shell TUI
	switch lower {
	case "exit", "quit", "q":
		return tea.Quit
	case "clear", "cls":
		m.outputLines = nil
		m.syncViewport()
		return nil
	case "help", "?":
		m.showHelp = true
		return nil
	case "pwd":
		m.addOutput(styleSuccess.Render(cwd), "out")
		m.syncViewport()
		return nil
	}

	if strings.HasPrefix(lower, "cd ") {
		dir := strings.TrimSpace(line[3:])
		if err := os.Chdir(dir); err != nil {
			m.addOutput(styleError.Render("✗ "+err.Error()), "err")
			m.syncViewport()
			return nil
		}
		newCwd, _ := os.Getwd()
		m.cwd = newCwd
		m.addOutput(styleMuted.Render("→ "+newCwd), "system")
		m.syncViewport()
		return fetchGitBranch()
	}

	// Executar em goroutine para não bloquear a UI
	captureCwd := cwd
	return func() tea.Msg {
		var buf bytes.Buffer
		result := m.runDuckOrShell(line, &buf)
		newCwd, _ := os.Getwd()
		if newCwd == captureCwd {
			newCwd = ""
		}
		return cmdResultMsg{
			output: buf.String(),
			err:    result,
			cmd:    line,
			cwd:    newCwd,
		}
	}
}

// ubuntuPromptLine retorna o prompt formatado igual ao Ubuntu:
// user@hostname:~/path (branch)$
func (m model) ubuntuPromptLine(cwd string) string {
	userHost := styleUbuntuUserHost.Render(m.username + "@" + m.hostname)
	colon := styleUbuntuColon.Render(":")
	path := styleUbuntuPath.Render(shortCwd(cwd))
	dollar := styleUbuntoDollar.Render("$")

	if m.gitBranch != "" {
		branch := styleUbuntuBranch.Render(" (" + m.gitBranch + ")")
		return userHost + colon + path + branch + dollar
	}
	return userHost + colon + path + dollar
}

func (m model) loadTabContent(tab tabKind, duckCmd string) tea.Cmd {
	return func() tea.Msg {
		args := strings.Fields(duckCmd)
		if len(args) > 1 {
			args = args[1:] // remove "duck"
		}
		var buf bytes.Buffer
		cmds := m.commands()
		cli.RunWithOutput("duck", cmds, args, &buf)
		return tabContentMsg{
			tab:     tab,
			content: buf.String(),
		}
	}
}

func (m model) runDuckOrShell(line string, buf *bytes.Buffer) error {
	parts, err := terminal.ParseLine(line)
	if err != nil {
		return err
	}
	if len(parts) == 0 {
		return nil
	}
	if parts[0] == "duck" {
		parts = parts[1:]
	}
	if len(parts) == 0 {
		return nil
	}

	cmds := m.commands()
	if terminal.IsDuckCommand(parts, cmds) {
		code := cli.RunWithOutput("duck", cmds, parts, buf)
		if code != 0 {
			return fmt.Errorf("comando retornou código %d", code)
		}
		return nil
	}

	// Comando de shell nativo — capturar output
	return runShellCapture(line, buf)
}

func (m model) View() string {
	if m.width == 0 {
		return "carregando..."
	}
	if m.showHelp {
		return m.renderHelp()
	}

	tabs := m.renderTabs()
	body := m.renderViewport()
	input := m.renderInput()

	return lipgloss.JoinVertical(lipgloss.Left,
		tabs,
		body,
		input,
	)
}

func (m model) renderHeader() string {
	left := styleHeader.Render(" duck ") +
		styleHeaderDim.Render("v"+version.Label()+"  "+shortCwd(m.cwd))
	right := styleHeaderDim.Render("F1 help  Ctrl+B sidebar  Tab focus ")

	contentW := m.width - lipgloss.Width(left) - lipgloss.Width(right)
	if contentW < 0 {
		contentW = 0
	}
	fill := styleHeaderBar.Render(strings.Repeat(" ", contentW))
	return left + fill + right
}

func (m model) renderTabs() string {
	var parts []string
	for i, name := range tabNames {
		if tabKind(i) == m.activeTab {
			parts = append(parts, styleTabActive.Render(name))
		} else {
			parts = append(parts, styleTabInactive.Render(name))
		}
		if i < len(tabNames)-1 {
			parts = append(parts, styleTabSep.Render("│"))
		}
	}
	bar := strings.Join(parts, "")
	pad := m.width - lipgloss.Width(bar)
	if pad > 0 {
		bar += styleTabBar.Render(strings.Repeat(" ", pad))
	}
	return bar
}

func (m model) renderBody() string {
	vpContent := m.renderViewport()
	if !m.showSidebar {
		return vpContent
	}
	sidebar := m.renderSidebar()
	return lipgloss.JoinHorizontal(lipgloss.Top, vpContent, sidebar)
}

func (m model) renderViewport() string {
	vpStyle := styleViewport
	if m.focus == focusViewport {
		vpStyle = styleViewportFocused
	}
	vpWidth := m.viewport.Width
	if vpWidth < 4 {
		vpWidth = 4
	}
	return vpStyle.Width(vpWidth).Render(m.viewport.View())
}

func (m model) renderSidebar() string {
	var sb strings.Builder

	sb.WriteString(styleSidebarTitle.Render("Atalhos"))
	sb.WriteString("\n")

	sections := sidebarSections(m.activeTab)
	for _, sec := range sections {
		sb.WriteString(styleSidebarCategory.Render(sec.title) + "\n")
		for _, item := range sec.items {
			key := styleSidebarKey.Render(item.key)
			desc := styleSidebarDesc.Render(item.desc)
			sb.WriteString(key + desc + "\n")
		}
		sb.WriteString("\n")
	}

	// Info contextual
	sb.WriteString(styleSidebarCategory.Render("Info") + "\n")
	sb.WriteString(styleMuted.Render(fmt.Sprintf("Cmds: %d", m.cmdCount)) + "\n")
	if m.lastCmd != "" {
		short := m.lastCmd
		if len(short) > 18 {
			short = short[:15] + "..."
		}
		sb.WriteString(styleMuted.Render("Último: "+short) + "\n")
	}
	switch m.lastStatus {
	case "ok":
		sb.WriteString(styleSuccess.Render("● ok") + "\n")
	case "error":
		sb.WriteString(styleError.Render("● erro") + "\n")
	case "running":
		sb.WriteString(styleWarn.Render("● executando") + "\n")
	}

	sidebarWidth := 30
	return styleSidebar.Width(sidebarWidth).Render(sb.String())
}

func (m model) renderInput() string {
	cwd, _ := os.Getwd()
	prompt := m.ubuntuPromptLine(cwd) + " "
	promptW := lipgloss.Width(prompt)

	// Largura disponível para o texto de input
	inputW := m.width - promptW - 4 // 4 = margem+bordas
	if inputW < 10 {
		inputW = 10
	}
	m.input.Width = inputW

	inner := prompt + m.input.View()
	// Borda apenas inferior, largura total
	return styleInputBox.Width(m.width - 2).Render(inner)
}

func (m model) renderStatusBar() string {
	statusIcon := styleSuccess.Render("●")
	if m.lastStatus == "error" {
		statusIcon = styleError.Render("●")
	} else if m.lastStatus == "running" {
		statusIcon = styleWarn.Render("…")
	}

	tabLabel := styleStatusItem.Render(tabNames[m.activeTab])

	scrollPct := "  0%"
	if m.viewport.TotalLineCount() > 0 {
		pct := int(100 * float64(m.viewport.YOffset) / float64(max(1, m.viewport.TotalLineCount()-m.viewport.Height)))
		scrollPct = fmt.Sprintf("%3d%%", pct)
	}

	hints := " enter exec  tab focus  f1 help  ctrl+c quit "

	left := styleStatusMuted.Render(" ") + statusIcon + styleStatusMuted.Render(" ") + tabLabel
	right := styleStatusMuted.Render(scrollPct + hints)

	gap := m.width - lipgloss.Width(left) - lipgloss.Width(right)
	if gap < 0 {
		gap = 0
	}
	fill := styleStatusBar.Render(strings.Repeat(" ", gap))

	return left + fill + right
}

func (m model) renderHelp() string {
	var sb strings.Builder

	sb.WriteString(styleHelpTitle.Render("🦆 Duck Shell — Ajuda") + "\n\n")

	sections := []struct {
		title string
		items [][2]string
	}{
		{
			title: "Navegação",
			items: [][2]string{
				{"Tab", "Alternar foco entre input e viewport"},
				{"Esc", "Voltar o foco para o input"},
				{"↑ / ↓", "Navegar histórico de comandos"},
				{"Ctrl+B", "Mostrar/ocultar sidebar"},
				{"F1 / Ctrl+H", "Esta janela de ajuda"},
			},
		},
		{
			title: "Abas",
			items: [][2]string{
				{"Alt+1 / F2", "Aba Shell (REPL)"},
				{"Alt+2 / F3", "Aba Docker"},
				{"Alt+3 / F4", "Aba Kubernetes"},
				{"Alt+4 / F5", "Aba AWS"},
				{"Alt+5 / F6", "Aba Git"},
				{"Alt+6 / F7", "Aba Terraform"},
			},
		},
		{
			title: "Viewport (foco no viewport)",
			items: [][2]string{
				{"↑ / ↓", "Rolar linha a linha"},
				{"PgUp / PgDn", "Rolar página"},
				{"g / G", "Início / fim"},
			},
		},
		{
			title: "Comandos Internos",
			items: [][2]string{
				{"clear / cls", "Limpar output"},
				{"Ctrl+L", "Limpar output (atalho)"},
				{"pwd", "Mostrar diretório atual"},
				{"cd <dir>", "Mudar diretório"},
				{"exit / quit", "Sair do Duck Shell"},
				{"help / ?", "Esta ajuda"},
			},
		},
		{
			title: "Execução",
			items: [][2]string{
				{"Enter", "Executar comando"},
				{"duck <cmd>", "Executar comando Duck"},
				{"$ <cmd>", "Forçar comando nativo do SO"},
				{"docker ps", "Listar containers"},
				{"kubectl get pods", "Listar pods"},
			},
		},
	}

	for _, sec := range sections {
		sb.WriteString(styleHelpSection.Render("▶ "+sec.title) + "\n")
		for _, item := range sec.items {
			key := styleHelpKey.Render(item[0])
			desc := styleHelpDesc.Render(item[1])
			sb.WriteString("  " + key + "  " + desc + "\n")
		}
		sb.WriteString("\n")
	}

	sb.WriteString(styleMuted.Render("Pressione F1 ou Esc para fechar"))

	w := min(m.width-8, 70)
	h := min(m.height-4, 40)
	overlay := styleHelpOverlay.Width(w).Height(h).Render(sb.String())

	// Centralizar
	topPad := (m.height - lipgloss.Height(overlay)) / 2
	leftPad := (m.width - lipgloss.Width(overlay)) / 2
	if topPad < 0 {
		topPad = 0
	}
	if leftPad < 0 {
		leftPad = 0
	}
	return strings.Repeat("\n", topPad) +
		strings.Repeat(" ", leftPad) + overlay
}

func shortCwd(cwd string) string {
	home, _ := os.UserHomeDir()
	if home != "" && strings.HasPrefix(cwd, home) {
		cwd = "~" + cwd[len(home):]
	}
	parts := strings.Split(strings.ReplaceAll(cwd, "\\", "/"), "/")
	if len(parts) > 3 {
		parts = append([]string{"..."}, parts[len(parts)-2:]...)
	}
	return strings.Join(parts, "/")
}

func welcomeMessage(username, hostname string) string {
	var sb strings.Builder
	// Banner ASCII estilo Ubuntu
	sb.WriteString(styleBold.Render("🦆 Duck Shell") + "  " + styleMuted.Render(fmt.Sprintf("v%s", version.Label())) + "\n")
	sb.WriteString(styleUbuntuUserHost.Render(username+"@"+hostname) + styleMuted.Render(" — terminal interativo estilo Ubuntu") + "\n")
	sb.WriteString("\n")
	sb.WriteString(styleSuccess.Render("  Comandos disponíveis:") + "\n")
	sb.WriteString(styleMuted.Render("  docker  kubectl  aws  git  terraform  helm") + "\n")
	sb.WriteString(styleMuted.Render("  status  config  profile  task  aliases  envcheck") + "\n")
	sb.WriteString("\n")
	sb.WriteString(styleWarn.Render("  Dicas: ") + "\n")
	sb.WriteString(styleMuted.Render("  • Tab = alternar foco input/viewport") + "\n")
	sb.WriteString(styleMuted.Render("  • F1  = ajuda completa") + "\n")
	sb.WriteString(styleMuted.Render("  • Alt+2~6 = abas Docker/K8s/AWS/Git/Terraform") + "\n")
	sb.WriteString(styleMuted.Render("  • $ <cmd> = executar comando nativo do SO") + "\n")
	sb.WriteString("\n")
	sb.WriteString(styleMuted.Render("─────────────────────────────────────────────────────") + "\n")
	return sb.String()
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
