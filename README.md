# Coder

A simple one-step AI code editor.

Coder is a TUI-based AI chat tool designed for terminal-centric workflows. It supports any OpenAI-compatible GenAI service (OpenAI, Groq, Ollama, vLLM, DeepSeek, Gemini via proxy, etc.) — just bring your API key and the server base URL.

## Core Philosophy

Coder is **not an autonomous agent**. It does not crawl your codebase, make executive decisions, or run loops in the background.

Instead, it is a **human-in-the-loop code editor**. You are the driver:

- **Manual Context**: You choose exactly which files or directories to provide to the AI using `:file` or `:tree`. ( load all by default )
- **Precise Guidance**: You guide the AI through prompts to perform specific tasks.
- **One-Step Application**: Coder interprets the AI's response to apply changes directly to your filesystem.

## How it Works

Coder uses a specialized output format to bridge the gap between chat and code. When the AI suggests changes, it can:

- **Edit**: Apply Unified Diff format to existing files.
- **Create / Delete / Rename**: Handle file lifecycle operations through specific Markdown blocks.

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

```sh
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
  url: https://api.openai.com/v1 # Base URL of the provider
  apikey: "" # (Optional) Leave empty to use env var

generation:
  modelcode: gpt-4o # The model ID used for chat
  titlemodelcode: gpt-4o-mini # The model ID used for session titles
```

### API Key

We recommend setting your API key via an environment variable for security:

```bash
export CODER_API_KEY="your-api-key-here"
```

## User Guide

### Global Shortcuts

| Shortcut       | Action                                                                                        |
| :------------- | :-------------------------------------------------------------------------------------------- |
| `Ctrl+J`       | Send message / Submit command                                                                 |
| `Ctrl+E`       | Edit current prompt in external editor (`$EDITOR`)                                            |
| `Ctrl+V`       | Paste from clipboard (supports images)                                                        |
| `Ctrl+A`       | Apply code changes from the last AI response (via [itf](https://github.com/sokinpui/itf.git)) |
| `Ctrl+H`       | View conversation history                                                                     |
| `Ctrl+N`       | Start a new chat session                                                                      |
| `Ctrl+F`       | Open command finder (fuzzy search all commands)                                               |
| `Ctrl+T`       | Open file tree to select context                                                              |
| `Ctrl+P`       | Fuzzy search current conversation                                                             |
| `Ctrl+Q`       | Jump to a specific user message                                                               |
| `Ctrl+L`       | Quick view of current project context (files read by AI)                                      |
| `Ctrl+U` / `D` | Scroll conversation view up / down                                                            |
| `Esc`          | Enter **Visual Mode**                                                                         |
| `Ctrl+C`       | Clear input (or double press on empty line to quit)                                           |
| `Tab`          | Autocomplete commands and arguments                                                           |

### Commands

Commands are prefixed with a colon `:`. You can pipe commands together using `|`.

- `:file [paths...]`: Add specific files or directories to the AI's context.
- `:exclude [paths...]`: Remove paths from the context.
- `:tree`: Interactive file tree for selecting context.
- `:list`: Show a summary of files currently in context.
- `:list-all`: Show a detailed list of every file being read by the AI.
- `:itf`: Manually trigger the code application tool on the last response.
- `:model [name]`: Switch the generation model on the fly.
- `:temp [0.0-2.0]`: Adjust the sampling temperature.
- `:new`: Reset the session but keep current configuration.
- `:history`: Browse and load previous conversations.
- `:rename [title]`: Manually set the session title.

### Visual Mode

Press `Esc` to enter Visual Mode. This allows you to interact with previous messages:

- `j` / `k`: Move cursor between messages.
- `v`: Select multiple messages.
- `y`: Yank (copy) selected messages to clipboard.
- `d`: Delete selected messages from the session.
- `g`: Regenerate the conversation starting from the selected message.
- `e`: Edit a previous user message and re-run the thread.
- `b`: Branch the conversation into a new session from the selected point.
- `Ctrl+A`: Apply code changes from the nearest AI response above the cursor.

## Configuration

On first run, a default `config.yaml` is created at `~/.config/coder/config.yaml`.

You can also create a project-specific configuration at `.coder/config.yaml` in your repository's root. This will override the global settings.
