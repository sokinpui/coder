package commands

import (
	"fmt"
	"github.com/sokinpui/coder/internal/types"
	"path/filepath"
	"strings"
)

func init() {
	registerCommand("exclude", excludeCmd, "exclude path from context", PathArgumentCompleter)
}

func excludeCmd(args string, s SessionController) (CommandOutput, bool) {
	paths := strings.Fields(args)

	if len(paths) == 0 {
		return CommandOutput{Type: types.MessagesUpdated, Payload: "Usage: /exclude <paths...>"}, false
	}

	pathsToRemove, _ := ExpandPaths(paths)

	pathsToModify := make(map[string]struct{})
	for _, p := range pathsToRemove {
		pathsToModify[filepath.ToSlash(p)] = struct{}{}
	}

	currentFiles := s.GetContextFiles()
	newFiles := filterPaths(currentFiles, pathsToModify)
	removedFilesCount := len(currentFiles) - len(newFiles)
	s.SetContextFiles(newFiles)

	currentDocs := s.GetContextDocuments()
	newDocs := filterPaths(currentDocs, pathsToModify)
	removedDocsCount := len(currentDocs) - len(newDocs)
	s.SetContextDocuments(newDocs)
	removedCount := removedFilesCount + removedDocsCount

	s.PurgeDocumentMessages(pathsToRemove)

	if err := s.LoadContext(); err != nil {
		return CommandOutput{Type: types.MessagesUpdated, Payload: fmt.Sprintf("Project source updated, but failed to reload context: %v", err)}, false
	}

	fileWord := "files"
	if removedCount == 1 {
		fileWord = "file"
	}
	return CommandOutput{Type: types.MessagesUpdated, Payload: fmt.Sprintf("Successfully excluded %s, %d %s removed.", args, removedCount, fileWord), IsContext: true}, true
}

func filterPaths(original []string, toRemove map[string]struct{}) []string {
	filtered := make([]string, 0, len(original))
	for _, p := range original {
		pathKey := p
		if idx := strings.Index(pathKey, " (pages:"); idx != -1 {
			pathKey = pathKey[:idx]
		}
		if _, found := toRemove[pathKey]; found {
			continue
		}
		if _, found := toRemove[p]; !found {
			filtered = append(filtered, p)
		}
	}
	return filtered
}
