package core

import (
	"fmt"
	"os/exec"
	"strings"

	"coder/internal/config"
)

type CommandResultType int

const (
	CommandResultString CommandResultType = iota
	CommandResultNewSession
	CommandResultGenerateMode
	CommandResultVisualMode
	CommandResultEditMode
	CommandResultBranchMode
	CommandResultHistoryMode
)

// CommandOutput is the structured result of a command execution.
type CommandOutput struct {
	Type    CommandResultType
	Payload string
}

// SessionChanger is an interface that allows commands to modify session state
// without creating a circular dependency between core and session packages.
type SessionChanger interface {
	SetTitle(title string)
}

type commandFunc func(args string, messages []Message, cfg *config.Config, sess SessionChanger) (CommandOutput, bool)

var commands = map[string]commandFunc{
	"model":   modelCmd,
	"itf":     itfCmd,
	"new":     newCmd,
	"mode":    modeCmd,
	"gen":     genCmd,
	"edit":    editModeCmd,
	"visual":  visualCmd,
	"branch":  branchCmd,
	"rename":  renameCmd,
	"history": historyCmd,
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

func newCmd(args string, messages []Message, cfg *config.Config, sess SessionChanger) (CommandOutput, bool) {
	return CommandOutput{Type: CommandResultNewSession}, true
}

func genCmd(args string, messages []Message, cfg *config.Config, sess SessionChanger) (CommandOutput, bool) {
	if !hasSelectableMessages(messages) {
		return CommandOutput{Type: CommandResultString, Payload: "Cannot enter generate mode: no messages to select."}, false
	}
	return CommandOutput{Type: CommandResultGenerateMode}, true
}

func editModeCmd(args string, messages []Message, cfg *config.Config, sess SessionChanger) (CommandOutput, bool) {
	if !hasSelectableMessages(messages) {
		return CommandOutput{Type: CommandResultString, Payload: "Cannot enter edit mode: no messages to select."}, false
	}
	return CommandOutput{Type: CommandResultEditMode}, true
}

func visualCmd(args string, messages []Message, cfg *config.Config, sess SessionChanger) (CommandOutput, bool) {
	if !hasSelectableMessages(messages) {
		return CommandOutput{Type: CommandResultString, Payload: "Cannot enter visual mode: no messages to select."}, false
	}
	return CommandOutput{Type: CommandResultVisualMode}, true
}

func branchCmd(args string, messages []Message, cfg *config.Config, sess SessionChanger) (CommandOutput, bool) {
	if !hasSelectableMessages(messages) {
		return CommandOutput{Type: CommandResultString, Payload: "Cannot enter branch mode: no messages to select."}, false
	}
	return CommandOutput{Type: CommandResultBranchMode}, true
}

func historyCmd(args string, messages []Message, cfg *config.Config, sess SessionChanger) (CommandOutput, bool) {
	return CommandOutput{Type: CommandResultHistoryMode}, true
}

// ExecuteItf runs the 'itf' command with the given content as stdin.
func ExecuteItf(content string, args string) (string, bool) {
	argSlice := strings.Fields(args)
	cmd := exec.Command("itf --no-animation", argSlice...)
	cmd.Stdin = strings.NewReader(content)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "Error executing itf: " + err.Error() + "\n" + string(output), false
	}
	return string(output), true
}

func itfCmd(args string, messages []Message, cfg *config.Config, sess SessionChanger) (CommandOutput, bool) {
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
		return CommandOutput{Type: CommandResultString, Payload: "No AI response found to pipe to itf."}, false
	}

	result, success := ExecuteItf(lastAIResponse, args)
	return CommandOutput{Type: CommandResultString, Payload: result}, success
}

func modelCmd(args string, messages []Message, cfg *config.Config, sess SessionChanger) (CommandOutput, bool) {
	if args == "" {
		var b strings.Builder
		fmt.Fprintf(&b, "Current model: %s\n", cfg.Generation.ModelCode)
		fmt.Fprintln(&b, "Available models:")
		for _, m := range config.AvailableModels {
			fmt.Fprintf(&b, "- %s\n", m)
		}
		fmt.Fprint(&b, "Usage: :model <model_name>")
		return CommandOutput{Type: CommandResultString, Payload: b.String()}, true
	}

	for _, m := range config.AvailableModels {
		if m == args {
			cfg.Generation.ModelCode = args
			return CommandOutput{Type: CommandResultString, Payload: fmt.Sprintf("Switched model to: %s", args)}, true
		}
	}

	return CommandOutput{Type: CommandResultString, Payload: fmt.Sprintf("Error: model '%s' not found. Use ':model' to see available models.", args)}, false
}

func modeCmd(args string, messages []Message, cfg *config.Config, sess SessionChanger) (CommandOutput, bool) {
	if args == "" {
		var b strings.Builder
		fmt.Fprintf(&b, "Current mode: %s\n", cfg.AppMode)
		fmt.Fprintln(&b, "Available modes:")
		for _, m := range config.AvailableAppModes {
			fmt.Fprintf(&b, "- %s\n", m)
		}
		fmt.Fprint(&b, "Usage: :mode <mode_name>")
		return CommandOutput{Type: CommandResultString, Payload: b.String()}, true
	}

	requestedMode := config.AppMode(args)
	for _, m := range config.AvailableAppModes {
		if m == requestedMode {
			cfg.AppMode = requestedMode
			return CommandOutput{Type: CommandResultString, Payload: fmt.Sprintf("Switched mode to: %s", args)}, true
		}
	}

	return CommandOutput{Type: CommandResultString, Payload: fmt.Sprintf("Error: mode '%s' not found. Use ':mode' to see available modes.", args)}, false
}

func renameCmd(args string, messages []Message, cfg *config.Config, sess SessionChanger) (CommandOutput, bool) {
	if args == "" {
		return CommandOutput{Type: CommandResultString, Payload: "Usage: :rename <new title>"}, false
	}
	sess.SetTitle(args)
	return CommandOutput{Type: CommandResultString, Payload: fmt.Sprintf("Session title renamed to: %s", args)}, true
}

// ProcessCommand tries to execute a command from the input string.
// It returns the result and a boolean indicating if it was a command.
func ProcessCommand(input string, messages []Message, cfg *config.Config, sess SessionChanger) (result CommandOutput, isCmd bool, success bool) {
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

	result, success = cmd(args, messages, cfg, sess)
	return result, true, success
}
