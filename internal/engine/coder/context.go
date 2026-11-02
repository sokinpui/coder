package coder

import (
	"fmt"
	"os"
	"time"

	"github.com/sokinpui/coder/internal/engine"
	coderprompt "github.com/sokinpui/coder/internal/engine/coder/prompt"
	"github.com/sokinpui/coder/internal/engine/source"
	"github.com/sokinpui/coder/internal/project"
	"github.com/sokinpui/coder/internal/types"
)

func (s *Session) NeedsContextReload() bool {
	if len(s.projectSourceFiles) == 0 && len(s.contextFiles) > 0 {
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
	return false
}

func (s *Session) LoadContext() error {
	if !s.NeedsContextReload() {
		return nil
	}

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
