package commands

import (
	"github.com/sokinpui/coder/internal/config"
	"fmt"
	"strings"
	"github.com/sokinpui/coder/internal/types"
)

var commands = make(map[string]commandFunc)
var commandArgumentCompleters = make(map[string]argumentCompleter)

// splitPipedCommands splits a string by the pipe symbol '|', respecting quotes and escapes.
func splitPipedCommands(input string) []string {
	var parts []string
	var currentPart strings.Builder
	inSingleQuote := false
	inDoubleQuote := false
	isEscaped := false

	for _, char := range input {
		if isEscaped {
			currentPart.WriteRune(char)
			isEscaped = false
			continue
		}

		if char == '\\' {
			isEscaped = true
			currentPart.WriteRune(char)
			continue
		}

		if char == '\'' && !inDoubleQuote {
			inSingleQuote = !inSingleQuote
		} else if char == '"' && !inSingleQuote {
			inDoubleQuote = !inDoubleQuote
		}

		if char == '|' && !inSingleQuote && !inDoubleQuote {
			parts = append(parts, currentPart.String())
			currentPart.Reset()
		} else {
			currentPart.WriteRune(char)
		}
	}
	parts = append(parts, currentPart.String())
	return parts
}

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

// processPipedCommands handles the logic for executing a series of commands linked by pipes.
func processPipedCommands(trimmedInput string, s SessionController) (CommandOutput, bool) {
	commandParts := splitPipedCommands(trimmedInput)
	var lastOutput CommandOutput
	var lastSuccess = true

	for i, part := range commandParts {
		part = strings.TrimSpace(part)
		if part == "" {
			return CommandOutput{Type: types.MessagesUpdated, Payload: "Invalid pipe syntax: empty command."}, false
		}

		var pipedArgs string
		if i > 0 { // For commands after the first one in the pipe
			if !lastSuccess {
				return CommandOutput{Type: types.MessagesUpdated, Payload: "Error: previous command in pipe failed."}, false
			}
			if lastOutput.Type != types.MessagesUpdated {
				return CommandOutput{Type: types.MessagesUpdated, Payload: "Error: command output is not pipeable."}, false
			}
			pipedArgs = lastOutput.Payload
		}

		parts := strings.Fields(part)
		if len(parts) == 0 {
			return CommandOutput{Type: types.MessagesUpdated, Payload: "Invalid command syntax."}, false
		}
		cmdName := parts[0]
		argsFromPart := strings.Join(parts[1:], " ")

		finalArgs := argsFromPart
		if pipedArgs != "" {
			// Normalize whitespace and newlines from the piped output to pass as arguments.
			normalizedPipedArgs := strings.Join(strings.Fields(pipedArgs), " ")
			if finalArgs != "" {
				finalArgs += " " + normalizedPipedArgs
			} else {
				finalArgs = normalizedPipedArgs
			}
		}

		cmd, exists := commands[cmdName]
		if !exists {
			return CommandOutput{Type: types.MessagesUpdated, Payload: fmt.Sprintf("Unknown command: %s", cmdName)}, false
		}

		lastOutput, lastSuccess = cmd(finalArgs, s)
	}
	return lastOutput, lastSuccess
}

// ProcessCommand tries to execute a command from the input string.
// It returns the result and a boolean indicating if it was a command.
func ProcessCommand(input string, s SessionController) (result CommandOutput, isCmd bool, success bool) {
	if !strings.HasPrefix(input, ":") {
		return CommandOutput{}, false, false // Not a command
	}
	trimmedInput := strings.TrimPrefix(input, ":")
	if strings.Contains(trimmedInput, "|") {
		result, success = processPipedCommands(trimmedInput, s)
		return result, true, success
	}

	// No pipe, original logic
	parts := strings.Fields(trimmedInput)
	if len(parts) == 0 {
		return CommandOutput{Type: types.MessagesUpdated, Payload: "Invalid command syntax. Use :<command> [args]"}, true, false
	}

	cmdName := parts[0]
	args := strings.Join(parts[1:], " ")

	cmd, exists := commands[cmdName]
	if !exists {
		return CommandOutput{Type: types.MessagesUpdated, Payload: fmt.Sprintf("Unknown command: %s", cmdName)}, true, false
	}

	result, success = cmd(args, s)
	return result, true, success
}
