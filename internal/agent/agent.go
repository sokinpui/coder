package agent

import (
	"github.com/sokinpui/itf.go/cli"
	"github.com/sokinpui/itf.go/itf"
)

// ExtractToolCalls extracts tool call JSON from a given string.
// It returns the JSON string of the tool calls if found, otherwise an empty string.
func ExtractToolCalls(content string) (string, error) {
	config := cli.Config{} // Use default config

	toolCallsJSON, err := itf.GetToolCall(content, config)
	if err != nil {
		// itf.GetToolCall returns an error if no tool call is found.
		// We treat this as a non-error case, meaning no tools to execute.
		return "", nil
	}

	return toolCallsJSON, nil
}
