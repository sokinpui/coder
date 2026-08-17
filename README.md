# Coder

A simple one-step AI code editor.

Coder is a TUI-based AI chat tool designed for terminal-centric workflows. It supports any OpenAI-compatible GenAI service.

Coder is **not an autonomous agent**.

You ask, AI edit the code.

- **Manual Context**: You choose exactly which files or directories to provide to the AI using `/file` or `/tree`.
- **Precise Guidance**: You guide the AI through prompts to perform specific tasks.
- **One-Step Application**: Coder interprets the AI's response to apply changes directly to your filesystem using `itf`.

## CLI Usage

Coder can be used as an interactive TUI or as a one-shot CLI tool.

```bash
coder [files...]          # Start TUI in coding mode with specified context
coder chat                # Start TUI in chat mode (no project context)
coder run -p "prompt"     # Execute a single AI request and output to shell
coder context [files...]  # Print the built prompt and context (for debugging)
coder apply [content]     # Apply code changes from piped input or argument
coder config -g           # Edit global configuration
```

## How it Works

Coder uses a specialized output format to bridge the gap between chat and code. When the AI suggests changes, it can:

- **Edit**: Apply Unified Diff format to existing files.
- **Create / Delete / Rename**: Handle file lifecycle operations through specific Markdown blocks.

## Coder Suite (Sub-Tools)

Coder relies on several highly optimized sub-tools that are designed to operate perfectly under the hood or as standalone CLI utilities for custom scripts:

- **[sf (Search Fast)](./pkg/sf/README.md)**: A blazing-fast directory walker/find tool (an alternative to `fd`) that respects `.gitignore` rules.
- **[pcat (Prompt Cat)](./pkg/pcat/README.md)**: A specialized concatenator to safely wrap files and code structures into formatted markdown blocks for LLM contexts.
- **[itf (Insert To File)](./pkg/itf/README.md)**: An "Insert To File" parser and patching utility that interprets edits, diffs, creations, renames, and deletions with local state undo/redo capabilities.

See the linked README files above for reference on how to use each tool independently.

## Installation

### Prerequisites

- **Common**: Go, Git.
- `pngpaste` (macOS) or `xclip` (Linux) or `wl-clipboard` (Linux Wayland) or [`sync-clip` (sync clipboard for ssh)](https://github.com/sokinpui/sync-clip) for image pasting.

### Installation

From source (recommended for correct versioning):

```bash
git clone https://github.com/sokinpui/coder.git
cd coder
./install.sh
```

Or via Go directly:

```bash
# To install sf
go install github.com/sokinpui/coder/cmd/sf@latest

# To install pcat
go install github.com/sokinpui/coder/cmd/pcat@latest

# To install itf
go install github.com/sokinpui/coder/cmd/itf@latest

# To install the main coder TUI
go install github.com/sokinpui/coder/cmd/coder@latest
```

### Usage

Run this command in your terminal

```bash
coder
```

## Configuration

### OpenAI Compatible Service

To use `coder` with your preferred provider, update your `config.yaml`:

```yaml
server:
  url: http://localhost:9001/v1 # Base URL of the OpenAI-compatible provider

generation:
  modelcode: gemini-3-flash-preview # The model ID used for chat
  titlemodelcode: gemini-2.5-flash-lite # The model ID used for session titles
  reasoning_effort: high # Reasoning effort for models that support it (minimal, low, medium, high)
```

### Custom Shell Commands

You can define custom commands in your `config.yaml` that execute shell scripts. These behave like built-in slash commands.

```yaml
shellcommands:
  test:
    description: "Run project tests"
    exec: "go test ./..."
    canAIsee: true
  grep:
    description: "Search in files"
    exec: "grep -r $1 ."
    canAIsee: true
```

- **exec**: The shell command to run. Use `$1`, `$2`, etc., for positional arguments.
- **canAIsee**: If `true`, the output is added to the conversation history, allowing the AI to see test results, file contents, or directory structures to assist in debugging or coding.
- **Usage**: Run them in the chat using `/[command_name] [args]`, e.g., `/grep Todo`.

### API Key

We recommend setting your API key via an environment variable for security:

```bash
export CODER_API_KEY="your-api-key-here"
```

## User Guide

### Global Shortcuts

| Shortcut       | Action                                                                                          |
| :------------- | :---------------------------------------------------------------------------------------------- |
| `Ctrl+J`       | Send message / Submit command                                                                   |
| `Ctrl+E`       | Edit current prompt in external editor (`$EDITOR`)                                              |
| `Ctrl+V`       | Paste from clipboard (supports images)                                                          |
| `Ctrl+A`       | Apply code changes from the last AI response (via [itf](https://github.com/sokinpui/coder.git)) |
| `Ctrl+H`       | View conversation history                                                                       |
| `Ctrl+N`       | Start a new chat session                                                                        |
| `Ctrl+F`       | Open model switcher                                                                             |
| `Ctrl+L`       | Quick view of current project context (files read by AI)                                        |
| `Ctrl+B`       | Branch the conversation into a new session                                                      |
| `Ctrl+U` / `D` | Scroll conversation view up / down                                                              |
| `Ctrl+Z`       | Suspend application                                                                             |
| `Esc`          | Enter **Visual Mode**                                                                           |
| `Ctrl+C`       | Clear input (or double press on empty line to quit)                                             |
| `Tab`          | Autocomplete commands and arguments                                                             |

### Commands

Commands are prefixed with a slash `/`.

- `/file [paths...]`: Add specific files or directories to the AI's context.
- `/exclude [paths...]`: Remove paths from the context.
- `/list`: Show a summary of files currently in context.
- `/undo`: Undo the last file changes applied by `itf`.
- `/itf`: Manually trigger the code application tool on the last response.
- `/model [name]`: Switch the generation model on the fly.
- `/new`: Reset the session but keep current configuration.
- `/history`: Browse and load previous conversations.
- `/active`: List and switch between active chat sessions.
- `/rename [title]`: Manually set the session title.
- `/fzf`: Open model switcher.
- `/config`: Open configuration file.
- `/editor [path]`: Open a file in external editor (alias: `/e`).
- `/branch`: Branch the conversation.
- `/edit`: Enter edit mode to modify a previous user message.
- `/help`: Show internal help documentation.
- `/quit`: Quit the application.

### Visual Mode

Press `Esc` to enter Visual Mode. This allows you to interact with previous messages:

- `j` / `k`: Move cursor between messages.
- `v`: Select multiple messages.
- `y`: Yank (copy) selected messages to clipboard.
- `d`: Delete selected messages from the session.
- `g`: Regenerate the conversation starting from the selected message.
- `e`: Edit a previous user message and re-run the thread.
- `b`: Branch the conversation into a new session from the selected point.
- `n`: Start a new chat session.
- `o`: Swap cursor position in selection.
- `i`: Exit visual mode.
- `Ctrl+A`: Apply code changes from the nearest AI response above the cursor.

## Configuration

Configuration files are not created automatically. You must explicitly create them using the command line:

- **Global Configuration**: Run `coder -g -c` to create and edit the global config at `~/.config/coder/config.yaml`.
- **Local Configuration**: Run `coder -c` within a Git repository to create and edit a project-specific config at `.coder/config.yaml`.

Local settings override global settings.

### OpenAI Compatible Service
