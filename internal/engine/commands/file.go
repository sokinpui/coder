package commands

import (
	"fmt"
	"github.com/sokinpui/coder/internal/engine/source"
	"github.com/sokinpui/coder/internal/types"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

func init() {
	registerCommand("file", fileCmd, "add path to context", PathArgumentCompleter)
}

func PathArgumentCompleter(s SessionController, prefix string) []string {
	if prefix == "~" {
		return []string{"~/"}
	}

	dir := "."
	if lastSlash := strings.LastIndexAny(prefix, "/\\"); lastSlash != -1 {
		dir = prefix[:lastSlash+1]
	}

	entries, err := os.ReadDir(ExpandHome(dir))
	if err != nil {
		return nil
	}

	var results []string
	for _, entry := range entries {
		name := entry.Name()
		fullPath := name
		if dir != "." {
			fullPath = filepath.Join(dir, name)
		}

		fullPath = filepath.ToSlash(fullPath)
		if entry.IsDir() {
			fullPath += "/"
		}

		results = append(results, fullPath)
	}
	return results
}

func fileCmd(args string, s SessionController) (CommandOutput, bool) {
	paths := strings.Fields(args)

	if len(paths) == 0 {
		s.SetContextFiles([]string{})
		s.SetContextDocuments([]string{})
		s.ClearAllDocumentMessages()
		if err := s.LoadContext(); err != nil {
			msg := fmt.Sprintf("Project context cleared, but failed to reload context: %v", err)
			return CommandOutput{Type: types.MessagesUpdated, Payload: msg}, false
		}
		return CommandOutput{Type: types.MessagesUpdated, Payload: "Project context cleared.", IsContext: true}, true
	}

	currentFiles := s.GetContextFiles()
	currentDocs := s.GetContextDocuments()

	cfg := s.GetConfig()
	allExclusions := append([]string{}, source.Exclusions...)
	allExclusions = append(allExclusions, cfg.Context.Exclusions...)

	newFiles, newDocs, invalidPaths := source.Add(currentFiles, currentDocs, paths, allExclusions)
	addedCount := len(newFiles) - len(currentFiles)
	s.SetContextFiles(newFiles)
	s.SetContextDocuments(newDocs)

	if err := s.LoadContext(); err != nil {
		return CommandOutput{Type: types.MessagesUpdated, Payload: fmt.Sprintf("Project context updated, but failed to reload context: %v", err)}, false
	}

	var pdfRenderNotes []string
	var pdfErrors []string
	for _, doc := range newDocs {
		if slices.Contains(currentDocs, doc) {
			continue
		}
		pages := s.GetDocumentPageCount(doc)
		if pages > 0 {
			pdfRenderNotes = append(pdfRenderNotes, fmt.Sprintf("%s (%d pages)", doc, pages))
			continue
		}
		pdfErrors = append(pdfErrors, fmt.Sprintf("%s (failed to render or 0 pages)", doc))
	}

	fileWord := "files"
	if addedCount == 1 {
		fileWord = "file"
	}
	msg := fmt.Sprintf("Successfully add: %s, %d %s added.", args, addedCount, fileWord)
	if len(pdfRenderNotes) > 0 {
		msg += fmt.Sprintf("\nPDF pages added: %s", strings.Join(pdfRenderNotes, ", "))
	}
	if len(pdfErrors) > 0 {
		msg += fmt.Sprintf("\nPDF rendering failed: %s", strings.Join(pdfErrors, ", "))
	}
	if len(invalidPaths) > 0 {
		msg += fmt.Sprintf("\nWarning: The following paths do not exist and were ignored: %s", strings.Join(invalidPaths, ", "))
	}

	return CommandOutput{Type: types.MessagesUpdated, Payload: msg, IsContext: true}, true
}
