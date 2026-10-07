package commands

import (
	"fmt"
	"strings"

	"github.com/sokinpui/coder/internal/engine"
	"github.com/sokinpui/coder/internal/types"
)

func init() {
	registerCommand("mode", modeCmd, "view or switch conversation mode (coder, chat, agent)", modeArgumentCompleter)
	registerCommand("coder", coderCmd, "switch conversation mode to coder", nil)
	registerCommand("chat", chatCmd, "switch conversation mode to chat", nil)
	registerCommand("agent", agentCmd, "switch conversation mode to agent", nil)
}

func modeArgumentCompleter(s SessionController, prefix string) []string {
	return []string{engine.ModeCoder, engine.ModeChat, engine.ModeAgent}
}

func modeCmd(args string, s SessionController) (CommandOutput, bool) {
	target := strings.TrimSpace(args)
	if target == "" {
		return CommandOutput{
			Type:    types.MessagesUpdated,
			Payload: fmt.Sprintf("Current mode: %s\nAvailable modes: %s, %s, %s", s.GetMode(), engine.ModeCoder, engine.ModeChat, engine.ModeAgent),
		}, true
	}
	norm := engine.NormalizeMode(target)
	if norm != engine.ModeCoder && norm != engine.ModeChat && norm != engine.ModeAgent {
		return CommandOutput{
			Type:    types.MessagesUpdated,
			Payload: fmt.Sprintf("Unknown mode '%s'. Valid modes: %s, %s, %s", target, engine.ModeCoder, engine.ModeChat, engine.ModeAgent),
		}, false
	}
	return switchMode(norm, s)
}

func coderCmd(args string, s SessionController) (CommandOutput, bool) {
	return switchMode(engine.ModeCoder, s)
}

func chatCmd(args string, s SessionController) (CommandOutput, bool) {
	return switchMode(engine.ModeChat, s)
}

func agentCmd(args string, s SessionController) (CommandOutput, bool) {
	return switchMode(engine.ModeAgent, s)
}

func switchMode(targetMode string, s SessionController) (CommandOutput, bool) {
	norm := engine.NormalizeMode(targetMode)
	if s.HasChatHistory() {
		return CommandOutput{
			Type:    types.MessagesUpdated,
			Payload: "Cannot switch mode in a non-empty session. Start a new session (/new [mode]) to switch modes.",
		}, false
	}

	if s.GetMode() == norm {
		return CommandOutput{
			Type:    types.MessagesUpdated,
			Payload: fmt.Sprintf("Already in %s mode.", norm),
		}, true
	}

	return CommandOutput{
		Type:    types.NewSessionStarted,
		Mode:    norm,
		Payload: fmt.Sprintf("Switched mode to: %s", norm),
	}, true
}
