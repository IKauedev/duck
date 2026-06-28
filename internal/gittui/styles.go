package gittui

import (
	"github.com/charmbracelet/lipgloss"
)

var (
	activeTabSt = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("229")).
			Background(lipgloss.Color("62")).
			Padding(0, 1)

	tabSt = lipgloss.NewStyle().
		Foreground(lipgloss.Color("245")).
		Padding(0, 1)

	selectedSt = lipgloss.NewStyle().
			Foreground(lipgloss.Color("229")).
			Background(lipgloss.Color("62"))

	rowSt = lipgloss.NewStyle().Foreground(lipgloss.Color("252"))

	mutedSt = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))

	errSt = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("196")).
		Background(lipgloss.Color("236")).
		Padding(0, 1)

	successSt = lipgloss.NewStyle().
			Foreground(lipgloss.Color("82")).
			Background(lipgloss.Color("236")).
			Padding(0, 1)

	hashSt = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("214"))

	refSt = lipgloss.NewStyle().
		Foreground(lipgloss.Color("81")).
		Background(lipgloss.Color("236")).
		Padding(0, 1)

	currentBranchSt = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("82"))

	detailTitleSt = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("117")).
			Background(lipgloss.Color("236")).
			Padding(0, 1)

	inputPromptSt = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("229")).
			Padding(0, 1)

	confirmSt = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("196")).
			Background(lipgloss.Color("236")).
			Padding(1, 2)

	modifiedSt  = lipgloss.NewStyle().Foreground(lipgloss.Color("214"))
	addedSt     = lipgloss.NewStyle().Foreground(lipgloss.Color("82"))
	deletedSt   = lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
	renamedSt   = lipgloss.NewStyle().Foreground(lipgloss.Color("81"))
	untrackedSt = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
)
