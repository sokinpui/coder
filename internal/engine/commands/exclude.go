package commands

import (
	"fmt"
	"strings"
	"github.com/sokinpui/coder/internal/engine/source"
	"github.com/sokinpui/coder/internal/types"
)

func init() {
	registerCommand("exclude", excludeCmd, "exclude path from context", excludeArgumentCompleter)
}

func excludeArgumentCompleter(s SessionController, prefix string) []string {
	if s == nil {
		return nil
	}
	return source.ContextSuggestions(s.GetContextFiles(), s.GetContextDocuments())
}

func excludeCmd(args string, s SessionController) (CommandOutput, bool) {
	paths := strings.Fields(args)

	if len(paths) == 0 {
		return CommandOutput{Type: types.MessagesUpdated, Payload: "Usage: /exclude <paths...>"}, false
	}

	newFiles, newDocs, removedFiles, removedDocs := source.Exclude(s.GetContextFiles(), s.GetContextDocuments(), paths)
	removedCount := len(removedFiles) + len(removedDocs)

	s.SetContextFiles(newFiles)
	s.SetContextDocuments(newDocs)
	s.PurgeDocumentMessages(removedDocs)

	if err := s.LoadContext(); err != nil {
		return CommandOutput{Type: types.MessagesUpdated, Payload: fmt.Sprintf("Project source updated, but failed to reload context: %v", err)}, false
	}

	fileWord := "files"
	if removedCount == 1 {
		fileWord = "file"
	}
	return CommandOutput{Type: types.MessagesUpdated, Payload: fmt.Sprintf("Successfully excluded %s, %d %s removed.", args, removedCount, fileWord), IsContext: true}, true
}
