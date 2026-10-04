package ui

import (
	"fmt"
	"io"
	"log"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/sokinpui/coder/internal/engine"
)

func Start(sess engine.EngineSession, prompt string) error {
	log.SetOutput(io.Discard)

	mainModel, err := NewModel(sess, prompt)
	if err != nil {
		return fmt.Errorf("error creating model: %w", err)
	}

	manager := NewManager(&mainModel)
	manager.Overlays = []Overlay{
		&QuickViewOverlay{},
		&SelectorOverlay{},
		&PaletteOverlay{},
	}

	p := tea.NewProgram(
		manager,
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
		tea.WithFilter(newSGRSequenceFilter()),
	)

	finalModel, err := p.Run()
	if err != nil {
		return fmt.Errorf("error starting program: %w", err)
	}

	// Save conversation on exit
	if m, ok := finalModel.(*Manager); ok {
		if m.Main != nil && m.Main.Session != nil {
			if err := m.Main.Session.SaveConversation(); err != nil {
				log.Printf("Error saving conversation history: %v", err)
			}
		}
	}
	return nil
}
