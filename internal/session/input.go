package session

import (
	"coder/internal/commands"
	"coder/internal/types"
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
		case commands.CommandResultNewSession:
			s.newSession()
			return types.Event{Type: types.NewSessionStarted}
		case commands.CommandResultVisualMode:
			return types.Event{Type: types.VisualModeStarted}
		case commands.CommandResultGenerateMode:
			return types.Event{Type: types.GenerateModeStarted}
		case commands.CommandResultEditMode:
			return types.Event{Type: types.EditModeStarted}
		case commands.CommandResultBranchMode:
			return types.Event{Type: types.BranchModeStarted}
		case commands.CommandResultHistoryMode:
			return types.Event{Type: types.HistoryModeStarted}
		case commands.CommandResultQuit:
			return types.Event{Type: types.Quit}
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
