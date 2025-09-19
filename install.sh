#!/bin/bash

# This script installs the Coder application (TUI or WebUI) to your Go environment.
# It handles dependencies like npm (for WebUI),
# builds the frontend (for WebUI), and installs the binary using 'go install'.

set -euo pipefail

# Initialize installation flags
INSTALL_TUI=false
INSTALL_WEB=false

# Function to display usage instructions
usage() {
    echo "Usage: $0 [tui] [web]"
    echo "  If no arguments are provided, both 'tui' and 'web' applications will be installed."
    echo "  tui: Installs the Coder TUI application."
    echo "  web: Installs the Coder Web application."
    exit 1
}

# Validate command-line arguments
if [ "$#" -eq 0 ]; then
    INSTALL_TUI=true
    INSTALL_WEB=true
else
    for arg in "$@"; do
        case "$arg" in
            "tui")
                INSTALL_TUI=true
                ;;
            "web")
                INSTALL_WEB=true
                ;;
            *)
                echo "Error: Invalid argument '$arg'."
                usage
                ;;
        esac
    done
fi

echo "Starting Coder installation..."

# --- Common Dependency Checks ---

echo "Checking for Go installation..."
if ! command -v go &> /dev/null; then
    echo "Error: Go is not installed or not in PATH. Please install Go 1.24.1 or later."
    exit 1
fi

# --- WebUI Specific Steps ---
if [ "$INSTALL_WEB" = true ]; then
    echo "Checking for Node.js and npm..."
    if ! command -v npm &> /dev/null; then
        echo "Error: npm is not installed or not in PATH. Please install Node.js (which includes npm)."
        exit 1
    fi

    echo "Building frontend assets for WebUI..."
    # Navigate to the web directory, install dependencies, and build
    (cd web && npm ci && npm run build)
fi

# --- Install Application Binary ---
echo "Installing Coder binaries to $(go env GOPATH)/bin..."
if [ "$INSTALL_TUI" = true ]; then
    echo "Installing Coder TUI..."
    go install ./cmd/coder
fi

if [ "$INSTALL_WEB" = true ]; then
    echo "Installing Coder Web..."
    go install ./cmd/coder-web
fi

echo "Installation complete."
if [ "$INSTALL_TUI" = true ] && [ "$INSTALL_WEB" = true ]; then
    echo "You can now run 'coder' and 'coder-web' from your terminal."
elif [ "$INSTALL_TUI" = true ]; then echo "You can now run 'coder' from your terminal."
elif [ "$INSTALL_WEB" = true ]; then echo "You can now run 'coder-web' from your terminal."
fi
echo "Ensure $(go env GOPATH)/bin is in your system's PATH."
