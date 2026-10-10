package coagent

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/sokinpui/coder/internal/config"
	"github.com/sokinpui/coder/internal/engine"
	"github.com/sokinpui/coder/internal/engine/coagent/tools"
	"github.com/sokinpui/coder/internal/engine/commands"
	"github.com/sokinpui/coder/internal/engine/history"
	"github.com/sokinpui/coder/internal/engine/token"
	"github.com/sokinpui/coder/internal/types"
)

var _ engine.EngineSession = (*Session)(nil)
var _ engine.ToolApprover = (*Session)(nil)
var _ engine.AgentFileReader = (*Session)(nil)

func (s *Session) GetMode() string {
	return engine.ModeAgent
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
		engine.CapShell |
		engine.CapReasoningSwitch
}

func (s *Session) GetMessages() []types.Message {
	s.mu.RLock()
	defer s.mu.RUnlock()
	msgs := make([]types.Message, len(s.Messages))
	copy(msgs, s.Messages)
	return msgs
}

func (s *Session) DeleteMessages(indices []int) {
	if len(indices) == 0 {
		return
	}
	s.mu.Lock()
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
	s.mu.Unlock()
}

func (s *Session) EditMessage(index int, newContent string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
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
	count := token.CountTokens(s.GetPrompt())
	if s.Runtime != nil && s.Runtime.Registry != nil {
		decls := s.Runtime.Registry.Declarations()
		for _, decl := range decls {
			if s.Runtime.Permissions == nil || s.Runtime.Permissions.IsToolEnabled(decl.Name) {
				if b, err := json.Marshal(decl); err == nil {
					count += token.CountTokens([]types.Message{{
						Type:    types.InstructionMessage,
						Content: string(b),
					}})
				}
			}
		}
	}
	return count
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
		s.mu.Lock()
		s.Messages = append(s.Messages, types.Message{Type: types.ShellCmdMessage, Content: input})
		s.mu.Unlock()
		return out, success
	}
	if out.IsAgentFileRead {
		return out, success
	}
	msgType := types.CommandMessage
	s.mu.Lock()
	s.Messages = append(s.Messages, types.Message{Type: msgType, Content: input})
	if success {
		s.Messages = append(s.Messages, types.Message{Type: types.CommandResultMessage, Content: out.Payload})
	} else {
		s.Messages = append(s.Messages, types.Message{Type: types.CommandErrorResultMessage, Content: out.Payload})
	}
	s.mu.Unlock()
	return out, success
}

func (s *Session) CreateNew(mode string) (engine.EngineSession, error) {
	if mode == "" {
		mode = engine.ModeAgent
	}
	return engine.NewSession(s.Config, mode, s.Instruction, nil)
}

func (s *Session) Branch(endMessageIndex int) (engine.EngineSession, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
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
	s.mu.Lock()
	if messageIndex < 0 || messageIndex >= len(s.Messages) {
		s.mu.Unlock()
		return nil, fmt.Errorf("invalid index for regeneration: %d", messageIndex)
	}
	s.Messages = s.Messages[:messageIndex+1]
	s.mu.Unlock()
	return s.runAgentLoop(context.Background())
}

func (s *Session) Submit(ctx context.Context, input string) (<-chan types.SessionEvent, error) {
	s.mu.Lock()
	s.Messages = append(s.Messages, types.Message{
		Type:    types.UserMessage,
		Content: input,
	})
	s.mu.Unlock()
	return s.runAgentLoop(ctx)
}

