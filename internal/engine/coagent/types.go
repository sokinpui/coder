package coagent

import "github.com/sokinpui/coder/internal/types"

type ToolCallInfo struct {
	CallID    string `json:"call_id"`
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

type ToolResultInfo struct {
	CallID string          `json:"call_id"`
	Name   string          `json:"name"`
	Output string          `json:"output"`
	Images []types.Message `json:"images,omitempty"`
}

type AgentStreamChunk struct {
	State            string
	Content          string
	ReasoningContent string
	ToolCall         *ToolCallInfo
	ToolResult       *ToolResultInfo
	ToolConfirm      *types.ToolConfirmRequest
	Messages         []types.Message
}
