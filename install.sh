#!/bin/bash

set -e

if ! command -v go &>/dev/null; then
  echo "Error: Go is not installed."
  exit 1
fi

if ! command -v git &>/dev/null; then
  echo "Error: Git is not installed."
  exit 1
fi

if [ ! -d "cmd/coder" ]; then
  TMP_DIR=$(mktemp -d 2>/dev/null || mktemp -d -t 'coder-install')
  trap 'rm -rf "$TMP_DIR"' EXIT
  echo "Cloning repository..."
  git clone https://github.com/sokinpui/coder.git "$TMP_DIR"
  cd "$TMP_DIR"
  LATEST_TAG=$(git tag -l --sort=-v:refname | head -n 1)
  if [ -n "$LATEST_TAG" ]; then
    echo "Checking out latest stable release ($LATEST_TAG)..."
    git checkout "$LATEST_TAG" --quiet
  fi
fi

LATEST_TAG=$(git describe --tags --abbrev=0 2>/dev/null || git tag -l --sort=-v:refname | head -n 1)
VERSION=${LATEST_TAG:-$(git describe --tags --always --dirty)}
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
