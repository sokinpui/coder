# Architecture

This document provides an overview of the Coder project's architecture, components, and data flow.

## High-Level Overview

Coder is a monorepo containing two main applications: a Terminal User Interface (`coder`) and a Web User Interface (`coder-web`). Both applications share a common core logic layer written in Go. The applications act as clients to an external gRPC-based AI service for code generation.

```
+------------------+      +-------------------+
|   TUI Client     |      |    Web Client     |
|     (coder)      |      |    (coder-web)    |
+------------------+      +-------------------+
        |                   |      ^
        |                   |      | (WebSocket)
        |                   V      |
        |             +-------------------+
        |             | Go WebSocket Srv  |
        |             +-------------------+
        |                   |
        +---------+---------+
                  |
        +-------------------+
        |    Core Logic     |
        |    (session)      |
        +-------------------+
                  |
        +-------------------+
        |  gRPC Client for  |
        |    AI Service     |
        +-------------------+
                  |
                  V
        +-------------------+
        | External AI Service|
        +-------------------+
```

## Component Breakdown

### Applications

-   **`cmd/coder` (TUI)**: A standalone, keyboard-driven terminal application built using the `bubbletea` framework. It directly interacts with the core session management logic.
-   **`cmd/coder-web` (Web UI)**: A Go application that serves a React-based single-page application (SPA) and handles communication with it via a WebSocket server.

### Core Logic (`internal/`)

This directory contains the shared business logic for both applications.

-   **`session`**: The central component for state management. It manages the conversation history, configuration, and orchestrates interactions between the UI, the core logic, and the generation service.
-   **`core`**: Defines fundamental types (`Message`) and handles the processing of user input, distinguishing between prompts, commands (`:mode`), and actions (`:pcat`).
-   **`generation`**: Contains the client for the external AI gRPC service. It is responsible for sending prompts and receiving generated content streams.
-   **`history`**: Manages the persistence of conversations. Sessions are saved as Markdown files with YAML frontmatter in the `.coder/history` directory at the root of the Git repository.
-   **`server`**: Implements the WebSocket server for `coder-web`. It manages client connections, translates WebSocket messages into calls to the `session` manager, and streams responses back to the web client.
-   **`ui`**: The implementation of the TUI, including all models, views, and updates for the `bubbletea` framework.

### Frontend (`web/`)

-   A React and TypeScript single-page application built with Vite.
-   It communicates with the `coder-web` backend exclusively through a WebSocket connection.
-   Components are built using Material-UI.

## Data Flow

### Context Gathering

On startup, and before each generation, the application gathers context:

1.  **Project Source**: The `source` package uses `fd` to list all source files (respecting `.gitignore` and custom exclusions) and pipes them to `pcat` to create a single formatted string of project code.
2.  **Context Directory**: The `contextdir` package reads all files from a user-created `Context/` directory at the repository root. These are treated as high-priority related documents.

This context is prepended to the prompt sent to the AI service.

### Conversation Flow (Web UI)

1.  The user types a message in the React UI and sends it.
2.  The `useWebSocket` hook sends a JSON message (`userInput`) over the WebSocket.
3.  The Go `server` receives the message and passes the input to the `session` manager.
4.  The `session` manager processes the input via the `core` package. For a prompt, it constructs the full prompt including system instructions, context, and conversation history.
5.  The `session` calls the `generation` client to send the prompt to the AI service.
6.  The `generation` client streams the response back to the `session`.
7.  The `session` forwards the stream to the `server`, which sends `generationChunk` messages over the WebSocket.
8.  The React UI receives the chunks and updates the display in real-time.
9.  Upon completion, the `session` saves the full conversation to the history file.

The TUI flow is similar but replaces the WebSocket communication with direct function calls between the `ui` and `session` packages.
