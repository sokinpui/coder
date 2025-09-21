# Usage Guide

This guide explains how to use the Coder TUI and Web UI applications. Both applications must be run from within a Git repository.

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
| `Ctrl+E`        | Edit the current prompt in an external editor (`$EDITOR`). |
| `Esc`           | Clear input or enter Visual Mode from empty input.   |
| `Ctrl+C`        | Clear input. Press again on empty input to quit.     |
| `Ctrl+D` / `Ctrl+U` | Scroll conversation view down/up.                    |
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
| `:edit`         | Enter Edit Mode to modify a previous prompt.         |
| `:branch`       | Enter Branch Mode to create a new session from a point. |
| `:visual`       | Enter Visual Mode for message selection.             |
| `:history`      | View and load past conversations.                    |

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

## Web UI (`coder-web`)

The Web UI provides a graphical interface with additional features like a file and Git browser.

### Starting the Web UI

Navigate to your project's root directory and run:

```sh
coder-web
```

The application will start a web server and print the URL to access in your browser, typically `http://localhost:<port>`.

### Interface Overview

- **Sidebar**: Provides navigation for creating a new chat, viewing history, and accessing the Code and Git browsers.
- **Top Bar**: Displays the current conversation title, token count, and controls for changing the mode and model.
- **Chat View**: The main area for conversation with the AI.
- **Code Browser**: A file tree of the project. You can view file contents directly in the UI.
- **Git Browser**: A viewer for the Git commit history and diffs.

### Features

- **Chat Interaction**: Send messages using the input box at the bottom. AI responses are rendered as Markdown.
- **Message Actions**: Hover over a message to access actions like Regenerate, Edit, Branch, and Delete.
- **Code Browser**: Navigate your project's file structure. Clicking a file displays its content. Markdown files are rendered, while code files are shown with syntax highlighting.
- **Git Browser**: Visualize the commit graph, view commit details, and inspect diffs in either a side-by-side or unified view.
- **History Management**: Load previous conversations from the history dialog, accessible via the sidebar.
