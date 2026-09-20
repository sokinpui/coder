package coagent

import (
	"context"
	"fmt"
	"strings"

	"github.com/sokinpui/coder/internal/config"
	coagentprompt "github.com/sokinpui/coder/internal/engine/coagent/prompt"
	"github.com/sokinpui/coder/internal/engine/generation"
	"github.com/sokinpui/coder/internal/types"
)

type AgentRuntime struct {
	Config            config.Generation
	Generator         *generation.Generator
	MaxToolIterations int
	Registry          *Registry
}

func NewAgentRuntime(cfg *config.Config, registry *Registry) (*AgentRuntime, error) {
	if registry == nil {
		registry = DefaultRegistry
	}
	gen, err := generation.New(cfg)
	if err != nil {
		return nil, err
	}
	return &AgentRuntime{
		Config:            cfg.Generation,
		Generator:         gen,
		MaxToolIterations: 20,
		Registry:          registry,
	}, nil
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

	for range maxIterations {
		if ctx.Err() != nil {
			return
		}

		instruction, chatMsgs := types.AssemblePrompt(currentMessages, instructions)
		toolDecls := ar.Registry.Declarations()

		genChan := make(chan types.StreamChunk, 100)
		go ar.Generator.GenerateTask(ctx, instruction, chatMsgs, toolDecls, genChan, &ar.Config)

		var turnText string
		var toolCalls []types.ToolCall
		var hasError bool

		for chunk := range genChan {
			if strings.HasPrefix(chunk.Content, "Error:") {
				hasError = true
			}
			if chunk.Content != "" {
				turnText += chunk.Content
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

		if turnText != "" || len(toolCalls) > 0 {
			currentMessages = append(currentMessages, types.Message{
				Type:      types.AIMessage,
				Content:   turnText,
				ToolCalls: toolCalls,
			})
		}

		if len(toolCalls) == 0 {
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

			output, err := ar.Registry.Execute(ctx, tc.Name, tc.Arguments)
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
		}
	}
}
