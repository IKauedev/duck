package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/IKauedev/duck/internal/config"
	"github.com/IKauedev/duck/internal/runner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type quitWithActionMsg struct {
	pending pendingAction
}

const (
	dockerView = iota
	kubeView
	awsView
)

type viewKind int

type pendingAction struct {
	binary      string
	args        []string
	interactive bool
	duckArgs    []string
}

type model struct {
	cfg config.Config
	run runner.Runner
	opts Options

	activeView viewKind
	mode       string // list, filter, detail, confirm, favorites, tasks, help
	width      int
	height     int

	filter     string
	filterInput string

	dockerRows     []dockerRow
	dockerCursor   int
	dockerVersion  string
	dockerShowAll  bool
	dockerErr      error
	dockerErrText  string

	kubeRows          []kubeRow
	kubeCursor        int
	kubeContext       string
	kubeClusterInfo   string
	kubeResource      kubeResourceKind
	kubeAllNamespaces bool
	kubeNamespace     string
	kubeEditTitle     string
	kubeEditInput     string
	kubeErr           error
	kubeErrText       string

	awsVersion         string
	awsIdentity        string
	awsErr             error
	awsErrText         string
	awsIdentityErr     error
	awsIdentityErrText string

	detailTitle string
	detailBody  string

	confirmAction string
	confirmTarget string

	message       string
	loading       bool
	quitting      bool
	pending       *pendingAction

	dockerBackend toolBackend
	kubeBackend   toolBackend
	awsBackend    toolBackend

	alerts     alertsState
	menuItems  []menuItem
	menuCursor int
}

type tickMsg time.Time

type detailLoadedMsg struct {
	title string
	body  string
	err   error
}

type actionDoneMsg struct {
	view   viewKind
	err    error
	output string
}

type kubeNamespaceSelectedMsg struct {
	namespace string
}

func newModel(cfg config.Config, run runner.Runner, opts Options) model {
	backend := toolBackend{cfg: cfg, run: run}
	return model{
		cfg:           cfg,
		run:           run,
		opts:          opts,
		activeView:    dockerView,
		mode:          "list",
		loading:       true,
		dockerBackend: backend,
		kubeBackend:   backend,
		awsBackend:    backend,
		kubeResource:  kubeResPods,
	}
}

func (m model) Init() tea.Cmd {
	return tea.Batch(m.loadActive(), loadAlerts(&m.dockerBackend, &m.kubeBackend, &m.awsBackend), m.tickCmd())
}

func (m model) tickCmd() tea.Cmd {
	return tea.Tick(m.opts.Refresh, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tickMsg:
		if m.mode == "list" && !m.loading {
			return m, tea.Batch(
				m.loadActive(),
				loadAlerts(&m.dockerBackend, &m.kubeBackend, &m.awsBackend),
			)
		}
		return m, m.tickCmd()

	case alertsLoadedMsg:
		m.alerts = msg.state
		return m, nil

	case dockerLoadedMsg:
		if m.activeView == dockerView {
			m.loading = false
			m.dockerRows = msg.rows
			m.dockerVersion = msg.version
			m.dockerErr = msg.err
			m.dockerErrText = msg.errText
			m.clampCursor()
		}
		return m, nil

	case kubeLoadedMsg:
		if m.activeView == kubeView {
			m.loading = false
			m.kubeRows = msg.rows
			m.kubeContext = msg.context
			m.kubeClusterInfo = msg.clusterInfo
			m.kubeErr = msg.err
			m.kubeErrText = msg.errText
			m.clampCursor()
		}
		return m, nil

	case awsLoadedMsg:
		if m.activeView == awsView {
			m.loading = false
			m.awsVersion = msg.version
			m.awsIdentity = msg.identity
			m.awsErr = msg.cliErr
			m.awsErrText = msg.errText
			m.awsIdentityErr = msg.identityErr
			m.awsIdentityErrText = ""
			if msg.identityErr != nil {
				m.awsIdentityErrText = msg.errText
			}
		}
		return m, nil

	case detailLoadedMsg:
		m.loading = false
		if msg.err != nil {
			m.message = msg.err.Error()
			m.mode = "list"
			return m, nil
		}
		m.detailTitle = msg.title
		m.detailBody = msg.body
		m.mode = "detail"
		return m, nil

	case actionDoneMsg:
		m.loading = false
		m.mode = "list"
		if msg.err != nil {
			m.message = strings.TrimSpace(msg.output)
			if m.message == "" {
				m.message = msg.err.Error()
			}
		} else if strings.TrimSpace(msg.output) != "" {
			m.message = strings.TrimSpace(msg.output)
		} else {
			m.message = "Acao concluida."
		}
		return m, m.loadForView(msg.view)

	case quitWithActionMsg:
		m.pending = &msg.pending
		m.quitting = true
		return m, tea.Quit

	case kubeNamespaceSelectedMsg:
		m.kubeNamespace = msg.namespace
		m.kubeAllNamespaces = false
		m.kubeResource = kubeResPods
		m.kubeCursor = 0
		m.mode = "list"
		m.message = "Namespace ativo: " + msg.namespace
		m.loading = true
		return m, loadKube(&m.kubeBackend, m.kubeLoadState())

	case tea.KeyMsg:
		return m.handleKey(msg)
	}

	return m, nil
}

