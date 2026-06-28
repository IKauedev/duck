package shelltui

import "github.com/charmbracelet/lipgloss"

const (
	colorBg      = "232" // preto terminal
	colorBgBar   = "234" // barra levemente mais clara
	colorBgInput = "233"
	colorText    = "252" // texto principal branco-acinzentado
	colorMuted   = "240" // texto esmaecido
	colorBorder  = "238" // bordas escuras
	colorActive  = "39"  // azul para elemento ativo
	colorSuccess = "83"
	colorError   = "203"
	colorWarn    = "221"
	colorPrompt  = "39"  // azul prompt
	colorBranch  = "240" // branch git esmaecido
)

var (
	styleHeaderBar = lipgloss.NewStyle().
		Background(lipgloss.Color(colorBgBar))

	styleHeader = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color(colorText)).
		Background(lipgloss.Color(colorBgBar))

	styleHeaderDim = lipgloss.NewStyle().
		Foreground(lipgloss.Color(colorMuted)).
		Background(lipgloss.Color(colorBgBar))

	styleTabActive = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color(colorText)).
		Background(lipgloss.Color(colorBg)).
		BorderBottom(true).
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color(colorActive)).
		Padding(0, 2)

	styleTabInactive = lipgloss.NewStyle().
		Foreground(lipgloss.Color(colorMuted)).
		Background(lipgloss.Color(colorBgBar)).
		Padding(0, 2)

	styleTabSep = lipgloss.NewStyle().
		Foreground(lipgloss.Color(colorBorder)).
		Background(lipgloss.Color(colorBgBar))

	styleTabBar = lipgloss.NewStyle().
		Background(lipgloss.Color(colorBgBar))

	styleViewport = lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color(colorBorder))

	styleViewportFocused = lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color(colorActive))

	styleSidebar = lipgloss.NewStyle().
		BorderLeft(true).
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color(colorBorder)).
		Padding(0, 1)

	styleSidebarTitle = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color(colorText))

	styleSidebarCategory = lipgloss.NewStyle().
		Foreground(lipgloss.Color(colorMuted))

	styleSidebarKey = lipgloss.NewStyle().
		Foreground(lipgloss.Color(colorActive)).
		Width(10)

	styleSidebarDesc = lipgloss.NewStyle().
		Foreground(lipgloss.Color(colorMuted))

	styleInputBox = lipgloss.NewStyle().
		BorderTop(true).
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color(colorBorder)).
		Background(lipgloss.Color(colorBgInput)).
		Padding(0, 1)

	stylePrompt      = lipgloss.NewStyle().Foreground(lipgloss.Color(colorPrompt))
	stylePromptDir   = lipgloss.NewStyle().Foreground(lipgloss.Color(colorMuted))

	styleStatusBar = lipgloss.NewStyle().
		Background(lipgloss.Color(colorBgBar)).
		Foreground(lipgloss.Color(colorMuted))

	styleStatusItem = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color(colorActive)).
		Background(lipgloss.Color(colorBgBar))

	styleStatusMuted = lipgloss.NewStyle().
		Foreground(lipgloss.Color(colorMuted)).
		Background(lipgloss.Color(colorBgBar))

	styleSuccess = lipgloss.NewStyle().Foreground(lipgloss.Color(colorSuccess))
	styleError   = lipgloss.NewStyle().Foreground(lipgloss.Color(colorError))
	styleWarn    = lipgloss.NewStyle().Foreground(lipgloss.Color(colorWarn))
	styleMuted   = lipgloss.NewStyle().Foreground(lipgloss.Color(colorMuted))
	styleBold    = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(colorText))

	styleNotification = lipgloss.NewStyle().
		Foreground(lipgloss.Color(colorWarn)).
		Background(lipgloss.Color(colorBgBar)).
		Padding(0, 1)

	styleHelpOverlay = lipgloss.NewStyle().
		Background(lipgloss.Color(colorBgBar)).
		Border(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color(colorBorder)).
		Padding(1, 3)

	styleHelpTitle = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color(colorText))

	styleHelpKey = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color(colorActive)).
		Width(18)

	styleHelpDesc = lipgloss.NewStyle().
		Foreground(lipgloss.Color(colorMuted))

	styleHelpSection = lipgloss.NewStyle().
		Foreground(lipgloss.Color(colorText)).
		MarginTop(1)

	// Prompt minimalista: user@host:path$
	styleUbuntuUserHost = lipgloss.NewStyle().
		Foreground(lipgloss.Color(colorActive))

	styleUbuntuColon = lipgloss.NewStyle().
		Foreground(lipgloss.Color(colorMuted))

	styleUbuntuPath = lipgloss.NewStyle().
		Foreground(lipgloss.Color(colorText))

	styleUbuntuBranch = lipgloss.NewStyle().
		Foreground(lipgloss.Color(colorBranch))

	styleUbuntoDollar = lipgloss.NewStyle().
		Foreground(lipgloss.Color(colorMuted))
)
