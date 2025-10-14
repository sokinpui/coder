package session

import (
	"coder/internal/commands"
	"coder/internal/core"
	"strings"
)

// HandleInput processes user input (prompts, commands, actions).
func (s *Session) HandleInput(input string) core.Event {
	if strings.TrimSpace(input) == "" {
		return core.Event{Type: core.NoOp}
	}

	if !strings.HasPrefix(input, ":") {
		// This is a new user prompt.
		s.messages = append(s.messages, core.Message{Type: core.UserMessage, Content: input})
		return s.StartGeneration()
	}

	cmdOutput, _, cmdSuccess := commands.ProcessCommand(input, s)
	// ProcessCommand returns isCmd=true for any string with ':', so we don't need to check it.

	if cmdSuccess {
		switch cmdOutput.Type {
		case commands.CommandResultNewSession:
			s.newSession()
			return core.Event{Type: core.NewSessionStarted}
		case commands.CommandResultVisualMode:
			return core.Event{Type: core.VisualModeStarted}
		case commands.CommandResultGenerateMode:
			return core.Event{Type: core.GenerateModeStarted}
		case commands.CommandResultEditMode:
			return core.Event{Type: core.EditModeStarted}
		case commands.CommandResultBranchMode:
			return core.Event{Type: core.BranchModeStarted}
		case commands.CommandResultHistoryMode:
			return core.Event{Type: core.HistoryModeStarted}
		}
	}

	s.generator.Config = s.config.Generation
	s.messages = append(s.messages, core.Message{Type: core.CommandMessage, Content: input})
	if cmdSuccess {
		s.messages = append(s.messages, core.Message{Type: core.CommandResultMessage, Content: cmdOutput.Payload})
	} else {
		s.messages = append(s.messages, core.Message{Type: core.CommandErrorResultMessage, Content: cmdOutput.Payload})
	}
	return core.Event{Type: core.MessagesUpdated}
}
