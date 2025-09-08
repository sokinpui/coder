package core

import (
	"strings"
)

type actionFunc func(args string) string

var actions = map[string]actionFunc{}

// ProcessAction tries to execute an action from the input string.
// It returns the result and a boolean indicating if it was an action.
func ProcessAction(input string) (result string, isAction bool, success bool) {
	if !strings.HasPrefix(input, "/") {
		return "", false, false
	}

	parts := strings.Fields(strings.TrimPrefix(input, "/"))
	if len(parts) == 0 {
		// It's a slash command, but malformed. We can't tell if it's an action or command.
		// Let's assume it's not an action and let ProcessCommand handle it.
		return "", false, false
	}

	actionName := parts[0]
	args := strings.Join(parts[1:], " ")

	action, exists := actions[actionName]
	if !exists {
		return "", false, false // Not an action, might be a command
	}

	// It is a known action.
	return action(args), true, true
}
