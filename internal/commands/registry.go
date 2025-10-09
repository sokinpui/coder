package commands

import (
	"coder/internal/config"
	"fmt"
	"strings"
)

var commands = make(map[string]commandFunc)
var commandArgumentCompleters = make(map[string]argumentCompleter)

func registerCommand(name string, fn commandFunc, completer argumentCompleter) {
	commands[name] = fn
	if completer != nil {
		commandArgumentCompleters[name] = completer
	}
}

// GetCommandArgumentSuggestions returns suggestions for a command's arguments.
func GetCommandArgumentSuggestions(cmdName string, cfg *config.Config) []string {
	if completer, ok := commandArgumentCompleters[cmdName]; ok {
		return completer(cfg)
	}
	return nil
}

// GetCommands returns a slice of available command names.
func GetCommands() []string {
	commandNames := make([]string, 0, len(commands))
	for name := range commands {
		commandNames = append(commandNames, name)
	}
	return commandNames
}

// ProcessCommand tries to execute a command from the input string.
// It returns the result and a boolean indicating if it was a command.
func ProcessCommand(input string, s SessionController) (result CommandOutput, isCmd bool, success bool) {
	if !strings.HasPrefix(input, ":") {
		return CommandOutput{}, false, false
	}

	parts := strings.Fields(strings.TrimPrefix(input, ":"))
	if len(parts) == 0 {
		return CommandOutput{Type: CommandResultString, Payload: "Invalid command syntax. Use :<command> [args]"}, true, false
	}

	cmdName := parts[0]
	args := strings.Join(parts[1:], " ")

	cmd, exists := commands[cmdName]
	if !exists {
		return CommandOutput{Type: CommandResultString, Payload: fmt.Sprintf("Unknown command: %s", cmdName)}, true, false
	}

	result, success = cmd(args, s)
	return result, true, success
}
