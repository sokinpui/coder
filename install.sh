#!/bin/bash

set -e

if ! command -v go &>/dev/null; then
  echo "Error: Go is not installed."
  exit 1
fi

VERSION=$(git describe --tags --always --dirty)
LD_FLAGS="-s -w -X github.com/sokinpui/coder/internal/utils.Version=$VERSION"

echo "Installing Coder ($VERSION)..."

go install -ldflags="$LD_FLAGS" ./cmd/coder

echo "Successfully installed to $(go env GOPATH)/bin/coder"
