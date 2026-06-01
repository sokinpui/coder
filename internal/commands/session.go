package commands

import (
	"fmt"
	"github.com/sokinpui/coder/internal/types"
)

func init() {
	registerCommand("gen", genCmd, "re-generate response", nil)
	registerCommand("edit", editModeCmd, "edit user prompt", nil)
	registerCommand("visual", visualCmd, "enter visual mode", nil)
	registerCommand("branch", branchCmd, "branch conversation", nil)
	registerCommand("history", historyCmd, "view chat history", nil)
	registerCommand("rename", renameCmd, "rename session title", nil)
	registerCommand("active", activeCmd, "view active sessions", nil)
}

func hasSelectableMessages(messages []types.Message) bool {
	for _, msg := range messages {
		if msg.Type.IsSelectable() {
			return true
		}
	}
	return false
}

func genCmd(args string, s SessionController) (CommandOutput, bool) {
	messages := s.GetMessages()
	if !hasSelectableMessages(messages) {
		return CommandOutput{Type: types.NoOp}, true
	}
	return CommandOutput{Type: types.GenerateModeStarted}, true
}

func editModeCmd(args string, s SessionController) (CommandOutput, bool) {
	messages := s.GetMessages()
	if !hasSelectableMessages(messages) {
		return CommandOutput{Type: types.NoOp}, true
	}
	return CommandOutput{Type: types.EditModeStarted}, true
}

func visualCmd(args string, s SessionController) (CommandOutput, bool) {
	messages := s.GetMessages()
	if !hasSelectableMessages(messages) {
		return CommandOutput{Type: types.NoOp}, true
	}
	return CommandOutput{Type: types.VisualModeStarted}, true
}

func branchCmd(args string, s SessionController) (CommandOutput, bool) {
	messages := s.GetMessages()
	if !hasSelectableMessages(messages) {
		return CommandOutput{Type: types.NoOp}, true
	}
	return CommandOutput{Type: types.BranchModeStarted}, true
}

func historyCmd(args string, s SessionController) (CommandOutput, bool) {
	return CommandOutput{Type: types.HistoryModeStarted}, true
}

func activeCmd(args string, s SessionController) (CommandOutput, bool) {
	return CommandOutput{Type: types.ActiveModeStarted}, true
}

func renameCmd(args string, s SessionController) (CommandOutput, bool) {
	if args == "" {
		return CommandOutput{Type: types.MessagesUpdated, Payload: "Usage: /rename <new title>"}, false
	}
	s.SetTitle(args)
	return CommandOutput{Type: types.MessagesUpdated, Payload: fmt.Sprintf("Session title renamed to: %s", args)}, true
}
