# Coder

Coder is an interactive terminal-based AI assistant designed to help developers with code generation and understanding within their Git repositories. It leverages a gRPC-based AI generation service to provide intelligent responses and allows users to interact with their project's codebase through a command-line interface.

## Features

- **AI-Powered Code Generation:** Chat with an AI model to generate code, understand existing code, or get help with development tasks.
- **Contextual Understanding:** Automatically loads system instructions, user-provided documents from a `Context` directory, and project source code (via `git ls-files` and `pcat`) to provide relevant AI responses.
- **Interactive Terminal UI:** Built with [Charmbracelet Bubble Tea](https://github.com/charmbracelet/bubbletea) for a rich and responsive user experience.
- **Markdown Rendering:** AI responses are rendered in markdown for readability.
- **Action Execution:** Execute predefined actions like `pcat` (project-aware cat and `itf` (interactive test framework - placeholder/example) directly from the chat.
- **Internal Commands:** Manage AI models, copy AI responses, and view token counts with internal commands.
- **Conversation History:** Automatically saves chat history to a `.coder/history` directory within your Git repository.
- **Token Counting:** Provides real-time token count estimation for prompts.

## Getting Started

### Prerequisites

- Go (1.21 or higher)
- Git
- `pcat` and `itf` (or equivalent tools if actions are modified) command-line tools installed and available in your PATH.
- Access to an AI generation gRPC service (e.g., a local server or a cloud endpoint).

### Installation

1.  **Clone the repository:**

    ```bash
    git clone https://github.com/your-org/coder.git
    cd coder
    ```

2.  **Install Go dependencies:**

    ```bash
    go mod tidy
    ```

3.  **Generate gRPC code:**

    ```bash
    ./gen.sh
    ```

    This requires `protoc` to be installed and in your PATH.

4.  **Build the application:**
    ```bash
    go build -o coder cmd/coder/main.go
    ```

### Usage

Run the `coder` executable from within a Git repository:

```bash
./coder
```

#### Key Bindings:

- `Enter`: New line in your prompt (when not a command).
- `Ctrl+J`: Send your message to the AI.
- `Ctrl+D` / `Ctrl+U`: Scroll down / up through the conversation.
- `Esc`: Clear the current input.
- `Ctrl+C`: Clear the current input. Press `Ctrl+C` again to quit.
- `:`: Type `:` to bring up the command/action palette.
- `Tab`/`Shift+Tab`: Navigate the command/action palette.
- `Enter` (in palette): Select and insert a command/action from the palette.

#### Commands:

- `:model [model_name]`: View or switch the active AI model.
- `:copy`: Copy the last AI response to the clipboard.
- `:echo <text>`: Echoes the provided text.

#### Actions:

- `:pcat <args>`: Execute the `pcat` command.
- `:itf <args>`: Execute the `itf` command.

## Configuration

The application uses `internal/config/config.go` for default settings.
The gRPC server address and AI generation parameters can be configured.

## Context Directory

Place any `.md` or other relevant documentation files in a directory named `Context` at the root of your Git repository. These files will be automatically loaded and provided to the AI as "Provided Documents" for contextual awareness.

Example:

```
.
├── Context/
│   └── myProjectOverview.md
│   └── importantSchema.md
├── main.go
└── coder
```

## Logging

`coder` logs its activity to `coder.log` in the directory where it's run.

## Contributing

See `develop.md` for information on how to contribute to this project.
