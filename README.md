# Coder

Coder is an AI coding assistant available as a Terminal User Interface (TUI) and a Web User Interface (Web UI). It is designed to integrate AI code generation and modification directly into the development workflow.

## Features

- **Dual Interfaces**: Choose between a fast, keyboard-driven TUI (`coder`) or a feature-rich Web UI (`coder-web`).
- **Context-Aware**: Automatically includes relevant project source code and files from a `Context` directory in prompts.
- **In-place Code Application**: Apply generated code changes directly using a diff viewer.
- **Conversation Management**: Sessions are automatically saved and can be browsed or resumed later.
- **Extensible Commands**: A command system (`:mode`, `:model`) allows for runtime configuration changes.

## Documentation

Full documentation for the project can be found in the `docs/` directory.

- **[Installation](./docs/Installation/README.md)**: How to install the application.
- **[Usage](./docs/Usage/README.md)**: How to configure and use the TUI and Web UI.
- **[Architecture](./docs/Architecture/README.md)**: An overview of the project's architecture.
- **[Developer Guide](./docs/Develop/README.md)**: Information for contributors.
- **[API Reference](./docs/Api/README.md)**: Details on the WebSocket API for the Web UI.

## Quick Start

### Prerequisites

- **Common**: Go (1.24+), Git.
- **For TUI**: `fd`, `pcat`, `itf`.
- **For Web UI**: Node.js and npm.

### Installation

The provided installation script can be used to install one or both applications.

```sh
# Install both TUI and Web UI
./install.sh

# Install only the TUI
./install.sh tui

# Install only the Web UI
./install.sh web
```

### Running

- **TUI**: Run `coder` from within a Git repository.
- **Web UI**: Run `coder-web` from within a Git repository. It will print the URL to access the UI in your browser.
