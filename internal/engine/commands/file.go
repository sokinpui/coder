package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/sokinpui/coder/internal/engine"
	"github.com/sokinpui/coder/internal/engine/pdf"
	"github.com/sokinpui/coder/internal/engine/source"
	"github.com/sokinpui/coder/internal/project"
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
		paths := SplitArgs(args)
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
		paths := SplitArgs(args)
		if len(paths) == 0 {
			return CommandOutput{Type: types.MessagesUpdated, Payload: "Usage: /file <paths...>"}, false
		}
		return reader.ReadAgentFiles("/file "+args, paths)
	}
	return addFileToContext(args, "/file", s)
}

func addFileToContext(args string, cmdPrefix string, s SessionController) (CommandOutput, bool) {
	paths := SplitArgs(args)
	if len(paths) == 0 {
		return CommandOutput{Type: types.MessagesUpdated, Payload: fmt.Sprintf("Usage: %s <paths...>", cmdPrefix)}, false
	}

	ctxCtrl, ok := s.(engine.ContextController)
	if !ok {
		return CommandOutput{Type: types.MessagesUpdated, Payload: "Context management not supported in this session"}, false
	}

	expanded, invalidPaths := ExpandPaths(paths)

	var codePaths []string
	var pdfPaths []string
	var imagePaths []string

	for _, p := range expanded {
		clean := p
		if idx := strings.Index(clean, " (pages:"); idx != -1 {
			clean = strings.TrimSpace(clean[:idx])
		}
		ext := strings.ToLower(filepath.Ext(clean))
		switch {
		case ext == ".pdf":
			pdfPaths = append(pdfPaths, p)
		case isImageExt(ext):
			imagePaths = append(imagePaths, p)
		default:
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
		invalidPaths = append(invalidPaths, newInvalid...)
	}

	var pdfRenderNotes []string
	var pdfErrors []string
	var imageNotes []string
	var imageErrors []string
	var mediaMessages []types.Message
	sessID := s.GetID()
	pdfMode := ""
	if cfg := s.GetConfig(); cfg != nil {
		pdfMode = cfg.PDFMode
	}
	isImageMode := pdf.NormalizeMode(pdfMode) == pdf.ModeImage

	for _, rawDoc := range pdfPaths {
		docPath := rawDoc
		pagesSpec := ""
		if idx := strings.Index(docPath, " (pages:"); idx != -1 {
			if endIdx := strings.Index(docPath[idx:], ")"); endIdx != -1 {
				pagesSpec = docPath[idx+8 : idx+endIdx]
			}
			docPath = strings.TrimSpace(docPath[:idx])
		}

		if isImageMode {
			pdfMsg, err := pdf.RenderPDF(sessID, docPath, pagesSpec)
			if err != nil {
				pdfErrors = append(pdfErrors, fmt.Sprintf("%s (%v)", docPath, err))
				continue
			}
			meta, _ := types.ParsePDFMetadata(pdfMsg.Content)
			if meta.PageCount == 0 {
				pdfErrors = append(pdfErrors, fmt.Sprintf("%s (0 pages)", docPath))
				continue
			}

			pdfRenderNotes = append(pdfRenderNotes, fmt.Sprintf("%s (%d pages)", docPath, meta.PageCount))
			mediaMessages = append(mediaMessages, pdfMsg)
			continue
		}

		pdfMsg, err := pdf.LoadNativePDF(docPath)
		if err != nil {
			pdfErrors = append(pdfErrors, fmt.Sprintf("%s (%v)", docPath, err))
			continue
		}
		pdfRenderNotes = append(pdfRenderNotes, docPath)
		mediaMessages = append(mediaMessages, pdfMsg)
	}

	for _, imgPath := range imagePaths {
		imgMsg, notFound, err := loadImageFile(imgPath)
		if err != nil {
			if notFound {
				invalidPaths = append(invalidPaths, imgPath)
				continue
			}
			imageErrors = append(imageErrors, fmt.Sprintf("%s (%v)", imgPath, err))
			continue
		}
		imageNotes = append(imageNotes, filepath.ToSlash(imgPath))
		mediaMessages = append(mediaMessages, imgMsg)
	}

	var msgParts []string
	if addedCount > 0 || (len(codePaths) > 0 && len(pdfPaths) == 0 && len(imagePaths) == 0) {
		fileWord := "files"
		if addedCount == 1 {
			fileWord = "file"
		}
		msgParts = append(msgParts, fmt.Sprintf("Successfully added %d %s to context.", addedCount, fileWord))
	}
	if len(pdfRenderNotes) > 0 {
		if isImageMode {
			msgParts = append(msgParts, fmt.Sprintf("PDF pages added to chat: %s", strings.Join(pdfRenderNotes, ", ")))
		} else {
			msgParts = append(msgParts, fmt.Sprintf("PDF added to chat: %s", strings.Join(pdfRenderNotes, ", ")))
		}
	}
	if len(imageNotes) > 0 {
		msgParts = append(msgParts, fmt.Sprintf("Image added to chat: %s", strings.Join(imageNotes, ", ")))
	}
	if len(pdfErrors) > 0 {
		msgParts = append(msgParts, fmt.Sprintf("PDF loading failed: %s", strings.Join(pdfErrors, ", ")))
	}
	if len(imageErrors) > 0 {
		msgParts = append(msgParts, fmt.Sprintf("Image loading failed: %s", strings.Join(imageErrors, ", ")))
	}
	if len(invalidPaths) > 0 {
		msgParts = append(msgParts, fmt.Sprintf("Warning: The following paths do not exist and were ignored: %s", strings.Join(invalidPaths, ", ")))
	}

	if len(msgParts) == 0 {
		msgParts = append(msgParts, "No files or documents added.")
	}

	success := (addedCount > 0) || (len(pdfRenderNotes) > 0) || (len(imageNotes) > 0)
	if len(codePaths) == 0 && len(pdfPaths) == 0 && len(imagePaths) == 0 {
		success = false
	}

	return CommandOutput{
		Type:      types.MessagesUpdated,
		Payload:   strings.Join(msgParts, "\n"),
		IsContext: true,
		Messages:  mediaMessages,
	}, success
}

func isImageExt(ext string) bool {
	switch ext {
	case ".png", ".jpg", ".jpeg", ".webp", ".gif", ".bmp":
		return true
	default:
		return false
	}
}

func loadImageFile(path string) (types.Message, bool, error) {
	info, err := os.Stat(path)
	if err != nil {
		return types.Message{}, os.IsNotExist(err), err
	}
	if info.IsDir() {
		return types.Message{}, false, fmt.Errorf("is a directory")
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return types.Message{}, false, err
	}
	if len(data) == 0 {
		return types.Message{}, false, fmt.Errorf("empty image file")
	}

	return types.Message{
		Type:    types.ImageMessage,
		Content: storeImagePath(path),
		Data:    data,
	}, false, nil
}

func storeImagePath(path string) string {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return filepath.ToSlash(path)
	}
	absPath = filepath.Clean(absPath)
	if repoRoot := project.Root(); repoRoot != "" {
		if rel, err := filepath.Rel(repoRoot, absPath); err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			return filepath.ToSlash(rel)
		}
	}
	return filepath.ToSlash(absPath)
}
