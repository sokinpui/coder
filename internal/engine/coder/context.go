package coder

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

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
		info, err := os.Stat(doc)
		if err != nil || info.ModTime().After(s.contextLoadedAt) {
			return true
		}
	}
	return false
}

func (s *Session) LoadContext() error {
	s.loadDocumentContext()

	if len(s.contextFiles) == 0 {
		s.projectSourceFiles = nil
		s.contextLoadedAt = time.Time{}
		return nil
	}

	if !s.NeedsContextReload() {
		return nil
	}

	files, err := source.LoadProjectSourceFiles(s.contextFiles)
	if err != nil {
		return fmt.Errorf("failed to load project source: %w", err)
	}

	if len(files) == 0 {
		s.projectSourceFiles = nil
		s.contextLoadedAt = time.Time{}
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
		cleanPath := filepath.ToSlash(doc)
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

		msgs, err := pdf.RenderPDFToMessages(cleanPath, "")
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
	case ModeCoding:
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
	case ModeChat:
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
