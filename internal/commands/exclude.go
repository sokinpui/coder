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
		return CommandOutput{Type: types.ExcludePickerStarted}, true
	}

	pathsToRemove, _ := ExpandPaths(paths)

	pathsToModify := make(map[string]struct{})
	for _, p := range pathsToRemove {
		pathsToModify[filepath.ToSlash(p)] = struct{}{}
	}

	currentFiles := s.GetContextFiles()
	newFiles := filterPaths(currentFiles, pathsToModify)
	removedCount := len(currentFiles) - len(newFiles)
	s.SetContextFiles(newFiles)

	if err := s.LoadContext(); err != nil {
		return CommandOutput{Type: types.MessagesUpdated, Payload: fmt.Sprintf("Project source updated, but failed to reload context: %v", err)}, false
	}

	fileWord := "files"
	if removedCount == 1 {
		fileWord = "file"
	}
	return CommandOutput{Type: types.MessagesUpdated, Payload: fmt.Sprintf("Successfully excluded %s, %d %s removed.", args, removedCount, fileWord)}, true
}

func filterPaths(original []string, toRemove map[string]struct{}) []string {
	filtered := make([]string, 0, len(original))
	for _, p := range original {
		if _, found := toRemove[p]; !found {
			filtered = append(filtered, p)
		}
	}
	return filtered
}
