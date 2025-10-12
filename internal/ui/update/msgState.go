package update

import "coder/internal/history"

type State int

const (
	StateIdle State = iota
	StateGenPending
	StateThinking
	StateGenerating
	StateCancelling
	StateVisualSelect
	StateFzf
	StateHistorySelect
)

type (
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
)
