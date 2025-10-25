# Installation

This document provides instructions for installing the Coder TUI application.

## Prerequisites

Before installation, ensure the following dependencies are installed and available in your system's `PATH`.

### Common Dependencies

- **Go**: Version 1.24 or later.
- **Git**: Required for context gathering and version control features.

### Dependencies

- **`fd`**: A simple, fast and user-friendly alternative to `find`.
- **`pcat`**: A syntax highlighting file viewer.
- **`itf`**: An interactive diff viewer used to apply code changes.
- **`fzf`**: A command-line fuzzy finder used for the command palette (`Ctrl+F`).
- **Clipboard Tools** (for pasting images with `Ctrl+V`):
  - **macOS**: `pngpaste`
  - **Linux (X11)**: `xclip`
  - **Linux (Wayland)**: `wl-clipboard`

## Installation Methods

### Using the Installation Script (Recommended)

The project includes a convenience script `install.sh` to automate the installation process. It checks for dependencies and installs the binary to your `GOPATH`.

1.  Navigate to the root of the project directory.
2.  Make the script executable: `chmod +x install.sh`
3.  Run the script:
    ```sh
    ./install.sh
    ```

4.  Ensure `$(go env GOPATH)/bin` is in your system's `PATH`.

### From Pre-built Binaries

Binary releases are available on the project's GitHub Releases page. Download the appropriate binary for your operating system and architecture, place it in a directory included in your system's `PATH`, and make it executable.

Note that the application still requires the dependencies to be installed.

### Manual Build from Source

You can also build and install the applications manually.

1.  **Clone the repository:**
    ```sh
    git clone https://github.com/your-org/coder.git
    cd coder
    ```

2.  **Install Go dependencies:**
    ```sh
    go mod tidy
    ```

3.  **Build and install the application:**
    ```sh
    go install ./cmd/coder
    ```

