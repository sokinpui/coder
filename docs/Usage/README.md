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
| `Ctrl+A`        | Apply the last AI response with the diff viewer (`:itf`). |
| `Ctrl+B`        | Branch the conversation from a specific point.       |
| `Enter`         | Insert a newline or execute a command.               |
| `Ctrl+E`        | Edit the current prompt in an external editor (`$EDITOR`). |
| `Ctrl+F`        | Open the command finder (`fzf`).                     |
| `Ctrl+N`        | Start a new chat session.                            |
| `Esc`           | Clear input or enter Visual Mode from empty input.   |
| `Ctrl+C`        | Clear input. Press again on empty input to quit.     |
| `Ctrl+D` / `Ctrl+U` | Scroll conversation view down/up.                    |
| `Ctrl+H`        | View and load past conversations.                    |
| `Tab` / `Shift+Tab` | Cycle through command/action completions.            |

### Commands

Commands start with a colon (`:`) and are entered in the input area.

| Command         | Description                                          |
| --------------- | ---------------------------------------------------- |
| `:new`          | Start a new chat session.                            |
| `:mode <name>`  | Switch the application mode (e.g., `Coding`, `Documenting`). |
| `:model <name>` | Switch the AI model.                                 |
| `:rename <title>`| Rename the current session title.                    |
| `:itf`          | Pipe the last AI response to the `itf` diff viewer.  |
| `:gen`          | Enter Generate Mode to re-run a previous prompt.     |
| `:file [paths...]`| Sets the project source context to the specified files/directories. If no paths are given, clears the context. |
| `:edit`         | Enter Edit Mode to modify a previous prompt.         |
| `:branch`       | Enter Branch Mode to create a new session from a point. |
| `:visual`       | Enter Visual Mode for message selection.             |
| `:history`      | View and load past conversations.                    |
| `:fzf`          | Open a fuzzy finder for commands.                    |

### Actions

Actions are commands that execute external tools.

| Action         | Description                                          |
| -------------- | ---------------------------------------------------- |
| `:pcat <file>` | Display a file with syntax highlighting.             |

### Visual Mode

Enter Visual Mode by pressing `Esc` on an empty input line. This mode allows you to select, copy, delete, or regenerate from messages in the history.

| Key | Action                               |
| --- | ------------------------------------ |
| `j`/`k` | Move cursor up/down.                 |
| `v`   | Start/end selection.                 |
| `y`   | Copy selected messages to clipboard. |
| `d`   | Delete selected messages.            |
| `g`   | Regenerate from the selected message.|
| `e`   | Edit the selected user prompt.       |
| `Esc` | Exit Visual Mode.                    |

