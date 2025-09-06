package ui

import "github.com/charmbracelet/lipgloss"

var (
	helpStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("208")) // Orange
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
)
