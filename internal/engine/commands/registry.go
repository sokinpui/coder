package commands

import (
	"fmt"
	"github.com/sokinpui/coder/internal/engine"
	"github.com/sokinpui/coder/internal/types"
	"strings"
)

var commands = make(map[string]commandFunc)
var commandDescriptions = make(map[string]string)
var commandArgumentCompleters = make(map[string]argumentCompleter)

func registerCommand(name string, fn commandFunc, desc string, completer argumentCompleter) {
	commands[name] = fn
	commandDescriptions[name] = desc
	if completer != nil {
		commandArgumentCompleters[name] = completer
	}
}

func GetCommandDescriptions() map[string]string {
	return commandDescriptions
}

func GetCommandArgumentSuggestions(cmdName string, s SessionController, prefix string) []string {
	if completer, ok := commandArgumentCompleters[cmdName]; ok {
		return completer(s, prefix)
	}
	return nil
}

func GetCommandsForSession(s engine.EngineSession) []string {
	caps := engine.Capability(0)
	if s != nil {
		caps = s.Capabilities()
	}
	commandNames := make([]string, 0, len(commands))
	for name := range commands {
		if !isCommandAllowed(name, caps) {
			continue
		}
		commandNames = append(commandNames, name)
	}
	return commandNames
}

func GetCommands() []string {
	commandNames := make([]string, 0, len(commands))
	for name := range commands {
		commandNames = append(commandNames, name)
	}
	return commandNames
}

func isCommandAllowed(cmdName string, caps engine.Capability) bool {
	switch cmdName {
	case "file", "exclude", "list", "chat", "coding":
		return caps.Has(engine.CapContextFiles)
	case "itf", "undo":
		return caps.Has(engine.CapITF)
	case "model":
		return caps.Has(engine.CapModelSwitch)
	case "shell":
		return caps.Has(engine.CapShell)
	case "branch":
		return caps.Has(engine.CapBranch)
	case "gen":
		return caps.Has(engine.CapRegenerate)
	default:
		return true
	}
}

func ProcessCommand(input string, s SessionController) (result CommandOutput, isCmd bool, success bool) {
	if !strings.HasPrefix(input, "/") {
		return CommandOutput{}, false, false // Not a command
	}
	trimmedInput := strings.TrimPrefix(input, "/")

	parts := strings.Fields(trimmedInput)
	if len(parts) == 0 {
		return CommandOutput{Type: types.MessagesUpdated, Payload: "Invalid command syntax. Use /<command> [args]"}, true, false
	}

	cmdName := parts[0]
	args := strings.Join(parts[1:], " ")

	caps := engine.Capability(0)
	if s != nil {
		caps = s.Capabilities()
	}

	cmd, exists := commands[cmdName]
	if exists && isCommandAllowed(cmdName, caps) {
		result, success = cmd(args, s)
		return result, true, success
	}
	return CommandOutput{Type: types.MessagesUpdated, Payload: fmt.Sprintf("Unknown command: %s", cmdName)}, true, false
}
