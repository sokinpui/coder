package modes

import (
	"coder/internal/config"
	"coder/internal/core"
	"coder/internal/tools"
	"context"
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

// GetRolePrompt returns the agent role.
func (m *AgentMode) GetRolePrompt() string {
	return core.AgentRole
}

// LoadContext does not load any context for agent mode.
func (m *AgentMode) LoadContext() (string, string, string, error) {
	return "", "", "", nil
}

// getAgentContext returns the role prompt and generation config for a given agent.
func (m *AgentMode) getAgentContext(s SessionController, agentName string) (string, *config.Generation, error) {
	agent := config.AgentName(agentName)

	rolePrompt, err := agentRoles[agent]
	if !err {
		return "", nil, fmt.Errorf("unknown agent: %s", agentName)
	}

	agentGenConfig, err := config.AgentConfigs[agent]
	if !err {
		return "", nil, fmt.Errorf("no config for agent: %s", agentName)
	}

	// Get base config from the session and create a copy to override.
	baseConfig := s.GetGenerator().Config
	newConfig := baseConfig
	newConfig.ModelCode = agentGenConfig.ModelCode
	newConfig.Temperature = agentGenConfig.Temperature

	return rolePrompt, &newConfig, nil
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

	s.AddMessage(core.Message{Type: core.ToolCallMessage, Content: toolCallsJSON})

	results, err := tools.ExecuteToolCalls(toolCallsJSON, lastMsg.Content)
	if err != nil {
		resultContent := fmt.Sprintf("Error executing tool calls: %v", err)
		s.AddMessage(core.Message{Type: core.ToolResultMessage, Content: resultContent})
		return core.Event{Type: core.MessagesUpdated}
	}

	var agentRequests []AgentRequest
	var normalResults []tools.ToolResult

	for _, res := range results {
		var req AgentRequest
		if res.Error == nil && json.Unmarshal([]byte(res.Output), &req) == nil && req.AgentName != "" {
			agentRequests = append(agentRequests, req)
		} else {
			normalResults = append(normalResults, res)
		}
	}

	// Prioritize agent requests. If any are present, handle the first one and ignore others for this turn.
	if len(agentRequests) > 0 {
		req := agentRequests[0]

		rolePrompt, genConfig, err := m.getAgentContext(s, req.AgentName)
		if err != nil {
			s.AddMessage(core.Message{Type: core.CommandErrorResultMessage, Content: err.Error()})
			return core.Event{Type: core.MessagesUpdated}
		}

		// The prompt for the sub-agent is the conversation history plus the new task.
		// We create a temporary message to represent the task for prompt building.
		tempMessages := append(s.GetMessages(), core.Message{Type: core.UserMessage, Content: req.Prompt})
		prompt := BuildPrompt(rolePrompt, "", "", "", "", tempMessages)

		streamChan := make(chan string)
		ctx, cancel := context.WithCancel(context.Background())
		s.SetCancelGeneration(cancel)
		go s.GetGenerator().GenerateTask(ctx, prompt, nil, streamChan, genConfig)

		s.AddMessage(core.Message{Type: core.AIMessage, Content: ""}) // Placeholder for AI

		return core.Event{
			Type: core.GenerationStarted,
			Data: streamChan,
		}
	}

	// If no agent requests, process normal tool results.
	if len(normalResults) > 0 {
		var resultBuilder strings.Builder
		resultBuilder.WriteString("[\n")
		for i, res := range normalResults {
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
				resultObj["output"] = res.Output
			}

			jsonBytes, _ := json.MarshalIndent(resultObj, "  ", "  ")
			resultBuilder.WriteString("  ")
			resultBuilder.Write(jsonBytes)
			if i < len(normalResults)-1 {
				resultBuilder.WriteString(",\n")
			}
		}
		resultBuilder.WriteString("\n]")
		s.AddMessage(core.Message{Type: core.ToolResultMessage, Content: resultBuilder.String()})
		return s.StartGeneration()
	}
	return core.Event{Type: core.NoOp}
}

// StartGeneration begins a new AI generation task using the agent-specific logic.
func (m *AgentMode) StartGeneration(s SessionController) core.Event {
	_, genConfig, err := m.getAgentContext(s, "main_agent")
	if err != nil {
		s.AddMessage(core.Message{Type: core.CommandErrorResultMessage, Content: err.Error()})
		return core.Event{Type: core.MessagesUpdated}
	}
	return StartGeneration(s, genConfig)
}

// BuildPrompt constructs the prompt for agent mode.
func (m *AgentMode) BuildPrompt(systemInstructions, relatedDocuments, projectSourceCode string, messages []core.Message) string {
	// Agent mode does not use file-based context.
	return BuildPrompt(m.GetRolePrompt(), "", "", "", "", messages)
}
