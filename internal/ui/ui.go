package ui

import (
	"coder/internal/config"
	"coder/internal/ui/update"
	"log"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
)

func Start() {
	cfg := config.Default()
	model, err := update.NewModel(cfg)
	if err != nil {
		fmt.Printf("Error creating model: %v\n", err)
		os.Exit(1)
	}

	p := tea.NewProgram(
		model,
		tea.WithAltScreen(),
	)

	finalModel, err := p.Run()
	if err != nil {
		fmt.Printf("Error starting program: %v\n", err)
		os.Exit(1)
	}

	// Save conversation on exit
	if m, ok := finalModel.(update.Model); ok {
		if m.Session != nil {
			if err := m.Session.SaveConversation(); err != nil {
				log.Printf("Error saving conversation history: %v", err)
			}
		}
	}
}
