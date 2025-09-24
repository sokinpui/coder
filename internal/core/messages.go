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
	DirectoryMessage
	ImageMessage
)

type Message struct {
	Type    MessageType
	Content string
}
