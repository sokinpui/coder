package coagent

import (
	"context"
	"fmt"

	"github.com/sokinpui/coder/internal/config"
	"github.com/sokinpui/coder/internal/engine"
	"github.com/sokinpui/coder/internal/engine/commands"
	"github.com/sokinpui/coder/internal/engine/history"
	"github.com/sokinpui/coder/internal/engine/token"
	"github.com/sokinpui/coder/internal/types"
)

var _ engine.EngineSession = (*Session)(nil)

func (s *Session) GetMode() string {
	return ModeCoAgent
}

func (s *Session) SetTitle(title string) {
	s.Title = title
	s.TitleGenerated = true
}

func (s *Session) IsTitleGenerated() bool {
	return s.TitleGenerated
}

func (s *Session) Capabilities() engine.Capability {
	return engine.CapToolToggle |
		engine.CapToolLoop |
		engine.CapModelSwitch |
		engine.CapBranch |
		engine.CapRegenerate |
		engine.CapShell
}

func (s *Session) GetMessages() []types.Message {
	return s.Messages
}

func (s *Session) DeleteMessages(indices []int) {
	if len(indices) == 0 {
		return
	}
	toDelete := make(map[int]struct{}, len(indices))
	for _, idx := range indices {
		if idx >= 0 && idx < len(s.Messages) {
			toDelete[idx] = struct{}{}
		}
	}
	var remaining []types.Message
	for i, msg := range s.Messages {
		if _, ok := toDelete[i]; !ok {
			remaining = append(remaining, msg)
		}
	}
	s.Messages = remaining
}

func (s *Session) EditMessage(index int, newContent string) error {
	if index < 0 || index >= len(s.Messages) {
		return fmt.Errorf("index out of bounds: %d", index)
	}
	if s.Messages[index].Type != types.UserMessage {
		return fmt.Errorf("can only edit user messages, got type %v", s.Messages[index].Type)
	}
	s.Messages[index].Content = newContent
	return nil
}

func (s *Session) TokenCount() int {
	return token.CountTokens(s.GetPrompt())
}

func (s *Session) Cancel() {
	if s.cancelFunc != nil {
		s.cancelFunc()
	}
	s.isStreaming = false
}

func (s *Session) IsStreaming() bool {
	return s.isStreaming
}

func (s *Session) GetSupportedCommands() []string {
	return commands.GetCommandsForSession(s)
}

func (s *Session) GetCommandDescriptions() map[string]string {
	return commands.GetCommandDescriptions()
}

func (s *Session) GetCommandSuggestions(cmdName, prefix string) []string {
	return commands.GetCommandArgumentSuggestions(cmdName, s, prefix)
}

func (s *Session) ExecuteCommand(input string) (commands.CommandOutput, bool) {
	out, _, success := commands.ProcessCommand(input, s)
	if out.Type == types.NewSessionStarted || out.Type == types.Quit || out.Type == types.NoOp {
		return out, success
	}
	if out.Type == types.ShellExecutionStarted {
		s.Messages = append(s.Messages, types.Message{Type: types.ShellCmdMessage, Content: input})
		return out, success
	}
	msgType := types.CommandMessage
	s.Messages = append(s.Messages, types.Message{Type: msgType, Content: input})
	if success {
		s.Messages = append(s.Messages, types.Message{Type: types.CommandResultMessage, Content: out.Payload})
	} else {
		s.Messages = append(s.Messages, types.Message{Type: types.CommandErrorResultMessage, Content: out.Payload})
	}
	return out, success
}

func (s *Session) CreateNew(mode string) (engine.EngineSession, error) {
	return NewSession(s.Config)
}

func (s *Session) Branch(endMessageIndex int) (engine.EngineSession, error) {
	if endMessageIndex < 0 || endMessageIndex >= len(s.Messages) {
		return nil, fmt.Errorf("invalid index for branching: %d", endMessageIndex)
	}
	newSess, err := NewSession(s.Config)
	if err != nil {
		return nil, err
	}
	newSess.Messages = append([]types.Message(nil), s.Messages[:endMessageIndex+1]...)
	newSess.Title = fmt.Sprintf("branch of %s", s.Title)
	newSess.TitleGenerated = true
	newSess.Instruction = s.Instruction
	return newSess, nil
}

