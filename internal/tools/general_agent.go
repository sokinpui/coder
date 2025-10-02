package tools

import (
	"encoding/json"
	"fmt"
)

func init() {
	RegisterTool(
		Definition{
			ToolName:    "general_agent",
			Description: "Delegates a general question or reasoning task to a specialized agent. Use this for tasks that do not involve coding or writing.",
			Args: []ArgumentDefinition{
				{
					Name:        "prompt",
					Type:        "string",
					Description: "The detailed prompt, question, or task for the general agent.",
				},
			},
		},
		generalAgent,
	)
}

func generalAgent(args map[string]interface{}, lastAIResponse string) (string, error) {
	prompt, ok := args["prompt"].(string)
	if !ok || prompt == "" {
		return "", fmt.Errorf("missing or invalid 'prompt' argument")
	}

	request := map[string]interface{}{
		"_special_agent_request": "general_agent",
		"prompt":                 prompt,
	}

	jsonBytes, err := json.Marshal(request)
	if err != nil {
		return "", fmt.Errorf("failed to create agent request: %w", err)
	}

	return string(jsonBytes), nil
}
