package commands

import (
	"coder/internal/core"
	"fmt"
)

func init() {
	registerCommand("gen", genCmd, nil)
	registerCommand("edit", editModeCmd, nil)
	registerCommand("visual", visualCmd, nil)
	registerCommand("branch", branchCmd, nil)
	registerCommand("history", historyCmd, nil)
	registerCommand("rename", renameCmd, nil)
}

func hasSelectableMessages(messages []core.Message) bool {
	for _, msg := range messages {
		switch msg.Type {
		case core.InitMessage, core.DirectoryMessage:
			continue
		default:
			return true
		}
	}
	return false
}

func genCmd(args string, s Session) (CommandOutput, bool) {
	messages := s.GetMessages()
	if !hasSelectableMessages(messages) {
		return CommandOutput{Type: CommandResultString, Payload: "Cannot enter generate mode: no messages to select."}, false
	}
	return CommandOutput{Type: CommandResultGenerateMode}, true
}

func editModeCmd(args string, s Session) (CommandOutput, bool) {
	messages := s.GetMessages()
	if !hasSelectableMessages(messages) {
		return CommandOutput{Type: CommandResultString, Payload: "Cannot enter edit mode: no messages to select."}, false
	}
	return CommandOutput{Type: CommandResultEditMode}, true
}

func visualCmd(args string, s Session) (CommandOutput, bool) {
	messages := s.GetMessages()
	if !hasSelectableMessages(messages) {
		return CommandOutput{Type: CommandResultString, Payload: "Cannot enter visual mode: no messages to select."}, false
	}
	return CommandOutput{Type: CommandResultVisualMode}, true
}

func branchCmd(args string, s Session) (CommandOutput, bool) {
	messages := s.GetMessages()
	if !hasSelectableMessages(messages) {
		return CommandOutput{Type: CommandResultString, Payload: "Cannot enter branch mode: no messages to select."}, false
	}
	return CommandOutput{Type: CommandResultBranchMode}, true
}

func historyCmd(args string, s Session) (CommandOutput, bool) {
	return CommandOutput{Type: CommandResultHistoryMode}, true
}

func renameCmd(args string, s Session) (CommandOutput, bool) {
	if args == "" {
		return CommandOutput{Type: CommandResultString, Payload: "Usage: :rename <new title>"}, false
	}
	s.SetTitle(args)
	return CommandOutput{Type: CommandResultString, Payload: fmt.Sprintf("Session title renamed to: %s", args)}, true
}
