package commands

import (
	"fmt"
	"github.com/sokinpui/coder/internal/engine"
	"github.com/sokinpui/coder/internal/engine/source"
	"github.com/sokinpui/coder/internal/types"
	"strings"
)

func init() {
	registerCommand("exclude", excludeCmd, "exclude path from context", excludeArgumentCompleter)
}

func excludeArgumentCompleter(s SessionController, prefix string) []string {
	if s == nil {
		return nil
	}
	ctxCtrl, ok := s.(engine.ContextController)
	if !ok {
		return nil
	}
	return source.ContextSuggestions(ctxCtrl.GetContextFiles(), ctxCtrl.GetContextDocuments())
}

func excludeCmd(args string, s SessionController) (CommandOutput, bool) {
	if !s.Capabilities().Has(engine.CapContextFiles) {
		return CommandOutput{Type: types.MessagesUpdated, Payload: "Unknown command: exclude"}, false
	}

	paths := strings.Fields(args)

	if len(paths) == 0 {
		return CommandOutput{Type: types.MessagesUpdated, Payload: "Usage: /exclude <paths...>"}, false
	}

	ctxCtrl, ok := s.(engine.ContextController)
	if !ok {
		return CommandOutput{Type: types.MessagesUpdated, Payload: "Context management not supported in this session"}, false
	}

	newFiles, newDocs, removedFiles, removedDocs := source.Exclude(ctxCtrl.GetContextFiles(), ctxCtrl.GetContextDocuments(), paths)
	removedCount := len(removedFiles) + len(removedDocs)

	ctxCtrl.SetContextFiles(newFiles)
	ctxCtrl.SetContextDocuments(newDocs)
	ctxCtrl.PurgeDocumentMessages(removedDocs)

	if err := ctxCtrl.LoadContext(); err != nil {
		return CommandOutput{Type: types.MessagesUpdated, Payload: fmt.Sprintf("Project source updated, but failed to reload context: %v", err)}, false
	}

	fileWord := "files"
	if removedCount == 1 {
		fileWord = "file"
	}
	return CommandOutput{Type: types.MessagesUpdated, Payload: fmt.Sprintf("Successfully excluded %s, %d %s removed.", args, removedCount, fileWord), IsContext: true}, true
}
