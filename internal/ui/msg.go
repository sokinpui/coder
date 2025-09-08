package ui

type state int

const (
	stateIdle state = iota
	stateThinking
	stateGenerating
	stateCancelling
)

type (
	streamResultMsg   string
	streamFinishedMsg struct{}
	renderTickMsg     struct{}
	errorMsg          struct{ error }
	ctrlCTimeoutMsg   struct{}
)
