package ui

type state int

const (
	stateIdle state = iota
	stateThinking
	stateGenerating
	stateCancelling
)

type messageType int

const (
	userMessage messageType = iota
	aiMessage
	commandResultMessage
	commandErrorResultMessage
	appMessage
)

type message struct {
	mType   messageType
	content string
}

type (
	streamResultMsg   string
	streamFinishedMsg struct{}
	renderTickMsg     struct{}
	errorMsg          struct{ error }
	ctrlCTimeoutMsg   struct{}
)
