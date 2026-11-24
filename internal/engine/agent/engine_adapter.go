package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/sokinpui/coder/internal/config"
	"github.com/sokinpui/coder/internal/engine"
	"github.com/sokinpui/coder/internal/engine/agent/tools"
	"github.com/sokinpui/coder/internal/engine/commands"
	"github.com/sokinpui/coder/internal/engine/history"
	"github.com/sokinpui/coder/internal/engine/pdf"
	"github.com/sokinpui/coder/internal/engine/token"
	"github.com/sokinpui/coder/internal/project"
	"github.com/sokinpui/coder/internal/types"
)

var _ engine.EngineSession = (*Session)(nil)
var _ engine.ToolApprover = (*Session)(nil)
var _ engine.AgentFileReader = (*Session)(nil)
var _ engine.SideQuestioner = (*Session)(nil)

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
		engine.CapCompact |
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
	defer s.mu.Unlock()

	expanded := types.ExpandGroupIndices(s.Messages, indices)
	if len(expanded) == 0 {
		return
	}

	repoRoot := project.Root()
	toDelete := make(map[int]struct{}, len(expanded))
	for _, idx := range expanded {
		if idx >= 0 && idx < len(s.Messages) {
			toDelete[idx] = struct{}{}
			msg := s.Messages[idx]
			if msg.Type == types.PDFMessage && repoRoot != "" {
				if meta, ok := types.ParsePDFMetadata(msg.Content); ok && meta.CacheDir != "" {
					pdf.EvictCache(meta.CacheDir)
					_ = os.RemoveAll(filepath.Join(repoRoot, meta.CacheDir))
				}
			}
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
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.cancelFunc != nil {
		s.cancelFunc()
	}
	s.isStreaming = false
	s.sealOpenToolCallsLocked("Interrupted by user.")
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
	cmdGroupID := fmt.Sprintf("cmd_%d", time.Now().UnixNano())
	msgType := types.CommandMessage
	s.mu.Lock()
	s.Messages = append(s.Messages, types.Message{GroupID: cmdGroupID, Type: msgType, Content: input})
	if success {
		s.Messages = append(s.Messages, types.Message{GroupID: cmdGroupID, Type: types.CommandResultMessage, Content: out.Payload})
	} else {
		s.Messages = append(s.Messages, types.Message{GroupID: cmdGroupID, Type: types.CommandErrorResultMessage, Content: out.Payload})
	}
	if len(out.Messages) > 0 {
		for _, extraMsg := range out.Messages {
			extraMsg.GroupID = cmdGroupID
			s.Messages = append(s.Messages, extraMsg)
		}
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
	runCtx = pdf.WithSessionID(runCtx, s.ID)
	if s.Config != nil {
		runCtx = pdf.WithMode(runCtx, s.Config.PDFMode)
	}
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

		var currentToolGroupID string
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
					if runCtx.Err() != nil {
						s.mu.Unlock()
						return
					}
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
					if runCtx.Err() != nil {
						s.mu.Unlock()
						return
					}
					delta := chunk.ToolCallDelta
					msgs := s.Messages
					var targetMsg *types.Message
					if len(msgs) > 0 && msgs[len(msgs)-1].Type == types.ToolCallMessage {
						targetMsg = &s.Messages[len(msgs)-1]
						currentToolGroupID = targetMsg.GroupID
					} else {
						currentToolGroupID = fmt.Sprintf("tc_%d", time.Now().UnixNano())
						s.Messages = append(s.Messages, types.Message{
							GroupID: currentToolGroupID,
							Type:    types.ToolCallMessage,
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
					if runCtx.Err() != nil {
						s.mu.Unlock()
						return
					}
					msgs := s.Messages
					var found bool
					if len(msgs) > 0 && msgs[len(msgs)-1].Type == types.ToolCallMessage {
						tcMsg := &s.Messages[len(msgs)-1]
						currentToolGroupID = tcMsg.GroupID
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
						currentToolGroupID = fmt.Sprintf("tc_%d", time.Now().UnixNano())
						s.Messages = append(s.Messages, types.Message{
							GroupID: currentToolGroupID,
							Type:    types.ToolCallMessage,
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
					if runCtx.Err() != nil {
						s.mu.Unlock()
						return
					}
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
						s.Messages = appendToolResultInGroup(s.Messages, currentToolGroupID, types.Message{
							GroupID:    currentToolGroupID,
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
					if runCtx.Err() != nil {
						s.mu.Unlock()
						return
					}
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
						s.Messages = appendToolResultInGroup(s.Messages, currentToolGroupID, types.Message{
							GroupID:    currentToolGroupID,
							Type:       types.ToolResultMessage,
							Content:    chunk.ToolResult.Output,
							ToolCallID: chunk.ToolResult.CallID,
						})
					}
					if len(chunk.ToolResult.Images) > 0 {
						s.Messages = append(s.Messages, chunk.ToolResult.Images...)
						for imgIdx := len(s.Messages) - len(chunk.ToolResult.Images); imgIdx < len(s.Messages); imgIdx++ {
							s.Messages[imgIdx].GroupID = currentToolGroupID
						}
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
					if runCtx.Err() != nil {
						s.mu.Unlock()
						return
					}
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

func (s *Session) SnapshotMessages() ([]types.Message, error) {
	prepared := s.PrepareMessages()
	return sealOpenToolCalls(prepared, "Tool call still running in main session."), nil
}

func (s *Session) AskSideQuestion(ctx context.Context, messages []types.Message) (<-chan types.StreamChunk, error) {
	if s.Runtime == nil || s.Runtime.Generator == nil {
		return nil, fmt.Errorf("runtime not initialized")
	}

	pdfMode := ""
	if s.Config != nil {
		pdfMode = s.Config.PDFMode
	}
	instr := s.getEffectiveInstruction()
	instruction, chatMsgs := types.AssemblePrompt(messages, instr, pdfMode)

	streamChan := make(chan types.StreamChunk, 100)
	cfg := s.Runtime.Config
	go s.Runtime.Generator.GenerateTask(ctx, instruction, chatMsgs, nil, streamChan, &cfg)
	return streamChan, nil
}

func (s *Session) sealOpenToolCallsLocked(reason string) {
	s.Messages = sealOpenToolCalls(s.Messages, reason)
}

func sealOpenToolCalls(msgs []types.Message, reason string) []types.Message {
	if len(msgs) == 0 {
		return nil
	}

	out := make([]types.Message, len(msgs))
	copy(out, msgs)
	for i := range out {
		if len(out[i].ToolCalls) > 0 {
			tcCopy := make([]types.ToolCall, len(out[i].ToolCalls))
			copy(tcCopy, out[i].ToolCalls)
			out[i].ToolCalls = tcCopy
		}
	}

	answered := make(map[string]struct{})
	for _, msg := range out {
		if msg.Type == types.ToolResultMessage && msg.ToolCallID != "" {
			answered[msg.ToolCallID] = struct{}{}
		}
	}

	for i := len(out) - 1; i >= 0; i-- {
		if i >= len(out) {
			continue
		}
		msg := out[i]
		if msg.Type == types.UserMessage || msg.Type == types.InterruptedMessage {
			break
		}
		if msg.Type != types.ToolCallMessage {
			continue
		}

		var unansweredIndices []int
		hasAnyResult := false
		hasIncompleteCall := false
		for idx, tc := range msg.ToolCalls {
			if tc.ID != "" {
				if _, ok := answered[tc.ID]; ok {
					hasAnyResult = true
					continue
				}
			}
			unansweredIndices = append(unansweredIndices, idx)
			if strings.TrimSpace(tc.Name) == "" || strings.TrimSpace(tc.ID) == "" || !json.Valid([]byte(tc.Arguments)) {
				hasIncompleteCall = true
			}
		}

		if len(unansweredIndices) == 0 {
			continue
		}

		if !hasAnyResult && hasIncompleteCall {
			groupID := msg.GroupID
			var filtered []types.Message
			for j, m := range out {
				if j == i || (groupID != "" && m.GroupID == groupID && m.Type == types.ToolCallMessage) {
					continue
				}
				filtered = append(filtered, m)
			}
			out = filtered
			continue
		}

		groupID := msg.GroupID
		for _, idx := range unansweredIndices {
			tc := &out[i].ToolCalls[idx]
			if strings.TrimSpace(tc.ID) == "" {
				tc.ID = fmt.Sprintf("call_interrupted_%d_%d", time.Now().UnixNano(), idx)
			}
			if !json.Valid([]byte(tc.Arguments)) {
				tc.Arguments = "{}"
			}
			out = appendToolResultInGroup(out, groupID, types.Message{
				GroupID:    groupID,
				Type:       types.ToolResultMessage,
				Content:    reason,
				ToolCallID: tc.ID,
			})
			answered[tc.ID] = struct{}{}
		}
	}

	return out
}

func appendToolResultInGroup(msgs []types.Message, groupID string, resMsg types.Message) []types.Message {
	if groupID == "" {
		return append(msgs, resMsg)
	}
	insertAt := len(msgs)
	for insertAt > 0 {
		prev := msgs[insertAt-1]
		if prev.GroupID != groupID {
			break
		}
		if prev.Type != types.ImageMessage && prev.Type != types.PDFMessage {
			break
		}
		insertAt--
	}
	if insertAt == len(msgs) {
		return append(msgs, resMsg)
	}
	msgs = append(msgs, types.Message{})
	copy(msgs[insertAt+1:], msgs[insertAt:])
	msgs[insertAt] = resMsg
	return msgs
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
	s.Runtime.PDFMode = cfg.PDFMode
	s.Runtime.Generator.TitleModel = cfg.Title.ModelCode
	s.Runtime.Generator.BaseURL = cfg.Server.URL
	s.Runtime.Generator.Protocol = cfg.Server.Protocol
	s.Runtime.Generator.APIKey = cfg.Server.APIKey
	if s.Runtime.Permissions != nil {
		s.Runtime.Permissions.SetToolsConfig(cfg.Agent.Tools)
		s.Runtime.Permissions.SetRegistry(s.Runtime.Registry)
	}
	if s.Runtime.Registry != nil {
		tools.ConfigureRegistry(s.Runtime.Registry, cfg.Agent.Tools, s.Runtime.Generator, &s.Runtime.SecondaryConfig)
	}
	return nil
}
