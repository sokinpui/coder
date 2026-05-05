package commands

import (
	"github.com/sokinpui/coder/internal/types"
	"strings"
)

func init() {
	registerCommand("editor", editorCmd, PathArgumentCompleter)
	registerCommand("e", editorCmd, PathArgumentCompleter)
}

func editorCmd(args string, s SessionController) (CommandOutput, bool) {
	paths := strings.Fields(args)
	if len(paths) == 0 {
		return CommandOutput{Type: types.MessagesUpdated, Payload: "Usage: :editor <file paths>"}, false
	}

	expanded, _ := ExpandPaths(paths)
	if len(expanded) == 0 {
		return CommandOutput{Type: types.MessagesUpdated, Payload: "Error: No valid files found."}, false
	}

	return CommandOutput{Type: types.ExternalEditorStarted, Payload: strings.Join(expanded, " ")}, true
}
