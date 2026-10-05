# Co - Autonomous Coding Agent

[简体中文](README.zh.md)

`co` is an autonomous terminal coding agent powered by an iterative LLM tool-calling loop. Unlike `coder`, which relies on human-guided diff applications, `co` autonomously explores the workspace, reads files, runs commands, and edits source code in real time.

## CLI Usage

```bash
co                         # Start interactive agent TUI in current directory
co "Fix test failures"     # Start agent with initial task
co -p "Refactor package"   # Explicit initial prompt flag
co -m gpt-4o               # Override agent generation model
co -i "Custom rules"       # Override agent system instruction
co -P responses            # Set API protocol ("responses" or "chat")
co --completion bash       # Generate autocompletion script
```

## Agent Tooling & Safety Loop

`co` executes an autonomous loop up to `max_iterations` (default: 50) using the following built-in tools:

- **`read`**: Reads files from the filesystem.
  - Plain text and source code: Returns raw content.
  - Images (`.png`, `.jpg`, `.jpeg`, `.webp`): Loaded into multi-modal vision context.
  - PDF documents (`.pdf`): Rendered into vision context pages on demand.
- **`edit`**: Replaces text blocks within files using exact string matching (`old_string` -> `new_string`).
  - **Guard Clause**: Rejects any edit attempt on a file that has not already been inspected with `read` in the current session.
  - Requires `old_string` to be unambiguous and unique.
- **`write`**: Creates new files or overwrites existing files. Automatically creates missing parent directories.
- **`bash`**: Executes shell commands on the host system with a 60-second default timeout and captured combined stdout/stderr.

## Loading Files Directly

Inside the `co` TUI, you can manually feed files into the agent's context using `@` or `/file`:

```text
@pkg/itf/patcher.go explain the hunk matching logic
/file internal/config/config.go
```

This immediately triggers the `read` tool and injects the file content into the conversation history.

## Shortcuts

| Shortcut       | Action                                                          |
| :------------- | :-------------------------------------------------------------- |
| `Ctrl+J`       | Send message / Submit command                                   |
| `Ctrl+E`       | Edit prompt in external editor (`$EDITOR`)                      |
| `Ctrl+V`       | Paste text or images from clipboard                             |
| `Ctrl+T`       | Toggle tool call display mode (**Compact** vs. **Expanded**)    |
| `Ctrl+H`       | Open conversation history selector                             |
| `Ctrl+N`       | Start a new agent session                                       |
| `Ctrl+B`       | Open Atomic Messages overlay for branching                     |
| `Ctrl+U` / `D` | Scroll conversation half-page up / down                         |
| `Ctrl+Z`       | Suspend application                                             |
| `Esc`          | Open **Atomic Messages** overlay                                |
| `Ctrl+C`       | Cancel active agent loop / Clear input                          |
| `Tab`          | Autocomplete commands and file arguments                        |

## Configuration

Configure the agent under the `agent:` block in `config.yaml`:

```yaml
agent:
  modelcode: aisrp/gemini-flash-lite-latest
  reasoningeffort: high
  max_iterations: 50
  keymap:
    submit: ctrl+j
    editor: ctrl+e
```
