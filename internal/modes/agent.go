package modes

import (
	"coder/internal/agent"
	"coder/internal/core"
	"coder/internal/tools"
	"encoding/json"
	"fmt"
	"strings"
)

// AgentMode is the strategy for the agent/tool-using mode.
type AgentMode struct{}

// GetRolePrompt returns the agent role.
func (m *AgentMode) GetRolePrompt() string {
	return core.AgentRole
}

// LoadContext does not load any context for agent mode.
func (m *AgentMode) LoadContext() (string, string, string, error) {
	return "", "", "", nil
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

	toolCallsJSON, _ := agent.ExtractToolCalls(lastMsg.Content)
	if toolCallsJSON == "" {
		return core.Event{Type: core.NoOp}
	}

	s.AddMessage(core.Message{Type: core.ToolCallMessage, Content: toolCallsJSON})

	results, err := tools.ExecuteToolCalls(toolCallsJSON, lastMsg.Content)
	if err != nil {
		// This is likely a JSON parsing error from ExecuteToolCalls.
		resultContent := fmt.Sprintf("Error parsing tool calls JSON: %v", err)
		s.AddMessage(core.Message{Type: core.ToolResultMessage, Content: resultContent})
		return core.Event{Type: core.MessagesUpdated}
	}

	var resultBuilder strings.Builder
	resultBuilder.WriteString("[\n")
	for i, res := range results {
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
		if i < len(results)-1 {
			resultBuilder.WriteString(",\n")
		}
	}
	resultBuilder.WriteString("\n]")
	s.AddMessage(core.Message{Type: core.ToolResultMessage, Content: resultBuilder.String()})
	return s.StartGeneration()
}
