package modes

import (
	"fmt"
	"github.com/sokinpui/coder/internal/source"
	"github.com/sokinpui/coder/internal/types"
)

type ChatMode struct {
	projectSourceCode string
}

func (m *ChatMode) GetRolePrompt() string {
	return ""
}

func (m *ChatMode) LoadSourceCode(files []string) error {
	projSource, srcErr := source.LoadProjectSource(files)
	if srcErr != nil {
		return fmt.Errorf("failed to load project source: %w", srcErr)
	}

	m.projectSourceCode = projSource
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

	if m.projectSourceCode != "" {
		result = append(result, types.Message{
			Type:    types.SourceCodeMessage,
			Content: ProjectSourceCodeHeader + m.projectSourceCode,
		})
	}

	for _, msg := range messages {
		if msg.Type == types.ShellCmdMessage || msg.Type == types.ShellCmdResultMessage {
			if canSee, ok := msg.Metadata["canAISee"].(bool); !ok || !canSee {
				continue
			}
		}
		result = append(result, msg)
	}

	return result
}
