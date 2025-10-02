package tools

import (
	"encoding/json"
	"fmt"
)

func init() {
	RegisterTool(
		Definition{
			ToolName:    "coding_agent",
			Description: "Delegates a coding task to a specialized agent. Use this for any code writing, modification, or explanation.",
			Args: []ArgumentDefinition{
				{
					Name:        "prompt",
					Type:        "string",
					Description: "The detailed prompt or task for the coding agent.",
				},
			},
		},
		codingAgent,
	)
}

func codingAgent(args map[string]interface{}, lastAIResponse string) (string, error) {
	prompt, ok := args["prompt"].(string)
	if !ok || prompt == "" {
		return "", fmt.Errorf("missing or invalid 'prompt' argument")
	}

	request := map[string]interface{}{
		"_special_agent_request": "coding_agent",
		"prompt":                 prompt,
	}

	jsonBytes, err := json.Marshal(request)
	if err != nil {
		return "", fmt.Errorf("failed to create agent request: %w", err)
	}

	return string(jsonBytes), nil
}
