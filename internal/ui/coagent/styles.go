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

	toolSuccessStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("78"))

	toolPathStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("255"))

	toolCommandStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("255")).
			Bold(true)

	toolMutedStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("244"))

	toolBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("240")).
			Padding(0, 1)

	diffRemovedStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("203"))

	diffAddedStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("78"))

	diffContextStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("244"))

	thinkingHeaderStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("213")).
				Bold(true)

	reasoningStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("244")).
			Italic(true)

	inputContainerStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("240")).
				Padding(0, 1)

	statusBarStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("244"))
)
