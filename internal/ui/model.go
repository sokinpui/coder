package ui

import (
	"coder/internal/config"
	"coder/internal/generation"
	"context"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
)

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// Model defines the state of the application's UI.
type Model struct {
	textArea           textarea.Model
	viewport           viewport.Model
	spinner            spinner.Model
	generator          *generation.Generator
	streamSub          chan string
	cancelGeneration   context.CancelFunc
	messages           []message
	state              state
	quitting           bool
	height             int
	width              int
	glamourRenderer    *glamour.TermRenderer
	isStreaming        bool
	lastRenderedAIPart string
	ctrlCPressed       bool
}

// NewModel creates a new UI model.
func NewModel(cfg *config.Config) (Model, error) {
	gen, err := generation.New(cfg)
	if err != nil {
		return Model{}, err
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

	return Model{
		textArea:           ta,
		viewport:           vp,
		spinner:            s,
		generator:          gen,
		state:              stateIdle,
		glamourRenderer:    renderer,
		isStreaming:        false,
		messages:           []message{},
		lastRenderedAIPart: "",
		ctrlCPressed:       false,
	}, nil
}
