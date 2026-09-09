package session

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/sokinpui/coder/internal/pdf"
	"github.com/sokinpui/coder/internal/prompt"
	"github.com/sokinpui/coder/internal/source"
	"github.com/sokinpui/coder/internal/types"
	"github.com/sokinpui/coder/internal/utils"
)

func (s *Session) NeedsContextReload() bool {
	if s.projectSourceCode == "" && len(s.contextFiles) > 0 {
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
		s.projectSourceCode = ""
		s.contextLoadedAt = time.Time{}
		return nil
	}

	if !s.NeedsContextReload() {
		return nil
	}

	projSource, err := source.LoadProjectSource(s.contextFiles)
	if err != nil {
		return fmt.Errorf("failed to load project source: %w", err)
	}

	if projSource == "" {
		s.projectSourceCode = ""
		s.contextLoadedAt = time.Time{}
		return nil
	}
	s.projectSourceCode = prompt.ProjectSourceCodeHeader + projSource
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
			instr = prompt.CoderInstructions
		}
		result = append(result, types.Message{Type: types.InstructionMessage, Content: instr})
		if dirInfo := utils.GetDirInfoContent(); dirInfo != "" {
			result = append(result, types.Message{Type: types.DirectoryMessage, Content: dirInfo})
		}
		if s.projectSourceCode != "" {
			result = append(result, types.Message{Type: types.SourceCodeMessage, Content: s.projectSourceCode})
		}
		result = append(result, s.documentMessages...)
	case ModeChat:
		instr := s.instruction
		if instr == "" {
			instr = prompt.ChatInstructions
		}
		result = append(result, types.Message{Type: types.InstructionMessage, Content: instr})
		if s.projectSourceCode != "" {
			result = append(result, types.Message{Type: types.SourceCodeMessage, Content: s.projectSourceCode})
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
