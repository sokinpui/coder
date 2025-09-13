package ui

import (
	"coder/internal/config"
	"coder/internal/core"
	"coder/internal/session"
	"sort"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
)

const welcomeMessage = `Welcome to Coder!

- Chat with the AI.
- Use Ctrl+J to send your message.
- Press Enter for a new line in your prompt (or to run a command).
- Use Esc or Ctrl+C to clear the input. Press Ctrl+C again on an empty line to quit.
- Use Ctrl+D and Ctrl+U to scroll the conversation.
- Type ':' to see available commands and actions.
- Place files in the 'Context' directory to provide them to the AI.`

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
	session            *session.Session
	streamSub          chan string
	state              state
	quitting           bool
	height             int
	width              int
	glamourRenderer    *glamour.TermRenderer
	isStreaming        bool
	lastRenderedAIPart string
	ctrlCPressed       bool
	tokenCount         int
	isCountingTokens   bool
	showPalette        bool
	availableActions   []string
	availableCommands  []string
	paletteFilteredActions  []string
	paletteFilteredCommands []string
	paletteCursor           int
	lastInteractionFailed   bool
}

func NewModel(cfg *config.Config) (Model, error) {
	sess, err := session.New(cfg)
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

	sess.AddMessage(core.Message{Type: core.InitMessage, Content: welcomeMessage})

	actions := core.GetActions()
	sort.Strings(actions)
	commands := core.GetCommands()
	sort.Strings(commands)

	return Model{
		textArea:                ta,
		viewport:                vp,
		spinner:                 s,
		session:                 sess,
		state:                   stateIdle,
		glamourRenderer:         renderer,
		isStreaming:             false,
		lastRenderedAIPart:      "",
		ctrlCPressed:            false,
		tokenCount:              0,
		isCountingTokens:        false,
		showPalette:             false,
		availableActions:        actions,
		availableCommands:       commands,
		paletteFilteredActions:  []string{},
		paletteFilteredCommands: []string{},
		paletteCursor:           0,
		lastInteractionFailed:   false,
	}, nil
}
