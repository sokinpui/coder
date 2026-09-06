package session

import (
	"fmt"
	"os"
	"time"

	"github.com/sokinpui/coder/internal/prompt"
	"github.com/sokinpui/coder/internal/source"
	"github.com/sokinpui/coder/internal/types"
	"github.com/sokinpui/coder/internal/utils"
)

func (s *Session) NeedsContextReload() bool {
	if s.projectSourceCode == "" && len(s.contextFiles) > 0 {
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
	if len(s.contextFiles) == 0 {
		s.projectSourceCode = ""
		s.contextLoadedAt = time.Now()
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
		s.contextLoadedAt = time.Now()
		return nil
	}
	s.projectSourceCode = prompt.ProjectSourceCodeHeader + projSource
	s.contextLoadedAt = time.Now()
	return nil
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
	case ModeChat:
		instr := s.instruction
		if instr == "" {
			instr = prompt.ChatInstructions
		}
		result = append(result, types.Message{Type: types.InstructionMessage, Content: instr})
		if s.projectSourceCode != "" {
			result = append(result, types.Message{Type: types.SourceCodeMessage, Content: s.projectSourceCode})
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
