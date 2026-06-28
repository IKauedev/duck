package gittui

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ── views ───────────────────────────────────────────────────────────────────

type viewKind int

const (
	viewLog viewKind = iota
	viewBranches
	viewStatus
	viewStash
)

var viewNames = []string{"Log", "Branches", "Status", "Stash"}
var viewKeys = []string{"1", "2", "3", "4"}

// ── data rows ────────────────────────────────────────────────────────────────

type commitRow struct {
	hash    string
	date    string
	author  string
	subject string
	refs    string
}

type branchRow struct {
	name    string
	current bool
	remote  bool
	ahead   int
	behind  int
	last    string
}

type statusRow struct {
	xy   string
	path string
}

type stashRow struct {
	index   int
	message string
	branch  string
}

// ── messages ─────────────────────────────────────────────────────────────────

type loadedMsg struct {
	view viewKind
	err  error

	commits  []commitRow
	branches []branchRow
	statuses []statusRow
	stashes  []stashRow
}

type actionDoneMsg struct {
	output string
	err    error
}

type tickMsg time.Time

// ── model ────────────────────────────────────────────────────────────────────

type model struct {
	width  int
	height int

	activeView viewKind
	loading    bool
	err        error

	commits  []commitRow
	branches []branchRow
	statuses []statusRow
	stashes  []stashRow

	cursor int

	mode         string // list | detail | input | confirm
	detailTitle  string
	detailBody   string
	detailVP     viewport.Model
	inputModel   textinput.Model
	inputPrompt  string
	inputPending string // action to run after input: "commit", "branch-new", "stash-save"
	confirmMsg   string
	confirmCmd   []string

	message    string
	messageErr bool
}

func newModel() model {
	vp := viewport.New(80, 20)
	ti := textinput.New()
	ti.CharLimit = 200

	return model{
		activeView: viewLog,
		loading:    true,
		mode:       "list",
		detailVP:   vp,
		inputModel: ti,
	}
}

// ── init ─────────────────────────────────────────────────────────────────────

func (m model) Init() tea.Cmd {
	return tea.Batch(loadView(viewLog), tickCmd())
}

func tickCmd() tea.Cmd {
	return tea.Tick(30*time.Second, func(t time.Time) tea.Msg { return tickMsg(t) })
}

// ── update ───────────────────────────────────────────────────────────────────

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.detailVP.Width = msg.Width - 4
		m.detailVP.Height = msg.Height - 6
		return m, nil

	case tickMsg:
		return m, tea.Batch(loadView(m.activeView), tickCmd())

	case loadedMsg:
		m.loading = false
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		switch msg.view {
		case viewLog:
			m.commits = msg.commits
		case viewBranches:
			m.branches = msg.branches
		case viewStatus:
			m.statuses = msg.statuses
		case viewStash:
			m.stashes = msg.stashes
		}
		if msg.view == m.activeView && m.cursor > m.rowCount()-1 {
			m.cursor = max(0, m.rowCount()-1)
		}
		return m, nil

	case actionDoneMsg:
		m.mode = "list"
		if msg.err != nil {
			m.message = "Erro: " + msg.err.Error()
			m.messageErr = true
		} else {
			m.message = strings.TrimSpace(msg.output)
			m.messageErr = false
		}
		return m, loadView(m.activeView)

	case tea.KeyMsg:
		return m.handleKey(msg)
	}

	// propagar para viewport ou input quando necessário
	if m.mode == "detail" {
		var cmd tea.Cmd
		m.detailVP, cmd = m.detailVP.Update(msg)
		return m, cmd
	}
	if m.mode == "input" {
		var cmd tea.Cmd
		m.inputModel, cmd = m.inputModel.Update(msg)
		return m, cmd
	}
	return m, nil
}

