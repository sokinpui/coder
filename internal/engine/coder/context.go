package coder

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/sokinpui/coder/internal/engine"
	coderprompt "github.com/sokinpui/coder/internal/engine/coder/prompt"
	"github.com/sokinpui/coder/internal/engine/pdf"
	"github.com/sokinpui/coder/internal/engine/source"
	"github.com/sokinpui/coder/internal/project"
	"github.com/sokinpui/coder/internal/types"
)

func (s *Session) NeedsContextReload() bool {
	if len(s.projectSourceFiles) == 0 && len(s.contextFiles) > 0 {
		return true
	}
	if len(s.documentMessages) == 0 && len(s.contextDocuments) > 0 {
		return true
	}
	if len(s.documentMessages) > 0 && len(s.contextDocuments) == 0 {
		return true
	}
	if s.contextLoadedAt.IsZero() {
		return true
	}

	for _, file := range s.contextFiles {
		info, err := os.Stat(file)
		if err != nil || info.ModTime().After(s.contextLoadedAt) {
			return true
		}
	}
	for _, doc := range s.contextDocuments {
		cleanDoc := doc
		if idx := strings.Index(cleanDoc, " (pages:"); idx != -1 {
			cleanDoc = strings.TrimSpace(cleanDoc[:idx])
		}
		info, err := os.Stat(cleanDoc)
		if err != nil || info.ModTime().After(s.contextLoadedAt) {
			return true
		}
	}
	return false
}

func (s *Session) LoadContext() error {
	if !s.NeedsContextReload() {
		return nil
	}

	s.loadDocumentContext()

	if len(s.contextFiles) == 0 {
		s.projectSourceFiles = nil
		s.contextLoadedAt = time.Now()
		return nil
	}

	files, err := source.LoadProjectSourceFiles(s.contextFiles)
	if err != nil {
		return fmt.Errorf("failed to load project source: %w", err)
	}

	if len(files) == 0 {
		s.projectSourceFiles = nil
		s.contextLoadedAt = time.Now()
		return nil
	}
	s.projectSourceFiles = files
	s.contextLoadedAt = time.Now()
	return nil
}

func (s *Session) loadDocumentContext() {
	if len(s.contextDocuments) == 0 {
		s.documentMessages = nil
		return
	}

	if s.cachedDocMessages == nil {
		s.cachedDocMessages = make(map[string][]types.Message)
	}
	if s.docModTimes == nil {
		s.docModTimes = make(map[string]time.Time)
	}

	var allDocMsgs []types.Message
	for _, doc := range s.contextDocuments {
		docPath := doc
		pagesSpec := ""
		if idx := strings.Index(docPath, " (pages:"); idx != -1 {
			if endIdx := strings.Index(docPath[idx:], ")"); endIdx != -1 {
				pagesSpec = docPath[idx+8 : idx+endIdx]
			}
			docPath = strings.TrimSpace(docPath[:idx])
		}

		cleanPath := filepath.ToSlash(docPath)
		info, err := os.Stat(cleanPath)
		if err != nil {
			continue
		}

		cached, ok := s.cachedDocMessages[cleanPath]
		modTime, modOk := s.docModTimes[cleanPath]
		if ok && modOk && modTime.Equal(info.ModTime()) {
			allDocMsgs = append(allDocMsgs, cached...)
			continue
		}

		msgs, err := pdf.RenderPDFToMessages(cleanPath, pagesSpec)
		if err != nil {
			continue
		}
		s.cachedDocMessages[cleanPath] = msgs
		s.docModTimes[cleanPath] = info.ModTime()
		allDocMsgs = append(allDocMsgs, msgs...)
	}

	s.documentMessages = allDocMsgs
}

func (s *Session) BuildPrompt(messages []types.Message) []types.Message {
	var result []types.Message

	switch s.mode {
	case engine.ModeCoder:
		instr := s.instruction
		if instr == "" {
			instr = coderprompt.Instructions
		}
		result = append(result, types.Message{Type: types.InstructionMessage, Content: instr})
		if dirInfo := project.DirInfo(); dirInfo != "" {
			result = append(result, types.Message{Type: types.DirectoryMessage, Content: dirInfo})
		}
		if len(s.projectSourceFiles) > 0 {
			result = append(result, types.Message{Type: types.SourceCodeMessage, Content: coderprompt.ProjectSourceCodeHeader})
			for _, fileContent := range s.projectSourceFiles {
				result = append(result, types.Message{Type: types.SourceCodeMessage, Content: fileContent})
			}
		}
		result = append(result, s.documentMessages...)
	case engine.ModeChat:
		instr := s.instruction
		if instr == "" {
			instr = coderprompt.ChatInstructions
		}
		result = append(result, types.Message{Type: types.InstructionMessage, Content: instr})
		if len(s.projectSourceFiles) > 0 {
			result = append(result, types.Message{Type: types.SourceCodeMessage, Content: coderprompt.ProjectSourceCodeHeader})
			for _, fileContent := range s.projectSourceFiles {
				result = append(result, types.Message{Type: types.SourceCodeMessage, Content: fileContent})
			}
		}
		result = append(result, s.documentMessages...)
	default:
	}

	for _, msg := range messages {
		if !msg.CanSendToAI() {
			continue
		}
		result = append(result, msg)
	}

	return result
}
