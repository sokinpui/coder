package commands

import (
	"fmt"
	"github.com/sokinpui/coder/internal/types"
)

func init() {
	registerCommand("gen", genCmd, "re-generate response", nil)
	registerCommand("edit", editModeCmd, "edit user prompt", nil)
	registerCommand("msg", msgCmd, "open atomic messages overlay", nil)
	registerCommand("branch", branchCmd, "branch conversation", nil)
	registerCommand("history", historyCmd, "view chat history", nil)
	registerCommand("rename", renameCmd, "rename session title", nil)
	registerCommand("active", activeCmd, "view active sessions", nil)
}

func genCmd(args string, s SessionController) (CommandOutput, bool) {
	return CommandOutput{Type: types.MessagesUpdated, Payload: "Interactive command: use in TUI mode."}, true
}

func editModeCmd(args string, s SessionController) (CommandOutput, bool) {
	return CommandOutput{Type: types.MessagesUpdated, Payload: "Interactive command: use in TUI mode."}, true
}

func msgCmd(args string, s SessionController) (CommandOutput, bool) {
	return CommandOutput{Type: types.MessagesUpdated, Payload: "Interactive command: use in TUI mode."}, true
}

func branchCmd(args string, s SessionController) (CommandOutput, bool) {
	return CommandOutput{Type: types.MessagesUpdated, Payload: "Interactive command: use in TUI mode or session/branch RPC."}, true
}

func historyCmd(args string, s SessionController) (CommandOutput, bool) {
	return CommandOutput{Type: types.MessagesUpdated, Payload: "Interactive command: use in TUI mode or history RPC endpoints."}, true
}

func activeCmd(args string, s SessionController) (CommandOutput, bool) {
	return CommandOutput{Type: types.MessagesUpdated, Payload: "Interactive command: use in TUI mode."}, true
}

func renameCmd(args string, s SessionController) (CommandOutput, bool) {
	if args == "" {
		return CommandOutput{Type: types.MessagesUpdated, Payload: "Usage: /rename <new title>"}, false
	}
	s.SetTitle(args)
	return CommandOutput{Type: types.MessagesUpdated, Payload: fmt.Sprintf("Session title renamed to: %s", args)}, true
}
