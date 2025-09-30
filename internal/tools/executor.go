package tools

import (
	"encoding/json"
	"fmt"
)

// To add a new tool:
// 1. Create a new file in the `tools` directory (e.g., `my_tool.go`).
// 2. In that file, define a function that matches the `ToolFunc` signature.
// 3. In an `init()` function in the same file, call `RegisterTool` to make it available.

// ToolCall represents a single tool call from the model.
// The model is expected to return a JSON array of these objects.
// Example:
// [
//
//	{"tool": "read_file", "args": {"path": "README.md"}},
//	{"tool": "list_files"}
//
// ]
type ToolCall struct {
	ToolName string         `json:"tool"`
	Args     map[string]any `json:"args"`
}

// ToolResult represents the result of a single tool execution.
type ToolResult struct {
	ToolCall ToolCall
	Output   string
	Error    error
}

type ToolFunc func(args map[string]any) (string, error)

var registry = make(map[string]ToolFunc)

// RegisterTool adds a new tool to the registry. This is not thread-safe
// and should be called from `init()` functions only.
func RegisterTool(name string, fn ToolFunc) {
	if _, exists := registry[name]; exists {
		panic(fmt.Sprintf("tool with name '%s' is already registered", name))
	}
	registry[name] = fn
}

func ExecuteToolCalls(toolCallsJSON string) ([]ToolResult, error) {
	var calls []ToolCall
	if err := json.Unmarshal([]byte(toolCallsJSON), &calls); err != nil {
		return nil, fmt.Errorf("failed to parse tool calls JSON: %w. Ensure it is a valid JSON array of tool call objects", err)
	}

	if len(calls) == 0 {
		return nil, nil
	}

	var results []ToolResult
	for _, call := range calls {
		output, err := executeTool(call)
		results = append(results, ToolResult{
			ToolCall: call,
			Output:   output,
			Error:    err,
		})
	}

	return results, nil
}

func executeTool(call ToolCall) (string, error) {
	toolFunc, exists := registry[call.ToolName]
	if !exists {
		return "", fmt.Errorf("tool '%s' not found", call.ToolName)
	}

	return toolFunc(call.Args)
}
