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

- **`cmd/coder` (TUI)**: A standalone, keyboard-driven terminal application built using the `bubbletea` framework. It directly interacts with the core session management logic.

### Core Logic (`internal/`)

This directory contains the shared business logic for both applications.

- **`session`**: The central component for state management. It manages the conversation history, configuration, and orchestrates interactions between the UI and other components.
- **`commands`**: Handles the processing of user input commands (e.g., `:mode`, `:file`).
- **`generation`**: Contains the client for the external AI gRPC service. It is responsible for sending prompts and receiving generated content streams.
- **`history`**: Manages the persistence of conversations. Sessions are saved as Markdown files with YAML frontmatter in the `.coder/history` directory at the root of the Git repository.
- **`ui`**: The implementation of the TUI, including all models, views, and updates for the `bubbletea` framework.

Other key packages include `types` for fundamental data structures, `modes` for handling application behavior, and `source` for context gathering.

## Data Flow

### Context Gathering

On startup, and before each generation, the application gathers context:

1.  **Project Source**: The `source` package uses the internal `sf` library to find source files based on the `context` configuration in `config.yaml`. This configuration defaults to the current directory and can be modified at runtime with the `:file` command. The file list is then processed by the `pcat` library to create a single formatted string of project code.

This context is combined with the system prompt and conversation history to form the full prompt sent to the AI service.

### Conversation Flow

The TUI flow uses direct function calls between the `ui` and `session` packages. When a user sends a prompt:

1. The `ui` package sends the input string to the `session` manager.
2. The `session` manager processes the input. It constructs the full prompt including system instructions, context, and conversation history using the current `mode` strategy.
3. The `session` calls the `generation` client to send the prompt to the AI service.
4. The `generation` client streams the response back to the `session`.
5. The `session` forwards the stream to the `ui`, which updates the display in real-time.
6. Upon completion, the `session` saves the full conversation to the history file.
