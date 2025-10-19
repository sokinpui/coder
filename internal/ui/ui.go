package ui

import (
	"coder/internal/config"
	"fmt"
	"log"
	"os"

	tea "github.com/charmbracelet/bubbletea"
)

func Start() {
	cfg := config.Default()
	mainModel, err := NewModel(cfg)
	if err != nil {
		fmt.Printf("Error creating model: %v\n", err)
		os.Exit(1)
	}

	manager := NewManager(&mainModel)
	manager.Overlays = []Overlay{&PaletteOverlay{}}

	p := tea.NewProgram(
		manager,
		tea.WithAltScreen(),
	)

	finalModel, err := p.Run()
	if err != nil {
		fmt.Printf("Error starting program: %v\n", err)
		os.Exit(1)
	}

	// Save conversation on exit
	if m, ok := finalModel.(*Manager); ok {
		if m.Main != nil && m.Main.Session != nil {
			if err := m.Main.Session.SaveConversation(); err != nil {
				log.Printf("Error saving conversation history: %v", err)
			}
		}
	}
}
