# Installation

This document provides instructions for installing the Coder TUI and Web applications.

## Prerequisites

Before installation, ensure the following dependencies are installed and available in your system's `PATH`.

### Common Dependencies

- **Go**: Version 1.24 or later.
- **Git**: Required for context gathering and version control features.

### TUI-Specific Dependencies

The following tools are required to run the TUI application (`coder`):

- **`fd`**: A simple, fast and user-friendly alternative to `find`.
- **`pcat`**: A syntax highlighting file viewer.
- **`itf`**: An interactive diff viewer used to apply code changes.

### Web UI-Specific Dependencies

The following tools are required to build and install the Web UI application (`coder-web`):

- **Node.js and npm**: Required to build the frontend assets.

## Installation Methods

### Using the Installation Script (Recommended)

The project includes a convenience script `install.sh` to automate the installation process. It checks for dependencies, builds frontend assets (for the Web UI), and installs the binaries to your `GOPATH`.

1.  Navigate to the root of the project directory.
2.  Make the script executable: `chmod +x install.sh`
3.  Run the script with one of the following options:

    ```sh
    # Install both the TUI and Web UI applications
    ./install.sh

    # Install only the TUI application
    ./install.sh tui

    # Install only the Web UI application
    ./install.sh web
    ```

4.  Ensure `$(go env GOPATH)/bin` is in your system's `PATH`.

### From Pre-built Binaries

Binary releases are available on the project's GitHub Releases page. Download the appropriate binary for your operating system and architecture, place it in a directory included in your system's `PATH`, and make it executable.

Note that the TUI application still requires the TUI-specific dependencies to be installed.

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

3.  **Build and install the TUI:**
    ```sh
    go install ./cmd/coder
    ```

4.  **Build and install the Web UI:**
    - Build the frontend assets:
      ```sh
      cd web
      npm ci
      npm run build
      cd ..
      ```
    - Install the Go binary:
      ```sh
      go install ./cmd/coder-web
      ```
