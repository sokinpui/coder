package coagent

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/sokinpui/coder/internal/config"
	coagentprompt "github.com/sokinpui/coder/internal/engine/coagent/prompt"
	"github.com/sokinpui/coder/internal/engine/coagent/skills"
	"github.com/sokinpui/coder/internal/engine/coagent/tools"
	"github.com/sokinpui/coder/internal/engine/generation"
	"github.com/sokinpui/coder/internal/types"
)

type AgentRuntime struct {
	Config            config.ModelConfig
	SecondaryConfig   config.ModelConfig
	Generator         *generation.Generator
	MaxToolIterations int
	Registry          *tools.Registry
	Permissions       *PermissionManager

	pendingMu sync.Mutex
	pending   map[string]chan types.ToolConfirmResponse
}

func NewAgentRuntime(cfg *config.Config, registry *tools.Registry) (*AgentRuntime, error) {
	if registry == nil {
		registry = tools.DefaultRegistry
	}
	gen, err := generation.New(cfg)
	if err != nil {
		return nil, err
	}
	maxIterations := cfg.Agent.MaxIterations
	if maxIterations <= 0 {
		maxIterations = 50
	}
	var permMap map[string]any
	if cfg != nil && cfg.Agent.Permission != nil {
		permMap = cfg.Agent.Permission
	}
	var toolsMap map[string]any
	if cfg != nil && cfg.Agent.Tools != nil {
		toolsMap = cfg.Agent.Tools
	}
	ar := &AgentRuntime{
		Config:            cfg.Agent.ModelConfig(),
		SecondaryConfig:   cfg.Agent.SecondaryModelConfig(),
		Generator:         gen,
		MaxToolIterations: maxIterations,
		Registry:          registry,
		Permissions:       NewPermissionManager(permMap, toolsMap),
		pending:           make(map[string]chan types.ToolConfirmResponse),
	}

	ar.Registry.Register(tools.NewWebFetchTool(gen, &ar.SecondaryConfig))

	return ar, nil
}

func (ar *AgentRuntime) registerConfirmation(callID string, ch chan types.ToolConfirmResponse) {
	ar.pendingMu.Lock()
	defer ar.pendingMu.Unlock()
	ar.pending[callID] = ch
}

func (ar *AgentRuntime) unregisterConfirmation(callID string) {
	ar.pendingMu.Lock()
	defer ar.pendingMu.Unlock()
	delete(ar.pending, callID)
}

func (ar *AgentRuntime) RespondConfirmation(callID string, resp types.ToolConfirmResponse) error {
	ar.pendingMu.Lock()
	ch, ok := ar.pending[callID]
	ar.pendingMu.Unlock()

	if !ok {
		return fmt.Errorf("no pending confirmation for call ID: %s", callID)
	}

	select {
	case ch <- resp:
		return nil
	default:
		return fmt.Errorf("confirmation already sent for call ID: %s", callID)
	}
}

