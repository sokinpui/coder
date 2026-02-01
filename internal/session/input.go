package session

import (
	"github.com/sokinpui/coder/internal/commands"
	"github.com/sokinpui/coder/internal/types"
	"strings"
)

// HandleInput processes user input (prompts, commands, actions).
func (s *Session) HandleInput(input string) types.Event {
	if strings.TrimSpace(input) == "" {
		return types.Event{Type: types.NoOp}
	}

	if !strings.HasPrefix(input, ":") {
		// This is a new user prompt.
		s.messages = append(s.messages, types.Message{Type: types.UserMessage, Content: input})
		return s.StartGeneration()
	}

	cmdOutput, _, cmdSuccess := commands.ProcessCommand(input, s)
	// ProcessCommand returns isCmd=true for any string with ':', so we don't need to check it.

	if cmdSuccess {
		switch cmdOutput.Type {
		case types.NewSessionStarted, types.Quit:
			// These events typically don't log to the message history
			return types.Event{Type: cmdOutput.Type}
		case types.MessagesUpdated, types.NoOp:
			// Fall through to standard logging below
		default:
			// Mode transition events: log the command call then return transition event.
			s.messages = append(s.messages, types.Message{Type: types.CommandMessage, Content: input})
			return types.Event{Type: cmdOutput.Type, Data: cmdOutput.Payload}
		}
	}

	s.generator.Config = s.config.Generation
	s.messages = append(s.messages, types.Message{Type: types.CommandMessage, Content: input})
	if cmdSuccess {
		s.messages = append(s.messages, types.Message{Type: types.CommandResultMessage, Content: cmdOutput.Payload})
	} else {
		s.messages = append(s.messages, types.Message{Type: types.CommandErrorResultMessage, Content: cmdOutput.Payload})
	}
	return types.Event{Type: types.MessagesUpdated}
}
