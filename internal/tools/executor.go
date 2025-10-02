package tools

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
)

// To add a new tool:
// 1. Create a new file in the `tools` directory (e.g., `my_tool.go`).
// 2. Define a function that matches the `ToolFunc` signature.
// 3. In an `init()` function, call `RegisterTool` with the tool's `Definition` and the function.

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
	ToolName string         `json:"tool,omitempty"`
	Args     map[string]any `json:"args,omitempty"`
	Shell    any            `json:"shell,omitempty"` // Can be string or []string
}

// ToolResult represents the result of a single tool execution.
type ToolResult struct {
	ToolCall ToolCall
	Output   string
	Error    error
}

type ToolFunc func(args map[string]any) (string, error)

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
		var output string
		var err error
		if call.ToolName != "" {
			output, err = executeTool(call)
		} else if call.Shell != nil {
			output, err = executeShell(call)
		} else {
			err = fmt.Errorf("invalid tool call object: must have 'tool' or 'shell' key")
		}

		results = append(results, ToolResult{
			ToolCall: call,
			Output:   output,
			Error:    err,
		})
	}

	return results, nil
}

func executeTool(call ToolCall) (string, error) {
	tool, exists := registry[call.ToolName]
	if !exists {
		return "", fmt.Errorf("tool '%s' not found", call.ToolName)
	}

	return tool.Function(call.Args)
}

func executeShell(call ToolCall) (string, error) {
	var commands []string
	switch shellCmd := call.Shell.(type) {
	case string:
		commands = []string{shellCmd}
	case []interface{}:
		for _, v := range shellCmd {
			if s, ok := v.(string); ok {
				commands = append(commands, s)
			} else {
				return "", fmt.Errorf("invalid shell command in array: not a string")
			}
		}
	default:
		return "", fmt.Errorf("invalid type for 'shell': expected string or array of strings")
	}

	if len(commands) == 0 {
		return "", nil
	}

	var output strings.Builder
	var firstErr error
	for _, command := range commands {
		// #nosec G204
		cmd := exec.Command("bash", "-c", command)
		out, err := cmd.CombinedOutput()
		output.Write(out)
		if err != nil && firstErr == nil {
			firstErr = fmt.Errorf("error executing '%s': %w", command, err)
		}
	}

	return output.String(), firstErr
}
