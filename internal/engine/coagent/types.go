package coagent

type ToolCallInfo struct {
	CallID    string `json:"call_id"`
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

type ToolResultInfo struct {
	CallID string `json:"call_id"`
	Name   string `json:"name"`
	Output string `json:"output"`
}

type AgentStreamChunk struct {
	Content          string
	ReasoningContent string
	ToolCall         *ToolCallInfo
	ToolResult       *ToolResultInfo
}
