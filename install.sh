#!/bin/bash

# This script installs the Coder application and its Go-based dependencies.
# It checks for required system dependencies and installs binaries using 'go install'.

set -euo pipefail

echo "Starting Coder installation..."

# --- Dependency Checks ---

echo "Checking for required commands..."

# Function to check for a command
check_command() {
  if ! command -v "$1" &>/dev/null; then
    echo "Error: Required command '$1' is not installed or not in PATH."
    echo "Please install it and try again."
    exit 1
  fi
}

check_command "go"
check_command "git"
if [ "$(uname)" == "Linux" ]; then
  if [ "$XDG_SESSION_TYPE" == "wayland" ]; then
    check_command "wl-clipboard"
  else
    check_command "xclip"
  fi

fi
if [ "$(uname)" == "Darwin" ]; then
  check_command "pngpaste"
fi

# --- Install Go Dependencies ---
echo "Installing Go-based dependencies to $(go env GOPATH)/bin..."

# if itf is not installed, install it
if ! command -v itf &>/dev/null; then
  echo "Installing itf (interactive diff tool)..."
  go install github.com/sokinpui/itf.go/cmd/itf@latest
else
  echo "itf is already installed"
fi

if ! command -v pcat &>/dev/null; then
  echo "Installing pcat (code to prompt tool)..."
  go install github.com/sokinpui/pcat.go/cmd/pcat@latest
else
  echo "pcat is already installed"
fi

if ! command -v sf &>/dev/null; then
  echo "Installing pcat (code to prompt tool)..."
  go install github.com/sokinpui/sf@latest
else
  echo "pcat is already installed"
fi

# --- Install Application Binary ---
echo "Installing Coder TUI..."
go install ./cmd/coder

# cp default config file if it does not exist
CONFIG_DIR="$HOME/.config/coder"
CONFIG_FILE="$CONFIG_DIR/config.yaml"
if [ ! -f "$CONFIG_FILE" ]; then
  echo "Creating default configuration file at $CONFIG_FILE..."
  mkdir -p "$CONFIG_DIR"
  cp ./internal/config/template.yaml "$CONFIG_FILE"
else
  echo "Configuration file already exists at $CONFIG_FILE"
fi

echo "Installation complete."
echo "You can now run 'coder' from your terminal."
echo "Ensure $(go env GOPATH)/bin is in your system's PATH."
