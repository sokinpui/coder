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

func (m *CodingMode) BuildPrompt(messages []types.Message) string {
	instr := m.instruction
	if instr == "" {
		instr = prompt.CoderInstructions
	}

	return BuildPrompt(PromptSectionArray{
		Sections: []PromptSection{
			RoleSection(m.GetRolePrompt(), instr),
			ProjectSourceCodeSection(m.projectSourceCode),
			ConversationHistorySection(messages),
		},
	})
}
