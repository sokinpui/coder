package tools

import (
	"encoding/json"
	"fmt"
	"strings"
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

func ExecuteToolCalls(toolCallsJSON string) (string, error) {
	var calls []ToolCall
	if err := json.Unmarshal([]byte(toolCallsJSON), &calls); err != nil {
		return "", fmt.Errorf("failed to parse tool calls JSON: %w. Ensure it is a valid JSON array of tool call objects", err)
	}

	if len(calls) == 0 {
		return "No tools were called.", nil
	}

	var results []string
	for _, call := range calls {
		output, err := executeTool(call)
		if err != nil {
			output = fmt.Sprintf("Error: %v", err)
		}
		// Format the output for clarity before returning to the model.
		result := fmt.Sprintf("Tool Call: %s\nTool Output:\n%s", call.ToolName, output)
		results = append(results, result)
	}

	return strings.Join(results, "\n\n"), nil
}

func executeTool(call ToolCall) (string, error) {
	toolFunc, exists := registry[call.ToolName]
	if !exists {
		return "", fmt.Errorf("tool '%s' not found", call.ToolName)
	}

	return toolFunc(call.Args)
}
