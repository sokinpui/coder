package modes

import (
	"github.com/sokinpui/coder/internal/config"
	"github.com/sokinpui/coder/internal/types"
)

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

func (m *ChatMode) BuildPrompt(messages []types.Message) []types.Message {
	var result []types.Message
	role := m.GetRolePrompt()
	if role != "" {
		result = append(result, types.Message{
			Type:    types.InitMessage,
			Content: role,
		})
	}
	result = append(result, messages...)
	return result
}
