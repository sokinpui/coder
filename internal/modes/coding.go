package modes

import (
	"coder/internal/prompt"
	"coder/internal/types"
	"coder/internal/config"
	"coder/internal/source"
	"fmt"
)

// CodingMode is the strategy for the standard coding assistant mode.
type CodingMode struct {
	projectSourceCode string
}

// GetRolePrompt returns the coding role.
func (m *CodingMode) GetRolePrompt() string {
	return prompt.CodingRole
}

// LoadSourceCode loads context from the Context/ directory and project source files.
func (m *CodingMode) LoadSourceCode(cfg *config.Config) error {
	projSource, srcErr := source.LoadProjectSource(&cfg.Sources)
	if srcErr != nil {
		return fmt.Errorf("failed to load project source: %w", srcErr)
	}

	m.projectSourceCode = projSource
	return nil
}

// StartGeneration begins a new AI generation task using the default logic.
func (m *CodingMode) StartGeneration(s SessionController) types.Event {
	return StartGeneration(s, nil)
}

// BuildPrompt constructs the prompt for coding mode.
func (m *CodingMode) BuildPrompt(messages []types.Message) string {
	return BuildPrompt(PromptSectionArray{
		Sections: []PromptSection{
			RoleSection(m.GetRolePrompt(), prompt.CoderInstructions),
			ProjectSourceCodeSection(m.projectSourceCode),
			ConversationHistorySection(messages),
		},
	})
}
