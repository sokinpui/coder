package coder

import (
	"context"
	"fmt"

	"github.com/sokinpui/coder/internal/engine"
	"github.com/sokinpui/coder/internal/engine/commands"
	"github.com/sokinpui/coder/internal/engine/token"
	"github.com/sokinpui/coder/internal/types"
)

var _ engine.EngineSession = (*Session)(nil)

func (s *Session) GetID() string {
	return s.ID
}

func (s *Session) Capabilities() engine.Capability {
	return engine.CapContextFiles |
		engine.CapITF |
		engine.CapModelSwitch |
		engine.CapBranch |
		engine.CapRegenerate |
		engine.CapShell |
		engine.CapDocumentContext
}

func (s *Session) TokenCount() int {
	return token.CountTokens(s.GetPrompt())
}

func (s *Session) Cancel() {
	s.CancelGeneration()
}

func (s *Session) GetSupportedCommands() []string {
	return commands.GetCommands()
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
		s.messages = append(s.messages, types.Message{Type: types.ShellCmdMessage, Content: input})
		return out, success
	}

	msgType := types.CommandMessage
	if out.IsFileApply {
		msgType = types.FileApplyCmdMessage
	} else if out.IsFileApplyUndo {
		msgType = types.FileApplyUndoCmdMessage
	} else if success && out.IsContext {
		msgType = types.ContextCmdMessage
	} else if out.IsShell {
		msgType = types.ShellCmdMessage
	}
	s.messages = append(s.messages, types.Message{Type: msgType, Content: input})

	if success {
		resType := types.CommandResultMessage
		if out.IsFileApply {
			resType = types.FileApplyCmdResultMessage
		} else if out.IsFileApplyUndo {
			resType = types.FileApplyUndoCmdResultMessage
		} else if out.IsShell {
			resType = types.ShellCmdResultMessage
		} else if out.IsContext {
			resType = types.ContextCmdResultMessage
		}
		s.messages = append(s.messages, types.Message{Type: resType, Content: out.Payload})
	} else {
		errType := types.CommandErrorResultMessage
		if out.IsFileApply {
			errType = types.FileApplyCmdErrorMessage
		} else if out.IsFileApplyUndo {
			errType = types.FileApplyUndoCmdErrorMessage
		}
		s.messages = append(s.messages, types.Message{Type: errType, Content: out.Payload})
	}
	return out, success
}

func (s *Session) CreateNew(mode string) (engine.EngineSession, error) {
	if mode == "" {
		mode = s.mode
	}
	return engine.NewSession(s.config, mode, s.instruction, s.contextFiles)
}

func (s *Session) Submit(ctx context.Context, input string) (<-chan types.SessionEvent, error) {
	s.messages = append(s.messages, types.Message{Type: types.UserMessage, Content: input})
	ev := s.StartGeneration()
	return s.bridgeStreamEvent(ctx, ev), nil
}

func (s *Session) Regenerate(messageIndex int) (<-chan types.SessionEvent, error) {
	ev := s.RegenerateFrom(messageIndex)
	return s.bridgeStreamEvent(context.Background(), ev), nil
}

func (s *Session) bridgeStreamEvent(ctx context.Context, ev types.Event) <-chan types.SessionEvent {
	eventChan := make(chan types.SessionEvent, 100)
	streamChan, ok := ev.Data.(chan types.StreamChunk)
	if !ok {
		close(eventChan)
		return eventChan
	}

	go func() {
		defer close(eventChan)
		for {
			select {
			case <-ctx.Done():
				s.Cancel()
				return
			case chunk, ok := <-streamChan:
				if !ok {
					eventChan <- types.SessionEvent{
						Kind:     types.EventComplete,
						Messages: s.GetMessages(),
					}
					return
				}
				if chunk.Error != nil {
					errorContent := fmt.Sprintf("\n**Error:**\n```\n%v\n```\n", chunk.Error)
					msgs := s.GetMessages()
					if len(msgs) > 0 && msgs[len(msgs)-1].Type == types.AIMessage && msgs[len(msgs)-1].Content == "" {
						s.ReplaceLastMessage(types.Message{Type: types.CommandErrorResultMessage, Content: errorContent})
					} else {
						s.AddMessages(types.Message{Type: types.CommandErrorResultMessage, Content: errorContent})
					}
					eventChan <- types.SessionEvent{
						Kind:  types.EventError,
						Error: chunk.Error,
					}
					return
				}
				if chunk.ReasoningContent != "" {
					eventChan <- types.SessionEvent{
						Kind:             types.EventThinking,
						ReasoningContent: chunk.ReasoningContent,
					}
				}
				if chunk.Content != "" {
					messages := s.GetMessages()
					aiIdx := len(messages) - 1
					if len(messages) > 0 && messages[aiIdx].Type == types.AIMessage {
						s.messages[aiIdx].Content += chunk.Content
					} else {
						s.AddMessages(types.Message{Type: types.AIMessage, Content: chunk.Content})
					}
					eventChan <- types.SessionEvent{
						Kind:    types.EventChunk,
						Content: chunk.Content,
					}
				}
				if chunk.ToolCall != nil {
					eventChan <- types.SessionEvent{
						Kind:          types.EventToolCall,
						ToolCallID:    chunk.ToolCall.ID,
						ToolName:      chunk.ToolCall.Name,
						ToolArguments: chunk.ToolCall.Arguments,
					}
				}
			}
		}
	}()
	return eventChan
}
