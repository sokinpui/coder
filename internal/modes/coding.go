package modes

import (
	"coder/internal/config"
	"coder/internal/contextdir"
	"coder/internal/core"
	"coder/internal/source"
	"fmt"
)

// CodingMode is the strategy for the standard coding assistant mode.
type CodingMode struct{}

// GetRolePrompt returns the coding role.
func (m *CodingMode) GetRolePrompt() string {
	return core.CodingRole
}

// LoadContext loads context from the Context/ directory and project source files.
func (m *CodingMode) LoadContext() (string, string, string, error) {
	sysInstructions, docs, ctxErr := contextdir.LoadContext()
	if ctxErr != nil {
		return "", "", "", fmt.Errorf("failed to load context directory: %w", ctxErr)
	}

	projSource, srcErr := source.LoadProjectSource(config.CodingMode)
	if srcErr != nil {
		return "", "", "", fmt.Errorf("failed to load project source: %w", srcErr)
	}

	return sysInstructions, docs, projSource, nil
}

// ProcessAIResponse does nothing in coding mode.
func (m *CodingMode) ProcessAIResponse(s SessionController) core.Event {
	return core.Event{Type: core.NoOp}
}

// StartGeneration begins a new AI generation task using the default logic.
func (m *CodingMode) StartGeneration(s SessionController) core.Event {
	return StartGeneration(s)
}

// BuildPrompt constructs the prompt for coding mode.
func (m *CodingMode) BuildPrompt(systemInstructions, relatedDocuments, projectSourceCode string, messages []core.Message) string {
	return BuildPrompt(m.GetRolePrompt(), core.CoderInstructions, systemInstructions, relatedDocuments, projectSourceCode, messages)
}
