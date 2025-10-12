package ui

import (
	"coder/internal/config"
	"coder/internal/ui/update"
	"fmt"
	"log"
	"os"

	tea "github.com/charmbracelet/bubbletea"
)

func Start() {
	cfg := config.Default()
	mainModel, err := update.NewModel(cfg)
	if err != nil {
		fmt.Printf("Error creating model: %v\n", err)
		os.Exit(1)
	}

	p := tea.NewProgram(
		update.NewManager(&mainModel),
		tea.WithAltScreen(),
	)

	finalModel, err := p.Run()
	if err != nil {
		fmt.Printf("Error starting program: %v\n", err)
		os.Exit(1)
	}

	// Save conversation on exit
	if m, ok := finalModel.(*update.Manager); ok {
		if m.Main != nil && m.Main.Session != nil {
			if err := m.Main.Session.SaveConversation(); err != nil {
				log.Printf("Error saving conversation history: %v", err)
			}
		}
	}
}
