package tui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

type awsLoadedMsg struct {
	version     string
	identity    string
	cliErr      error
	identityErr error
	errText     string
}

func loadAWS(backend *toolBackend) tea.Cmd {
	return func() tea.Msg {
		cfg := backend.cfg
		version, err := backend.output(cfg.AWSBin, []string{"--version"})
		if err != nil {
			return awsLoadedMsg{
				cliErr:  err,
				version: strings.TrimSpace(version),
				errText: formatCommandError(version, err),
			}
		}
		identity, err := backend.output(cfg.AWSBin, []string{"sts", "get-caller-identity", "--output", "table"})
		if err != nil {
			return awsLoadedMsg{
				version:     strings.TrimSpace(version),
				identity:    strings.TrimSpace(identity),
				identityErr: err,
				errText:     formatCommandError(identity, err),
			}
		}
		return awsLoadedMsg{
			version:  strings.TrimSpace(version),
			identity: strings.TrimSpace(identity),
		}
	}
}

func (m *model) renderAWS() string {
	if m.awsErr != nil {
		return errorStyle.Render("AWS CLI indisponivel: " + m.awsErrorText())
	}

	var builder strings.Builder
	builder.WriteString(detailTitleStyle.Render("AWS CLI"))
	builder.WriteString("\n")
	builder.WriteString(detailBodyStyle.Render(m.awsVersion))
	builder.WriteString("\n\n")
	builder.WriteString(detailTitleStyle.Render("Identidade"))
	builder.WriteString("\n")

	if m.awsIdentityErr != nil {
		builder.WriteString(statusBad.Render(actionableError("aws", m.awsIdentityErrorText())))
	} else {
		builder.WriteString(detailBodyStyle.Render(m.awsIdentity))
	}
	return builder.String()
}

func (m *model) awsHelpLine() string {
	return "r atualizar | tab troca aba | q sai"
}
