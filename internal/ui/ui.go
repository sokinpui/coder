package ui

import (
	"coder/internal/config"
	"context"
	"fmt"
	"log"
	"os"
	"slices"

	"github.com/sokinpui/synapse.go/v2/client"
	tea "github.com/charmbracelet/bubbletea"
)

func Start() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Printf("Error loading configuration: %v\n", err)
		os.Exit(1)
	}

	// Dynamic model resolution
	fmt.Println("Connecting to synapse server to fetch available models...")
	c, err := client.New(cfg.GRPC.Addr)
	if err != nil {
		fmt.Printf("Error connecting to server: %v\n", err)
		os.Exit(1)
	}

	models, err := c.ListModels(context.Background())
	c.Close()
	if err != nil {
		fmt.Printf("Error fetching models from server: %v\n", err)
		os.Exit(1)
	}
	if len(models) == 0 {
		fmt.Println("Warning: Server returned no available models.")
	}

	cfg.AvailableModels = models

	// Validate config default model
	if len(models) > 0 {
		hasError := false
		if !slices.Contains(models, cfg.Generation.ModelCode) {
			fmt.Printf("Error: Configured chat model '%s' is not in the available list.\n", cfg.Generation.ModelCode)
			hasError = true
		}
		if !slices.Contains(models, cfg.Generation.TitleModelCode) {
			fmt.Printf("Error: Configured title model '%s' is not in the available list.\n", cfg.Generation.TitleModelCode)
			hasError = true
		}
		if hasError {
			fmt.Printf("Available models: %v\n", models)
		}
	}

	mainModel, err := NewModel(cfg)
	if err != nil {
		fmt.Printf("Error creating model: %v\n", err)
		os.Exit(1)
	}

	manager := NewManager(&mainModel)
	manager.Overlays = []Overlay{
		&PaletteOverlay{},
		&FinderOverlay{},
		&SearchOverlay{},
		&QuickViewOverlay{},
		&TreeOverlay{},
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
