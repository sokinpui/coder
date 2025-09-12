# Coder - Developer Guide

This document provides an overview of the `coder` project's structure, key components, and internal workflows to help developers understand, maintain, and contribute to the codebase.

## Project Goal

`coder` is an interactive, AI-powered assistant designed to be run from within a Git repository. Its primary function is to assist developers by generating code, providing explanations, and interacting with the project context, leveraging a gRPC-based AI generation service.

## Project Structure

The project follows a standard Go project layout with a `cmd` directory for the main application entry point and an `internal` directory for private packages.

```
.
├── .gitignore               # Ignored files (grpc/, test.go, .coder/, context/)
├── README.md                # Project overview
├── arch.md                  # Project architecture and UI design
├── cmd/
│   └── coder/
│       └── main.go          # Application entry point
├── develop.md               # This document
├── gen.sh                   # Script to generate gRPC code
├── go.mod                   # Go module definition
├── go.sum                   # Go module checksums
├── grpc/                    # Generated gRPC code (ignored by git)
├── internal/
│   ├── config/              # Application configuration
│   │   └── config.go
│   ├── contextdir/          # Logic for loading documents from 'Context' directory
│   │   └── contextdir.go
│   ├── core/                # Core application logic
│   │   ├── SystemInstructions.md # Embedded AI system instructions
│   │   ├── action.go        # Defines external actions (e.g., pcat, itf)
│   │   ├── command.go       # Defines internal commands (e.g., model, copy)
│   │   ├── messages.go      # Defines message types and structures for conversation
│   │   ├── prompt.go        # Logic for building AI prompts
│   │   └── systemInstructions.go # Embeds SystemInstructions.md
│   ├── generation/          # Handles gRPC communication with AI service
│   │   └── generator.go
│   ├── history/             # Manages saving conversation history
│   │   └── history.go
│   ├── logger/              # Simple file-based logging
│   │   └── logger.go
│   ├── source/              # Logic for loading project source code via git
│   │   └── source.go
│   ├── token/               # Token counting utilities
│   │   └── token.go
│   ├── ui/                  # Terminal User Interface (TUI) implementation
│   │   ├── model.go         # The main Bubble Tea model (state)
│   │   ├── msg.go           # Custom Bubble Tea messages
│   │   ├── style.go         # Lipgloss styling definitions
│   │   ├── ui.go            # UI entry point and program setup
│   │   ├── update.go        # Bubble Tea update logic (state transitions)
│   │   └── utils.go         # UI utility functions
│   └── utils/               # General utility functions
│       └── utils.go
└── protos/
    └── generate.proto       # Protocol Buffer definition for AI generation service
```

## Key Packages and Components

### `cmd/coder/main.go`

The application's entry point. It performs initial checks (e.g., if inside a Git repository), initializes the logger, and starts the Bubble Tea UI.

### `internal/config`

Manages application-wide configuration, including gRPC server address and AI generation parameters (model, temperature, top-p, top-k, output length).

- **`Config` struct**: Holds `GRPC` and `Generation` substructs.
- **`Default()`**: Provides a sensible default configuration.

### `internal/contextdir`

Responsible for discovering and loading external `Context` files.

- **`LoadContext()`**: Walks the `Context` directory, reads files, and formats their content as provided documents for the AI.

### `internal/core`

Contains the core business logic, including AI system instructions, handling external actions, internal commands, conversation message structures, and prompt construction.

- **`SystemInstructions.md`**: Embedded markdown file containing the core directives for the AI.
- **`action.go`**:
  - `actionFunc` type: Defines the signature for executable actions.
  - `actions` map: Registers available external shell commands (e.g., `pcat`, `itf`).
  - `ProcessAction()`: Parses user input to identify and execute actions.
- **`command.go`**:
  - `commandFunc` type: Defines the signature for internal commands.
  - `commands` map: Registers internal application commands (e.g., `:model`, `:copy`).
  - `ProcessCommand()`: Parses user input to identify and execute internal commands.
- **`messages.go`**:
  - `MessageType` enum: Categorizes different types of messages in the conversation (User, AI, Action, Command, etc.).
  - `Message` struct: Represents a single entry in the conversation history with its type and content.
- **`prompt.go`**:
  - `BuildPrompt()`: Constructs the complete prompt string sent to the AI, combining system instructions, provided documents, project source, and conversation history. It adheres to a structured format for clarity.
- **`systemInstructions.go`**: Uses `//go:embed` to embed `SystemInstructions.md` into the binary.

### `internal/generation`

Manages the client-side communication with the gRPC AI generation service.

- **`Generator` struct**: Holds the gRPC client and generation configuration.
- **`New()`**: Establishes a gRPC connection.
- **`GenerateTask()`**: Sends a prompt to the AI service and streams the response back to a channel.

### `internal/history`

Handles the persistence of conversation history.

- **`Manager` struct**: Manages saving conversations.
- **`NewManager()`**: Initializes the history manager by finding the Git repository root and creating a `.coder/history` directory.
- **`SaveConversation()`**: Writes the formatted conversation (from `core.BuildPrompt`) to a markdown file with a timestamp in the history directory.

### `internal/logger`

Provides a simple file-based logger.

- **`Init()`**: Configures the global `log` package to write to `coder.log`.

### `internal/source`

Loads the project's source code by interacting with Git.

- **`LoadProjectSource()`**: Executes `git ls-tree --full-tree -r --name-only HEAD | pcat` to get a formatted view of the current project's tracked files.

### `internal/token`

Provides utilities for counting tokens in a given text.

- Uses `github.com/tiktoken-go/tokenizer` for accurate tokenization.
- **`CountTokens()`**: Returns the token count for a string.

### `internal/ui`

