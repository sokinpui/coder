package modes

import (
	"fmt"
	"github.com/sokinpui/coder/internal/prompt"
	"github.com/sokinpui/coder/internal/source"
	"github.com/sokinpui/coder/internal/types"
)

type CodingMode struct {
	projectSourceCode string
	instruction       string
}

func (m *CodingMode) GetRolePrompt() string {
	return ""
}

func (m *CodingMode) LoadSourceCode(files []string) error {
	projSource, srcErr := source.LoadProjectSource(files)
	if srcErr != nil {
		return fmt.Errorf("failed to load project source: %w", srcErr)
	}

	m.projectSourceCode = projSource
	return nil
}

func (m *CodingMode) StartGeneration(s SessionController) types.Event {
	return StartGeneration(s, nil)
}

func (m *CodingMode) BuildPrompt(messages []types.Message) []types.Message {
	instr := m.instruction
	if instr == "" {
		instr = prompt.CoderInstructions
	}

	var result []types.Message

	result = append(result, types.Message{
		Type:    types.InstructionMessage,
		Content: instr,
	})

	if m.projectSourceCode != "" {
		result = append(result, types.Message{
			Type:    types.SourceCodeMessage,
			Content: ProjectSourceCodeHeader + m.projectSourceCode,
		})
	}

	for _, msg := range messages {
		if msg.Type == types.ShellCmdMessage || msg.Type == types.ShellCmdResultMessage {
			if canSee, ok := msg.Metadata["canAISee"].(bool); !ok || !canSee {
				continue
			}
		}
		result = append(result, msg)
	}

	return result
}