func (m model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()

	if m.mode == "help" {
		if key == "esc" || key == "?" || key == "q" {
			m.mode = "list"
		}
		return m, nil
	}

	if m.mode == "favorites" || m.mode == "tasks" {
		switch key {
		case "esc", "q":
			m.mode = "list"
			m.menuCursor = 0
		case "up", "k":
			if m.menuCursor > 0 {
				m.menuCursor--
			}
		case "down", "j":
			if m.menuCursor < len(m.menuItems)-1 {
				m.menuCursor++
			}
		case "enter":
			if m.mode == "favorites" {
				if m.menuCursor >= 0 && m.menuCursor < len(m.menuItems) {
					item := favoriteItem{
						Label: m.menuItems[m.menuCursor].Name,
						Value: m.menuItems[m.menuCursor].Command,
					}
					m.jumpToFavorite(item)
					m.mode = "list"
					m.message = "Favorito selecionado."
					if m.activeView == kubeView {
						m.loading = true
						return m, loadKube(&m.kubeBackend, m.kubeLoadState())
					}
				}
				return m, nil
			}
			if duckArgs := m.selectedTaskDuckArgs(); duckArgs != nil {
				return m, m.runDuckCommand(duckArgs)
			}
		}
		return m, nil
	}

	if m.mode == "confirm" {
		switch key {
		case "y", "Y":
			m.mode = "list"
			m.loading = true
			switch m.activeView {
			case dockerView:
				return m, m.runDockerAction(m.confirmAction)
			case kubeView:
				return m, m.runKubeAction(m.confirmAction)
			}
		case "n", "N", "esc", "q":
			m.mode = "list"
			m.confirmAction = ""
			m.confirmTarget = ""
		}
		return m, nil
	}

	if m.mode == "detail" {
		switch key {
		case "e":
			if isYAMLDetail(m.detailTitle) {
				if path, err := exportYAMLContent(m.detailTitle, m.detailBody); err != nil {
					m.message = err.Error()
				} else {
					m.message = "YAML exportado: " + path
				}
			}
		case "esc", "q", "enter":
			m.mode = "list"
			m.detailTitle = ""
			m.detailBody = ""
		}
		return m, nil
	}

	if m.mode == "kube-input" {
		switch key {
		case "esc":
			m.mode = "list"
			m.kubeEditInput = ""
			m.kubeEditTitle = ""
		case "enter":
			m.mode = "list"
			switch m.kubeEditTitle {
			case "Imagem (container=image:tag)":
				return m, m.runKubeAction("set-image")
			case "Port-forward (local:remoto ou porta)":
				return m, m.runKubeAction("port-forward")
			}
		case "backspace":
			if len(m.kubeEditInput) > 0 {
				m.kubeEditInput = m.kubeEditInput[:len(m.kubeEditInput)-1]
			}
		default:
			if len(key) == 1 || strings.Contains(key, "=") || strings.Contains(key, ":") || strings.Contains(key, "/") {
				if key == "space" {
					m.kubeEditInput += " "
				} else if len(key) == 1 {
					m.kubeEditInput += key
				}
			}
		}
		return m, nil
	}

	if m.mode == "filter" {
		switch key {
		case "esc":
			m.mode = "list"
			m.filterInput = ""
		case "enter":
			m.mode = "list"
			m.filter = m.filterInput
			m.filterInput = ""
			m.clampCursor()
		case "backspace":
			if len(m.filterInput) > 0 {
				m.filterInput = m.filterInput[:len(m.filterInput)-1]
			}
		default:
			if len(key) == 1 {
				m.filterInput += key
			}
		}
		return m, nil
	}

	switch key {
	case "ctrl+c", "q":
		m.quitting = true
		return m, tea.Quit
	case "tab", "right":
		m.activeView = (m.activeView + 1) % 3
		return m.switchView()
	case "shift+tab", "left":
		m.activeView--
		if m.activeView < 0 {
			m.activeView = 2
		}
		return m.switchView()
	case "1":
		m.activeView = dockerView
		return m.switchView()
	case "2":
		m.activeView = kubeView
		return m.switchView()
	case "3":
		m.activeView = awsView
		return m.switchView()
	case "/":
		m.mode = "filter"
		m.filterInput = m.filter
		return m, nil
	case "r":
		m.loading = true
		m.message = ""
		return m, m.loadActive()
	case "up", "k":
		m.moveCursor(-1)
	case "down", "j":
		m.moveCursor(1)
	case "g":
		m.setCursor(0)
	case "G":
		m.setCursor(m.visibleCount() - 1)
	case "?":
		m.mode = "help"
		return m, nil
	case "e":
		if path, err := m.exportCurrent("json"); err != nil {
			m.message = err.Error()
		} else {
			m.message = m.renderExportMessage(path)
		}
		return m, nil
	case "E":
		if path, err := m.exportCurrent("csv"); err != nil {
			m.message = err.Error()
		} else {
			m.message = m.renderExportMessage(path)
		}
		return m, nil
	case "F":
		items, err := m.loadFavoriteItems()
		if err != nil {
			m.message = err.Error()
			return m, nil
		}
		m.menuItems = nil
		for _, item := range items {
			m.menuItems = append(m.menuItems, menuItem{Kind: "favorite", Name: item.Label, Command: item.Value})
		}
		m.menuCursor = 0
		m.mode = "favorites"
		return m, nil
	case "T":
		items, err := listTasksAndAliases()
		if err != nil {
			m.message = err.Error()
			return m, nil
		}
		m.menuItems = items
		m.menuCursor = 0
		m.mode = "tasks"
		return m, nil
	case "ctrl+s":
		kind, key, value, ok := m.currentFavoriteValue()
		if !ok {
			m.message = "Nenhum item selecionado para favoritar."
			return m, nil
		}
		if err := saveFavorite(kind, key, value); err != nil {
			m.message = err.Error()
		} else {
			m.message = "Favorito salvo: " + key
		}
		return m, nil
	}

	switch m.activeView {
	case dockerView:
		return m.handleDockerKeys(key)
	case kubeView:
		return m.handleKubeKeys(key)
	}

	return m, nil
}

