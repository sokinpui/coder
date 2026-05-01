package modes

import (
	"fmt"
	"github.com/sokinpui/coder/internal/config"
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

func (m *CodingMode) LoadSourceCode(cfg *config.Config) error {
	projSource, srcErr := source.LoadProjectSource(&cfg.Context)
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

	// System message contains instructions and source code context
	systemContent := instr
	if m.projectSourceCode != "" {
		systemContent += "\n\n" + ProjectSourceCodeHeader + m.projectSourceCode
	}

	result = append(result, types.Message{
		Type:    types.InitMessage, // We'll treat Init as System role in generator
		Content: systemContent,
	})

	result = append(result, messages...)
	return result
}
