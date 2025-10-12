# Developer Guide

This guide provides instructions for setting up the development environment, building the applications, and contributing to the project.

## Development Environment Setup

### Prerequisites

Ensure you have the following installed:

-   **Go**: Version 1.24 or later.
-   **Node.js and npm**: For frontend development.
-   **TUI Tools**: `fd`, `pcat`, `itf` for running and testing the TUI.

### Getting Started

1.  **Clone the repository:**
    ```sh
    go mod tidy
    ```

3.  **Install frontend dependencies:**
    ```sh
    cd web
    npm install
    cd ..

## Building and Running

To run the TUI application for development, execute the following command from within a Git repository:
```sh
go run ./cmd/coder
```

## CI/CD

The project uses GitHub Actions for continuous integration and releases. Workflows are defined in the `.github/workflows/` directory:

-   `go-tui.yml`: Builds and tests the TUI application on push and pull requests.
-   `release.yml`: Triggers on new version tags (`v*.*.*`), builds cross-platform binaries, and creates a GitHub Release.