func (m model) handleDockerKeys(key string) (tea.Model, tea.Cmd) {
	if m.opts.Readonly && isMutatingKey(key) {
		m.message = "Modo somente leitura ativo."
		return m, nil
	}
	switch key {
	case "a":
		m.dockerShowAll = !m.dockerShowAll
		m.loading = true
		return m, loadDocker(&m.dockerBackend, m.dockerShowAll)
	case "l":
		m.message = ""
		return m, m.runDockerAction("logs")
	case "s":
		m.message = ""
		return m, m.runDockerAction("shell")
	case "i", "d":
		m.loading = true
		m.message = ""
		return m, m.runDockerAction("inspect")
	case "S":
		return m.requestDockerAction("start")
	case "x":
		return m.requestDockerAction("stop")
	case "R":
		return m.requestDockerAction("restart")
	case "ctrl+d":
		return m.requestDockerAction("delete")
	}
	return m, nil
}

func (m model) requestDockerAction(action string) (tea.Model, tea.Cmd) {
	row, ok := m.selectedDockerRow()
	if !ok {
		return m, nil
	}
	if m.opts.needsConfirm(action) {
		m.mode = "confirm"
		m.confirmAction = action
		m.confirmTarget = row.Name
		return m, nil
	}
	m.message = ""
	return m, m.runDockerAction(action)
}

