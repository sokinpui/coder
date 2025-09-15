package ui

type state int

const (
	stateIdle state = iota
	stateThinking
	stateGenerating
	stateCancelling
	stateVisualSelect
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
)
