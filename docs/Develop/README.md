# Developer Guide

This guide provides instructions for setting up the development environment, building the applications, and contributing to the project.

## Development Environment Setup

### Prerequisites

Ensure you have the following installed:

-   **Go**: Version 1.24 or later.
-   **Git**: For version control.
-   **TUI Tools**: `fd`, `pcat`, `itf` for running and testing the TUI.

### Getting Started

1.  **Clone the repository:**
    ```sh
    git clone https://github.com/your-org/coder.git
    cd coder
    ```

2.  **Install Go dependencies:**
    ```sh
    go mod tidy
    ```

## Building and Running

### TUI (`coder`)

-   **Run for development:**
    ```sh
    go run ./cmd/coder
    ```
    This command compiles and runs the TUI application. It must be run from within a Git repository.

-   **Build a binary:**
    ```sh
    go build -o coder ./cmd/coder
    ```

## CI/CD

The project uses GitHub Actions for continuous integration and releases. Workflows are defined in the `.github/workflows/` directory:

-   `go-tui.yml`: Builds and tests the TUI application.
-   `release.yml`: Triggers on new version tags (`v*.*.*`), builds cross-platform binaries, and creates a GitHub Release.
