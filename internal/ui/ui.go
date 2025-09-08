package ui

import (
	"log"
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

	finalModel, err := p.Run()
	if err != nil {
		fmt.Printf("Error starting program: %v\n", err)
		os.Exit(1)
	}

	// Save conversation on exit
	if m, ok := finalModel.(Model); ok {
		if m.historyManager != nil {
			if err := m.historyManager.SaveConversation(m.messages, m.systemInstructions, m.providedDocuments); err != nil {
				log.Printf("Error saving conversation history: %v", err)
			}
		}
	}
}
