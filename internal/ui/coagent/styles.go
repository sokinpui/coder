package coagentui

import "github.com/charmbracelet/lipgloss"

var (
	promptPrefixStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("51")).
				Bold(true)

	userHeaderStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("42")).
			Bold(true)

	assistantHeaderStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("213")).
				Bold(true)

	toolCallStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("221")).
			Bold(true)

	toolResultStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("244"))

	toolErrorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("196")).
			Bold(true)

	spinnerStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("244")).
			Italic(true)

	systemNoteStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("240")).
			Italic(true)
)
