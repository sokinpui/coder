# Architecture

This document provides an overview of the Coder project's architecture, components, and data flow.

## High-Level Overview

Coder is a Terminal User Interface (TUI) application written in Go. It acts as a client to an external AI service for code generation, communicating via the OpenAI-compatible API protocol. It is designed to be run locally from within a Git repository to provide context-aware assistance.

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
|   HTTP Client for|
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
- **`generation`**: Contains the client for the external AI service via HTTP. It is responsible for sending prompts and receiving generated content streams using Server-Sent Events (SSE) logic.
- **`history`**: Manages the persistence of conversations. Sessions are saved as Markdown files with YAML frontmatter in the `.coder/history` directory.
- **`prompt`**: Manages system instructions and specialized prompts (like title generation).
- **`token`**: Handles local token counting using a Gemini-compatible tokenizer (Gemma) to provide accurate context window estimates.
- **`ui`**: The implementation of the TUI, including all models, views, and updates for the `bubbletea` framework.

Other key packages include `types` for fundamental data structures, `modes` for handling application behavior (Chat vs. Coding), and `source` for context gathering.

## Data Flow

### Context Gathering

On startup, and before each generation, the application gathers context:

1.  **Project Source**: The `source` package uses the internal `sf` library to find source files based on the `context` configuration in `config.yaml`. This configuration defaults to the current directory and can be modified at runtime with the `:file` command. The file list is then processed by the `pcat` library to create a single formatted string of project code.

This context is combined with the system prompt and conversation history to form the full prompt sent to the AI service.

### Conversation Flow

The TUI flow uses direct function calls between the `ui` and `session` packages. When a user sends a prompt:

1.  The `ui` package sends the input string (or command) to the `session` manager.
2.  The `session` manager processes the input. It constructs the full prompt including system instructions, project source code, and conversation history.
3.  If images were pasted (`Ctrl+V`), they are encoded as base64 and included in the payload.
4.  The `session` calls the `generation` client, which formats the data into an OpenAI-compatible JSON payload (supporting `messages` with `role` and `content`).
5.  The `generation` client sends an HTTP POST request and processes the response as a Server-Sent Events (SSE) stream.
6.  The `session` forwards the stream to the `ui`, which updates the display in real-time.
7.  Once the stream finishes, the `session` triggers an asynchronous title generation (if new) and saves the full conversation to the history file.
8.  The `token` package calculates the total token usage of the current context and history to update the status bar.
