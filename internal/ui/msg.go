package ui

import "coder/internal/history"

type state int

const (
	stateIdle state = iota
	stateThinking
	stateGenerating
	stateCancelling
	stateVisualSelect
	stateHistorySelect
)

type (
	streamResultMsg   string
	streamFinishedMsg struct{}
	renderTickMsg     struct{}
	errorMsg          struct{ error }
	ctrlCTimeoutMsg   struct{}
	tokenCountResultMsg int
	initialContextLoadedMsg struct{ err error }
	editorFinishedMsg struct {
		content string
		err     error
	}
	clearStatusBarMsg struct{}
	titleGeneratedMsg   struct{}
	historyListResultMsg struct {
		items []history.ConversationInfo
		err   error
	}
	conversationLoadedMsg struct {
		err error
	}
)
