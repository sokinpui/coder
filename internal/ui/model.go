package ui

import (
	"github.com/sokinpui/coder/internal/commands"
	"github.com/sokinpui/coder/internal/config"
	"github.com/sokinpui/coder/internal/history"
	"github.com/sokinpui/coder/internal/session"
	"github.com/sokinpui/coder/internal/types"
	"github.com/sokinpui/coder/internal/utils"
	"sort"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	"github.com/charmbracelet/glamour"
)

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
	IsFetchingModels         bool
	PaletteFilteredCommands  []string
	PaletteCursor            int
	LastInteractionFailed    bool
	PaletteFilteredArguments []string
	IsCyclingCompletions     bool
	VisualMode               visualMode
	Finder                   FinderModel
	Search                   SearchModel
	messageLineOffsets       map[int]int
	SearchQuery              string
	SearchFocusMsgIndex      int
	SearchFocusLineNum       int
	Tree                     TreeModel
	QuickView                *QuickViewModel
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

func NewModel(cfg *config.Config, mode string) (Model, error) {
	sess, err := session.New(cfg, mode)
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
		glamour.WithStandardStyle("dark"),
		glamour.WithWordWrap(vp.Width),
	)
	if err != nil {
		return Model{}, err
	}

	sess.AddMessages(types.Message{Type: types.InitMessage, Content: utils.WelcomeMessage})

	dirMsg := utils.GetDirInfoContent()
	sess.AddMessages(types.Message{Type: types.DirectoryMessage, Content: dirMsg})
	availableCommands := commands.GetCommands()
	sort.Strings(availableCommands)

	return Model{
		TextArea:                 ta,
		Viewport:                 vp,
		Spinner:                  s,
		Session:                  sess,
		State:                    stateInitializing,
		GlamourRenderer:          renderer,
		IsStreaming:              false,
		LastRenderedAIPart:       "",
		CtrlCPressed:             false,
		TokenCount:               0,
		IsCountingTokens:         false,
		ShowPalette:              false,
		IsFetchingModels:         true,
		AvailableCommands:        availableCommands,
		PaletteFilteredCommands:  []string{},
		PaletteCursor:            0,
		LastInteractionFailed:    false,
		PaletteFilteredArguments: []string{},
		IsCyclingCompletions:     false,
		VisualMode:               visualModeNone,
		VisualIsSelecting:        false,
		Search:                   NewSearch(),
		Finder:                   NewFinder(),
		Tree:                     NewTree(),
		messageLineOffsets:       make(map[int]int),
		SearchQuery:              "",
		SearchFocusMsgIndex:      -1,
		SearchFocusLineNum:       -1,
		QuickView:                NewQuickView(),
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