func (m model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()

	// fechar sobreposições
	if m.mode == "detail" {
		switch key {
		case "q", "esc", "backspace":
			m.mode = "list"
		default:
			var cmd tea.Cmd
			m.detailVP, cmd = m.detailVP.Update(msg)
			return m, cmd
		}
		return m, nil
	}

	if m.mode == "confirm" {
		switch key {
		case "y", "Y", "s", "S":
			cmd := m.confirmCmd
			m.mode = "list"
			m.confirmMsg = ""
			return m, runGit(cmd...)
		default:
			m.mode = "list"
			m.confirmMsg = ""
			m.message = "Cancelado."
			m.messageErr = false
		}
		return m, nil
	}

	if m.mode == "input" {
		switch key {
		case "enter":
			val := strings.TrimSpace(m.inputModel.Value())
			if val == "" {
				m.message = "Valor não pode ser vazio."
				m.messageErr = true
				m.mode = "list"
				return m, nil
			}
			pending := m.inputPending
			m.mode = "list"
			m.inputModel.Reset()
			return m, m.executeInputAction(pending, val)
		case "esc":
			m.mode = "list"
			m.inputModel.Reset()
			m.message = "Cancelado."
			m.messageErr = false
		default:
			var cmd tea.Cmd
			m.inputModel, cmd = m.inputModel.Update(msg)
			return m, cmd
		}
		return m, nil
	}

	// navegação global por tab
	switch key {
	case "1":
		return m.switchView(viewLog)
	case "2":
		return m.switchView(viewBranches)
	case "3":
		return m.switchView(viewStatus)
	case "4":
		return m.switchView(viewStash)
	case "q", "ctrl+c":
		return m, tea.Quit
	case "?", "h":
		return m.showHelp()
	case "r":
		m.loading = true
		return m, loadView(m.activeView)
	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}
		m.message = ""
	case "down", "j":
		if m.cursor < m.rowCount()-1 {
			m.cursor++
		}
		m.message = ""
	case "g", "home":
		m.cursor = 0
	case "G", "end":
		m.cursor = max(0, m.rowCount()-1)
	case "enter", " ":
		return m.handleEnter()
	case "d":
		return m.handleDiff()
	case "c":
		if m.activeView == viewStatus {
			return m.startInput("Mensagem do commit:", "commit")
		}
	case "p":
		if m.activeView == viewLog || m.activeView == viewBranches {
			return m, m.confirmAction("Fazer push da branch atual? (y/N)", "git", "push")
		}
	case "P":
		if m.activeView == viewLog || m.activeView == viewBranches {
			return m, runGit("pull", "--rebase")
		}
	case "n":
		if m.activeView == viewBranches {
			return m.startInput("Nome da nova branch:", "branch-new")
		}
	case "x":
		if m.activeView == viewStash {
			return m.handleStashDrop()
		}
		if m.activeView == viewStatus {
			return m.handleDiscard()
		}
	case "a":
		if m.activeView == viewStatus {
			return m, runGit("add", "-A")
		}
	case "s":
		if m.activeView == viewStatus {
			return m.startInput("Mensagem do stash (opcional):", "stash-save")
		}
	case "A":
		if m.activeView == viewStash {
			return m.handleStashApply()
		}
	}
	return m, nil
}

func (m model) switchView(v viewKind) (model, tea.Cmd) {
	m.activeView = v
	m.cursor = 0
	m.loading = true
	m.message = ""
	return m, loadView(v)
}

func (m model) rowCount() int {
	switch m.activeView {
	case viewLog:
		return len(m.commits)
	case viewBranches:
		return len(m.branches)
	case viewStatus:
		return len(m.statuses)
	case viewStash:
		return len(m.stashes)
	}
	return 0
}

func (m model) handleEnter() (model, tea.Cmd) {
	switch m.activeView {
	case viewLog:
		if len(m.commits) == 0 {
			return m, nil
		}
		hash := m.commits[m.cursor].hash
		return m.openDetail("Commit "+hash, gitOutput("show", "--stat", "--patch", hash))
	case viewBranches:
		if len(m.branches) == 0 {
			return m, nil
		}
		br := m.branches[m.cursor]
		if br.current {
			m.message = "Já está na branch " + br.name
			return m, nil
		}
		name := br.name
		if br.remote {
			// strip "origin/" prefix for local checkout
			name = strings.TrimPrefix(name, "origin/")
		}
		return m, m.confirmAction(fmt.Sprintf("Fazer checkout para '%s'? (y/N)", name), "git", "switch", name)
	case viewStash:
		if len(m.stashes) == 0 {
			return m, nil
		}
		ref := fmt.Sprintf("stash@{%d}", m.stashes[m.cursor].index)
		return m.openDetail("Stash "+ref, gitOutput("stash", "show", "-p", ref))
	}
	return m, nil
}

