# Agent Mode - Autonomous Coding Agent

[简体中文](README.zh.md)

`coder -a` (or `coder --agent`) launches Coder in autonomous terminal coding agent mode powered by an iterative LLM tool-calling loop. Unlike Coder mode, which relies on human-guided diff applications, Agent mode autonomously explores the workspace, reads files, runs commands, and edits source code in real time.

## CLI Usage

```bash
coder -a                         # Start interactive agent TUI in current directory
coder -a "Fix test failures"     # Start agent with initial task
coder -a -p "Refactor package"   # Explicit initial prompt flag
coder -a -m gpt-4o               # Override agent generation model
coder -a -i "Custom rules"       # Override agent system instruction
coder -a -P responses            # Set API protocol ("responses" or "chat")
coder --completion bash          # Generate autocompletion script
```

## Agent Tooling & Safety Loop

The agent executes an autonomous loop up to `max_iterations` (default: 50) using the following built-in tools:

- **`read`**: Reads files from the filesystem.
  - Plain text and source code: Returns raw content.
  - Images (`.png`, `.jpg`, `.jpeg`, `.webp`): Loaded into multi-modal vision context.
  - PDF documents (`.pdf`): Rendered into vision context pages on demand.
- **`edit`**: Replaces text blocks within files using exact string matching (`old_string` -> `new_string`).
  - **Guard Clause**: Rejects any edit attempt on a file that has not already been inspected with `read` in the current session.
  - Requires `old_string` to be unambiguous and unique.
- **`write`**: Creates new files or overwrites existing files. Automatically creates missing parent directories.
- **`bash`**: Executes shell commands on the host system with a 60-second default timeout and captured combined stdout/stderr.

## Tool Permissions & Approval

Control tool execution security under `agent.permission` in your configuration. Each tool or pattern supports three actions:

- **`allow`**: Executes the tool immediately without prompting.
- **`ask`**: Pauses and presents an interactive confirmation modal before execution.
- **`deny`**: Blocks execution immediately and returns a permission-denied message to the agent.

### Approval Prompts

When a tool triggers an `ask` rule, an approval overlay appears with the following options:

- `y`: **Allow** — Grant permission for this specific execution.
- `a`: **Always Allow** — Whitelist this tool for the remainder of the session.
- `n` / `Esc`: **Deny** — Reject execution and inform the agent.

### Target Matching Rules

Permissions can be configured globally (`*`), per tool, or granularly by target:

- **`bash`**: Target matches the shell command string (e.g. `rm -rf *`, `npm test`).
- **`edit`, `write`, `read`**: Target matches the file path (e.g. `*.go`, `.env*`, `internal/*`).
- **Wildcards (`*`)**: Supported across paths and commands. When multiple patterns match, the longest and most specific rule takes precedence.
- **Default**: When no matching rule is specified, tools default to `allow`.

## Loading Files Directly

Inside the agent TUI, you can manually feed files into the agent's context using `@` or `/file`:

```text
@pkg/itf/patcher.go explain the hunk matching logic
/file internal/config/config.go
```

This immediately triggers the `read` tool and injects the file content into the conversation history.

## Shortcuts

| Shortcut       | Action                                                       |
| :------------- | :----------------------------------------------------------- |
| `Ctrl+J`       | Send message / Submit command                                |
| `Ctrl+E`       | Edit prompt in external editor (`$EDITOR`)                   |
| `Ctrl+V`       | Paste text or images from clipboard                          |
| `Ctrl+T`       | Toggle tool call display mode (**Compact** vs. **Expanded**) |
| `Ctrl+H`       | Open conversation history selector                           |
| `Ctrl+N`       | Start a new agent session                                    |
| `Ctrl+B`       | Open Atomic Messages overlay for branching                   |
| `Ctrl+U` / `D` | Scroll conversation half-page up / down                      |
| `Ctrl+Z`       | Suspend application                                          |
| `Esc`          | Open **Atomic Messages** overlay                             |
| `Ctrl+C`       | Cancel active agent loop / Clear input                       |
| `Tab`          | Autocomplete commands and file arguments                     |

## Configuration

Configure the agent under the `agent:` block in `config.yaml`:

```yaml
agent:
  modelcode: aisrp/gemini-flash-lite-latest
  reasoningeffort: high
  max_iterations: 50
  permission:
    bash: ask
    write:
      ".env*": deny
      "src/*": allow
      "*": ask
    edit:
      "*.go": allow
      "*": ask
  keymap:
    submit: ctrl+j
    editor: ctrl+e
```
