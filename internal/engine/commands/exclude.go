package commands

import (
	"fmt"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"github.com/sokinpui/coder/internal/types"
)

func init() {
	registerCommand("exclude", excludeCmd, "exclude path from context", excludeArgumentCompleter)
}

func excludeArgumentCompleter(s SessionController, prefix string) []string {
	if s == nil {
		return nil
	}

	files := s.GetContextFiles()
	docs := s.GetContextDocuments()
	if len(files) == 0 && len(docs) == 0 {
		return nil
	}

	seen := make(map[string]struct{})
	var suggestions []string

	addPath := func(p string) {
		p = filepath.ToSlash(filepath.Clean(p))
		if p == "." || p == "" {
			return
		}

		for i := 0; i < len(p); i++ {
			if p[i] == '/' {
				dir := p[:i+1]
				if dir != "/" {
					if _, ok := seen[dir]; !ok {
						seen[dir] = struct{}{}
						suggestions = append(suggestions, dir)
					}
				}
			}
		}

		if _, ok := seen[p]; !ok {
			seen[p] = struct{}{}
			suggestions = append(suggestions, p)
		}
	}

	for _, f := range files {
		addPath(f)
	}
	for _, d := range docs {
		cleanDoc := d
		if idx := strings.Index(cleanDoc, " (pages:"); idx != -1 {
			cleanDoc = cleanDoc[:idx]
		}
		addPath(strings.TrimSpace(cleanDoc))
	}

	sort.Strings(suggestions)
	return suggestions
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

	var removedDocsList []string
	for _, doc := range currentDocs {
		if !slices.Contains(newDocs, doc) {
			cleanDoc := doc
			if idx := strings.Index(cleanDoc, " (pages:"); idx != -1 {
				cleanDoc = cleanDoc[:idx]
			}
			removedDocsList = append(removedDocsList, cleanDoc)
		}
	}
	s.PurgeDocumentMessages(append(pathsToRemove, removedDocsList...))

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
		if shouldRemovePath(p, toRemove) {
			continue
		}
		filtered = append(filtered, p)
	}
	return filtered
}

func shouldRemovePath(path string, toRemove map[string]struct{}) bool {
	pathKey := path
	if idx := strings.Index(pathKey, " (pages:"); idx != -1 {
		pathKey = pathKey[:idx]
	}

	if _, found := toRemove[pathKey]; found {
		return true
	}
	if _, found := toRemove[path]; found {
		return true
	}

	cleanTarget := filepath.ToSlash(filepath.Clean(pathKey))
	for removePath := range toRemove {
		cleanRemove := strings.TrimSuffix(filepath.ToSlash(filepath.Clean(removePath)), "/")
		if cleanRemove == "" {
			continue
		}
		if cleanTarget == cleanRemove {
			return true
		}
		if cleanRemove == "." {
			if !strings.HasPrefix(cleanTarget, "../") && !filepath.IsAbs(cleanTarget) {
				return true
			}
		}
		if strings.HasPrefix(cleanTarget, cleanRemove+"/") {
			return true
		}
	}
	return false
}
