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
		case commands.CommandResultChatMode:
			s.startChatSession()
			return types.Event{Type: types.NewSessionStarted}
		case commands.CommandResultQuit:
			return types.Event{Type: types.Quit}
		}

		modeEvents := map[commands.CommandResultType]types.EventType{
			commands.CommandResultVisualMode:   types.VisualModeStarted,
			commands.CommandResultGenerateMode: types.GenerateModeStarted,
			commands.CommandResultEditMode:     types.EditModeStarted,
			commands.CommandResultBranchMode:   types.BranchModeStarted,
			commands.CommandResultSearchMode:   types.SearchModeStarted,
			commands.CommandResultHistoryMode:  types.HistoryModeStarted,
			commands.CommandResultTreeMode:     types.TreeModeStarted,
			commands.CommandResultFzfMode:      types.FzfModeStarted,
		}

		if eventType, ok := modeEvents[cmdOutput.Type]; ok {
			s.messages = append(s.messages, types.Message{Type: types.CommandMessage, Content: input})
			return types.Event{Type: eventType, Data: cmdOutput.Payload}
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
