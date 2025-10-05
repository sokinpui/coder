#!/bin/bash

# This script installs the Coder application to your Go environment.
# It installs the binary using 'go install'.

set -euo pipefail

echo "Starting Coder installation..."

# --- Dependency Checks ---

echo "Checking for Go installation..."
if ! command -v go &> /dev/null; then
    echo "Error: Go is not installed or not in PATH. Please install Go 1.24.1 or later."
    exit 1
fi

# --- Install Application Binary ---
echo "Installing Coder binaries to $(go env GOPATH)/bin..."
echo "Installing Coder TUI..."
go install ./cmd/coder

echo "Installation complete."
echo "You can now run 'coder' from your terminal."
echo "Ensure $(go env GOPATH)/bin is in your system's PATH."
