package commands

import (
	"fmt"
	"strings"
)

func init() {
	registerCommand("help", helpCmd, nil)
}

type helpEntry struct {
	key  string
	desc string
}

type helpGroup []helpEntry

type helpSection struct {
	name  string
	group helpGroup
}

var behaviorGroup = helpGroup{
	{key: "Code Read by AI", desc: "Markdown files are not read by AI by default, you would need `:file` let AI read them."},
}

var commandGroup = helpGroup{
	{key: "branch", desc: "Enter branch mode to branch from a message."},
	{key: "edit", desc: "Enter edit mode to edit a user prompt."},
	{key: "file", desc: "Set project source files/directories. If no arguments, then clears all."},
	{key: "gen", desc: "Enter generate mode to re-generate a response."},
	{key: "help", desc: "Show this help message."},
	{key: "history", desc: "View conversation history."},
	{key: "itf", desc: "Pipe the last AI response to `itf` for applying changes."},
	{key: "list", desc: "List the current project source files/directories."},
	{key: "mode", desc: "Switch application mode (e.g., :mode Coding)."},
	{key: "model", desc: "Switch generation model (e.g., :model gemini-2.5-pro)."},
	{key: "new", desc: "Start a new chat session."},
	{key: "q", desc: "Quit the application."},
	{key: "quit", desc: "Quit the application."},
	{key: "rename", desc: "Rename the current session title."},
	{key: "shell", desc: "Execute a shell command."},
	{key: "visual", desc: "Enter visual mode for message selection."},
}

var globalGroup = helpGroup{
	{key: "Ctrl+J", desc: "Send message."},
	{key: "Ctrl+E", desc: "Edit prompt in external editor ($EDITOR)."},
	{key: "Ctrl+V", desc: "Paste from clipboard (supports images)."},
	{key: "Ctrl+H", desc: "View conversation history."},
	{key: "Ctrl+N", desc: "Start a new chat session."},
	{key: "Ctrl+B", desc: "Enter branch mode."},
	{key: "Ctrl+F", desc: "Open command finder (fzf)."},
	{key: "Ctrl+A", desc: "Apply last AI response with `itf`."},
	{key: "Ctrl+U / D", desc: "Scroll conversation view up / down."},
	{key: "Ctrl+Z", desc: "Suspend the application."},
	{key: "Tab", desc: "Autocomplete commands and arguments."},
	{key: "Esc", desc: "Enter visual mode."},
	{key: "Ctrl+C", desc: "Clear input, or double press on empty line to quit."},
}

var visualModeGroup = helpGroup{
	{key: "j / k", desc: "Move cursor down / up."},
	{key: "v", desc: "Start/stop selection."},
	{key: "y", desc: "Yank (copy) selected messages."},
	{key: "d", desc: "Delete selected messages."},
	{key: "g", desc: "Regenerate from the selected user message."},
	{key: "e", desc: "Edit the selected user message."},
	{key: "b", desc: "Enter branch mode from the selected message."},
	{key: "Ctrl+A", desc: "Apply code changes from nearest AI response above."},
	{key: "i / Esc", desc: "Exit visual mode."},
}

var historyViewGropu = helpGroup{
	{key: "j / k", desc: "Move cursor down / up."},
	{key: "gg / G", desc: "Go to top / bottom."},
	{key: "Enter", desc: "Load selected conversation."},
	{key: "Esc", desc: "Close history view."},
}

var helpPageDesc = []helpSection{
	{name: "Behavior", group: behaviorGroup},
	{name: "Global", group: globalGroup},
	{name: "Command", group: commandGroup},
	{name: "Visual mode", group: visualModeGroup},
	{name: "Chat History", group: historyViewGropu},
}

func helpCmd(args string, s SessionController) (CommandOutput, bool) {
	var b strings.Builder

	fmt.Fprintln(&b, "Coder Help")
	fmt.Fprintln(&b)

	// Shortcuts
	fmt.Fprintln(&b, "Shortcuts:")

	for _, section := range helpPageDesc {
		fmt.Fprintf(&b, "\n%s:\n", section.name)
		for _, item := range section.group {
			fmt.Fprintf(&b, "  %-12s %s\n", item.key, item.desc)
		}
	}

	return CommandOutput{Type: CommandResultString, Payload: strings.TrimSpace(b.String())}, true
}
