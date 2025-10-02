package tools

import (
	"encoding/json"
	"fmt"
)

func init() {
	RegisterTool(
		Definition{
			ToolName:    "writing_agent",
			Description: "Delegates a writing task to a specialized agent. Use this for drafting, editing, or proofreading text.",
			Args: []ArgumentDefinition{
				{
					Name:        "prompt",
					Type:        "string",
					Description: "The detailed prompt or writing task for the writing agent.",
				},
			},
		},
		writingAgent,
	)
}

func writingAgent(args map[string]interface{}, lastAIResponse string) (string, error) {
	prompt, ok := args["prompt"].(string)
	if !ok || prompt == "" {
		return "", fmt.Errorf("missing or invalid 'prompt' argument")
	}

	request := map[string]interface{}{
		"_special_agent_request": "writing_agent",
		"prompt":                 prompt,
	}

	jsonBytes, err := json.Marshal(request)
	if err != nil {
		return "", fmt.Errorf("failed to create agent request: %w", err)
	}

	return string(jsonBytes), nil
}
