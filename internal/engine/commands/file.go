package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/sokinpui/coder/internal/engine"
	"github.com/sokinpui/coder/internal/engine/pdf"
	"github.com/sokinpui/coder/internal/engine/source"
	"github.com/sokinpui/coder/internal/types"
)

func init() {
	registerCommand("file", fileCmd, "add path to context", PathArgumentCompleter)
	registerCommand("@", atCmd, "add path to context (alias for /file)", PathArgumentCompleter)
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

func atCmd(args string, s SessionController) (CommandOutput, bool) {
	if !s.Capabilities().Has(engine.CapContextFiles) && !s.Capabilities().Has(engine.CapToolLoop) {
		return CommandOutput{Type: types.MessagesUpdated, Payload: "Unknown command: @"}, false
	}
	if s.Capabilities().Has(engine.CapToolLoop) {
		reader, ok := s.(engine.AgentFileReader)
		if !ok {
			return CommandOutput{Type: types.MessagesUpdated, Payload: "Agent file reading not supported"}, false
		}
		paths := strings.Fields(args)
		if len(paths) == 0 {
			return CommandOutput{Type: types.MessagesUpdated, Payload: "Usage: @ <paths...>"}, false
		}
		return reader.ReadAgentFiles("@ "+args, paths)
	}
	return addFileToContext(args, "@", s)
}

func fileCmd(args string, s SessionController) (CommandOutput, bool) {
	if !s.Capabilities().Has(engine.CapContextFiles) && !s.Capabilities().Has(engine.CapToolLoop) {
		return CommandOutput{Type: types.MessagesUpdated, Payload: "Unknown command: file"}, false
	}

	if s.Capabilities().Has(engine.CapToolLoop) {
		reader, ok := s.(engine.AgentFileReader)
		if !ok {
			return CommandOutput{Type: types.MessagesUpdated, Payload: "Agent file reading not supported"}, false
		}
		paths := strings.Fields(args)
		if len(paths) == 0 {
			return CommandOutput{Type: types.MessagesUpdated, Payload: "Usage: /file <paths...>"}, false
		}
		return reader.ReadAgentFiles("/file "+args, paths)
	}
	return addFileToContext(args, "/file", s)
}

func addFileToContext(args string, cmdPrefix string, s SessionController) (CommandOutput, bool) {
	paths := strings.Fields(args)
	if len(paths) == 0 {
		return CommandOutput{Type: types.MessagesUpdated, Payload: fmt.Sprintf("Usage: %s <paths...>", cmdPrefix)}, false
	}

	ctxCtrl, ok := s.(engine.ContextController)
	if !ok {
		return CommandOutput{Type: types.MessagesUpdated, Payload: "Context management not supported in this session"}, false
	}

	expanded, invalidPaths := ExpandPaths(paths)
	allPaths := append(expanded, invalidPaths...)

	var codePaths []string
	var pdfPaths []string

	for _, p := range allPaths {
		clean := p
		if idx := strings.Index(clean, " (pages:"); idx != -1 {
			clean = strings.TrimSpace(clean[:idx])
		}
		if strings.EqualFold(filepath.Ext(clean), ".pdf") {
			pdfPaths = append(pdfPaths, p)
		} else {
			codePaths = append(codePaths, p)
		}
	}

	var addedCount int
	if len(codePaths) > 0 {
		currentFiles := ctxCtrl.GetContextFiles()
		cfg := s.GetConfig()
		allExclusions := append([]string{}, source.Exclusions...)
		allExclusions = append(allExclusions, cfg.Coder.Context.Exclusions...)

		newFiles, newInvalid := source.Add(currentFiles, codePaths, allExclusions)
		addedCount = len(newFiles) - len(currentFiles)
		ctxCtrl.SetContextFiles(newFiles)
		if err := ctxCtrl.LoadContext(); err != nil {
			return CommandOutput{Type: types.MessagesUpdated, Payload: fmt.Sprintf("Project context updated, but failed to reload context: %v", err)}, false
		}
		invalidPaths = newInvalid
	}

	var pdfRenderNotes []string
	var pdfErrors []string
	var imageMessages []types.Message

	for _, rawDoc := range pdfPaths {
		docPath := rawDoc
		pagesSpec := ""
		if idx := strings.Index(docPath, " (pages:"); idx != -1 {
			if endIdx := strings.Index(docPath[idx:], ")"); endIdx != -1 {
				pagesSpec = docPath[idx+8 : idx+endIdx]
			}
			docPath = strings.TrimSpace(docPath[:idx])
		}

		msgs, err := pdf.RenderPDFToMessages(docPath, pagesSpec)
		if err != nil {
			pdfErrors = append(pdfErrors, fmt.Sprintf("%s (%v)", docPath, err))
			continue
		}
		if len(msgs) == 0 {
			pdfErrors = append(pdfErrors, fmt.Sprintf("%s (0 pages)", docPath))
			continue
		}

		pdfRenderNotes = append(pdfRenderNotes, fmt.Sprintf("%s (%d pages)", docPath, len(msgs)))
		imageMessages = append(imageMessages, msgs...)
	}

	var msgParts []string
	if addedCount > 0 || (len(codePaths) > 0 && len(pdfPaths) == 0) {
		fileWord := "files"
		if addedCount == 1 {
			fileWord = "file"
		}
		msgParts = append(msgParts, fmt.Sprintf("Successfully added %d %s to context.", addedCount, fileWord))
	}
	if len(pdfRenderNotes) > 0 {
		msgParts = append(msgParts, fmt.Sprintf("PDF pages added to chat: %s", strings.Join(pdfRenderNotes, ", ")))
	}
	if len(pdfErrors) > 0 {
		msgParts = append(msgParts, fmt.Sprintf("PDF rendering failed: %s", strings.Join(pdfErrors, ", ")))
	}
	if len(invalidPaths) > 0 {
		msgParts = append(msgParts, fmt.Sprintf("Warning: The following paths do not exist and were ignored: %s", strings.Join(invalidPaths, ", ")))
	}

	if len(msgParts) == 0 {
		msgParts = append(msgParts, "No files or documents added.")
	}

	success := (addedCount > 0) || (len(pdfRenderNotes) > 0)
	if len(codePaths) == 0 && len(pdfPaths) == 0 {
		success = false
	}

	return CommandOutput{
		Type:      types.MessagesUpdated,
		Payload:   strings.Join(msgParts, "\n"),
		IsContext: true,
		Messages:  imageMessages,
	}, success
}
