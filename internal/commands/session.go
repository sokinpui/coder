package commands

import (
	"github.com/sokinpui/coder/internal/types"
	"fmt"
)

func init() {
	registerCommand("gen", genCmd, nil)
	registerCommand("edit", editModeCmd, nil)
	registerCommand("visual", visualCmd, nil)
	registerCommand("branch", branchCmd, nil)
	registerCommand("history", historyCmd, nil)
	registerCommand("rename", renameCmd, nil)
	registerCommand("jump", jumpCmd, nil)
}

func hasSelectableMessages(messages []types.Message) bool {
	for _, msg := range messages {
		switch msg.Type {
		case types.InitMessage, types.DirectoryMessage:
			continue
		default:
			return true
		}
	}
	return false
}

func genCmd(args string, s SessionController) (CommandOutput, bool) {
	messages := s.GetMessages()
	if !hasSelectableMessages(messages) {
		return CommandOutput{Type: types.MessagesUpdated, Payload: "Cannot enter generate mode: no messages to select."}, false
	}
	return CommandOutput{Type: types.GenerateModeStarted}, true
}

func editModeCmd(args string, s SessionController) (CommandOutput, bool) {
	messages := s.GetMessages()
	if !hasSelectableMessages(messages) {
		return CommandOutput{Type: types.MessagesUpdated, Payload: "Cannot enter edit mode: no messages to select."}, false
	}
	return CommandOutput{Type: types.EditModeStarted}, true
}

func visualCmd(args string, s SessionController) (CommandOutput, bool) {
	messages := s.GetMessages()
	if !hasSelectableMessages(messages) {
		return CommandOutput{Type: types.MessagesUpdated, Payload: "Cannot enter visual mode: no messages to select."}, false
	}
	return CommandOutput{Type: types.VisualModeStarted}, true
}

func branchCmd(args string, s SessionController) (CommandOutput, bool) {
	messages := s.GetMessages()
	if !hasSelectableMessages(messages) {
		return CommandOutput{Type: types.MessagesUpdated, Payload: "Cannot enter branch mode: no messages to select."}, false
	}
	return CommandOutput{Type: types.BranchModeStarted}, true
}

func historyCmd(args string, s SessionController) (CommandOutput, bool) {
	return CommandOutput{Type: types.HistoryModeStarted}, true
}

func jumpCmd(args string, s SessionController) (CommandOutput, bool) {
	messages := s.GetMessages()
	if !hasSelectableMessages(messages) {
		return CommandOutput{Type: types.MessagesUpdated, Payload: "Cannot enter jump mode: no messages to select."}, false
	}
	return CommandOutput{Type: types.JumpModeStarted}, true
}

func renameCmd(args string, s SessionController) (CommandOutput, bool) {
	if args == "" {
		return CommandOutput{Type: types.MessagesUpdated, Payload: "Usage: :rename <new title>"}, false
	}
	s.SetTitle(args)
	return CommandOutput{Type: types.MessagesUpdated, Payload: fmt.Sprintf("Session title renamed to: %s", args)}, true
}
