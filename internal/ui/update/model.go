package update

import (
	"coder/internal/commands"
	"coder/internal/config"
	"coder/internal/history"
	"coder/internal/session"
	"coder/internal/types"
	"coder/internal/utils"
	"sort"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	"github.com/charmbracelet/glamour"
)

const welcomeMessage = `Welcome to Coder!

- Chat with the AI.
- Press Enter for a new line in your prompt (or to run a command).
- Use Ctrl+J to send your message.
- Use Ctrl+E to edit your prompt in an external editor ($EDITOR).
- Use Ctrl+F to open a command finder.
- Use Ctrl+D and Ctrl+U to scroll the conversation.
- Use Ctrl+H to view conversation history.
- Use Esc or Ctrl+C to clear the input. Press Ctrl+C again on an empty line to quit.
- During generation, press Ctrl+C to cancel.
- Type ':help' for a list of all commands and shortcuts.
`

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
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
	TextArea                 textarea.Model
	Viewport                 viewport.Model
	Spinner                  spinner.Model
	Session                  *session.Session
	StreamSub                chan string
	State                    state
	Quitting                 bool
	Height                   int
	Width                    int
	GlamourRenderer          *glamour.TermRenderer
	IsStreaming              bool
	LastRenderedAIPart       string
	CtrlCPressed             bool
	TokenCount               int
	IsCountingTokens         bool
	ShowPalette              bool
	AvailableCommands        []string
	PaletteFilteredCommands  []string
	PaletteCursor            int
	LastInteractionFailed    bool
	PaletteFilteredArguments []string
	IsCyclingCompletions     bool
	ClearedInputBuffer       string
	VisualMode               visualMode
	VisualIsSelecting        bool
	SelectableBlocks         []messageBlock
	VisualSelectCursor       int
	VisualSelectStart        int
	StatusBarMessage         string
	AnimatingTitle           bool
	FullGeneratedTitle       string
	DisplayedTitle           string
	EditingMessageIndex      int
	HistoryItems             []history.ConversationInfo
	HistoryCussorPos         int
	HistoryGGPressed         bool
	PreserveInputOnSubmit    bool
	CommandHistory           []string
	CommandHistoryCursor     int
	commandHistoryModified   string
}

func NewModel(cfg *config.Config) (Model, error) {
	sess, err := session.New(cfg)
	if err != nil {
		return Model{}, err
	}

	s := spinner.New()
	s.Spinner = typingSpinner

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

	sess.AddMessages(types.Message{Type: types.InitMessage, Content: welcomeMessage})

	dirMsg := utils.GetDirInfoContent()
	sess.AddMessages(types.Message{Type: types.DirectoryMessage, Content: dirMsg})
	availableCommands := commands.GetCommands()
	sort.Strings(availableCommands)

	return Model{
		TextArea:                 ta,
		Viewport:                 vp,
		Spinner:                  s,
		Session:                  sess,
		State:                    stateIdle,
		GlamourRenderer:          renderer,
		IsStreaming:              false,
		LastRenderedAIPart:       "",
		CtrlCPressed:             false,
		TokenCount:               0,
		IsCountingTokens:         false,
		ShowPalette:              false,
		AvailableCommands:        availableCommands,
		PaletteFilteredCommands:  []string{},
		PaletteCursor:            0,
		LastInteractionFailed:    false,
		PaletteFilteredArguments: []string{},
		IsCyclingCompletions:     false,
		ClearedInputBuffer:       "",
		VisualMode:               visualModeNone,
		VisualIsSelecting:        false,
		SelectableBlocks:         []messageBlock{},
		VisualSelectCursor:       0,
		VisualSelectStart:        0,
		StatusBarMessage:         "",
		AnimatingTitle:           false,
		FullGeneratedTitle:       "",
		DisplayedTitle:           "",
		EditingMessageIndex:      -1,
		HistoryItems:             nil,
		HistoryCussorPos:         0,
		HistoryGGPressed:         false,
		PreserveInputOnSubmit:    false,
		CommandHistory:           []string{},
		CommandHistoryCursor:     0,
		commandHistoryModified:   "",
	}, nil
}
