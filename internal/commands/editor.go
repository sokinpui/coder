package commands

import (
	"github.com/sokinpui/coder/internal/types"
	"strings"
)

func init() {
	registerCommand("editor", editorCmd, "open files in $EDITOR", PathArgumentCompleter)
	registerCommand("e", editorCmd, "alias for /editor", PathArgumentCompleter)
}

func editorCmd(args string, s SessionController) (CommandOutput, bool) {
	if strings.Contains(args, "%s") {
		lastModified := s.GetLastModifiedFiles()
		if len(lastModified) == 0 {
			return CommandOutput{Type: types.MessagesUpdated, Payload: "Error: No files were affected by the last itf run."}, false
		}

		args = strings.ReplaceAll(args, "%s", strings.Join(lastModified, " "))
	}

	paths := strings.Fields(args)
	if len(paths) == 0 {
		return CommandOutput{Type: types.MessagesUpdated, Payload: "Usage: /editor <file paths>"}, false
	}

	expanded, _ := ExpandPaths(paths)
	if len(expanded) == 0 {
		return CommandOutput{Type: types.MessagesUpdated, Payload: "Error: No valid files found."}, false
	}

	return CommandOutput{Type: types.ExternalEditorStarted, Payload: strings.Join(expanded, " ")}, true
}
