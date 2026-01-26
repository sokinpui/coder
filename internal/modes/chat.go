package modes

import (
	"github.com/sokinpui/coder/internal/config"
	"github.com/sokinpui/coder/internal/types"
)

// ChatMode is a strategy for a pure chat session without project context or instructions.
type ChatMode struct{}

func (m *ChatMode) GetRolePrompt() string {
	return ""
}

func (m *ChatMode) LoadSourceCode(cfg *config.Config) error {
	// No source code loaded for pure chat mode.
	return nil
}

func (m *ChatMode) StartGeneration(s SessionController) types.Event {
	return StartGeneration(s, nil)
}

func (m *ChatMode) BuildPrompt(messages []types.Message) string {
	return BuildPrompt(PromptSectionArray{
		Sections: []PromptSection{
			ConversationHistorySection(messages),
		},
	})
}
