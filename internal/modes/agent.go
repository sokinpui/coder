package modes

import (
	"coder/internal/config"
	"coder/internal/core"
	"coder/internal/tools"
	"encoding/json"
	"fmt"
	"strings"
)

// AgentMode is the strategy for the agent/tool-using mode.
type AgentMode struct{}

// AgentRequest represents the parsed JSON from a special agent tool call.
type AgentRequest struct {
	AgentName string `json:"_special_agent_request"`
	Prompt    string `json:"prompt"`
}

var agentRoles = map[config.AgentName]string{
	config.CodingAgent:  core.AgentCodingRole,
	config.WritingAgent: core.AgentWritingRole,
	config.GeneralAgent: core.AgentGeneralRole,
	config.MainAgent:    core.AgentRole,
}

// GetRolePrompt returns the main agent role.
func (m *AgentMode) GetRolePrompt() string {
	return agentRoles[config.MainAgent]
}

// LoadContext does not load any context for agent mode.
func (m *AgentMode) LoadContext() (string, string, string, error) {
	return "", "", "", nil
}

// getAgentConfig returns the generation config for a given agent.
func (m *AgentMode) getAgentConfig(s SessionController, agentName config.AgentName) (*config.Generation, error) {
	agentGenConfig, ok := config.AgentConfigs[agentName]
	if !ok {
		return nil, fmt.Errorf("no config for agent: %s", agentName)
	}

	// Get base config from the session and create a copy to override.
	baseConfig := s.GetGenerator().Config
	newConfig := baseConfig
	newConfig.ModelCode = agentGenConfig.ModelCode
	newConfig.Temperature = agentGenConfig.Temperature

	return &newConfig, nil
}

// buildAgentPrompt constructs a prompt for a specific agent and message history.
func (m *AgentMode) buildAgentPrompt(messages []core.Message, agentName config.AgentName) (string, error) {
	rolePrompt, ok := agentRoles[agentName]
	if !ok {
		return "", fmt.Errorf("unknown agent: %s", agentName)
	}

	agentConfig, ok := config.AgentConfigs[agentName]
	if !ok {
		return "", fmt.Errorf("no config for agent: %s", agentName)
	}

	var toolDocs string
	if len(agentConfig.Tools) > 0 {
		toolDocsJSON, err := tools.GenerateToolDocsJSONForTools(agentConfig.Tools)
		if err != nil {
			return "", fmt.Errorf("failed to generate tool docs: %w", err)
		}
		if toolDocsJSON != "" {
			toolDocs = strings.Replace(core.ToolCallPrompt, "{{TOOLS_DOCUMENTATION}}", toolDocsJSON, 1)
		}
	}

	// Agent mode does not use file-based context.
	return BuildPrompt(rolePrompt, "", "", "", "", toolDocs, messages), nil
}

// ProcessAIResponse checks for tool calls in the last AI message, executes them,
// and starts a new generation cycle with the results.
func (m *AgentMode) ProcessAIResponse(s SessionController) core.Event {
	messages := s.GetMessages()
	if len(messages) == 0 {
		return core.Event{Type: core.NoOp}
	}
	lastMsg := messages[len(messages)-1]
	if lastMsg.Type != core.AIMessage || lastMsg.Content == "" {
		return core.Event{Type: core.NoOp}
	}

	toolCallsJSON, _ := tools.ExtractToolCalls(lastMsg.Content)
	if toolCallsJSON == "" {
		return core.Event{Type: core.NoOp}
	}

	results, err := tools.ExecuteToolCalls(toolCallsJSON, lastMsg.Content)
	if err != nil {
		resultContent := fmt.Sprintf("Error executing tool calls: %v", err)
		s.AddMessage(core.Message{Type: core.ToolResultMessage, Content: resultContent})
		return s.StartGeneration()
	}

	if len(results) == 0 {
		return core.Event{Type: core.NoOp}
	}

	// Sequentially add each tool call and its result to the message history.
	for _, res := range results {
		// Add the tool call message for this specific call.
		// We marshal the single ToolCall struct back into JSON for the message.
		toolCallBytes, err := json.Marshal(res.ToolCall)
		if err != nil {
			// If marshalling fails, create an error result message and continue.
			errorMsg := fmt.Sprintf("{\"tool\": \"%s\", \"error\": \"failed to marshal tool call: %v\"}", res.ToolCall.ToolName, err)
			s.AddMessage(core.Message{Type: core.ToolResultMessage, Content: errorMsg})
			continue
		}
		s.AddMessage(core.Message{Type: core.ToolCallMessage, Content: string(toolCallBytes)})

		// Add the tool result message.
		var toolName string
		if res.ToolCall.ToolName != "" {
			toolName = res.ToolCall.ToolName
		} else if res.ToolCall.Shell != nil {
			toolName = "shell"
		}

		resultObj := map[string]interface{}{"tool": toolName}
		if res.Error != nil {
			resultObj["error"] = res.Error.Error()
		}
		if res.Output != "" {
			// For agent calls, the output is the special JSON request.
			// We treat it like any other tool output.
			resultObj["output"] = res.Output
		}

		resultBytes, err := json.MarshalIndent(resultObj, "", "  ")
		if err != nil {
			errorMsg := fmt.Sprintf("{\"tool\": \"%s\", \"error\": \"failed to marshal tool result: %v\"}", toolName, err)
			s.AddMessage(core.Message{Type: core.ToolResultMessage, Content: errorMsg})
			continue
		}
		s.AddMessage(core.Message{Type: core.ToolResultMessage, Content: string(resultBytes)})
	}

	// After all tools are executed and results are appended, start a new generation.
	return s.StartGeneration()
}

// StartGeneration begins a new AI generation task using the agent-specific logic.
func (m *AgentMode) StartGeneration(s SessionController) core.Event {
	genConfig, err := m.getAgentConfig(s, config.MainAgent)
	if err != nil {
		s.AddMessage(core.Message{Type: core.CommandErrorResultMessage, Content: err.Error()})
		return core.Event{Type: core.MessagesUpdated}
	}
	return StartGeneration(s, genConfig)
}

// BuildPrompt constructs the prompt for agent mode.
func (m *AgentMode) BuildPrompt(systemInstructions, relatedDocuments, projectSourceCode string, messages []core.Message) string {
	// Agent mode does not use file-based context.
	// This will never fail as MainAgent is always in the map.
	prompt, _ := m.buildAgentPrompt(messages, config.MainAgent)
	return prompt
}
