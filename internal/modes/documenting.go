package modes

import (
	"coder/internal/prompt"
	"coder/internal/types"
	"coder/internal/config"
	"coder/internal/source"
	"fmt"
)

// DocumentingMode is the strategy for the documentation generation mode.
type DocumentingMode struct {
	projectSourceCode string
}

// GetRolePrompt returns the documenting role.
func (m *DocumentingMode) GetRolePrompt() string {
	return prompt.DocumentingRole
}

// LoadSourceCode loads context and project source, including markdown files.
func (m *DocumentingMode) LoadSourceCode(cfg *config.Config) error {

	projSource, srcErr := source.LoadProjectSource(&cfg.Sources)
	if srcErr != nil {
		return fmt.Errorf("failed to load project source: %w", srcErr)
	}

	m.projectSourceCode = projSource
	return nil
}

// StartGeneration begins a new AI generation task using the default logic.
func (m *DocumentingMode) StartGeneration(s SessionController) types.Event {
	return StartGeneration(s, nil)
}

// BuildPrompt constructs the prompt for documenting mode.
func (m *DocumentingMode) BuildPrompt(messages []types.Message) string {
	return BuildPrompt(PromptSectionArray{
		Sections: []PromptSection{
			RoleSection(m.GetRolePrompt(), prompt.CoderInstructions),
			ProjectSourceCodeSection(m.projectSourceCode),
			ConversationHistorySection(messages),
		},
	})
}
