package ui

import (
	"github.com/sokinpui/coder/internal/history"
	"github.com/sokinpui/coder/internal/session"
)

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
)

type modelsFetchedMsg struct {
	models []string
	err    error
}

type (
	tokenizerInitializedMsg struct{ err error }
	startGenerationMsg      struct{}
	streamResultMsg         string
	streamFinishedMsg       struct{}
	streamAnimeMsg          struct{}
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
		sess *session.Session
		err  error
	}
	switchActiveSessionMsg struct {
		sess *session.Session
	}
	pasteResultMsg struct {
		isImage bool
		content string
		err     error
	}
	finderResultMsg struct {
		result string
	}
)
