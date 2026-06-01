package commands

import (
	"fmt"
	"github.com/sokinpui/coder/internal/config"
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

func IsBuiltIn(name string) bool {
	_, exists := commands[name]
	return exists
}

func GetCommandArgumentSuggestions(cmdName string, cfg *config.Config, prefix string) []string {
	if completer, ok := commandArgumentCompleters[cmdName]; ok {
		return completer(cfg, prefix)
	}
	return nil
}

func GetCommands() []string {
	commandNames := make([]string, 0, len(commands))
	for name := range commands {
		commandNames = append(commandNames, name)
	}
	return commandNames
}

func errorOutput(msg string) (CommandOutput, bool) {
	return CommandOutput{Type: types.MessagesUpdated, Payload: msg}, false
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

	cmd, exists := commands[cmdName]
	if exists {
		result, success = cmd(args, s)
		return result, true, success
	}
	return CommandOutput{Type: types.MessagesUpdated, Payload: fmt.Sprintf("Unknown command: %s", cmdName)}, true, false
}
