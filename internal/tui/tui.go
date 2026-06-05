package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/IKauedev/duck/internal/cli"
	"github.com/IKauedev/duck/internal/config"
	"github.com/IKauedev/duck/internal/runner"
)

type tab struct {
	title string
	load  func() string
}

type model struct {
	tabs    []tab
	active  int
	content string
	err     error
}

type loadedMsg string

var (
	activeTabStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("229")).Background(lipgloss.Color("62")).Padding(0, 1)
	tabStyle       = lipgloss.NewStyle().Foreground(lipgloss.Color("245")).Padding(0, 1)
	helpStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	errorStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
)

func Command(cfg config.Config, run runner.Runner) cli.Command {
	return cli.Command{
		Name:        "tui",
		Description: "Abre interface TUI com abas Docker, K8s e AWS",
		Usage:       "tui",
		Run: func(_ cli.Context, args []string) error {
			if len(args) > 0 {
				return cli.UsageError("tui nao recebe argumentos")
			}
			_, err := tea.NewProgram(newModel(cfg, run)).Run()
			return err
		},
		Examples: []string{
			"tui",
		},
	}
}

func newModel(cfg config.Config, run runner.Runner) model {
	return model{
		tabs: []tab{
			{title: "Docker", load: dockerStatus(cfg, run)},
			{title: "K8s", load: kubeStatus(cfg, run)},
			{title: "AWS", load: awsStatus(cfg, run)},
		},
	}
}

func (m model) Init() tea.Cmd {
	return m.loadActive()
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "esc", "ctrl+c":
			return m, tea.Quit
		case "tab", "right", "l":
			m.active = (m.active + 1) % len(m.tabs)
			m.content = "Carregando..."
			return m, m.loadActive()
		case "shift+tab", "left", "h":
			m.active--
			if m.active < 0 {
				m.active = len(m.tabs) - 1
			}
			m.content = "Carregando..."
			return m, m.loadActive()
		case "r":
			m.content = "Atualizando..."
			return m, m.loadActive()
		}
	case loadedMsg:
		m.content = string(msg)
		m.err = nil
	}
	return m, nil
}

func (m model) View() string {
	var tabs []string
	for index, tab := range m.tabs {
		if index == m.active {
			tabs = append(tabs, activeTabStyle.Render(tab.title))
			continue
		}
		tabs = append(tabs, tabStyle.Render(tab.title))
	}

	body := strings.TrimSpace(m.content)
	if body == "" {
		body = "Carregando..."
	}
	if m.err != nil {
		body = errorStyle.Render(m.err.Error())
	}

	return fmt.Sprintf(
		"%s\n\n%s\n\n%s\n",
		lipgloss.JoinHorizontal(lipgloss.Top, tabs...),
		body,
		helpStyle.Render("tab/shift+tab alterna abas | r atualiza | q sai"),
	)
}

func (m model) loadActive() tea.Cmd {
	active := m.tabs[m.active]
	return func() tea.Msg {
		return loadedMsg(active.load())
	}
}

func dockerStatus(cfg config.Config, run runner.Runner) func() string {
	return func() string {
		version, err := run.Output(cfg.DockerBin, []string{"version", "--format", "Cliente: {{.Client.Version}} | Servidor: {{.Server.Version}}"})
		if err != nil {
			return "Docker indisponivel: " + strings.TrimSpace(versionOrError(version, err))
		}
		containers, err := run.Output(cfg.DockerBin, []string{"ps", "-a", "--format", "table {{.Names}}\t{{.Status}}\t{{.Image}}\t{{.Ports}}"})
		if err != nil {
			containers = versionOrError(containers, err)
		}
		return "Docker\n" + strings.TrimSpace(version) + "\n\nContainers\n" + strings.TrimSpace(containers)
	}
}

func kubeStatus(cfg config.Config, run runner.Runner) func() string {
	return func() string {
		context, err := run.Output(cfg.KubectlBin, []string{"config", "current-context"})
		if err != nil {
			return "Kubernetes indisponivel: " + strings.TrimSpace(versionOrError(context, err))
		}
		pods, err := run.Output(cfg.KubectlBin, []string{"get", "pods", "-A"})
		if err != nil {
			pods = versionOrError(pods, err)
		}
		return "Contexto\n" + strings.TrimSpace(context) + "\n\nPods\n" + strings.TrimSpace(pods)
	}
}

func awsStatus(cfg config.Config, run runner.Runner) func() string {
	return func() string {
		version, err := run.Output(cfg.AWSBin, []string{"--version"})
		if err != nil {
			return "AWS indisponivel: " + strings.TrimSpace(versionOrError(version, err))
		}
		identity, err := run.Output(cfg.AWSBin, []string{"sts", "get-caller-identity", "--output", "table"})
		if err != nil {
			identity = versionOrError(identity, err)
		}
		return "AWS CLI\n" + strings.TrimSpace(version) + "\n\nIdentidade\n" + strings.TrimSpace(identity)
	}
}

func versionOrError(output string, err error) string {
	output = strings.TrimSpace(output)
	if output != "" {
		return output
	}
	return err.Error()
}
