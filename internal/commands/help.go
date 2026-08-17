package commands

import (
	"fmt"
	"github.com/sokinpui/coder/internal/types"
	"strings"
)

func init() {
	registerCommand("help", helpCmd, "show help message", nil)
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
	{key: "Code Read by AI", desc: "Markdown files are not read by AI by default, you would need `/file` let AI read them."},
	{key: "Shell Commands", desc: "You can define custom slash commands in your config.yaml using the 'shellcommands' section."},
}

var commandGroup = helpGroup{
	{key: "branch", desc: "Enter branch mode to branch from a message."},
	{key: "chat", desc: "Start a new chat session with no context/instructions."},
	{key: "config", desc: "Print the current configuration."},
	{key: "edit", desc: "Enter edit mode to edit a user prompt."},
	{key: "editor", desc: "Open file(s) in external editor (alias: /e)."},
	{key: "exclude", desc: "Exclude a file/directory from the project source."},
	{key: "file", desc: "Set project source files/directories. If no arguments, then clears all."},
	{key: "fzf", desc: "Open model switcher."},
	{key: "gen", desc: "Enter generate mode to re-generate a response."},
	{key: "help", desc: "Show this help message."},
	{key: "history", desc: "View conversation history."},
	{key: "itf", desc: "Pipe the last AI response to `itf` for applying changes."},
	{key: "list", desc: "List the current project source files/directories."},
	{key: "mode", desc: "Switch conversation mode (coding/chat)."},
	{key: "model", desc: "Switch generation model (e.g., /model gemini-2.5-pro)."},
	{key: "msg", desc: "Open atomic messages overlay."},
	{key: "new", desc: "Start a new chat session."},
	{key: "q", desc: "Quit the application."},
	{key: "quit", desc: "Quit the application."},
	{key: "rename", desc: "Rename the current session title."},
	{key: "undo", desc: "Undo the last file changes applied by itf."},
}

var globalGroup = helpGroup{
	{key: "Ctrl+J", desc: "Send message."},
	{key: "Ctrl+E", desc: "Edit prompt in external editor ($EDITOR)."},
	{key: "Ctrl+V", desc: "Paste from clipboard (supports images)."},
	{key: "Ctrl+H", desc: "View conversation history."},
	{key: "Ctrl+N", desc: "Start a new chat session."},
	{key: "Ctrl+B", desc: "Enter branch mode."},
	{key: "Ctrl+F", desc: "Open model switcher (fzf)."},
	{key: "Ctrl+L", desc: "Quick view of project context (/list)."},
	{key: "Ctrl+A", desc: "Apply last AI response with `itf`."},
	{key: "Ctrl+U / D", desc: "Scroll conversation view up / down."},
	{key: "Ctrl+Z", desc: "Suspend the application."},
	{key: "Tab", desc: "Autocomplete commands and arguments."},
	{key: "Esc", desc: "Open atomic messages overlay."},
	{key: "Ctrl+C", desc: "Clear input, or double press on empty line to quit."},
}

var atomicMsgGroup = helpGroup{
	{key: "j / k", desc: "Move cursor between atomic messages."},
	{key: "v", desc: "Toggle multi-message selection for copy/delete."},
	{key: "o / O", desc: "Swap cursor and anchor in multi-selection."},
	{key: "y", desc: "Yank (copy) selected message(s) to clipboard."},
	{key: "d", desc: "Delete selected message(s)."},
	{key: "a", desc: "Apply code changes from AI response with itf."},
	{key: "e", desc: "Edit selected user message in external editor."},
	{key: "r", desc: "Regenerate conversation starting from message."},
	{key: "b", desc: "Branch conversation into a new session."},
	{key: "Esc / Ctrl+C", desc: "Exit atomic messages overlay."},
}

var historyViewGroup = helpGroup{
	{key: "j / k", desc: "Move cursor down / up."},
	{key: "u / d", desc: "Half-page up / down."},
	{key: "Ctrl+U / D", desc: "Half-page up / down."},
	{key: "gg / G", desc: "Go to top / bottom."},
	{key: "/", desc: "Fuzzy search history."},
	{key: "Enter", desc: "Load selected conversation."},
	{key: "q / Esc", desc: "Close history view."},
}

var helpPageDesc = []helpSection{
	{name: "Behavior", group: behaviorGroup},
	{name: "Global", group: globalGroup},
	{name: "Command", group: commandGroup},
	{name: "Atomic Messages", group: atomicMsgGroup},
	{name: "Chat History", group: historyViewGroup},
}

func helpCmd(args string, s SessionController) (CommandOutput, bool) {
	var b strings.Builder

	fmt.Fprintln(&b, "Coder Help")
	fmt.Fprintln(&b)

	fmt.Fprintln(&b, "Shortcuts:")

	for _, section := range helpPageDesc {
		fmt.Fprintf(&b, "\n%s:\n", section.name)
		for _, item := range section.group {
			if section.name == "Command" {
				fmt.Fprintf(&b, "  /%-11s %s\n", item.key, item.desc)
			} else {
				fmt.Fprintf(&b, "  %-12s %s\n", item.key, item.desc)
			}
		}
	}

	return CommandOutput{Type: types.HelpViewerStarted, Payload: strings.TrimSpace(b.String())}, true
}
