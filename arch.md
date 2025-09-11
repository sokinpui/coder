# Coder - Architecture and Design

This document outlines the high-level architecture, modular design, and user interface (UI) design principles of the `coder` project.

## Overall Architecture

`coder` follows a client-server architecture, where the `coder` application itself acts as a rich terminal-based client that interacts with an external AI generation service via gRPC. The application is designed to be run from within a Git repository, allowing it to provide highly contextualized responses based on the project's codebase and user-defined documents.

```
+-------------------+      +-----------------------+      +-------------------------+
|     Git Repo      |      |                       |      |                         |
|  (Local Context)  |      |   Coder Application   |      |   AI Generation Service |
|                   |      |    (gRPC Client)      |      |    (gRPC Server)        |
| - Source Code     |      |                       |      |                         |
| - Context Dir     | <----| - Terminal UI (TUI)   | <--->| - Large Language Model  |
| - .coder/history  |      | - Command/Action Proc.|      | - Generation Logic      |
+-------------------+      | - Prompt Builder      |      |                         |
                           | - Context Loader      |      |                         |
                           | - History Manager     |      |                         |
                           +-----------------------+      +-------------------------+
                                 ^         ^
                                 |         |
                                 v         v
                             (External Tools: git, pcat, itf)
```

## Modular Design

The project is structured into several internal packages, each with a focused responsibility, promoting maintainability, reusability, and scalability.

- **`cmd/coder`**: The main application entry point. Responsible for initial setup, environment validation (Git repository check), logger initialization, and starting the UI.
- **`internal/config`**: Centralized management of application settings, including network addresses for services and AI generation parameters. This ensures consistent configuration across the application.
- **`internal/contextdir`**: Handles the discovery and loading of user-provided documentation from a `Context` directory. This abstraction separates file system interaction from core logic.
- **`internal/core`**: Contains the core business logic of the application, independent of the UI or external services.
  - **System Instructions**: Embedded directives for the AI, ensuring consistent behavior.
  - **Actions**: Definition and execution of external command-line tools (e.g., `pcat`, `itf`).
  - **Commands**: Definition and execution of internal application commands (e.g., `/model`, `/copy`).
  - **Messages**: Standardized data structures for representing conversation turns.
  - **Prompt Building**: Logic to assemble all contextual information into a coherent prompt for the AI.
- **`internal/generation`**: Acts as the gRPC client to communicate with the external AI generation service. It handles request/response serialization (using `protos`), streaming, and configuration of generation parameters.
- **`internal/history`**: Manages the persistence of user conversations. It's responsible for finding the repository root and saving chat logs in a structured format.
- **`internal/logger`**: A simple wrapper for file-based logging, centralizing logging configuration.
- **`internal/source`**: Interacts with Git to extract the project's source code, formatted using external tools like `pcat`. This provides the AI with awareness of the current codebase.
- **`internal/token`**: Provides utilities for token counting using `tiktoken-go`, essential for managing AI context windows and cost estimation.
- **`internal/ui`**: The entire Terminal User Interface implementation. This is the primary interaction layer for the user. Its design is detailed in the next section.
- **`internal/utils`**: A collection of general utility functions, such as finding the Git repository root.
- **`protos`**: Contains the Protocol Buffer definitions (`.proto` files) for the gRPC service, acting as the contract between `coder` and the AI generation service.
- **`grpc`**: Contains the auto-generated Go code from the `protos` definitions, providing the concrete gRPC client implementations.

## User Interface (UI) Design (`internal/ui`)