func (ar *AgentRuntime) AgentLoop(ctx context.Context, systemInstruction string, messages []types.Message, streamChan chan<- AgentStreamChunk) {
	defer close(streamChan)

	currentMessages := append([]types.Message(nil), messages...)
	maxIterations := ar.MaxToolIterations
	if maxIterations <= 0 {
		maxIterations = 20
	}

	instructions := systemInstruction
	if instructions == "" {
		instructions = coagentprompt.Instructions
	}
	instructions = skills.AppendSkillsPrompt(instructions)

	for range maxIterations {
		if ctx.Err() != nil {
			return
		}

		select {
		case <-ctx.Done():
			return
		case streamChan <- AgentStreamChunk{State: "thinking"}:
		}

		instruction, chatMsgs := types.AssemblePrompt(currentMessages, instructions)
		toolDecls := ar.Registry.Declarations()

		var activeDecls []types.ToolDeclaration
		for _, decl := range toolDecls {
			if ar.Permissions == nil || ar.Permissions.IsToolEnabled(decl.Name) {
				activeDecls = append(activeDecls, decl)
			}
		}

		genChan := make(chan types.StreamChunk, 100)
		go ar.Generator.GenerateTask(ctx, instruction, chatMsgs, activeDecls, genChan, &ar.Config)

		var turnText strings.Builder
		var toolCalls []types.ToolCall
		var hasError bool

		for chunk := range genChan {
			if chunk.Error != nil {
				hasError = true
				select {
				case <-ctx.Done():
					return
				case streamChan <- AgentStreamChunk{Content: fmt.Sprintf("Error: %v", chunk.Error)}:
				}
				continue
			}
			if chunk.Content != "" {
				turnText.WriteString(chunk.Content)
				select {
				case <-ctx.Done():
					return
				case streamChan <- AgentStreamChunk{Content: chunk.Content}:
				}
			}
			if chunk.ReasoningContent != "" {
				select {
				case <-ctx.Done():
					return
				case streamChan <- AgentStreamChunk{ReasoningContent: chunk.ReasoningContent}:
				}
			}
			if chunk.ToolCall != nil {
				toolCalls = append(toolCalls, *chunk.ToolCall)
			}
		}

		if hasError || ctx.Err() != nil {
			return
		}

		if turnText.String() != "" {
			currentMessages = append(currentMessages, types.Message{
				Type:    types.AIMessage,
				Content: turnText.String(),
			})
		}

		if len(toolCalls) > 0 {
			currentMessages = append(currentMessages, types.Message{
				Type:      types.ToolCallMessage,
				ToolCalls: toolCalls,
			})
		}

		if len(toolCalls) == 0 {
			select {
			case <-ctx.Done():
			case streamChan <- AgentStreamChunk{Messages: currentMessages}:
			}
			return
		}

		for _, tc := range toolCalls {
			if ctx.Err() != nil {
				return
			}

			callInfo := ToolCallInfo{
				CallID:    tc.ID,
				Name:      tc.Name,
				Arguments: tc.Arguments,
			}
			streamChan <- AgentStreamChunk{ToolCall: &callInfo}

			if ar.Permissions != nil && !ar.Permissions.IsToolEnabled(tc.Name) {
				output := fmt.Sprintf("Permission denied: tool %s is disabled", tc.Name)
				resInfo := ToolResultInfo{
					CallID: tc.ID,
					Name:   tc.Name,
					Output: output,
				}
				streamChan <- AgentStreamChunk{ToolResult: &resInfo}
				currentMessages = append(currentMessages, types.Message{
					Type:       types.ToolResultMessage,
					Content:    output,
					ToolCallID: tc.ID,
				})
				continue
			}

			action := ActionAllow
			if ar.Permissions != nil {
				action = ar.Permissions.Check(tc.Name, tc.Arguments)
			}

			if action == ActionDeny {
				output := fmt.Sprintf("Permission denied: execution of %s is denied by configuration", tc.Name)
				resInfo := ToolResultInfo{
					CallID: tc.ID,
					Name:   tc.Name,
					Output: output,
				}
				streamChan <- AgentStreamChunk{ToolResult: &resInfo}
				currentMessages = append(currentMessages, types.Message{
					Type:       types.ToolResultMessage,
					Content:    output,
					ToolCallID: tc.ID,
				})
				continue
			}

			if action == ActionAsk {
				replyChan := make(chan types.ToolConfirmResponse, 1)
				ar.registerConfirmation(tc.ID, replyChan)
				confirmReq := &types.ToolConfirmRequest{
					CallID:    tc.ID,
					ToolName:  tc.Name,
					Arguments: tc.Arguments,
				}
				streamChan <- AgentStreamChunk{ToolConfirm: confirmReq}

				var resp types.ToolConfirmResponse
				select {
				case <-ctx.Done():
					ar.unregisterConfirmation(tc.ID)
					return
				case resp = <-replyChan:
					ar.unregisterConfirmation(tc.ID)
				}

				if resp.AlwaysAllow && ar.Permissions != nil {
					ar.Permissions.AlwaysAllow(tc.Name)
				}

				if !resp.Approved {
					output := fmt.Sprintf("Tool execution rejected by user: permission denied for %s", tc.Name)
					resInfo := ToolResultInfo{
						CallID: tc.ID,
						Name:   tc.Name,
						Output: output,
					}
					streamChan <- AgentStreamChunk{ToolResult: &resInfo}
					currentMessages = append(currentMessages, types.Message{
						Type:       types.ToolResultMessage,
						Content:    output,
						ToolCallID: tc.ID,
					})
					continue
				}
			}

			output, images, err := ar.Registry.Execute(ctx, tc.Name, tc.Arguments)
			if err != nil {
				output = fmt.Sprintf("Error: %v", err)
			}

			resInfo := ToolResultInfo{
				CallID: tc.ID,
				Name:   tc.Name,
				Output: output,
			}
			streamChan <- AgentStreamChunk{ToolResult: &resInfo}

			currentMessages = append(currentMessages, types.Message{
				Type:       types.ToolResultMessage,
				Content:    output,
				ToolCallID: tc.ID,
			})
			if len(images) > 0 {
				currentMessages = append(currentMessages, images...)
			}
		}

		select {
		case <-ctx.Done():
			return
		case streamChan <- AgentStreamChunk{Messages: currentMessages}:
		}
	}

	select {
	case <-ctx.Done():
	case streamChan <- AgentStreamChunk{Messages: currentMessages}:
	}
}
