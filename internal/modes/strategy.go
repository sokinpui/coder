package modes

import (
	"github.com/sokinpui/coder/internal/config"
	"github.com/sokinpui/coder/internal/types"
)

type ModeStrategy interface {
	GetRolePrompt() string

	LoadSourceCode(cfg *config.Config) error

	StartGeneration(s SessionController) types.Event

	BuildPrompt(messages []types.Message) []types.Message
}