func (m model) handleKubeKeys(key string) (tea.Model, tea.Cmd) {
	if m.opts.Readonly && isKubeMutatingKey(key) {
		m.message = "Modo somente leitura ativo."
		return m, nil
	}
	switch key {
	case "[", "p":
		m.kubeResource = m.kubeResource.prev()
		m.kubeCursor = 0
		m.loading = true
		return m, loadKube(&m.kubeBackend, m.kubeLoadState())
	case "]", "P":
		m.kubeResource = m.kubeResource.next()
		m.kubeCursor = 0
		m.loading = true
		return m, loadKube(&m.kubeBackend, m.kubeLoadState())
	case "n":
		m.kubeResource = kubeResNamespaces
		m.kubeCursor = 0
		m.loading = true
		return m, loadKube(&m.kubeBackend, m.kubeLoadState())
	case "c":
		m.kubeResource = kubeResContexts
		m.kubeCursor = 0
		m.loading = true
		return m, loadKube(&m.kubeBackend, m.kubeLoadState())
	case "a":
		m.kubeAllNamespaces = !m.kubeAllNamespaces
		if m.kubeAllNamespaces {
			m.kubeNamespace = ""
		}
		m.loading = true
		return m, loadKube(&m.kubeBackend, m.kubeLoadState())
	case "l":
		m.message = ""
		return m, m.runKubeAction("logs")
	case "s":
		m.message = ""
		return m, m.runKubeAction("shell")
	case "d":
		m.loading = true
		m.message = ""
		return m, m.runKubeAction("describe")
	case "y":
		m.loading = true
		m.message = ""
		return m, m.runKubeAction("yaml")
	case "Y":
		m.loading = true
		m.message = ""
		return m, m.runKubeAction("export-yaml")
	case "U":
		return m.requestKubeAction("redeploy")
	case "shift+U":
		return m.requestKubeAction("force-redeploy")
	case "E":
		m.message = ""
		return m, m.runKubeAction("edit")
	case "I":
		if m.kubeResource != kubeResDeployments {
			m.message = "Imagem disponivel apenas em Deployments."
			return m, nil
		}
		m = m.beginKubeInput("Imagem (container=image:tag)", "app=nginx:latest")
		return m, nil
	case "f":
		if m.kubeResource != kubeResPods && m.kubeResource != kubeResServices {
			m.message = "Port-forward disponivel em Pods e Services."
			return m, nil
		}
		m = m.beginKubeInput("Port-forward (local:remoto ou porta)", "8080")
		return m, nil
	case "R":
		return m.requestKubeAction("restart")
	case "+", "=":
		return m.requestKubeAction("scale-up")
	case "-", "_":
		return m.requestKubeAction("scale-down")
	case "ctrl+d":
		return m.requestKubeAction("delete")
	case "enter":
		switch m.kubeResource {
		case kubeResNamespaces:
			return m, m.runKubeAction("use-namespace")
		case kubeResContexts:
			return m.requestKubeAction("use-context")
		default:
			break
		}
	}
	return m, nil
}

func isKubeMutatingKey(key string) bool {
	switch key {
	case "S", "x", "R", "U", "shift+U", "ctrl+d", "+", "=", "-", "_", "I", "E", "Y", "enter":
		return true
	default:
		return isMutatingKey(key)
	}
}

