package modes

import (
	"github.com/sokinpui/coder/internal/types"
)

type ModeStrategy interface {
	GetRolePrompt() string

	LoadSourceCode(files []string) error

	StartGeneration(s SessionController) types.Event

	BuildPrompt(messages []types.Message) []types.Message
}
