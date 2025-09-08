package ui

import (
	"coder/internal/config"
	"coder/internal/core"
	"coder/internal/history"
	"coder/internal/contextdir"
	"coder/internal/generation"
	"context"
	"fmt"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
)

const welcomeMessage = `Welcome to Coder!

- Chat with the AI
- Press Enter for a new line in your prompt.
- Press Ctrl+J to send your message to the AI.
- Use /<command> to execute commands. Press Enter to run a command.
- Place files in the 'Context' directory.`

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

type Model struct {
	textArea           textarea.Model
	viewport           viewport.Model
	spinner            spinner.Model
	generator          *generation.Generator
	historyManager     *history.Manager
	streamSub          chan string
	cancelGeneration   context.CancelFunc
	messages           []core.Message
	state              state
	quitting           bool
	height             int
	width              int
	glamourRenderer    *glamour.TermRenderer
	isStreaming        bool
	lastRenderedAIPart string
	ctrlCPressed       bool
	systemInstructions string
	providedDocuments  string
}

func NewModel(cfg *config.Config) (Model, error) {
	gen, err := generation.New(cfg)
	if err != nil {
		return Model{}, err
	}

	hist, err := history.NewManager()
	if err != nil {
		return Model{}, fmt.Errorf("failed to initialize history manager: %w", err)
	}

	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	ta := textarea.New()
	ta.Placeholder = "Enter your prompt..."
	ta.Focus()
	ta.CharLimit = 0
	ta.SetHeight(1)
	ta.MaxHeight = 0
	ta.MaxWidth = 0
	ta.Prompt = ""
	ta.ShowLineNumbers = false

	vp := viewport.New(80, 20) // Initial size, will be updated

	renderer, err := glamour.NewTermRenderer(
		glamour.WithAutoStyle(),
		glamour.WithWordWrap(vp.Width),
	)
	if err != nil {
		return Model{}, err
	}

	sysInstructions, docs, err := contextdir.LoadContext()
	if err != nil {
		return Model{}, fmt.Errorf("failed to load context: %w", err)
	}

	initialMessages := []core.Message{
		{Type: core.InitMessage, Content: welcomeMessage},
	}

	return Model{
		textArea:           ta,
		viewport:           vp,
		spinner:            s,
		generator:          gen,
		historyManager:     hist,
		state:              stateIdle,
		glamourRenderer:    renderer,
		isStreaming:        false,
		messages:           initialMessages,
		lastRenderedAIPart: "",
		ctrlCPressed:       false,
		systemInstructions: sysInstructions,
		providedDocuments:  docs,
	}, nil
}
