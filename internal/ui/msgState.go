package ui

import "coder/internal/history"

type state int

const (
	stateInitializing state = iota
	stateIdle
	stateGenPending
	stateThinking
	stateGenerating
	stateCancelling
	stateVisualSelect
	stateHistorySelect
	stateFinder
	stateSearch
)

type (
	tokenizerInitializedMsg struct{ err error }
	startGenerationMsg      struct{}
	streamResultMsg         string
	streamFinishedMsg       struct{}
	renderTickMsg           struct{}
	errorMsg                struct{ error }
	ctrlCTimeoutMsg         struct{}
	tokenCountResultMsg     int
	initialContextLoadedMsg struct{ err error }
	editorFinishedMsg       struct {
		content         string
		originalContent string
		err             error
	}
	clearStatusBarMsg    struct{}
	titleGeneratedMsg    struct{ title string }
	animateTitleTickMsg  struct{}
	historyListResultMsg struct {
		items []history.ConversationInfo
		err   error
	}
	conversationLoadedMsg struct {
		err error
	}
	pasteResultMsg struct {
		isImage bool
		content string
		err     error
	}
	finderResultMsg struct {
		result string
	}
	searchResultMsg struct {
		item SearchItem
	}
)