func (m model) handleDiff() (model, tea.Cmd) {
	switch m.activeView {
	case viewStatus:
		if len(m.statuses) == 0 {
			return m, nil
		}
		path := m.statuses[m.cursor].path
		return m.openDetail("Diff: "+path, gitOutput("diff", "--", path))
	case viewLog:
		if len(m.commits) == 0 {
			return m, nil
		}
		hash := m.commits[m.cursor].hash
		return m.openDetail("Diff: "+hash, gitOutput("show", hash))
	}
	return m, nil
}

func (m model) handleStashDrop() (model, tea.Cmd) {
	if len(m.stashes) == 0 {
		return m, nil
	}
	ref := fmt.Sprintf("stash@{%d}", m.stashes[m.cursor].index)
	return m, m.confirmAction(fmt.Sprintf("Remover '%s'? (y/N)", ref), "git", "stash", "drop", ref)
}

func (m model) handleStashApply() (model, tea.Cmd) {
	if len(m.stashes) == 0 {
		return m, nil
	}
	ref := fmt.Sprintf("stash@{%d}", m.stashes[m.cursor].index)
	return m, m.confirmAction(fmt.Sprintf("Aplicar '%s'? (y/N)", ref), "git", "stash", "apply", ref)
}

func (m model) handleDiscard() (model, tea.Cmd) {
	if len(m.statuses) == 0 {
		return m, nil
	}
	path := m.statuses[m.cursor].path
	return m, m.confirmAction(fmt.Sprintf("Descartar mudanças em '%s'? (y/N)", path), "git", "checkout", "--", path)
}

func (m model) openDetail(title, body string) (model, tea.Cmd) {
	m.mode = "detail"
	m.detailTitle = title
	m.detailBody = body
	m.detailVP.SetContent(body)
	m.detailVP.GotoTop()
	return m, nil
}

func (m *model) startInput(prompt, action string) (model, tea.Cmd) {
	m.mode = "input"
	m.inputPrompt = prompt
	m.inputPending = action
	m.inputModel.Reset()
	m.inputModel.Focus()
	return *m, nil
}

func (m model) confirmAction(msg string, gitArgs ...string) tea.Cmd {
	m.confirmMsg = msg
	m.confirmCmd = gitArgs
	return nil // caller sets mode
}

// confirmAction sets mode and stores data, returns model + nil cmd
// called inline:
func (m model) confirmActionM(msg string, gitArgs ...string) (model, tea.Cmd) {
	m.mode = "confirm"
	m.confirmMsg = msg
	m.confirmCmd = gitArgs
	return m, nil
}

// override: re-route through confirmActionM
func (m model) confirmActionCmd(msg string, gitArgs ...string) (model, tea.Cmd) {
	return m.confirmActionM(msg, gitArgs...)
}

func (m model) executeInputAction(action, value string) tea.Cmd {
	switch action {
	case "commit":
		return tea.Batch(runGit("add", "-A"), runGit("commit", "-m", value))
	case "branch-new":
		return runGit("checkout", "-b", value)
	case "stash-save":
		if value == "" {
			return runGit("stash")
		}
		return runGit("stash", "save", value)
	}
	return nil
}

