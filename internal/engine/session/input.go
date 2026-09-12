package session

import (
	"github.com/sokinpui/coder/internal/engine/commands"
	"github.com/sokinpui/coder/internal/types"
	"strings"
)

func (s *Session) HandleInput(input string) types.Event {
	return s.processInput(input, false)
}

func (s *Session) HandleShortcut(input string) types.Event {
	return s.processInput(input, true)
}

func (s *Session) processInput(input string, silent bool) types.Event {
	if strings.TrimSpace(input) == "" {
		return types.Event{Type: types.NoOp}
	}

	if !strings.HasPrefix(input, "/") {
		// Prompts are never silent
		// This is a new user prompt.
		s.messages = append(s.messages, types.Message{Type: types.UserMessage, Content: input})
		return s.StartGeneration()
	}

	cmdOutput, _, cmdSuccess := commands.ProcessCommand(input, s)
	// ProcessCommand returns isCmd=true for any string with '/', so we don't need to check it.

	switch cmdOutput.Type {
	case types.NewSessionStarted, types.Quit:
		return types.Event{Type: cmdOutput.Type, Mode: cmdOutput.Mode}
	case types.TermExecutionStarted:
		if !silent {
			s.messages = append(s.messages, types.Message{Type: types.ShellCmdMessage, Content: input})
		}
		return types.Event{Type: types.TermExecutionStarted, Data: cmdOutput.Payload}
	case types.NoOp:
		return types.Event{Type: types.NoOp}
	}

	s.generator.Config = s.config.Generation
	if !silent {
		msgType := types.CommandMessage
		if cmdOutput.IsFileApply {
			msgType = types.FileApplyCmdMessage
		} else if cmdOutput.IsFileApplyUndo {
			msgType = types.FileApplyUndoCmdMessage
		} else if cmdSuccess && cmdOutput.IsContext {
			msgType = types.ContextCmdMessage
		} else if cmdOutput.IsShell {
			msgType = types.ShellCmdMessage
		}
		s.messages = append(s.messages, types.Message{Type: msgType, Content: input})
	}

	if cmdSuccess {
		msgType := types.CommandResultMessage
		if cmdOutput.IsFileApply {
			msgType = types.FileApplyCmdResultMessage
		} else if cmdOutput.IsFileApplyUndo {
			msgType = types.FileApplyUndoCmdResultMessage
		} else if cmdOutput.IsShell {
			msgType = types.ShellCmdResultMessage
		} else if cmdOutput.IsContext {
			msgType = types.ContextCmdResultMessage
		}
		s.messages = append(s.messages, types.Message{Type: msgType, Content: cmdOutput.Payload})
	} else {
		msgType := types.CommandErrorResultMessage
		if cmdOutput.IsFileApply {
			msgType = types.FileApplyCmdErrorMessage
		} else if cmdOutput.IsFileApplyUndo {
			msgType = types.FileApplyUndoCmdErrorMessage
		}
		s.messages = append(s.messages, types.Message{Type: msgType, Content: cmdOutput.Payload})
	}
	return types.Event{Type: types.MessagesUpdated}
}
