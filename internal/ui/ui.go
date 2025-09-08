package ui

import (
	"coder/internal/config"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
)

func Start() {
	cfg := config.Default()
	model, err := NewModel(cfg)
	if err != nil {
		fmt.Printf("Error creating model: %v\n", err)
		os.Exit(1)
	}

	p := tea.NewProgram(
		model,
		tea.WithAltScreen(),
	)

	if _, err := p.Run(); err != nil {
		fmt.Printf("Error starting program: %v\n", err)
		os.Exit(1)
	}
}
