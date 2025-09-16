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

// GenerateModeResult signals the UI to enter visual generate mode.
const GenerateModeResult = "---_GENERATE_MODE_---"

// VisualModeResult signals the UI to enter visual mode.
const VisualModeResult = "---_VISUAL_MODE_---"

// EditModeResult signals the UI to enter visual edit mode.
const EditModeResult = "---_EDIT_MODE_---"

// BranchModeResult signals the UI to enter visual branch mode.
const BranchModeResult = "---_BRANCH_MODE_---"

type commandFunc func(args string, messages []Message, cfg *config.Config) (string, bool)

var commands = map[string]commandFunc{
	"model":  modelCmd,
	"itf":    itfCmd,
	"new":    newCmd,
	"mode":   modeCmd,
	"gen":    genCmd,
	"edit":   editModeCmd,
	"visual": visualCmd,
	"branch": branchCmd,
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

func hasSelectableMessages(messages []Message) bool {
	for _, msg := range messages {
		switch msg.Type {
		case InitMessage, DirectoryMessage:
			continue
		default:
			return true
		}
	}
	return false
}

func newCmd(args string, messages []Message, cfg *config.Config) (string, bool) {
	return NewSessionResult, true
}

func genCmd(args string, messages []Message, cfg *config.Config) (string, bool) {
	if !hasSelectableMessages(messages) {
		return "Cannot enter generate mode: no messages to select.", false
	}
	return GenerateModeResult, true
}

func editModeCmd(args string, messages []Message, cfg *config.Config) (string, bool) {
	if !hasSelectableMessages(messages) {
		return "Cannot enter edit mode: no messages to select.", false
	}
	return EditModeResult, true
}

func visualCmd(args string, messages []Message, cfg *config.Config) (string, bool) {
	if !hasSelectableMessages(messages) {
		return "Cannot enter visual mode: no messages to select.", false
	}
	return VisualModeResult, true
}

func branchCmd(args string, messages []Message, cfg *config.Config) (string, bool) {
	if !hasSelectableMessages(messages) {
		return "Cannot enter branch mode: no messages to select.", false
	}
	return BranchModeResult, true
}

// ExecuteItf runs the 'itf' command with the given content as stdin.
func ExecuteItf(content string, args string) (string, bool) {
	argSlice := strings.Fields(args)
	cmd := exec.Command("itf", argSlice...)
	cmd.Stdin = strings.NewReader(content)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "Error executing itf: " + err.Error() + "\n" + string(output), false
	}
	return string(output), true
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

	return ExecuteItf(lastAIResponse, args)
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
