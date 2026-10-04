package commands

import (
	"fmt"
	"github.com/sokinpui/coder/internal/engine"
	"github.com/sokinpui/coder/internal/types"
	"strings"
)

func init() {
	registerCommand("help", helpCmd, "show help message", nil)
}

type helpEntry struct {
	key  string
	desc string
	cap  engine.Capability
}

type helpGroup []helpEntry

type helpSection struct {
	name  string
	group helpGroup
}

var behaviorGroup = helpGroup{
	{key: "Code Read by AI", desc: "Markdown files are not read by AI by default, you would need `/file` let AI read them.", cap: engine.CapContextFiles},
}

var commandGroup = helpGroup{
	{key: "active", desc: "View active chat sessions."},
	{key: "branch", desc: "Enter branch mode to branch from a message.", cap: engine.CapBranch},
	{key: "chat", desc: "Switch conversation mode to chat.", cap: engine.CapContextFiles},
	{key: "clear_context", desc: "Clear all context files and documents.", cap: engine.CapContextFiles},
	{key: "coding", desc: "Switch conversation mode to coding.", cap: engine.CapContextFiles},
	{key: "config", desc: "Print the current configuration."},
	{key: "edit", desc: "Enter edit mode to edit a user prompt."},
	{key: "exclude", desc: "Exclude a file/directory from the project source.", cap: engine.CapContextFiles},
	{key: "file", desc: "Add file(s) to context (alias: @)."},
	{key: "gen", desc: "Enter generate mode to re-generate a response.", cap: engine.CapRegenerate},
	{key: "help", desc: "Show this help message."},
	{key: "history", desc: "View conversation history."},
	{key: "itf", desc: "Pipe the last AI response to `itf` for applying changes.", cap: engine.CapITF},
	{key: "list", desc: "List the current project source files/directories.", cap: engine.CapContextFiles},
	{key: "model", desc: "Switch generation model (e.g., /model gemini-2.5-pro).", cap: engine.CapModelSwitch},
	{key: "msg", desc: "Open atomic messages overlay."},
	{key: "new", desc: "Start a new chat session."},
	{key: "q", desc: "Quit the application."},
	{key: "quit", desc: "Quit the application."},
	{key: "rename", desc: "Rename the current session title."},
	{key: "shell", desc: "Run interactive terminal command or open subshell (alias: !).", cap: engine.CapShell},
	{key: "undo", desc: "Undo the last file changes applied by itf.", cap: engine.CapITF},
}

var globalGroup = helpGroup{
	{key: "Ctrl+J", desc: "Send message."},
	{key: "Ctrl+E", desc: "Edit prompt in external editor ($EDITOR)."},
	{key: "Ctrl+V", desc: "Paste from clipboard (supports images)."},
	{key: "Ctrl+H", desc: "View conversation history."},
	{key: "Ctrl+N", desc: "Start a new chat session."},
	{key: "Ctrl+B", desc: "Enter branch mode.", cap: engine.CapBranch},
	{key: "Ctrl+T", desc: "Toggle tool expansion (compact/expanded).", cap: engine.CapToolToggle},
	{key: "Ctrl+T", desc: "Search files/dirs and add to context (/file).", cap: engine.CapContextFiles},
	{key: "Ctrl+F", desc: "Search context files and open in editor.", cap: engine.CapContextFiles},
	{key: "Ctrl+L", desc: "Quick view of project context (/list).", cap: engine.CapContextFiles},
	{key: "Ctrl+A", desc: "Apply last AI response with `itf`.", cap: engine.CapITF},
	{key: "Ctrl+U / D", desc: "Scroll conversation view up / down."},
	{key: "Ctrl+Z", desc: "Suspend the application."},
	{key: "Tab", desc: "Autocomplete commands and arguments."},
	{key: "Esc", desc: "Open atomic messages overlay."},
	{key: "Ctrl+C", desc: "Clear input, or double press on empty line to quit."},
}

var atomicMsgGroup = helpGroup{
	{key: "j / k", desc: "Move cursor between atomic messages."},
	{key: "gg / G", desc: "Go to top / bottom."},
	{key: "v", desc: "Toggle multi-message selection for copy/delete."},
	{key: "o / O", desc: "Swap cursor and anchor in multi-selection."},
	{key: "y", desc: "Yank (copy) selected message(s) to clipboard."},
	{key: "d", desc: "Delete selected message(s)."},
	{key: "a", desc: "Apply code changes from AI response with itf.", cap: engine.CapITF},
	{key: "e", desc: "Edit selected user message in external editor."},
	{key: "r", desc: "Regenerate conversation starting from message.", cap: engine.CapRegenerate},
	{key: "b", desc: "Branch conversation into a new session.", cap: engine.CapBranch},
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

	caps := s.Capabilities()
	for _, section := range helpPageDesc {
		var filtered helpGroup
		for _, item := range section.group {
			if item.cap != 0 && !caps.Has(item.cap) {
				continue
			}
			filtered = append(filtered, item)
		}

		if len(filtered) == 0 {
			continue
		}

		fmt.Fprintf(&b, "\n%s:\n", section.name)
		for _, item := range filtered {
			if section.name == "Command" {
				fmt.Fprintf(&b, "  /%-11s %s\n", item.key, item.desc)
			} else {
				fmt.Fprintf(&b, "  %-12s %s\n", item.key, item.desc)
			}
		}
	}

	return CommandOutput{Type: types.MessagesUpdated, Payload: strings.TrimSpace(b.String())}, true
}