The UI is built using the [Charmbracelet Bubble Tea](https://github.com/charmbracelet/bubbletea) framework, which adheres to a functional, immutable model-view-update (MVU) architecture.

### MVU Architecture

- **`Model` (`model.go`)**: This is the core state of the application's UI. It's a single struct that holds everything necessary to describe the UI at any given moment:

  - Conversation history (`[]core.Message`)
  - Input `textarea` state
  - Conversation `viewport` state
  - Spinner animation state
  - References to `Generator`, `history.Manager`, and application `config`
  - Loaded context data (system instructions, provided documents, project source)
  - UI dimensions (`width`, `height`)
  - Various flags and states (e.g., `isStreaming`, `showPalette`, `quitting`, `state`).
  - The `NewModel` function initializes this state.

- **`Update` (`update.go`)**: This is a pure function that takes the current `Model` and an incoming `tea.Msg` (message) and returns a new `Model` and optionally a `tea.Cmd` (command).

  - **Message Handling**: It's a large switch statement that processes all types of messages: user input (`tea.KeyMsg`), events from asynchronous operations (`streamResultMsg`, `initialContextLoadedMsg`), timer ticks (`spinner.TickMsg`, `renderTickMsg`), etc.
  - **State Transitions**: Based on the message, it updates the `Model`'s fields, reflecting changes like adding a user message, appending AI response chunks, changing the UI state (e.g., `stateThinking` to `stateGenerating`), or updating the command palette.
  - **Side Effects**: Any operation that has side effects (e.g., calling the gRPC service, reading files, setting timers) is encapsulated in a `tea.Cmd` and returned for the Bubble Tea runtime to execute.

- **`View` (`view.go`)**: This is another pure function that takes the current `Model` and returns a string, which is the rendered UI.
  - **Composition**: It composes different parts of the UI (viewport, palette, textarea, help bar) into a single string.
  - **`renderConversation()`**: Iterates through the `messages` in the `Model`, formats them according to their `MessageType` (e.g., user input, AI response, action result), and applies `lipgloss` styles. `glamour` is used to render markdown in AI responses.
  - **`paletteView()`**: Renders the dynamic command/action palette, highlighting the currently selected item.
  - **`helpView()`**: Renders the status bar at the bottom, including context-sensitive help, current model, and token count.

### Custom Message Types (`msg.go`)

The UI heavily relies on custom `tea.Msg` types to communicate events and data back to the `Update` function from asynchronous operations (e.g., `streamResultMsg` for AI response chunks, `initialContextLoadedMsg` for loaded project context, `errorMsg` for failures).

### Styling (`style.go`)

- The `lipgloss` library is used for all UI styling. Styles are defined centrally in `style.go` as `lipgloss.Style` objects, ensuring consistency and easy modification of the application's look and feel.

### Asynchronous Operations (`utils.go`)

- Long-running or non-blocking tasks (like gRPC streaming, loading large files, or token counting) are executed in separate goroutines and communicate their results back to the `Update` loop via custom `tea.Msg` objects created by `tea.Cmd` functions.
- Utility functions in `utils.go` encapsulate the logic for creating these commands, making the `Update` function cleaner and focused on state transitions.

### User Interaction Flow

1.  **Initialization**: `ui.Start()` creates and runs a `tea.Program`. The `Model`'s `Init()` method triggers `loadInitialContextCmd()` to load system instructions, provided documents, and project source in parallel.
2.  **User Input**: The user types into the `textarea`. Key presses are sent as `tea.KeyMsg` to `Update`.
3.  **Command Palette**: Typing `/` in the `textarea` triggers the command palette (`paletteView`), which dynamically filters `availableActions` and `availableCommands` based on the input. `Tab` and `Shift+Tab` navigate the palette.
4.  **Submission**:
    - `Ctrl+J` (or `Enter` for slash commands) triggers `handleSubmit()`.
    - `handleSubmit()` first checks if the input is an `Action` (via `core.ProcessAction()`) or a `Command` (via `core.ProcessCommand()`). If so, it executes locally and updates the conversation.
    - If not an action/command, it's considered an AI prompt. The prompt is built (`core.BuildPrompt()`), a `stateThinking` state is set, a spinner starts, and `generator.GenerateTask()` is called in a goroutine.
5.  **AI Generation (Streaming)**:
    - `listenForStream()` continuously receives chunks from the gRPC stream, sending `streamResultMsg` to `Update`.
    - `Update` appends chunks to the last `AIMessage` in the `messages` slice.
    - `renderTick()` periodically triggers UI redraws, ensuring the streaming response is visible to the user as it arrives.
6.  **Generation Completion/Cancellation**:
    - Upon stream completion, `streamFinishedMsg` is sent. `Update` finalizes the AI message, resets the UI state to `stateIdle`, and triggers `countTokensCmd()`.
    - If `Ctrl+C` is pressed during generation, `cancelGeneration()` is called, and the UI enters `stateCancelling`.
7.  **Display**: `View()` is called by Bubble Tea runtime whenever the `Model` changes, rendering the updated state (conversation, input, status) to the terminal.
8.  **Scrolling**: `Ctrl+U` and `Ctrl+D` allow scrolling the `viewport` content, which displays the conversation history.

This architecture ensures a clear separation of concerns, robust handling of asynchronous operations, and a highly interactive user experience within the terminal.
