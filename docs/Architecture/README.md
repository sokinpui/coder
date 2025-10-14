# Architecture

This document provides an overview of the Coder project's architecture, components, and data flow.

## High-Level Overview

Coder is a Terminal User Interface (TUI) application written in Go. It acts as a client to an external gRPC-based AI service for code generation, and is designed to be run locally from within a Git repository.

```
+------------------+
|   TUI Client     |
|     (coder)      |
+------------------+
         |
+------------------+
|    Core Logic    |
|    (session)     |
+------------------+
         |
+------------------+
|  gRPC Client for |
|    AI Service    |
+------------------+
```

## Component Breakdown

### Applications

-   **`cmd/coder` (TUI)**: A standalone, keyboard-driven terminal application built using the `bubbletea` framework. It directly interacts with the core session management logic.

### Core Logic (`internal/`)

This directory contains the shared business logic for both applications.

-   **`session`**: The central component for state management. It manages the conversation history, configuration, and orchestrates interactions between the UI, the core logic, and the generation service.
-   **`core`**: Defines fundamental types (`Message`) and handles the processing of user input, distinguishing between prompts, commands (`:mode`), and actions (`:pcat`).
-   **`generation`**: Contains the client for the external AI gRPC service. It is responsible for sending prompts and receiving generated content streams.
-   **`history`**: Manages the persistence of conversations. Sessions are saved as Markdown files with YAML frontmatter in the `.coder/history` directory at the root of the Git repository.
-   **`ui`**: The implementation of the TUI, including all models, views, and updates for the `bubbletea` framework.

## Data Flow

### Context Gathering

On startup, and before each generation, the application gathers context:

1.  **Project Source**: The `source` package uses `fd` to list all source files (respecting `.gitignore` and custom exclusions) and pipes them to `pcat` to create a single formatted string of project code.
2.  **Context Directory**: The `contextdir` package reads all files from a user-created `Context/` directory at the repository root. These are treated as high-priority related documents.

This context is prepended to the prompt sent to the AI service.

### Conversation Flow

The TUI flow uses direct function calls between the `ui` and `session` packages. When a user sends a prompt:
1. The `ui` package sends the input string to the `session` manager.
2. The `session` manager processes the input via the `core` package. It constructs the full prompt including system instructions, context, and conversation history.
3. The `session` calls the `generation` client to send the prompt to the AI service.
4. The `generation` client streams the response back to the `session`.
5. The `session` forwards the stream to the `ui`, which updates the display in real-time.
6. Upon completion, the `session` saves the full conversation to the history file.
