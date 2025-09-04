package ui

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
)

// Start initializes and runs the Bubble Tea program for the coder application.
func Start() {
	p := tea.NewProgram(NewModel())

	if _, err := p.Run(); err != nil {
		fmt.Printf("Error starting program: %v\n", err)
		os.Exit(1)
	}
}
