package ui

import (
	"fmt"
	"strings"
)

type commandFunc func(args string) string

var commands = map[string]commandFunc{
	"echo": echoCmd,
}

func echoCmd(args string) string {
	return args
}

// processCommand tries to execute a command from the input string.
// It returns the result and a boolean indicating if it was a command.
func processCommand(input string) (result string, isCmd bool, success bool) {
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

	return cmd(args), true, true
}