func (m model) requestKubeAction(action string) (tea.Model, tea.Cmd) {
	row, ok := m.selectedKubeRow()
	if !ok {
		return m, nil
	}
	if m.opts.needsConfirm(action) {
		m.mode = "confirm"
		m.confirmAction = action
		m.confirmTarget = row.Namespace + "/" + row.Name
		return m, nil
	}
	m.message = ""
	return m, m.runKubeAction(action)
}

func isMutatingKey(key string) bool {
	switch key {
	case "S", "x", "R", "ctrl+d":
		return true
	default:
		return false
	}
}

func (m model) runDuckCommand(args []string) tea.Cmd {
	return func() tea.Msg {
		return quitWithActionMsg{pending: pendingAction{duckArgs: args, interactive: true}}
	}
}

func (m model) switchView() (tea.Model, tea.Cmd) {
	m.mode = "list"
	m.filter = ""
	m.filterInput = ""
	m.message = ""
	m.loading = true
	m.dockerCursor = 0
	m.kubeCursor = 0
	return m, m.loadActive()
}

func (m *model) moveCursor(delta int) {
	switch m.activeView {
	case dockerView:
		m.dockerCursor += delta
	case kubeView:
		m.kubeCursor += delta
	}
	m.clampCursor()
}

func (m *model) setCursor(index int) {
	switch m.activeView {
	case dockerView:
		m.dockerCursor = index
	case kubeView:
		m.kubeCursor = index
	}
	m.clampCursor()
}

func (m *model) clampCursor() {
	count := m.visibleCount()
	if count == 0 {
		switch m.activeView {
		case dockerView:
			m.dockerCursor = 0
		case kubeView:
			m.kubeCursor = 0
		}
		return
	}
	switch m.activeView {
	case dockerView:
		if m.dockerCursor < 0 {
			m.dockerCursor = 0
		}
		if m.dockerCursor >= count {
			m.dockerCursor = count - 1
		}
	case kubeView:
		if m.kubeCursor < 0 {
			m.kubeCursor = 0
		}
		if m.kubeCursor >= count {
			m.kubeCursor = count - 1
		}
	}
}

func (m model) visibleCount() int {
	switch m.activeView {
	case dockerView:
		return len(m.dockerVisibleRows())
	case kubeView:
		return len(m.kubeVisibleRows())
	default:
		return 0
	}
}

func (m model) loadActive() tea.Cmd {
	return m.loadForView(m.activeView)
}

func (m model) loadForView(view viewKind) tea.Cmd {
	switch view {
	case dockerView:
		return loadDocker(&m.dockerBackend, m.dockerShowAll)
	case kubeView:
		return loadKube(&m.kubeBackend, m.kubeLoadState())
	case awsView:
		return loadAWS(&m.awsBackend)
	default:
		return nil
	}
}

func (m model) showDetail(load func() (string, error), title string) tea.Cmd {
	return func() tea.Msg {
		body, err := load()
		return detailLoadedMsg{title: title, body: strings.TrimSpace(body), err: err}
	}
}

func (m model) runExternal(backend *toolBackend, binary string, args []string, interactive bool) tea.Cmd {
	resolvedBin, resolvedArgs := backend.resolvedCommand(binary, args)
	return func() tea.Msg {
		return quitWithActionMsg{
			pending: pendingAction{
				binary:      resolvedBin,
				args:        resolvedArgs,
				interactive: interactive,
			},
		}
	}
}

func (m model) runActionThenRefresh(backend *toolBackend, binary string, args []string, view viewKind) tea.Cmd {
	return func() tea.Msg {
		err := backend.runCommand(binary, args, runner.DefaultOptions())
		return actionDoneMsg{view: view, err: err}
	}
}

func (m model) renderMainBody() []string {
	if m.opts.Compact {
		return []string{m.renderCompactSummary(), "", m.renderCompactBody()}
	}
	return []string{m.renderBody()}
}

