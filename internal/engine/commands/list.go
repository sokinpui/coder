package commands

import (
	"github.com/sokinpui/coder/internal/types"
)

func init() {
	registerCommand("list", listCmd, "list context summary", nil)
}

func listCmd(args string, s SessionController) (CommandOutput, bool) {
	allFiles := s.GetContextFiles()
	allDocs := s.GetContextDocuments()

	if len(allFiles) == 0 && len(allDocs) == 0 {
		return CommandOutput{Type: types.MessagesUpdated, Payload: "No project source files or documents are in current context.", IsContext: true}, true
	}

	overview := formatFileListSummary(allFiles, allDocs)
	summary := "Current project context:\n" + overview

	return CommandOutput{Type: types.MessagesUpdated, Payload: summary, IsContext: true}, true
}
