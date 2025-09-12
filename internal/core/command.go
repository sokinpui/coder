package core

import (
	"coder/internal/config"
	"fmt"
	"os/exec"
	"strings"

	"github.com/atotto/clipboard"
)

// NewSessionResult is a special string returned by the /new command
// to signal the UI to start a new session.
const NewSessionResult = "---_NEW_SESSION_---"

// RegenerateResult is a special string returned by the /gen command
const RegenerateResult = "---_REGENERATE_---"

type commandFunc func(args string, messages []Message, cfg *config.Config) (string, bool)

var commands = map[string]commandFunc{
	"echo":  echoCmd,
	"copy":  copyCmd,
	"model": modelCmd,
	"itf":   itfCmd,
	"new":   newCmd,
	"gen":   genCmd,
}

// GetCommands returns a slice of available command names.
func GetCommands() []string {
	commandNames := make([]string, 0, len(commands))
	for name := range commands {
		commandNames = append(commandNames, name)
	}
	return commandNames
}

func newCmd(args string, messages []Message, cfg *config.Config) (string, bool) {
	return NewSessionResult, true
}

func genCmd(args string, messages []Message, cfg *config.Config) (string, bool) {
	return RegenerateResult, true
}

func itfCmd(args string, messages []Message, cfg *config.Config) (string, bool) {
	argSlice := strings.Fields(args)
	cmd := exec.Command("itf", argSlice...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "Error executing itf: " + err.Error() + "\n" + string(output), false
	}
	return string(output), true
}

func modelCmd(args string, messages []Message, cfg *config.Config) (string, bool) {
	if args == "" {
		var b strings.Builder
		fmt.Fprintf(&b, "Current model: %s\n", cfg.Generation.ModelCode)
		fmt.Fprintln(&b, "Available models:")
		for _, m := range config.AvailableModels {
			fmt.Fprintf(&b, "- %s\n", m)
		}
		fmt.Fprint(&b, "Usage: /model <model_name>")
		return b.String(), true
	}

	for _, m := range config.AvailableModels {
		if m == args {
			cfg.Generation.ModelCode = args
			return fmt.Sprintf("Switched model to: %s", args), true
		}
	}

	return fmt.Sprintf("Error: model '%s' not found. Use '/model' to see available models.", args), false
}

func echoCmd(args string, messages []Message, cfg *config.Config) (string, bool) {
	return args, true
}

func copyCmd(args string, messages []Message, cfg *config.Config) (string, bool) {
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
func ProcessCommand(input string, messages []Message, cfg *config.Config) (result string, isCmd bool, success bool) {
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

	result, success = cmd(args, messages, cfg)
	return result, true, success
}
