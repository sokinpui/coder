package ui

import (
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	"github.com/sokinpui/coder/internal/types"
	"time"
)

type cachedRender struct {
	lines      []string
	content    string
	width      int
	isVisual   bool
	isCursorOn bool
	isSelected bool
}

type ChatModel struct {
	TextArea                 textarea.Model
	Viewport                 viewport.Model
	Spinner                  spinner.Model
	StreamSub                chan types.StreamChunk
	IsStreaming              bool
	StreamBuffer             string
	StreamDone               bool
	IsStreamAnime            bool
	LastRenderedAIPart       string
	CtrlCPressed             bool
	LastInteractionFailed    bool
	ShowPalette              bool
	PaletteFilteredCommands  []string
	PaletteFilteredArguments []string
	PaletteCursor            int
	PaletteOffset            int
	IsCyclingCompletions     bool
	IsFetchingModels         bool
	AnimatingTitle           bool
	FullGeneratedTitle       string
	DisplayedTitle           string
	EditingMessageIndex      int
	MessageLineOffsets       map[int]int
	PreserveInputOnSubmit    bool
	RenderCache              map[int]cachedRender
	StateStartTime           time.Time
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
	ta.CharLimit = 0
	ta.MaxHeight = 0
	ta.MaxWidth = 0

	return ChatModel{
		TextArea:            ta,
		Viewport:            viewport.New(80, 20),
		Spinner:             s,
		IsFetchingModels:    true,
		MessageLineOffsets:  make(map[int]int),
		EditingMessageIndex: -1,
		RenderCache:         make(map[int]cachedRender),
	}
}
