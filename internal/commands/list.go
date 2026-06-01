package commands

import (
	"github.com/sokinpui/coder/internal/types"
	"github.com/sokinpui/pcat"
)

func init() {
	registerCommand("list", listCmd, "list context summary", nil)
	registerCommand("list-all", listFullCmd, "list all context files", nil)
}

func listFullCmd(args string, s SessionController) (CommandOutput, bool) {
	allFiles := s.GetContextFiles()

	if len(allFiles) == 0 {
		return CommandOutput{Type: types.MessagesUpdated, Payload: "No project source files are in current context."}, true
	}

	fileList, err := pcat.Run(
		allFiles, // specificFiles
		nil,      // directories
		nil,      // extensions
		nil,      // excludePatterns
		false,    // withLineNumbers
		true,     // hidden
		true,     // listOnly
	)
	if err != nil {
		return CommandOutput{Type: types.MessagesUpdated, Payload: "Error listing files"}, false
	}

	overview := formatFileListSummary(allFiles)

	summary := "Current project context:\n" + overview + "\n\n" + "Files read by AI:\n" + fileList

	return CommandOutput{Type: types.MessagesUpdated, Payload: summary}, true
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
