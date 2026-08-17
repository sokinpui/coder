package session

import (
	"fmt"
	"github.com/sokinpui/coder/internal/prompt"
	"github.com/sokinpui/coder/internal/source"
	"github.com/sokinpui/coder/internal/types"
	"github.com/sokinpui/coder/internal/utils"
)

func (s *Session) LoadContext() error {
	if len(s.contextFiles) == 0 {
		s.projectSourceCode = ""
		return nil
	}

	projSource, err := source.LoadProjectSource(s.contextFiles)
	if err != nil {
		return fmt.Errorf("failed to load project source: %w", err)
	}

	s.projectSourceCode = projSource
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
			result = append(result, types.Message{Type: types.SourceCodeMessage, Content: prompt.ProjectSourceCodeHeader + s.projectSourceCode})
		}
	case ModeChat:
		if s.projectSourceCode != "" {
			result = append(result, types.Message{Type: types.SourceCodeMessage, Content: prompt.ProjectSourceCodeHeader + s.projectSourceCode})
		}
	default:
	}

	for _, msg := range messages {
		if !msg.CanSendToAI() {
			continue
		}
		result = append(result, msg)
		result = append(result, msg)
	}

	return result
}
