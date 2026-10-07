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
	case "file", "@":
		return caps.Has(engine.CapContextFiles) || caps.Has(engine.CapToolLoop)
	case "exclude", "list", "clear_context":
		return caps.Has(engine.CapContextFiles)
	case "itf", "undo":
		return caps.Has(engine.CapITF)
	case "model":
		return caps.Has(engine.CapModelSwitch)
	case "shell", "!":
		return caps.Has(engine.CapShell)
	case "branch":
		return caps.Has(engine.CapBranch)
	case "gen":
		return caps.Has(engine.CapRegenerate)
	default:
		return true
	}
}

func IsCommand(input string) bool {
	trimmed := strings.TrimSpace(input)
	if trimmed == "" || trimmed == "@" || trimmed == "/@" {
		return false
	}
	return strings.HasPrefix(trimmed, "/") || strings.HasPrefix(trimmed, "@") || strings.HasPrefix(trimmed, "!")
}

func ParseCommand(input string) (cmdName string, args string, isCmd bool) {
	trimmed := strings.TrimSpace(input)
	if trimmed == "" || trimmed == "@" || trimmed == "/@" {
		return "", "", false
	}

	if strings.HasPrefix(trimmed, "/@") {
		return "@", strings.TrimSpace(trimmed[2:]), true
	}
	if strings.HasPrefix(trimmed, "/!") {
		return "shell", strings.TrimSpace(trimmed[2:]), true
	}
	if strings.HasPrefix(trimmed, "@") {
		return "@", strings.TrimSpace(trimmed[1:]), true
	}
	if strings.HasPrefix(trimmed, "!") {
		return "shell", strings.TrimSpace(trimmed[1:]), true
	}
	if !strings.HasPrefix(trimmed, "/") {
		return "", "", false
	}

	trimmedCmd := strings.TrimPrefix(trimmed, "/")
	parts := strings.Fields(trimmedCmd)
	if len(parts) == 0 {
		return "", "", true
	}

	cmdName = parts[0]
	args = strings.TrimSpace(trimmedCmd[len(cmdName):])
	if cmdName == "!" {
		cmdName = "shell"
	}
	return cmdName, args, true
}

func ProcessCommand(input string, s SessionController) (result CommandOutput, isCmd bool, success bool) {
	cmdName, args, isCommand := ParseCommand(input)
	if !isCommand {
		return CommandOutput{}, false, false
	}
	if cmdName == "" {
		return CommandOutput{Type: types.MessagesUpdated, Payload: "Invalid command syntax. Use /<command> [args]"}, true, false
	}

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
