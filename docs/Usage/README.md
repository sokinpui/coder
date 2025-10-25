# Usage Guide

This guide explains how to use the Coder TUI application. The application must be run from within a Git repository.

## Application Modes

Coder provides different modes to tailor the AI's behavior for specific tasks like coding or writing documentation. You can switch modes using the `:mode` command.

For a detailed explanation of each mode and how it affects the AI's context and responses, see the [Modes Guide](./Modes.md).

## TUI (`coder`)

The TUI provides a keyboard-centric interface for interacting with the AI.

### Starting the TUI

Navigate to your project's root directory (or any subdirectory within a Git repository) and run:

```sh
coder
```

### Interface Overview

- **Conversation View**: The main area displaying the chat history.
- **Input Area**: A text box at the bottom for entering prompts and commands.
- **Status Bar**: A two-line area at the bottom displaying the session title, token count, current mode/model, and other contextual information.

### Keybindings

| Key             | Action                                               |
| --------------- | ---------------------------------------------------- |
| `Ctrl+J`        | Send the message in the input area.                  |
| `Enter`         | Insert a newline or execute a command.               |
| `Tab` / `Shift+Tab` | Cycle through command/action completions.            |
| `Esc`           | Clear input or enter Visual Mode from empty input.   |
| `Ctrl+C`        | Clear input. Press again on empty input to quit.     |
| `Ctrl+A`        | Apply the last AI response with the diff viewer (`:itf`). |
| `Ctrl+B`        | Enter Branch Mode to start a new session from a point. |
| `Ctrl+E`        | Edit the current prompt in an external editor (`$EDITOR`). |
| `Ctrl+F`        | Open the command finder (`fzf`).                     |
| `Ctrl+H`        | View and load past conversations.                    |
| `Ctrl+N`        | Start a new chat session.                            |
| `Ctrl+P`        | Fuzzy search the current conversation.               |
| `Ctrl+Q`        | Show a quick view of the last few messages.          |
| `Ctrl+D` / `Ctrl+U` | Scroll conversation view down/up.                    |
| `Ctrl+V`        | Paste from clipboard (supports images).              |
| `Ctrl+Z`        | Suspend the application.                             |

### Commands

Commands start with a colon (`:`) and are entered in the input area.

| Command         | Description                                          |
| --------------- | ---------------------------------------------------- |
| `:branch`       | Enter Branch Mode to create a new session from a point. |
| `:edit`         | Enter Edit Mode to modify a previous prompt.         |
| `:file [paths...]`| Set specific source files/directories for context. No paths clears the list. |
| `:gen`          | Enter Generate Mode to re-run a previous prompt.     |
| `:help`         | Show the help message with all commands and keybindings. |
| `:history`      | View and load past conversations.                    |
| `:itf`          | Pipe the last AI response to the `itf` diff viewer.  |
| `:list`         | List the current project source files being read by the AI. |
| `:mode <name>`  | Switch the application mode (e.g., `Coding`, `Documenting`). |
| `:model <name>` | Switch the AI model.                                 |
| `:new`          | Start a new chat session.                            |
| `:q` / `:quit`  | Quit the application.                                |
| `:rename <title>`| Rename the current session title.                    |
| `:search <query>`| Fuzzy search the conversation for a specific query.  |
| `:shell <cmd>`  | Execute a shell command and see the output.          |
| `:temp <value>` | Set the generation temperature (e.g., 0.0 to 2.0).   |
| `:visual`       | Enter Visual Mode for message selection.             |

### Visual Mode

Enter Visual Mode by pressing `Esc` on an empty input line. This mode allows you to select, copy, delete, or regenerate from messages in the history.

| Key | Action                               |
| --- | ------------------------------------ |
| `j`/`k` | Move cursor up/down.                 |
| `o`/`O` | Swap cursor with selection start.    |
| `v`   | Start/end selection.                 |
| `y`   | Copy selected messages to clipboard. |
| `d`   | Delete selected messages.            |
| `g`   | Regenerate from the selected message.|
| `e`   | Edit the selected user prompt.       |
| `b`   | Branch the conversation from the selected message. |
| `Ctrl+A` | Apply code changes from the nearest AI response above the cursor. |
| `Esc` | Exit Visual Mode.                    |

