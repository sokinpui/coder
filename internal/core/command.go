package core

import (
	"fmt"
	"os/exec"
	"strings"

	"coder/internal/config"
)

// NewSessionResult is a special string returned by the /new command
// to signal the UI to start a new session.
const NewSessionResult = "---_NEW_SESSION_---"

// CopyModeResult signals the UI to enter visual copy mode.
const CopyModeResult = "---_COPY_MODE_---"

// DeleteModeResult signals the UI to enter visual delete mode.
const DeleteModeResult = "---_DELETE_MODE_---"

// GenerateModeResult signals the UI to enter visual generate mode.
const GenerateModeResult = "---_GENERATE_MODE_---"

type commandFunc func(args string, messages []Message, cfg *config.Config) (string, bool)

var commands = map[string]commandFunc{
	"echo":  echoCmd,
	"model": modelCmd,
	"itf":   itfCmd,
	"new":   newCmd,
	"mode":  modeCmd,
	"gen":   genCmd,
	"copy":   copyModeCmd,
	"delete": deleteModeCmd,
}

type argumentCompleter func(cfg *config.Config) []string

var commandArgumentCompleters = map[string]argumentCompleter{
	"model": modelArgumentCompleter,
	"mode":  modeArgumentCompleter,
}

func modelArgumentCompleter(cfg *config.Config) []string {
	return config.AvailableModels
}

func modeArgumentCompleter(cfg *config.Config) []string {
	modes := make([]string, len(config.AvailableAppModes))
	for i, m := range config.AvailableAppModes {
		modes[i] = string(m)
	}
	return modes
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

func newCmd(args string, messages []Message, cfg *config.Config) (string, bool) {
	return NewSessionResult, true
}

func genCmd(args string, messages []Message, cfg *config.Config) (string, bool) {
	return GenerateModeResult, true
}

func copyModeCmd(args string, messages []Message, cfg *config.Config) (string, bool) {
	return CopyModeResult, true
}

func deleteModeCmd(args string, messages []Message, cfg *config.Config) (string, bool) {
	return DeleteModeResult, true
}

func itfCmd(args string, messages []Message, cfg *config.Config) (string, bool) {
	var lastAIResponse string
	found := false
	for i := len(messages) - 1; i >= 0; i-- {
		if messages[i].Type == AIMessage {
			lastAIResponse = messages[i].Content
			found = true
			break
		}
	}

	if !found {
		return "No AI response found to pipe to itf.", false
	}

	argSlice := strings.Fields(args)
	cmd := exec.Command("itf", argSlice...)
	cmd.Stdin = strings.NewReader(lastAIResponse)
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
		fmt.Fprint(&b, "Usage: :model <model_name>")
		return b.String(), true
	}

	for _, m := range config.AvailableModels {
		if m == args {
			cfg.Generation.ModelCode = args
			return fmt.Sprintf("Switched model to: %s", args), true
		}
	}

	return fmt.Sprintf("Error: model '%s' not found. Use ':model' to see available models.", args), false
}

func modeCmd(args string, messages []Message, cfg *config.Config) (string, bool) {
	if args == "" {
		var b strings.Builder
		fmt.Fprintf(&b, "Current mode: %s\n", cfg.AppMode)
		fmt.Fprintln(&b, "Available modes:")
		for _, m := range config.AvailableAppModes {
			fmt.Fprintf(&b, "- %s\n", m)
		}
		fmt.Fprint(&b, "Usage: :mode <mode_name>")
		return b.String(), true
	}

	requestedMode := config.AppMode(args)
	for _, m := range config.AvailableAppModes {
		if m == requestedMode {
			cfg.AppMode = requestedMode
			return fmt.Sprintf("Switched mode to: %s", args), true
		}
	}

	return fmt.Sprintf("Error: mode '%s' not found. Use ':mode' to see available modes.", args), false
}

func echoCmd(args string, messages []Message, cfg *config.Config) (string, bool) {
	return args, true
}

// ProcessCommand tries to execute a command from the input string.
// It returns the result and a boolean indicating if it was a command.
func ProcessCommand(input string, messages []Message, cfg *config.Config) (result string, isCmd bool, success bool) {
	if !strings.HasPrefix(input, ":") {
		return "", false, false
	}

	parts := strings.Fields(strings.TrimPrefix(input, ":"))
	if len(parts) == 0 {
		return "Invalid command syntax. Use :<command> [args]", true, false
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
