package coderui

import (
	"github.com/sokinpui/coder/internal/engine/coder"
	"github.com/sokinpui/coder/internal/engine/history"
	"github.com/sokinpui/coder/internal/types"
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
)

type modelsFetchedMsg struct {
	models []string
	err    error
}

type (
	streamResultMsg struct {
		sessID string
		chunk  types.StreamChunk
		sub    chan types.StreamChunk
	}
	streamFinishedMsg struct {
		sessID string
	}
	aiRenderedMsg struct {
		sessID  string
		msgIdx  int
		content string
		lines   []string
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
		sess *coder.Session
		err  error
	}
	switchActiveSessionMsg struct {
		sess *coder.Session
	}
	pasteResultMsg struct {
		isImage bool
		content string
		err     error
	}
	termFinishedMsg struct {
		cmdStr string
		output string
		err    error
	}
)
