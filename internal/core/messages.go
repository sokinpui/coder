package core

type MessageType int

const (
	UserMessage MessageType = iota
	AIMessage
	ActionMessage
	ActionResultMessage
	ActionErrorResultMessage
	CommandMessage
	CommandResultMessage
	CommandErrorResultMessage
	InitMessage
)

type Message struct {
	Type    MessageType
	Content string
}
