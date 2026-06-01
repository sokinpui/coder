package commands

import (
	"github.com/sokinpui/coder/internal/types"
)

func init() {
	registerCommand("list", listCmd, "list context summary", nil)
}

func listCmd(args string, s SessionController) (CommandOutput, bool) {
	allFiles := s.GetContextFiles()

	if len(allFiles) == 0 {
		return CommandOutput{Type: types.MessagesUpdated, Payload: "No project source files are in current context."}, true
	}

	overview := formatFileListSummary(allFiles)

	summary := "Current project context:\n" + overview

	payload := summary

	return CommandOutput{Type: types.MessagesUpdated, Payload: payload}, true
}
