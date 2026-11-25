package ui

import (
	"github.com/sokinpui/coder/internal/engine"
	"github.com/sokinpui/coder/internal/engine/history"
	"github.com/sokinpui/coder/internal/types"
	"github.com/sokinpui/coder/internal/ui/markdown"
)

type state int

const (
	stateIdle state = iota
	stateAsking
	stateThinking
	stateGenerating
)

type overlayMode int

const (
	overlayNone overlayMode = iota
	overlaySelector
	overlayQuickView
	overlayConfirm
	overlayBtw
	overlayCommandPalette
)

type modelsFetchedMsg struct {
	models []string
	err    error
}

type (
	sessionEventMsg struct {
		sessID string
		event  types.SessionEvent
		sub    <-chan types.SessionEvent
	}
	sessionFinishedMsg struct {
		sessID string
	}
	aiRenderedMsg struct {
		sessID  string
		msgIdx  int
		content string
		lines   []string
		width   int
	}
	markdownBatchRenderedMsg struct {
		sessID  string
		results []markdown.RenderResult
		width   int
	}
	errorMsg struct {
		sessID string
		error  error
	}
	tokenCountResultMsg struct {
		sessID string
		count  int
	}
	toolRenderResult struct {
		key   ToolCacheKey
		value string
	}
	toolsBatchRenderedMsg struct {
		sessID  string
		results []toolRenderResult
		width   int
	}
	mediaBatchRenderedMsg struct {
		sessID  string
		indices []int
		hasMore bool
	}
	ctrlCTimeoutMsg         struct{}
	initialContextLoadedMsg struct{ err error }
	editorFinishedMsg       struct {
		content         string
		originalContent string
		err             error
	}
	fileEditorFinishedMsg struct {
		err error
	}
	clearStatusBarMsg    struct{}
	titleGeneratedMsg    struct{ title string }
	animateTitleTickMsg  struct{}
	historyListResultMsg struct {
		items []history.ConversationInfo
		err   error
	}
	addFilesListResultMsg struct {
		items []string
	}
	conversationLoadedMsg struct {
		sess engine.EngineSession
		err  error
	}
	switchActiveSessionMsg struct {
		sess engine.EngineSession
	}
	conversationCompactedMsg struct {
		sessID  string
		newSess engine.EngineSession
		summary string
		err     error
	}
	pasteResultMsg   = PasteResultMsg
	shellFinishedMsg struct {
		cmdStr string
		output string
		err    error
	}
	MouseWheelScrollMsg struct {
		Lines int
		X     int
		Y     int
	}
	btwStreamChunkMsg struct {
		sub   <-chan types.StreamChunk
		chunk types.StreamChunk
	}
	btwStreamFinishedMsg struct {
		sub <-chan types.StreamChunk
	}
	btwPasteResultMsg struct {
		content string
		err     error
	}
	commandPalettePasteResultMsg struct {
		content string
		err     error
	}
)