Implements the entire Terminal User Interface using the [Charmbracelet Bubble Tea](https://github.com/charmbracelet/bubbletea) framework.

- **`model.go`**:
  - **`Model` struct**: The central state of the UI. It holds `textarea.Model`, `viewport.Model`, `spinner.Model`, references to `Generator` and `history.Manager`, conversation `messages`, UI state (`state`), dimensions, `glamourRenderer`, configuration, loaded context (system instructions, provided documents, project source), token count, command palette state, and more.
  - **`NewModel()`**: Initializes all UI components and dependencies.
- **`msg.go`**: Defines custom `tea.Msg` types used for asynchronous communication within the Bubble Tea `Update` loop (e.g., `streamResultMsg`, `streamFinishedMsg`, `initialContextLoadedMsg`, `errorMsg`).
- **`style.go`**: Defines `lipgloss.Style` objects for consistent and visually appealing UI elements (e.g., `initMessageStyle`, `helpStyle`, `userInputStyle`).
- **`ui.go`**:
  - **`Start()`**: The entry point for the UI, sets up and runs the Bubble Tea program.
- **`update.go`**: Contains the `Update` method of the `Model`. This is the core logic for state transitions.
  - Handles all `tea.Msg` types, including user key presses (`tea.KeyMsg`), spinner ticks, stream results, context loading results, and window size changes.
  - Manages interaction with the `textarea`, `viewport`, and AI generation process.
  - Routes user input to `core.ProcessAction()` or `core.ProcessCommand()` or to the AI generation.
  - Manages the command palette's filtering and selection logic.
- **`utils.go`**: Provides helper functions for the UI, mostly for creating `tea.Cmd` to perform asynchronous tasks.
  - `listenForStream()`: Creates a `tea.Cmd` to read from the AI generation stream.
  - `countTokensCmd()`: Creates a `tea.Cmd` to calculate token count.
  - `loadInitialContextCmd()`: Creates a `tea.Cmd` to load initial context in parallel.
  - `renderTick()`: Creates a `tea.Cmd` for periodic UI redraws during streaming.
  - `ctrlCTimeout()`: Creates a `tea.Cmd` for the double Ctrl+C timeout.
- **`view.go`**: Contains the `View` method of the `Model`. This method is responsible for rendering the entire UI based on the current state.
  - `renderConversation()`: Formats and renders all messages in the conversation history, applying appropriate styles and markdown rendering for AI messages.
  - `paletteView()`: Renders the command/action palette.
  - `helpView()`: Renders the dynamic help and status bar.

### `internal/utils`

General utility functions used across different packages.

- **`FindRepoRoot()`**: Locates the root directory of the current Git repository using `git rev-parse`.

### `protos`

Contains the Protocol Buffer definition for the `Generate` service.

- **`generate.proto`**: Defines the `Generate` service, `Request`, `Response`, and `GenerationConfig` messages.

### `grpc`

This directory contains the Go code generated by `protoc` from `protos/generate.proto`. It provides the gRPC client and server stubs for interaction with the AI generation service.

## Contribution Guidelines

1.  **Fork and Clone:** Fork the repository and clone your fork.
2.  **Set up Development Environment:** Ensure Go, Git, `protoc`, and necessary external tools (`pcat`, `itf`) are installed.
3.  **Understand the Code:** Familiarize yourself with the package structure and component responsibilities outlined above.
4.  **Issue Tracking:** Look for open issues or propose new features/bug fixes.
5.  **Branching:** Create a new branch for your work (e.g., `feature/my-new-feature` or `bugfix/fix-issue-123`).
6.  **Code Changes:** Implement your changes, adhering to the existing coding style.
    - **Self-documenting code:** Prefer clear variable names, function names, and logical structure over extensive comments.
    - **Modularity:** Keep concerns separated within packages.
    - **Robustness:** Handle errors gracefully, especially in I/O and external command execution.
    - **Scalability/Reusability:** Design components that can be extended or reused.
7.  **Generate gRPC Code:** If you modify `protos/generate.proto`, run `./gen.sh` to regenerate the Go gRPC stubs.
8.  **Testing:** Write unit tests for new functionality or bug fixes.
9.  **Commit:** Commit your changes with clear, concise commit messages.
10. **Pull Request:** Push your branch to your fork and open a pull request against the `main` branch of the upstream repository. Explain your changes and their purpose.

### Working with the UI (`internal/ui`)

- **State Management:** All UI state is managed within the `Model` struct in `model.go`. Avoid global variables for state.
- **Immutability:** When updating state in `Update`, return a new `Model` or a pointer to a modified `Model`.
- **Messages (`tea.Msg`):** All interactions, external events, and asynchronous results should be communicated via `tea.Msg` objects. Define custom message types in `msg.go` for specific events.
- **Commands (`tea.Cmd`):** Use `tea.Cmd` for side effects and asynchronous operations. Functions in `ui/utils.go` are good examples of how to create commands.
- **Rendering (`View`):** The `View` method should be a pure function of the `Model`'s state. It should not modify the state.
- **Styling:** Use `lipgloss` for all UI styling. Define styles in `style.go` for consistency.

## External Dependencies

- `github.com/charmbracelet/bubbletea`: Core TUI framework.
- `github.com/charmbracelet/bubbles`: UI components (textarea, viewport, spinner).
- `github.com/charmbracelet/glamour`: Markdown renderer for AI output.
- `github.com/charmbracelet/lipgloss`: Styling library.
- `github.com/tiktoken-go/tokenizer`: Token counting.
- `github.com/atotto/clipboard`: Clipboard access.
- `google.golang.org/grpc`: gRPC framework.
- `google.golang.org/protobuf`: Protocol buffers.

By following these guidelines, contributors can ensure `coder` remains maintainable, robust, and continues to evolve as a powerful development tool.