func (m model) View() string {
	if m.quitting {
		return ""
	}

	var sections []string
	sections = append(sections, m.renderHeader())
	sections = append(sections, "")

	switch m.mode {
	case "confirm":
		sections = append(sections, m.renderConfirm())
	case "detail":
		sections = append(sections, m.renderDetail())
	case "help":
		sections = append(sections, m.renderHelpOverlay())
	case "favorites":
		sections = append(sections, m.renderFavorites())
	case "tasks":
		sections = append(sections, m.renderTasksMenu())
	case "kube-input":
		sections = append(sections, m.renderKubeInputBar())
		sections = append(sections, m.renderMainBody()...)
	case "filter":
		sections = append(sections, m.renderFilterBar())
		sections = append(sections, m.renderMainBody()...)
	case "list":
		sections = append(sections, m.renderMainBody()...)
	}

	if m.loading {
		sections = append(sections, "", msgStyle.Render("Carregando..."))
	}
	if m.message != "" {
		sections = append(sections, "", msgStyle.Render(m.message))
	}

	sections = append(sections, "", m.renderFooter())
	return strings.Join(sections, "\n")
}

func (m model) renderHeader() string {
	tabs := []string{
		m.renderTab("Docker", dockerView, m.dockerCountLabel(), m.alerts.dockerAlert()),
		m.renderTab("Kubernetes", kubeView, m.kubeCountLabel(), m.alerts.kubeAlert()),
		m.renderTab("AWS", awsView, m.awsCountLabel(), m.alerts.awsAlert()),
	}

	subtitle := m.subtitle()
	header := lipgloss.JoinHorizontal(lipgloss.Top, tabs...)
	if subtitle != "" {
		header += "\n" + helpStyle.Render(subtitle)
	}
	return headerStyle.Render(header)
}

func (m model) renderTab(title string, view viewKind, badge string, alert bool) string {
	label := title
	if badge != "" {
		label += " (" + badge + ")"
	}
	if alert {
		label = "● " + label
	}
	return alertTabStyle(alert, m.activeView == view, label)
}

func (m model) subtitle() string {
	switch m.activeView {
	case dockerView:
		if m.dockerErr != nil {
			return "Docker indisponivel"
		}
		parts := []string{platformLabel(), m.dockerVersion}
		if m.dockerBackend.viaWSL {
			parts = append(parts, "via WSL")
		}
		if m.dockerShowAll {
			parts = append(parts, "todos os containers")
		} else {
			parts = append(parts, "containers ativos")
		}
		if m.filter != "" {
			parts = append(parts, "filtro: "+m.filter)
		}
		return strings.Join(parts, " | ")
	case kubeView:
		if m.kubeErr != nil {
			if m.kubeContext != "" {
				return "contexto: " + m.kubeContext + " | cluster indisponivel"
			}
			return "Kubernetes indisponivel"
		}
		parts := []string{platformLabel(), "contexto: " + m.kubeContext, kubeResourceLabel(m.kubeResource)}
		if m.kubeBackend.viaWSL {
			parts = append(parts, "via WSL")
		}
		if m.kubeClusterInfo != "" {
			parts = append(parts, m.kubeClusterInfo)
		}
		if m.kubeAllNamespaces {
			parts = append(parts, "todos os namespaces")
		}
		if m.filter != "" {
			parts = append(parts, "filtro: "+m.filter)
		}
		return strings.Join(parts, " | ")
	default:
		return ""
	}
}

func (m model) renderKubeError() string {
	msg := friendlyKubeError(m.kubeErrText)
	if msg == "" {
		msg = m.kubeErrorText()
	}
	body := m.renderActionableError("kube", msg)
	if m.kubeContext != "" {
		return body + "\n\n" + helpStyle.Render("Contexto atual: "+m.kubeContext)
	}
	return body
}

func (m model) kubeErrorText() string {
	if m.kubeErrText != "" {
		return m.kubeErrText
	}
	if m.kubeErr != nil {
		return m.kubeErr.Error()
	}
	return ""
}

func (m model) awsErrorText() string {
	if m.awsErrText != "" {
		return friendlyAWSError(m.awsErrText)
	}
	if m.awsErr != nil {
		return m.awsErr.Error()
	}
	return ""
}

