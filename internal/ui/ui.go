package ui

import (
	"fmt"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/sokinpui/coder/internal/commands"
	"github.com/sokinpui/coder/internal/config"
	"github.com/sokinpui/coder/internal/types"
	"log"
	"os"
)

func Start(mode string, initialInput string, contextFiles []string, instruction string) {
	cfg, err := config.Load()
	if err != nil {
		fmt.Printf("Error loading configuration: %v\n", err)
		os.Exit(1)
	}

	shellRegErrors := commands.RegisterShellCommands(cfg)

	mainModel, err := NewModel(cfg, mode, initialInput, contextFiles, instruction)
	if err != nil {
		fmt.Printf("Error creating model: %v\n", err)
		os.Exit(1)
	}
	for _, regErr := range shellRegErrors {
		mainModel.Session.AddMessages(types.Message{
			Type:    types.CommandErrorResultMessage,
			Content: regErr,
		})
	}

	manager := NewManager(&mainModel)
	manager.Overlays = []Overlay{
		&PaletteOverlay{},
		&FinderOverlay{},
		&QuickViewOverlay{},
	}

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