func (s *Session) runAgentLoop(ctx context.Context) (<-chan types.SessionEvent, error) {
	runCtx, cancel := context.WithCancel(ctx)
	s.cancelFunc = cancel
	s.isStreaming = true

	chunkChan := make(chan AgentStreamChunk, 100)
	eventChan := make(chan types.SessionEvent, 100)

	prepared := s.PrepareMessages()
	instr := s.getEffectiveInstruction()
	go func() {
		s.Runtime.AgentLoop(runCtx, instr, prepared, chunkChan)
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
					s.mu.Lock()
					s.Messages = chunk.Messages
					s.mu.Unlock()
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
				if chunk.ToolCallDelta != nil {
					s.mu.Lock()
					delta := chunk.ToolCallDelta
					msgs := s.Messages
					var targetMsg *types.Message
					if len(msgs) > 0 && msgs[len(msgs)-1].Type == types.ToolCallMessage {
						targetMsg = &s.Messages[len(msgs)-1]
					} else {
						s.Messages = append(s.Messages, types.Message{
							Type: types.ToolCallMessage,
						})
						targetMsg = &s.Messages[len(s.Messages)-1]
					}
					for len(targetMsg.ToolCalls) <= delta.Index {
						targetMsg.ToolCalls = append(targetMsg.ToolCalls, types.ToolCall{})
					}
					if delta.ID != "" {
						targetMsg.ToolCalls[delta.Index].ID = delta.ID
					}
					if delta.Name != "" {
						targetMsg.ToolCalls[delta.Index].Name = delta.Name
					}
					if delta.ArgumentsDelta != "" {
						targetMsg.ToolCalls[delta.Index].Arguments += delta.ArgumentsDelta
					}
					currTC := targetMsg.ToolCalls[delta.Index]
					s.mu.Unlock()
					eventChan <- types.SessionEvent{
						Kind:          types.EventToolCallDelta,
						ToolCallIndex: delta.Index,
						ToolCallID:    currTC.ID,
						ToolName:      currTC.Name,
						ToolArguments: currTC.Arguments,
					}
				}
				if chunk.ToolCall != nil {
					s.mu.Lock()
					msgs := s.Messages
					var found bool
					if len(msgs) > 0 && msgs[len(msgs)-1].Type == types.ToolCallMessage {
						tcMsg := &s.Messages[len(msgs)-1]
						for i, existing := range tcMsg.ToolCalls {
							if (existing.ID != "" && existing.ID == chunk.ToolCall.CallID) ||
								(existing.Name != "" && existing.Name == chunk.ToolCall.Name) {
								tcMsg.ToolCalls[i] = types.ToolCall{
									ID:        chunk.ToolCall.CallID,
									Name:      chunk.ToolCall.Name,
									Arguments: chunk.ToolCall.Arguments,
								}
								found = true
								break
							}
						}
					}
					if !found {
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
					}
					s.mu.Unlock()
					eventChan <- types.SessionEvent{
						Kind:          types.EventToolCall,
						ToolCallID:    chunk.ToolCall.CallID,
						ToolName:      chunk.ToolCall.Name,
						ToolArguments: chunk.ToolCall.Arguments,
					}
				}
				if chunk.ToolConfirm != nil {
					eventChan <- types.SessionEvent{
						Kind:    types.EventToolConfirm,
						Confirm: chunk.ToolConfirm,
					}
				}
				if chunk.ToolOutputChunk != "" && chunk.ToolCall != nil {
					s.mu.Lock()
					msgs := s.Messages
					var found bool
					for i := len(msgs) - 1; i >= 0; i-- {
						if msgs[i].Type == types.ToolResultMessage && msgs[i].ToolCallID == chunk.ToolCall.CallID {
							s.Messages[i].Content += chunk.ToolOutputChunk
							found = true
							break
						}
					}
					if !found {
						s.Messages = append(s.Messages, types.Message{
							Type:       types.ToolResultMessage,
							Content:    chunk.ToolOutputChunk,
							ToolCallID: chunk.ToolCall.CallID,
						})
					}
					s.mu.Unlock()
					eventChan <- types.SessionEvent{
						Kind:            types.EventToolOutputChunk,
						ToolCallID:      chunk.ToolCall.CallID,
						ToolName:        chunk.ToolCall.Name,
						ToolOutputChunk: chunk.ToolOutputChunk,
					}
				}
				if chunk.ToolResult != nil {
					s.mu.Lock()
					msgs := s.Messages
					var found bool
					for i := len(msgs) - 1; i >= 0; i-- {
						if msgs[i].Type == types.ToolResultMessage && msgs[i].ToolCallID == chunk.ToolResult.CallID {
							s.Messages[i].Content = chunk.ToolResult.Output
							found = true
							break
						}
					}
					if !found {
						s.Messages = append(s.Messages, types.Message{
							Type:       types.ToolResultMessage,
							Content:    chunk.ToolResult.Output,
							ToolCallID: chunk.ToolResult.CallID,
						})
					}
					if len(chunk.ToolResult.Images) > 0 {
						s.Messages = append(s.Messages, chunk.ToolResult.Images...)
					}
					s.mu.Unlock()
					eventChan <- types.SessionEvent{
						Kind:       types.EventToolResult,
						ToolCallID: chunk.ToolResult.CallID,
						ToolName:   chunk.ToolResult.Name,
						ToolOutput: chunk.ToolResult.Output,
					}
				}
				if chunk.Content != "" {
					s.mu.Lock()
					msgs := s.Messages
					if len(msgs) > 0 && msgs[len(msgs)-1].Type == types.AIMessage {
						s.Messages[len(msgs)-1].Content += chunk.Content
					} else {
						s.Messages = append(s.Messages, types.Message{
							Type:    types.AIMessage,
							Content: chunk.Content,
						})
					}
					s.mu.Unlock()
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
	s.Runtime.SecondaryConfig = cfg.Agent.SecondaryModelConfig()
	s.Runtime.Generator.TitleModel = cfg.Title.ModelCode
	s.Runtime.Generator.BaseURL = cfg.Server.URL
	s.Runtime.Generator.Protocol = cfg.Server.Protocol
	s.Runtime.Generator.APIKey = cfg.Server.APIKey
	if s.Runtime.Permissions != nil {
		s.Runtime.Permissions.SetConfig(cfg.Agent.Permission)
		s.Runtime.Permissions.SetToolsConfig(cfg.Agent.Tools)
	}
	if s.Runtime.Registry != nil {
		s.Runtime.Registry.Register(tools.NewWebFetchTool(s.Runtime.Generator, &s.Runtime.SecondaryConfig))
	}
	return nil
}
