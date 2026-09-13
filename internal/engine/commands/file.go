package commands

import (
	"fmt"
	"github.com/sokinpui/coder/internal/config"
	"github.com/sokinpui/coder/internal/engine/source"
	"github.com/sokinpui/coder/internal/types"
	"os"
	"path/filepath"
	"strings"
)

func init() {
	registerCommand("file", fileCmd, "add path to context", PathArgumentCompleter)
}

func PathArgumentCompleter(cfg *config.Config, prefix string) []string {
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

	var files []string
	var dirs []string
	var invalidPaths []string

	expandedPaths, invalidPatterns := ExpandPaths(paths)
	invalidPaths = append(invalidPaths, invalidPatterns...)

	for _, p := range expandedPaths {
		info, err := os.Stat(p)
		if err != nil {
			if os.IsNotExist(err) {
				invalidPaths = append(invalidPaths, p)
				continue
			}
			return CommandOutput{Type: types.MessagesUpdated, Payload: fmt.Sprintf("Error accessing path %s: %v", p, err)}, false
		}
		if info.IsDir() {
			dirs = append(dirs, p)
		} else {
			files = append(files, p)
		}
	}

	var pdfFiles []string
	var codeFiles []string
	for _, f := range files {
		if strings.EqualFold(filepath.Ext(f), ".pdf") {
			pdfFiles = append(pdfFiles, f)
			continue
		}
		codeFiles = append(codeFiles, f)
	}
	files = codeFiles

	var addedDocs []string
	for _, pdfPath := range pdfFiles {
		addedDocs = append(addedDocs, filepath.ToSlash(pdfPath))
	}

	if len(addedDocs) > 0 {
		s.SetContextDocuments(AppendUnique(s.GetContextDocuments(), addedDocs))
	}

	currentFiles := s.GetContextFiles()
	cfg := s.GetConfig()
	allExclusions := append([]string{}, source.Exclusions...)
	allExclusions = append(allExclusions, cfg.Context.Exclusions...)

	newResolvedFiles, _ := source.ResolveFileList(dirs, files, allExclusions)
	updatedFiles := AppendUnique(currentFiles, newResolvedFiles)
	addedCount := len(updatedFiles) - len(currentFiles)
	s.SetContextFiles(updatedFiles)

	if err := s.LoadContext(); err != nil {
		return CommandOutput{Type: types.MessagesUpdated, Payload: fmt.Sprintf("Project context updated, but failed to reload context: %v", err)}, false
	}

	var pdfRenderNotes []string
	var pdfErrors []string
	for _, doc := range addedDocs {
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
