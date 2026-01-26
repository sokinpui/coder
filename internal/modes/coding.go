package modes

import (
	"github.com/sokinpui/coder/internal/config"
	"github.com/sokinpui/coder/internal/prompt"
	"github.com/sokinpui/coder/internal/source"
	"github.com/sokinpui/coder/internal/types"
	"fmt"
)

// CodingMode is the strategy for the standard coding assistant mode.
type CodingMode struct {
	projectSourceCode string
}

// GetRolePrompt returns the coding role.
func (m *CodingMode) GetRolePrompt() string {
	return ""
}

// LoadSourceCode loads context from the Context/ directory and project source files.
func (m *CodingMode) LoadSourceCode(cfg *config.Config) error {
	projSource, srcErr := source.LoadProjectSource(&cfg.Context)
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
