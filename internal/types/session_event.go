package types

type EventType int

const (
	NoOp EventType = iota
	MessagesUpdated
	GenerationStarted
	NewSessionStarted
	ShellExecutionStarted
	Quit
)

type SessionEventKind int

const (
	EventChunk SessionEventKind = iota
	EventThinking
	EventToolCall
	EventToolCallDelta
	EventToolResult
	EventToolOutputChunk
	EventComplete
	EventError
	EventToolConfirm
)

type ToolConfirmResponse struct {
	Approved    bool
	AlwaysAllow bool
}

type ToolConfirmRequest struct {
	CallID    string `json:"callId"`
	ToolName  string `json:"toolName"`
	Arguments string `json:"arguments"`
}

type SessionEvent struct {
	Kind             SessionEventKind
	Content          string
	ReasoningContent string
	ToolCallIndex    int
	ToolCallID       string
	ToolName         string
	ToolArguments    string
	ToolOutput       string
	ToolOutputChunk  string
	Error            error
	Messages         []Message
	Confirm          *ToolConfirmRequest
}

// Event is returned by session methods to inform the UI about what happened.
type Event struct {
	Type EventType
	Data any // Can be a stream channel for GenerationStarted or an error for ErrorOccurred
	Mode string
}