func (m model) showHelp() (model, tea.Cmd) {
	help := `  NAVEGAÇÃO
  ──────────────────────────────────────
  ↑/k, ↓/j   Mover cursor
  g/Home      Ir para o início
  G/End       Ir para o fim
  1-4         Trocar aba (Log/Branches/Status/Stash)
  Enter       Selecionar / Checkout / Expandir
  r           Recarregar
  q           Sair
  ?/h         Esta ajuda

  LOG
  ──────────────────────────────────────
  Enter       Ver detalhes do commit
  d           Ver diff do commit
  p           Push branch
  P           Pull --rebase

  BRANCHES
  ──────────────────────────────────────
  Enter       Checkout para branch
  n           Criar nova branch
  p           Push
  P           Pull --rebase

  STATUS
  ──────────────────────────────────────
  a           git add -A (adicionar tudo)
  c           Commit (abre campo de mensagem)
  d           Ver diff do arquivo
  s           Stash mudanças
  x           Descartar mudanças no arquivo

  STASH
  ──────────────────────────────────────
  Enter       Ver conteúdo do stash
  A           Aplicar stash selecionado
  x           Remover stash selecionado
`
	return m.openDetail("Ajuda — Git TUI", help)
}

// ── view ─────────────────────────────────────────────────────────────────────

func (m model) View() string {
	if m.width == 0 {
		return "Carregando..."
	}

	header := m.renderHeader()
	footer := m.renderFooter()
	bodyH := m.height - lipgloss.Height(header) - lipgloss.Height(footer) - 1

	var body string
	switch m.mode {
	case "detail":
		m.detailVP.Height = bodyH
		body = renderDetailView(m.detailTitle, m.detailVP)
	case "input":
		body = m.renderInputView(bodyH)
	case "confirm":
		body = m.renderConfirmView(bodyH)
	default:
		body = m.renderListView(bodyH)
	}

	return lipgloss.JoinVertical(lipgloss.Left, header, body, footer)
}

func (m model) renderHeader() string {
	var tabs []string
	for i, name := range viewNames {
		label := fmt.Sprintf(" %s %s ", viewKeys[i], name)
		if viewKind(i) == m.activeView {
			tabs = append(tabs, activeTabSt.Render(label))
		} else {
			tabs = append(tabs, tabSt.Render(label))
		}
	}
	tabBar := lipgloss.JoinHorizontal(lipgloss.Top, tabs...)

	branch := gitBranchCurrent()
	repoName := gitRepoName()
	right := mutedSt.Render(fmt.Sprintf(" %s  %s ", repoName, branch))

	gap := m.width - lipgloss.Width(tabBar) - lipgloss.Width(right)
	if gap < 0 {
		gap = 0
	}
	return tabBar + strings.Repeat(" ", gap) + right
}

func (m model) renderFooter() string {
	msg := ""
	if m.message != "" {
		if m.messageErr {
			msg = errSt.Render(" " + m.message + " ")
		} else {
			msg = successSt.Render(" " + m.message + " ")
		}
	}
	if m.loading {
		msg = mutedSt.Render(" Carregando...")
	}
	if m.err != nil {
		msg = errSt.Render(" Erro: " + m.err.Error())
	}
	keyHint := mutedSt.Render(" q:sair  ?:ajuda  r:recarregar  1-4:abas ")
	gap := m.width - lipgloss.Width(msg) - lipgloss.Width(keyHint)
	if gap < 0 {
		gap = 0
	}
	return msg + strings.Repeat(" ", gap) + keyHint
}

func (m model) renderListView(height int) string {
	lines := m.buildRows()
	if len(lines) == 0 {
		return padLines(mutedSt.Render("  (vazio)"), height)
	}

	visibleStart := 0
	if m.cursor >= height {
		visibleStart = m.cursor - height + 1
	}
	visibleEnd := visibleStart + height
	if visibleEnd > len(lines) {
		visibleEnd = len(lines)
	}

	var sb strings.Builder
	for i := visibleStart; i < visibleEnd; i++ {
		if i == m.cursor {
			sb.WriteString(selectedSt.Width(m.width).Render(lines[i]))
		} else {
			sb.WriteString(rowSt.Render(lines[i]))
		}
		sb.WriteByte('\n')
	}
	return sb.String()
}

