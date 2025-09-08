package ui

import "github.com/charmbracelet/lipgloss"

var (
	helpStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("208")) // Orange
	textAreaStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("240"))
	userInputStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("240")).
			Padding(0, 1)
	generatingHelpStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("51")). // Cyan
				Italic(true)
	commandInputStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("240")).
				Padding(0, 1).
				BorderTop(false).
				BorderBottom(false).
				BorderRight(false)
	cmdResultStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("99")).
			Padding(0, 1).
			BorderTop(false).
			BorderBottom(false).
			BorderRight(false)
	cmdErrorStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("9")). // Red
			Foreground(lipgloss.Color("9")).       // Red
			Padding(0, 1).
			BorderTop(false).
			BorderBottom(false).
			BorderRight(false)
	appMessageStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("244")). // A muted gray
			Padding(0, 1)
)
