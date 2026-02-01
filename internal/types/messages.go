package types

type MessageType int

const (
	UserMessage MessageType = iota
	AIMessage
	CommandMessage
	CommandResultMessage
	CommandErrorResultMessage
	InitMessage
	DirectoryMessage
	ImageMessage
)

type Message struct {
	Type    MessageType
	Content string // For text content, or file path for images (for prompt)
}