func (m model) awsIdentityErrorText() string {
	if m.awsIdentityErrText != "" {
		return friendlyAWSError(m.awsIdentityErrText)
	}
	if m.awsIdentityErr != nil {
		return m.awsIdentityErr.Error()
	}
	return "nao foi possivel obter a identidade AWS"
}

func (m model) dockerErrorText() string {
	if m.dockerErrText != "" {
		if msg := friendlyDockerError(m.dockerErrText); msg != "" {
			return msg
		}
		return m.dockerErrText
	}
	if m.dockerErr != nil {
		return m.dockerErr.Error()
	}
	return ""
}

func (m model) dockerCountLabel() string {
	if m.dockerErr != nil {
		return "!"
	}
	return fmt.Sprintf("%d", len(m.dockerVisibleRows()))
}

func (m model) kubeCountLabel() string {
	if m.kubeErr != nil {
		return "!"
	}
	return fmt.Sprintf("%d", len(m.kubeVisibleRows()))
}

func (m model) awsCountLabel() string {
	if m.awsErr != nil {
		return "!"
	}
	if m.awsIdentityErr != nil {
		return "?"
	}
	return "ok"
}

func (m model) renderBody() string {
	width := m.width
	if width <= 0 {
		width = 100
	}

	switch m.activeView {
	case dockerView:
		if m.dockerErr != nil {
			return m.renderActionableError("docker", m.dockerErrorText())
		}
		return m.renderDockerTable(width)
	case kubeView:
		if m.kubeErr != nil {
			return m.renderKubeError()
		}
		return m.renderKubeTable(width)
	case awsView:
		return m.renderAWS()
	default:
		return ""
	}
}

func (m model) renderFilterBar() string {
	return filterStyle.Render("/ " + m.filterInput + "█")
}

func (m model) renderDetail() string {
	var builder strings.Builder
	builder.WriteString(detailTitleStyle.Render(m.detailTitle))
	builder.WriteString("\n\n")
	body := m.detailBody
	maxLines := m.height - 8
	if isYAMLDetail(m.detailTitle) && maxLines < 30 {
		maxLines = 30
	}
	if m.height > 0 && maxLines > 0 {
		lines := strings.Split(body, "\n")
		if len(lines) > maxLines {
			lines = lines[:maxLines]
			body = strings.Join(lines, "\n") + "\n... (use Y ou e para exportar o arquivo completo)"
		}
	}
	builder.WriteString(detailBodyStyle.Render(body))
	builder.WriteString("\n\n")
	if isYAMLDetail(m.detailTitle) {
		builder.WriteString(helpStyle.Render("e exportar yaml | esc volta"))
	} else {
		builder.WriteString(helpStyle.Render("esc volta"))
	}
	return builder.String()
}

func (m model) renderConfirm() string {
	action := m.confirmAction
	switch action {
	case "redeploy":
		action = "redeploy (recriar pod / rollout)"
	case "force-redeploy":
		action = "redeploy forçado (apaga pods antigos)"
	}
	return confirmStyle.Render(fmt.Sprintf(
		"Confirmar %s em %s?\n\n[y] sim  [n] nao",
		action,
		m.confirmTarget,
	))
}

func (m model) renderFooter() string {
	var help string
	switch m.activeView {
	case dockerView:
		help = m.dockerHelpLine()
	case kubeView:
		help = m.kubeHelpLine()
	case awsView:
		help = m.awsHelpLine()
	}
	extra := "? ajuda | e/E export | F favoritos | T tasks"
	if m.opts.Readonly {
		extra += " | readonly"
	}
	common := platformLabel() + " | tab/1-3 | j/k | r refresh " + fmt.Sprintf("(%s)", m.opts.Refresh) + " | q sai"
	if m.dockerBackend.viaWSL || m.kubeBackend.viaWSL || m.awsBackend.viaWSL {
		common = platformLabel() + " via WSL | tab/1-3 | j/k | q sai"
	}
	return helpStyle.Render(help + " | " + extra + " | " + common)
}