func (m model) buildRows() []string {
	switch m.activeView {
	case viewLog:
		rows := make([]string, len(m.commits))
		for i, c := range m.commits {
			refs := ""
			if c.refs != "" {
				refs = refSt.Render(" " + c.refs + " ")
			}
			rows[i] = fmt.Sprintf(" %s  %s  %s%s",
				hashSt.Render(c.hash),
				mutedSt.Render(c.date),
				c.subject,
				refs,
			)
		}
		return rows
	case viewBranches:
		rows := make([]string, len(m.branches))
		for i, b := range m.branches {
			marker := "  "
			if b.current {
				marker = currentBranchSt.Render("* ")
			}
			track := ""
			if b.ahead > 0 || b.behind > 0 {
				track = mutedSt.Render(fmt.Sprintf(" ↑%d ↓%d", b.ahead, b.behind))
			}
			kind := ""
			if b.remote {
				kind = mutedSt.Render(" remote")
			}
			rows[i] = fmt.Sprintf("%s%-40s%s%s  %s", marker, b.name, kind, track, mutedSt.Render(b.last))
		}
		return rows
	case viewStatus:
		rows := make([]string, len(m.statuses))
		for i, s := range m.statuses {
			rows[i] = fmt.Sprintf("  %s  %s", statusXYStyle(s.xy), s.path)
		}
		return rows
	case viewStash:
		rows := make([]string, len(m.stashes))
		for i, s := range m.stashes {
			rows[i] = fmt.Sprintf("  stash@{%d}  %s  %s", s.index, mutedSt.Render(s.branch), s.message)
		}
		return rows
	}
	return nil
}

func renderDetailView(title string, vp viewport.Model) string {
	header := detailTitleSt.Render(" " + title + " ")
	divider := mutedSt.Render(strings.Repeat("─", vp.Width))
	return lipgloss.JoinVertical(lipgloss.Left, header, divider, vp.View())
}

func (m model) renderInputView(height int) string {
	prompt := inputPromptSt.Render(" " + m.inputPrompt)
	field := "  " + m.inputModel.View()
	hint := mutedSt.Render("  Enter confirmar  ·  Esc cancelar")
	body := lipgloss.JoinVertical(lipgloss.Left,
		strings.Repeat("\n", height/3),
		prompt, field, hint,
	)
	return body
}

func (m model) renderConfirmView(height int) string {
	msg := confirmSt.Render(" " + m.confirmMsg)
	body := lipgloss.JoinVertical(lipgloss.Left,
		strings.Repeat("\n", height/3),
		msg,
		mutedSt.Render("  y/s = confirmar  ·  qualquer tecla = cancelar"),
	)
	return body
}

// ── helpers ──────────────────────────────────────────────────────────────────

func padLines(s string, height int) string {
	lines := strings.Count(s, "\n") + 1
	if lines < height {
		s += strings.Repeat("\n", height-lines)
	}
	return s
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func gitOutput(args ...string) string {
	out, err := exec.Command("git", args...).Output()
	if err != nil {
		return fmt.Sprintf("(erro: %v)", err)
	}
	return string(out)
}

func gitBranchCurrent() string {
	out, err := exec.Command("git", "branch", "--show-current").Output()
	if err != nil {
		return "?"
	}
	return strings.TrimSpace(string(out))
}

func gitRepoName() string {
	out, err := exec.Command("git", "rev-parse", "--show-toplevel").Output()
	if err != nil {
		return "?"
	}
	parts := strings.Split(strings.TrimSpace(string(out)), string(os.PathSeparator))
	if len(parts) == 0 {
		return "?"
	}
	return parts[len(parts)-1]
}

func statusXYStyle(xy string) string {
	xy = strings.TrimSpace(xy)
	switch {
	case strings.Contains(xy, "M"):
		return modifiedSt.Render(xy)
	case strings.Contains(xy, "A"):
		return addedSt.Render(xy)
	case strings.Contains(xy, "D"):
		return deletedSt.Render(xy)
	case strings.Contains(xy, "?"):
		return untrackedSt.Render(xy)
	case strings.Contains(xy, "R"):
		return renamedSt.Render(xy)
	}
	return xy
}

// ── git data loading ──────────────────────────────────────────────────────────

func loadView(v viewKind) tea.Cmd {
	return func() tea.Msg {
		msg := loadedMsg{view: v}
		var err error
		switch v {
		case viewLog:
			msg.commits, err = fetchLog()
		case viewBranches:
			msg.branches, err = fetchBranches()
		case viewStatus:
			msg.statuses, err = fetchStatus()
		case viewStash:
			msg.stashes, err = fetchStash()
		}
		msg.err = err
		return msg
	}
}

func fetchLog() ([]commitRow, error) {
	out, err := exec.Command("git", "log",
		"--pretty=format:%h\x1f%ad\x1f%an\x1f%s\x1f%D",
		"--date=short",
		"-n", "200",
	).Output()
	if err != nil {
		return nil, fmt.Errorf("git log: %w", err)
	}
	var rows []commitRow
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "\x1f", 5)
		if len(parts) < 5 {
			continue
		}
		rows = append(rows, commitRow{
			hash:    parts[0],
			date:    parts[1],
			author:  parts[2],
			subject: parts[3],
			refs:    parts[4],
		})
	}
	return rows, nil
}