func (s *Session) Regenerate(messageIndex int) (<-chan types.SessionEvent, error) {
	if messageIndex < 0 || messageIndex >= len(s.Messages) {
		return nil, fmt.Errorf("invalid index for regeneration: %d", messageIndex)
	}
	s.Messages = s.Messages[:messageIndex+1]
	return s.runAgentLoop(context.Background())
}

func (s *Session) Submit(ctx context.Context, input string) (<-chan types.SessionEvent, error) {
	s.Messages = append(s.Messages, types.Message{
		Type:    types.UserMessage,
		Content: input,
	})
	return s.runAgentLoop(ctx)
}

func (s *Session) runAgentLoop(ctx context.Context) (<-chan types.SessionEvent, error) {
	runCtx, cancel := context.WithCancel(ctx)
	s.cancelFunc = cancel
	s.isStreaming = true

	chunkChan := make(chan AgentStreamChunk, 100)
	eventChan := make(chan types.SessionEvent, 100)

	prepared := s.PrepareMessages()
	go func() {
		s.Runtime.AgentLoop(runCtx, s.Instruction, prepared, chunkChan)
	}()

	go func() {
		defer close(eventChan)

		for {
			select {
			case <-runCtx.Done():
				eventChan <- types.SessionEvent{
					Kind:  types.EventError,
					Error: runCtx.Err(),
				}
				return
			case chunk, ok := <-chunkChan:
				if !ok {
					eventChan <- types.SessionEvent{
						Kind:     types.EventComplete,
						Messages: s.Messages,
					}
					return
				}

				if len(chunk.Messages) > 0 {
					s.Messages = chunk.Messages
				}

				if chunk.State != "" {
					eventChan <- types.SessionEvent{
						Kind:    types.EventThinking,
						Content: chunk.State,
					}
				}
				if chunk.ReasoningContent != "" {
					eventChan <- types.SessionEvent{
						Kind:             types.EventThinking,
						ReasoningContent: chunk.ReasoningContent,
					}
				}
				if chunk.ToolCall != nil {
					s.Messages = append(s.Messages, types.Message{
						Type: types.ToolCallMessage,
						ToolCalls: []types.ToolCall{
							{
								ID:        chunk.ToolCall.CallID,
								Name:      chunk.ToolCall.Name,
								Arguments: chunk.ToolCall.Arguments,
							},
						},
					})
					eventChan <- types.SessionEvent{
						Kind:          types.EventToolCall,
						ToolCallID:    chunk.ToolCall.CallID,
						ToolName:      chunk.ToolCall.Name,
						ToolArguments: chunk.ToolCall.Arguments,
					}
				}
				if chunk.ToolResult != nil {
					s.Messages = append(s.Messages, types.Message{
						Type:       types.ToolResultMessage,
						Content:    chunk.ToolResult.Output,
						ToolCallID: chunk.ToolResult.CallID,
					})
					eventChan <- types.SessionEvent{
						Kind:       types.EventToolResult,
						ToolCallID: chunk.ToolResult.CallID,
						ToolName:   chunk.ToolResult.Name,
						ToolOutput: chunk.ToolResult.Output,
					}
				}
				if chunk.Content != "" {
					msgs := s.Messages
					if len(msgs) > 0 && msgs[len(msgs)-1].Type == types.AIMessage {
						s.Messages[len(msgs)-1].Content += chunk.Content
					} else {
						s.Messages = append(s.Messages, types.Message{
							Type:    types.AIMessage,
							Content: chunk.Content,
						})
					}
					eventChan <- types.SessionEvent{
						Kind:    types.EventChunk,
						Content: chunk.Content,
					}
				}
			}
		}
	}()

	return eventChan, nil
}

func (s *Session) GetHistoryManager() *history.Manager {
	return s.HistoryManager
}

func (s *Session) ReloadConfig() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	s.Config = cfg
	s.Runtime.Config = cfg.Agent.ModelConfig()
	s.Runtime.Generator.Config = cfg.Agent.ModelConfig()
	s.Runtime.Generator.TitleModel = cfg.Title.ModelCode
	s.Runtime.Generator.BaseURL = cfg.Server.URL
	s.Runtime.Generator.Protocol = cfg.Server.Protocol
	s.Runtime.Generator.APIKey = cfg.Server.APIKey
	return nil
}
