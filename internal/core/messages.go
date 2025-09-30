package core

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
	ToolCallMessage
	ToolResultMessage
)

type Message struct {
	Type    MessageType
	Content string // For text content, or file path for images (for prompt)
	DataURL string // For image base64 data (for UI rendering)
}
