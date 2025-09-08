package ui

import "github.com/charmbracelet/lipgloss"

var (
	initMessageStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("244")).
				Italic(true).
				Bold(true).
				Padding(0, 1)
	helpStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("208")) // Orange
	textAreaStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("240"))
	userInputStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("244")).
			Padding(0, 1)
	generatingHelpStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("51")). // Cyan
				Italic(true)
	actionInputStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("99")).
				Padding(0, 1)
	actionResultStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("99")).
				Padding(0, 1).
				BorderTop(false).
				BorderBottom(false).
				BorderRight(false)
	actionErrorStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("9")). // Red
				Foreground(lipgloss.Color("9")).       // Red
				Padding(0, 1).
				BorderTop(false).
				BorderBottom(false).
				BorderRight(false)
	commandInputStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("99")).
				Padding(0, 1)
	commandResultStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("99")).
				Padding(0, 1).
				BorderTop(false).
				BorderBottom(false).
				BorderRight(false)
	commandErrorStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("9")). // Red
				Foreground(lipgloss.Color("9")).       // Red
				Padding(0, 1).
				BorderTop(false).
				BorderBottom(false).
				BorderRight(false)
)
