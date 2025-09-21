# Developer Guide

This guide provides instructions for setting up the development environment, building the applications, and contributing to the project.

## Development Environment Setup

### Prerequisites

Ensure you have the following installed:

-   **Go**: Version 1.24 or later.
-   **Node.js and npm**: For frontend development.
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

3.  **Install frontend dependencies:**
    ```sh
    cd web
    npm install
    cd ..
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

### Web UI (`coder-web`)

The Web UI consists of a Go backend and a React frontend. For development, it's recommended to run them separately to benefit from frontend hot-reloading.

1.  **Run the frontend development server:**
    In one terminal, navigate to the `web` directory and start the Vite dev server. This will proxy WebSocket requests to the Go backend.
    ```sh
    cd web
    npm run dev
    ```
    The frontend will be accessible at `http://localhost:5173` (or another port if 5173 is in use).

2.  **Run the Go backend server:**
    In another terminal, run the Go backend. The `vite.config.ts` is configured to proxy to port `8084` by default.
    ```sh
    go run ./cmd/coder-web -addr :8084
    ```

3.  **Access the application:**
    Open your browser to the URL provided by the Vite dev server (e.g., `http://localhost:5173`).

### Building for Production

To build the Web UI for production (embedding the frontend assets into the Go binary):

1.  **Build frontend assets:**
    ```sh
    cd web
    npm run build
    cd ..
    ```

2.  **Build the Go binary:**
    ```sh
    go build -o coder-web ./cmd/coder-web
    ```
    You can then run the self-contained `./coder-web` binary.

## CI/CD

The project uses GitHub Actions for continuous integration and releases. Workflows are defined in the `.github/workflows/` directory:

-   `go-tui.yml`: Builds and tests the TUI application.
-   `go-web.yml`: Builds and tests the Web application (including frontend).
-   `web-ui.yml`: Lints and builds the React frontend.
-   `release.yml`: Triggers on new version tags (`v*.*.*`), builds cross-platform binaries, and creates a GitHub Release.
