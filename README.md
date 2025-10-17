# Coder

Coder is a wrapper of LLM chat interface with a few shortcutes to help apply code generated

## Screenshot

![](./attachments/1.png)
![](./attachments/2.png)
![](./attachments/3.png)

## Features

- **Tui Interfaces**: A fast, keyboard-driven TUI built with Bubble Tea.
- **Context-Aware**: Automatically includes relevant project source code and files from a `Context` directory in prompts.
- **In-place Code Application**: Apply generated code changes directly using a diff viewer.
- **Conversation Management**: Sessions are automatically saved and can be browsed or resumed later.
- **Extensible Commands**: A command system (`:mode`, `:model`) allows for runtime configuration changes.

## Documentation

Full documentation for the project can be found in the `docs/` directory.

- **[Installation](./docs/Installation/README.md)**: How to install the application.
- **[Usage](./docs/Usage/README.md)**: How to configure and use the TUI.
- **[Architecture](./docs/Architecture/README.md)**: An overview of the project's architecture.
- **[Developer Guide](./docs/Develop/README.md)**: Information for contributors.

## Quick Start

### Prerequisites

- **Common**: Go, Git.
- `fd`, `pcat`, `itf`, `fzf`
- `pngpaste` (macOS) or `xclip` (Linux) or `wl-clipboard` (Linux Wayland) for image pasting.

### Installation

The provided installation script can be used to install one or both applications.

```sh
./install.sh
```

### Running

- **TUI**: Run `coder` from within a Git repository.

# Web UI

mantain in another branch: https://github.com/sokinpui/coder/tree/web-ui

Latest feature will available in TUI first.
