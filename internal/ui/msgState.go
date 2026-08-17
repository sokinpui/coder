package ui

import (
	"github.com/sokinpui/coder/internal/history"
	"github.com/sokinpui/coder/internal/session"
	"github.com/sokinpui/coder/internal/types"
)

type state int

const (
	stateIdle state = iota
	stateAsking
	stateThinking
	stateGenerating
	stateCancelling
)

type overlayMode int

const (
	overlayNone overlayMode = iota
	overlayHistory
	overlayFinder
	overlayAtomicMsg
	overlayQuickView
)

type modelsFetchedMsg struct {
	models []string
	err    error
}

type (
	streamResultMsg         types.StreamChunk
	streamFinishedMsg       struct{}
	errorMsg                struct{ error }
	ctrlCTimeoutMsg         struct{}
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
