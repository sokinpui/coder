package core

import (
	"os/exec"
	"strings"
)

func pcatAction(args string) (string, bool) {
	argSlice := strings.Fields(args)
	cmd := exec.Command("pcat", argSlice...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "Error executing pcat: " + err.Error() + "\n" + string(output), false
	}
	return string(output), true
}

// actionFunc defines the signature for an action.
// It returns the output string and a boolean indicating success.
type actionFunc func(args string) (string, bool)

var actions = map[string]actionFunc{
	"pcat": pcatAction,
}

// GetActions returns a slice of available action names.
func GetActions() []string {
	actionNames := make([]string, 0, len(actions))
	for name := range actions {
		actionNames = append(actionNames, name)
	}
	return actionNames
}

// ProcessAction tries to execute an action from the input string.
// It returns the result and a boolean indicating if it was an action.
func ProcessAction(input string) (result string, isAction bool, success bool) {
	if !strings.HasPrefix(input, ":") {
		return "", false, false
	}

	parts := strings.Fields(strings.TrimPrefix(input, ":"))
	if len(parts) == 0 {
		// It's a colon command, but malformed. We can't tell if it's an action or command.
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
	result, success = action(args)
	return result, true, success
}