func fetchBranches() ([]branchRow, error) {
	// local branches
	out, err := exec.Command("git", "branch", "-vv").Output()
	if err != nil {
		return nil, fmt.Errorf("git branch: %w", err)
	}
	var rows []branchRow
	for _, line := range strings.Split(string(out), "\n") {
		if line == "" {
			continue
		}
		current := false
		if strings.HasPrefix(line, "*") {
			current = true
		}
		line = strings.TrimPrefix(line, "* ")
		line = strings.TrimPrefix(line, "  ")
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		name := fields[0]
		// parse ahead/behind from "[origin/x: ahead N, behind M]"
		ahead, behind := 0, 0
		rest := strings.Join(fields[2:], " ")
		if idx := strings.Index(rest, "ahead"); idx >= 0 {
			fmt.Sscanf(rest[idx+6:], "%d", &ahead)
		}
		if idx := strings.Index(rest, "behind"); idx >= 0 {
			fmt.Sscanf(rest[idx+7:], "%d", &behind)
		}
		rows = append(rows, branchRow{
			name:    name,
			current: current,
			ahead:   ahead,
			behind:  behind,
		})
	}

	// remote branches
	outR, err := exec.Command("git", "branch", "-r").Output()
	if err == nil {
		for _, line := range strings.Split(string(outR), "\n") {
			line = strings.TrimSpace(line)
			if line == "" || strings.Contains(line, "->") {
				continue
			}
			rows = append(rows, branchRow{name: line, remote: true})
		}
	}
	return rows, nil
}

func fetchStatus() ([]statusRow, error) {
	out, err := exec.Command("git", "status", "--porcelain").Output()
	if err != nil {
		return nil, fmt.Errorf("git status: %w", err)
	}
	var rows []statusRow
	for _, line := range strings.Split(string(out), "\n") {
		if len(line) < 3 {
			continue
		}
		rows = append(rows, statusRow{
			xy:   line[0:2],
			path: strings.TrimSpace(line[3:]),
		})
	}
	return rows, nil
}

func fetchStash() ([]stashRow, error) {
	out, err := exec.Command("git", "stash", "list", "--format=%gd\x1f%gs\x1f%gD").Output()
	if err != nil {
		return nil, fmt.Errorf("git stash list: %w", err)
	}
	var rows []stashRow
	for i, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "\x1f", 3)
		msg := ""
		branch := ""
		if len(parts) >= 2 {
			msg = parts[1]
		}
		if idx := strings.Index(msg, "On "); idx >= 0 {
			rest := msg[idx+3:]
			if colon := strings.Index(rest, ":"); colon >= 0 {
				branch = rest[:colon]
				msg = strings.TrimSpace(rest[colon+1:])
			}
		}
		rows = append(rows, stashRow{
			index:   i,
			message: msg,
			branch:  branch,
		})
	}
	return rows, nil
}

// ── git actions ───────────────────────────────────────────────────────────────

func runGit(args ...string) tea.Cmd {
	return func() tea.Msg {
		out, err := exec.Command("git", args...).CombinedOutput()
		return actionDoneMsg{output: string(out), err: err}
	}
}
