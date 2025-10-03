package session

import (
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

	cmdOutput, _, cmdSuccess := core.ProcessCommand(input, s.messages, s.config, s)
	// ProcessCommand returns isCmd=true for any string with ':', so we don't need to check it.

	if cmdSuccess {
		switch cmdOutput.Type {
		case core.CommandResultNewSession:
			s.newSession()
			return core.Event{Type: core.NewSessionStarted}
		case core.CommandResultVisualMode:
			return core.Event{Type: core.VisualModeStarted}
		case core.CommandResultGenerateMode:
			return core.Event{Type: core.GenerateModeStarted}
		case core.CommandResultEditMode:
			return core.Event{Type: core.EditModeStarted}
		case core.CommandResultBranchMode:
			return core.Event{Type: core.BranchModeStarted}
		case core.CommandResultHistoryMode:
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
