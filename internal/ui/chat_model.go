package ui

import (
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
)

type ChatModel struct {
	TextArea               textarea.Model
	Viewport               viewport.Model
	Spinner                spinner.Model
	StreamSub              chan string
	IsStreaming            bool
	StreamBuffer           string
	StreamDone             bool
	IsStreamAnime          bool
	LastRenderedAIPart     string
	CtrlCPressed           bool
	LastInteractionFailed  bool
	ShowPalette            bool
	PaletteFilteredCommands []string
	PaletteFilteredArguments []string
	PaletteCursor          int
	IsCyclingCompletions   bool
	IsFetchingModels       bool
	AnimatingTitle         bool
	FullGeneratedTitle     string
	DisplayedTitle         string
	EditingMessageIndex    int
	SearchQuery            string
	SearchFocusMsgIndex    int
	SearchFocusLineNum     int
	MessageLineOffsets     map[int]int
	CommandHistory         []string
	CommandHistoryCursor   int
	CommandHistoryModified string
	PreserveInputOnSubmit  bool
}

func NewChat(initialInput string) ChatModel {
	s := spinner.New()
	s.Spinner = typingSpinner

	ta := textarea.New()
	ta.Placeholder = "Enter your prompt..."
	ta.Focus()
	ta.SetValue(initialInput)
	ta.CursorEnd()
	ta.SetHeight(1)
	ta.Prompt = ""
	ta.ShowLineNumbers = false

	return ChatModel{
		TextArea:           ta,
		Viewport:           viewport.New(80, 20),
		Spinner:            s,
		IsFetchingModels:   true,
		MessageLineOffsets: make(map[int]int),
		SearchFocusMsgIndex: -1,
		SearchFocusLineNum: -1,
		EditingMessageIndex: -1,
	}
}
