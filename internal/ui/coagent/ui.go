package coagentui

import (
	"fmt"
	"io"
	"log"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/sokinpui/coder/internal/config"
)

func Start(cfg *config.Config, initialPrompt string) error {
	log.SetOutput(io.Discard)

	m, err := New(cfg, initialPrompt)
	if err != nil {
		return fmt.Errorf("failed to initialize coagent UI: %w", err)
	}

	p := tea.NewProgram(m)
	_, err = p.Run()
	return err
}
