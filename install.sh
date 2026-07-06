#!/bin/bash

set -e

if ! command -v go &>/dev/null; then
  echo "Error: Go is not installed."
  exit 1
fi

VERSION=$(git describe --tags --always --dirty)
LD_FLAGS="-s -w -X github.com/sokinpui/coder/pkg/version.Version=$VERSION"

echo "Installing Coder Suite ($VERSION)..."

GOWORK=off go install -ldflags="$LD_FLAGS" ./cmd/coder
GOWORK=off go install -ldflags="$LD_FLAGS" ./cmd/itf
GOWORK=off go install -ldflags="$LD_FLAGS" ./cmd/sf
GOWORK=off go install -ldflags="$LD_FLAGS" ./cmd/pcat

echo "Successfully installed to $(go env GOPATH)/bin:"
echo "  - coder"
echo "  - itf"
echo "  - sf"
echo "  - pcat"
