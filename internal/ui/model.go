package ui

import (
	"coder/internal/config"
	"coder/internal/history"
	"coder/internal/core"
	"coder/internal/session"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
)

const welcomeMessage = `Welcome to Coder!

- Chat with the AI.
- Use Ctrl+J to send your message.
- Use Ctrl+E to edit your prompt in an external editor ($EDITOR).
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

type visualMode int

const (
	visualModeNone visualMode = iota
	visualModeGenerate
	visualModeEdit
	visualModeBranch
)

// messageBlock represents a single selectable unit in the conversation view.
// It holds the start and end indices (inclusive) of messages in the session's
// message slice that form a single logical block.
type messageBlock struct {
	startIdx int
	endIdx   int
}

type Model struct {
	textArea                textarea.Model
	viewport                viewport.Model
	spinner                 spinner.Model
	session                 *session.Session
	streamSub               chan string
	state                   state
	quitting                bool
	height                  int
	width                   int
	glamourRenderer         *glamour.TermRenderer
	isStreaming             bool
	lastRenderedAIPart      string
	ctrlCPressed            bool
	tokenCount              int
	isCountingTokens        bool
	showPalette             bool
	availableActions        []string
	availableCommands       []string
	paletteFilteredActions  []string
	paletteFilteredCommands []string
	paletteCursor           int
	lastInteractionFailed   bool
	paletteFilteredArguments []string
	isCyclingCompletions    bool
	clearedInputBuffer      string
	visualMode              visualMode
	visualIsSelecting       bool
	selectableBlocks        []messageBlock
	visualSelectCursor      int
	visualSelectStart       int
	statusBarMessage        string
	editingMessageIndex     int
	historyItems            []history.ConversationInfo
	historySelectCursor     int
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

	wd, err := os.Getwd()
	if err != nil {
		wd = "an unknown directory"
	} else {
		home, err := os.UserHomeDir()
		if err == nil && home != "" && strings.HasPrefix(wd, home) {
			wd = "~" + strings.TrimPrefix(wd, home)
		}
	}
	dirMsg := fmt.Sprintf("Currently in: %s", wd)
	sess.AddMessage(core.Message{Type: core.DirectoryMessage, Content: dirMsg})

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
		paletteFilteredArguments: []string{},
		isCyclingCompletions:    false,
		clearedInputBuffer:      "",
		visualMode:              visualModeNone,
		visualIsSelecting:       false,
		selectableBlocks:        []messageBlock{},
		visualSelectCursor:      0,
		visualSelectStart:       0,
		statusBarMessage:        "",
		editingMessageIndex:     -1,
		historyItems:            nil,
		historySelectCursor:     0,
	}, nil
}
