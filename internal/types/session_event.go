package types

type EventType int

const (
	NoOp EventType = iota
	MessagesUpdated
	GenerationStarted
	NewSessionStarted
	TermExecutionStarted
	Quit
)

type SessionEventKind int

const (
	EventChunk SessionEventKind = iota
	EventThinking
	EventToolCall
	EventToolResult
	EventComplete
	EventError
)

type SessionEvent struct {
	Kind             SessionEventKind
	Content          string
	ReasoningContent string
	ToolCallID       string
	ToolName         string
	ToolArguments    string
	ToolOutput       string
	Error            error
	Messages         []Message
}

// Event is returned by session methods to inform the UI about what happened.
type Event struct {
	Type EventType
	Data any // Can be a stream channel for GenerationStarted or an error for ErrorOccurred
	Mode string
}
