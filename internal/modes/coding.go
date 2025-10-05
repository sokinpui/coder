package modes

import (
	"coder/internal/contextdir"
	"coder/internal/core"
	"coder/internal/source"
	"coder/internal/utils"
	"fmt"
)

// CodingMode is the strategy for the standard coding assistant mode.
type CodingMode struct {
	systemInstructions string
	relatedDocuments   string
	projectSourceCode  string
}

// GetRolePrompt returns the coding role.
func (m *CodingMode) GetRolePrompt() string {
	return core.CodingRole
}

// LoadContext loads context from the Context/ directory and project source files.
func (m *CodingMode) LoadContext() error {
	sysInstructions, docs, ctxErr := contextdir.LoadContext()
	if ctxErr != nil {
		return fmt.Errorf("failed to load context directory: %w", ctxErr)
	}

	fileSources := &source.FileSources{
		FileDirs: []string{"."},
	}
	projSource, srcErr := source.LoadProjectSource(fileSources)
	if srcErr != nil {
		return fmt.Errorf("failed to load project source: %w", srcErr)
	}

	m.systemInstructions = sysInstructions
	m.relatedDocuments = docs
	m.projectSourceCode = projSource
	return nil
}

// ProcessAIResponse does nothing in coding mode.
func (m *CodingMode) ProcessAIResponse(s SessionController) core.Event {
	return core.Event{Type: core.NoOp}
}

// StartGeneration begins a new AI generation task using the default logic.
func (m *CodingMode) StartGeneration(s SessionController) core.Event {
	return StartGeneration(s, nil)
}

// BuildPrompt constructs the prompt for coding mode.
func (m *CodingMode) BuildPrompt(messages []core.Message) string {

	dirInfoContent := utils.GetDirInfoContent()
	dirInfoSection := PromptSection{
		Content:   dirInfoContent,
		Separator: "\n\n",
	}

	return BuildPrompt(PromptSectionArray{
		Sections: []PromptSection{
			RoleSection(m.GetRolePrompt(), core.CoderInstructions),
			dirInfoSection,
			SystemInstructionsSection(m.systemInstructions),
			RelatedDocumentsSection(m.relatedDocuments),
			ProjectSourceCodeSection(m.projectSourceCode),
			ConversationHistorySection(messages),
		},
	})
}
