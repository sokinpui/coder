package core

type MessageType int

const (
	UserMessage MessageType = iota
	AIMessage
	CommandResultMessage
	CommandErrorResultMessage
	InitMessage
)

type Message struct {
	Type    MessageType
	Content string
}
