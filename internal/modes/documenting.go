package modes

import (
	"coder/internal/config"
	"coder/internal/contextdir"
	"coder/internal/core"
	"coder/internal/source"
	"fmt"
)

// DocumentingMode is the strategy for the documentation generation mode.
type DocumentingMode struct {
	systemInstructions string
	relatedDocuments   string
	projectSourceCode  string
}

// GetRolePrompt returns the documenting role.
func (m *DocumentingMode) GetRolePrompt() string {
	return core.DocumentingRole
}

// LoadContext loads context and project source, including markdown files.
func (m *DocumentingMode) LoadContext(cfg *config.Config) error {
	sysInstructions, docs, ctxErr := contextdir.LoadContext()
	if ctxErr != nil {
		return fmt.Errorf("failed to load context directory: %w", ctxErr)
	}

	projSource, srcErr := source.LoadProjectSource(&cfg.Sources)
	if srcErr != nil {
		return fmt.Errorf("failed to load project source: %w", srcErr)
	}

	m.systemInstructions = sysInstructions
	m.relatedDocuments = docs
	m.projectSourceCode = projSource
	return nil
}

// ProcessAIResponse does nothing in documenting mode.
func (m *DocumentingMode) ProcessAIResponse(s SessionController) core.Event {
	return core.Event{Type: core.NoOp}
}

// StartGeneration begins a new AI generation task using the default logic.
func (m *DocumentingMode) StartGeneration(s SessionController) core.Event {
	return StartGeneration(s, nil)
}

// BuildPrompt constructs the prompt for documenting mode.
func (m *DocumentingMode) BuildPrompt(messages []core.Message) string {
	return BuildPrompt(PromptSectionArray{
		Sections: []PromptSection{
			RoleSection(m.GetRolePrompt(), core.CoderInstructions),
			SystemInstructionsSection(m.systemInstructions),
			RelatedDocumentsSection(m.relatedDocuments),
			ProjectSourceCodeSection(m.projectSourceCode),
			ConversationHistorySection(messages),
		},
	})
}
