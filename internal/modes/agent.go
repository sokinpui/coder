package modes

import (
	"coder/internal/config"
	"coder/internal/core"
	"coder/internal/tools"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"sort"
	"strings"
)

// AgentMode is the strategy for the agent/tool-using mode.
type AgentMode struct {
	activeAgent config.AgentName
}

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
	return agentRoles[m.activeAgent]
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

	// add rough directory information
	var dirInfoBuilder strings.Builder
	cwd, err := os.Getwd()
	if err != nil {
		cwd = "unknown"
	}
	dirInfoBuilder.WriteString(fmt.Sprintf("current working directory: %s\n", cwd))
	dirInfoBuilder.WriteString("shell command> ls\n")

	cmd := exec.Command("ls", "-l", "-h", "-F")
	lsOutput, err := cmd.CombinedOutput()
	if err != nil {
		dirInfoBuilder.WriteString(fmt.Sprintf("<error executing ls: %v>\n", err))
	} else {
		dirInfoBuilder.WriteString(string(lsOutput))
	}

	// Agent mode does not use file-based context.
	return BuildPrompt(
		RoleSection(rolePrompt, ""),
		DirectoryInformationSection(dirInfoBuilder.String()),
		ExternalToolsSection(toolDocs),
		ConversationHistorySection(messages),
	), nil
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

	results, _ := tools.ExecuteToolCalls(toolCallsJSON, lastMsg.Content)

	var agentReq *AgentRequest
	var agentToolCall *tools.ToolCall

	for _, res := range results {
		var currentAgentReq AgentRequest
		isAgentCall := res.Output != "" && json.Unmarshal([]byte(res.Output), &currentAgentReq) == nil && currentAgentReq.AgentName != ""

		if isAgentCall {
			agentReq = &currentAgentReq
			tc := res.ToolCall
			agentToolCall = &tc
		} else {
			var toolCallContent string
			if res.ToolCall.ToolName != "" {
				var argsParts []string
				for k, v := range res.ToolCall.Args {
					argsParts = append(argsParts, fmt.Sprintf("%s=%v", k, v))
				}
				sort.Strings(argsParts)
				toolCallContent = strings.TrimSpace(fmt.Sprintf("%s %s", res.ToolCall.ToolName, strings.Join(argsParts, " ")))
			} else if res.ToolCall.Shell != nil {
				var shellCommands []string
				switch shellCmd := res.ToolCall.Shell.(type) {
				case string:
					shellCommands = append(shellCommands, shellCmd)
				case []interface{}:
					for _, v := range shellCmd {
						if s, ok := v.(string); ok {
							shellCommands = append(shellCommands, s)
						}
					}
				}
				toolCallContent = fmt.Sprintf("shell %q", strings.Join(shellCommands, "; "))
			}
			s.AddMessage(core.Message{Type: core.ToolCallMessage, Content: toolCallContent})

			var resultContent string
			if res.Error != nil {
				resultContent = fmt.Sprintf("Error: %v", res.Error)
			} else {
				resultContent = res.Output
			}
			s.AddMessage(core.Message{Type: core.ToolResultMessage, Content: resultContent})
		}
	}

	if agentReq != nil {
		toolCallBytes, _ := json.Marshal(*agentToolCall)
		s.AddMessage(core.Message{Type: core.ToolCallMessage, Content: string(toolCallBytes)})

		agentName := config.AgentName(agentReq.AgentName)
		m.activeAgent = agentName

		s.AddMessage(core.Message{Type: core.UserMessage, Content: agentReq.Prompt})
	}

	return s.StartGeneration()
}

// StartGeneration begins a new AI generation task using the agent-specific logic.
func (m *AgentMode) StartGeneration(s SessionController) core.Event {
	genConfig, err := m.getAgentConfig(s, m.activeAgent)
	if err != nil {
		s.AddMessage(core.Message{Type: core.CommandErrorResultMessage, Content: err.Error()})
		return core.Event{Type: core.MessagesUpdated}
	}
	return StartGeneration(s, genConfig)
}

// BuildPrompt constructs the prompt for agent mode.
func (m *AgentMode) BuildPrompt(systemInstructions, relatedDocuments, projectSourceCode string, messages []core.Message) string {
	// Agent mode does not use file-based context.
	prompt, _ := m.buildAgentPrompt(messages, m.activeAgent)
	return prompt
}
