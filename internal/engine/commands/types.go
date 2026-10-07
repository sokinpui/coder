package commands

import (
	"github.com/sokinpui/coder/internal/engine"
)

type CommandOutput = engine.CommandOutput

type SessionController interface {
	engine.EngineSession
}

type commandFunc func(args string, s SessionController) (CommandOutput, bool)

type argumentCompleter func(s SessionController, prefix string) []string
