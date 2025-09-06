package ui

type state int

const (
	stateIdle state = iota
	stateThinking
	stateGenerating
)

type message struct {
	isUser  bool
	content string
}

type (
	streamResultMsg   string
	streamFinishedMsg struct{}
	renderTickMsg     struct{}
	errorMsg          struct{ error }
)
