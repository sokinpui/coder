package rpc

import "encoding/json"

type Request struct {
	JSONRPC string           `json:"jsonrpc"`
	ID      *json.RawMessage `json:"id,omitempty"`
	Method  string           `json:"method"`
	Params  json.RawMessage  `json:"params,omitempty"`
}

type Response struct {
	JSONRPC string           `json:"jsonrpc"`
	ID      *json.RawMessage `json:"id,omitempty"`
	Result  any              `json:"result,omitempty"`
	Error   *ResponseError   `json:"error,omitempty"`
}

type ResponseError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type Notification struct {
	JSONRPC string `json:"jsonrpc"`
	Method  string `json:"method"`
	Params  any    `json:"params"`
}

// Session parameters
type InitParams struct {
	Mode         string   `json:"mode,omitempty"`
	Instruction  string   `json:"instruction,omitempty"`
	ContextFiles []string `json:"contextFiles,omitempty"`
	Model        string   `json:"model,omitempty"`
}

type SendPromptParams struct {
	Content string `json:"content"`
	Silent  bool   `json:"silent,omitempty"`
}

type ContextModifyParams struct {
	Paths []string `json:"paths"`
}

type ApplyItfParams struct {
	Content string `json:"content,omitempty"`
	Args    string `json:"args,omitempty"`
}

// Stream Notification Params
type StreamChunkNotification struct {
	Content          string `json:"content,omitempty"`
	ReasoningContent string `json:"reasoningContent,omitempty"`
	Done             bool   `json:"done"`
	Error            string `json:"error,omitempty"`
}
