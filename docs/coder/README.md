# Coder - User Manual

[简体中文](README.zh.md)

Coder is an interactive TUI-based, human-in-the-loop AI code editor designed for terminal workflows. It operates on a deterministic, one-step edit model: you provide exact file context, the AI suggests unified diffs and file lifecycle operations, and you inspect and apply them with one keystroke via `itf`.

## CLI Usage

```bash
coder [files...]          # Launch interactive TUI with specified files in context
coder -c                  # Launch interactive TUI in chat mode (no files)
coder -a [prompt]         # Launch interactive TUI in autonomous agent mode
coder -p "prompt" [files] # Start session with an initial prompt
coder -e -p "prompt"      # Single-shot execution output directly to stdout
coder --config            # Open local (.coder/config.yaml) or global config in $EDITOR
coder --config -g         # Open global config (~/.config/coder/config.yaml)
coder --headless          # Run as headless JSON-RPC server over stdio
coder --port 9005         # Run headless JSON-RPC server on TCP port
coder --ws --port 9005    # Run headless WebSocket/HTTP server
```

## Core Workflow

1. **Curate Context**: Specify files directly as CLI arguments (`coder main.go internal/`), or add them dynamically inside the TUI with `/file <path>` or `@<path>`.
2. **Review Instructions**: Coder loads relevant files and directories while adhering to exclusion rules. PDF documents are converted to visual context.
3. **Generate Changes**: The model outputs modifications as unified diffs, file creates, deletes, or renames.
4. **Apply Changes**: Press `Ctrl+A` or execute `/itf` to parse and apply changes to your filesystem. If needed, `/undo` reverts the patch.

## Shortcuts

| Shortcut       | Action                                                           |
| :------------- | :--------------------------------------------------------------- |
| `Ctrl+J`       | Send message / Submit command                                    |
| `Ctrl+E`       | Edit prompt in external editor (`$EDITOR`)                       |
| `Ctrl+V`       | Paste from clipboard (supports plain text and images)            |
| `Ctrl+A`       | Apply code changes from the last AI response (via `itf`)         |
| `Ctrl+H`       | Open conversation history selector                               |
| `Ctrl+N`       | Start a new chat session                                         |
| `Ctrl+B`       | Open Atomic Messages overlay for branching                       |
| `Ctrl+F`       | Search context files and open in external editor                 |
| `Ctrl+T`       | Search project files and directories to add to context (`/file`) |
| `Ctrl+L`       | Quick view of current project context (`/list`)                  |
| `Ctrl+U` / `D` | Scroll conversation half-page up / down                          |
| `Ctrl+Z`       | Suspend application                                              |
| `Esc`          | Open **Atomic Messages** overlay                                 |
| `Ctrl+C`       | Clear prompt line (press twice on empty prompt to quit)          |
| `Tab`          | Autocomplete commands and path arguments                         |

## Commands

- `/file [paths...]` (alias: `@`): Add files or directories to active context.
- `/exclude [paths...]`: Remove paths from active context.
- `/list`: Display current project files and PDF documents in context.
- `/clear_context`: Remove all files and documents from context.
- `/itf [args]`: Apply diffs and changes from the last AI response.
- `/undo`: Revert the last file changes applied by `itf`.
- `/mode [name]`: View or switch conversation mode (coder, chat, agent), or open mode selector if empty.
- `/model [name]`: Switch generation model on the fly, or list available models.
- `/new [mode]`: Reset conversation and start new session, optionally specifying mode.
- `/history`: Browse and load saved session history.
- `/active`: List and switch between running active sessions.
- `/rename [title]`: Change current session title.
- `/shell [cmd]` (alias: `!`): Execute interactive terminal command or launch subshell.
- `/config [reload]`: View current configuration or reload from disk.
- `/help`: Display internal help screen.
- `/quit` (alias: `/q`): Exit Coder.

## Atomic Messages Overlay (`Esc`)

Press `Esc` to inspect individual messages in the conversation stack:

- `j` / `k`: Navigate between messages.
- `v`: Toggle multi-message selection.
- `o` / `O`: Swap cursor and anchor ends during multi-selection.
- `y`: Yank (copy) message text or image to clipboard.
- `d`: Delete message(s).
- `a`: Apply code changes from the nearest AI response above via `itf`.
- `e`: Edit selected user prompt in external editor (`$EDITOR`).
- `r`: Regenerate conversation starting from selected point.
- `b`: Branch conversation into a new session from selected point.
- `Esc` / `Ctrl+C`: Exit overlay.

## Configuration

Coder is configured via `config.yaml` located globally at `~/.config/coder/config.yaml` or locally at `.coder/config.yaml`:

```yaml
coder:
  modelcode: aisrp/gemini-flash-latest
  reasoningeffort: high
  context:
    files: []
    dirs: ["."]
    exclusions: []
  keymap:
    submit: ctrl+j
    editor: ctrl+e
    applyitf: ctrl+a
```
