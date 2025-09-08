package core

import (
	"fmt"
	"strings"

	"github.com/atotto/clipboard"
)

type commandFunc func(args string, messages []Message) (string, bool)

var commands = map[string]commandFunc{
	"echo": echoCmd,
	"copy": copyCmd,
}

func echoCmd(args string, messages []Message) (string, bool) {
	return args, true
}

func copyCmd(args string, messages []Message) (string, bool) {
	for i := len(messages) - 1; i >= 0; i-- {
		if messages[i].Type == AIMessage {
			contentToCopy := messages[i].Content
			if err := clipboard.WriteAll(contentToCopy); err != nil {
				return fmt.Sprintf("Error copying to clipboard: %v", err), false
			}
			return "Copied last AI response to clipboard.", true
		}
	}
	return "No AI response found to copy.", false
}

// ProcessCommand tries to execute a command from the input string.
// It returns the result and a boolean indicating if it was a command.
func ProcessCommand(input string, messages []Message) (result string, isCmd bool, success bool) {
	if !strings.HasPrefix(input, "/") {
		return "", false, false
	}

	parts := strings.Fields(strings.TrimPrefix(input, "/"))
	if len(parts) == 0 {
		return "Invalid command syntax. Use /<command> [args]", true, false
	}

	cmdName := parts[0]
	args := strings.Join(parts[1:], " ")

	cmd, exists := commands[cmdName]
	if !exists {
		return fmt.Sprintf("Unknown command: %s", cmdName), true, false
	}

	result, success = cmd(args, messages)
	return result, true, success
}
