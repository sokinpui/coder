#!/bin/bash

set -e

BINARIES="coder co itf sf pcat pti"

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

VERSION=$(git describe --tags --always --dirty 2>/dev/null || echo "devel")
LD_FLAGS="-s -w -X github.com/sokinpui/coder/pkg/version.Version=$VERSION"
TARGET_DIR="${XDG_BIN_HOME:-$HOME/.local/bin}"
mkdir -p "$TARGET_DIR"

echo "Installing Coder Suite ($VERSION) to $TARGET_DIR..."

for bin in $BINARIES; do
  GOWORK=off go build -trimpath -ldflags="$LD_FLAGS" -o "$TARGET_DIR/$bin" "./cmd/$bin"
done

echo ""
echo "Successfully installed to ${TARGET_DIR}:"
for bin in $BINARIES; do
  echo "  - $bin"
done

case ":$PATH:" in
  *":$TARGET_DIR:"*) ;;
  *)
    echo ""
    echo "Notice: ${TARGET_DIR} is not in your \$PATH."
    echo "Add it by placing the following line in your shell profile (~/.bashrc, ~/.zshrc, etc.):"
    echo ""
    echo "  export PATH=\"${TARGET_DIR}:\$PATH\""
    ;;
esac
