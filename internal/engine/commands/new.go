package commands

import (
	"fmt"
	"strings"

	"github.com/sokinpui/coder/internal/engine"
	"github.com/sokinpui/coder/internal/types"
)

func init() {
	registerCommand("new", newCmd, "start new session (e.g., /new [mode])", newArgumentCompleter)
}

func newArgumentCompleter(s SessionController, prefix string) []string {
	return []string{engine.ModeCoder, engine.ModeChat, engine.ModeAgent}
}

func newCmd(args string, s SessionController) (CommandOutput, bool) {
	mode := engine.ModeCoder
	if s != nil && s.GetMode() != "" {
		mode = s.GetMode()
	}
	target := strings.TrimSpace(args)
	if target != "" {
		norm := engine.NormalizeMode(target)
		if norm != engine.ModeCoder && norm != engine.ModeChat && norm != engine.ModeAgent {
			return CommandOutput{
				Type:    types.MessagesUpdated,
				Payload: fmt.Sprintf("Unknown mode '%s'. Valid modes: %s, %s, %s", target, engine.ModeCoder, engine.ModeChat, engine.ModeAgent),
			}, false
		}
		mode = norm
	}
	return CommandOutput{Type: types.NewSessionStarted, Mode: mode}, true
}
