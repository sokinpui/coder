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
	Data    []byte // For raw image data
}

func (t MessageType) String() string {
	switch t {
	case UserMessage:
		return "User"
	case AIMessage:
		return "AI"
	case CommandMessage:
		return "Command"
	case CommandResultMessage:
		return "Command Result"
	case CommandErrorResultMessage:
		return "Command Error"
	case InitMessage:
		return "System"
	case DirectoryMessage:
		return "Directory"
	case ImageMessage:
		return "Image"
	default:
		return "Unknown"
	}
}
