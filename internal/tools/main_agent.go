package tools

import (
	"encoding/json"
	"fmt"
)

func init() {
	RegisterTool(
		Definition{
			ToolName:    "main_agent",
			Description: "Returns control to the main agent to continue the conversation or summarize results. Use this after a specialized agent has completed its task.",
			Args: []ArgumentDefinition{
				{
					Name:        "prompt",
					Type:        "string",
					Description: "The prompt or summary to pass back to the main agent.",
				},
			},
		},
		mainAgent,
	)
}

func mainAgent(args map[string]interface{}, lastAIResponse string) (string, error) {
	prompt, ok := args["prompt"].(string)
	if !ok || prompt == "" {
		return "", fmt.Errorf("missing or invalid 'prompt' argument")
	}

	request := map[string]interface{}{
		"_special_agent_request": "main_agent",
		"prompt":                 prompt,
	}

	jsonBytes, err := json.Marshal(request)
	if err != nil {
		return "", fmt.Errorf("failed to create agent request: %w", err)
	}

	return string(jsonBytes), nil
}
