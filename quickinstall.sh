#!/bin/sh

set -e

REPO="sokinpui/coder"

detect_os() {
  case "$(uname -s)" in
    Darwin) echo "darwin" ;;
    Linux)  echo "linux" ;;
    *)
      echo "Error: Unsupported operating system $(uname -s)" >&2
      exit 1
      ;;
  esac
}

detect_arch() {
  case "$(uname -m)" in
    x86_64|amd64) echo "amd64" ;;
    arm64|aarch64) echo "arm64" ;;
    *)
      echo "Error: Unsupported architecture $(uname -m)" >&2
      exit 1
      ;;
  esac
}

download_file() {
  url="$1"
  dest="$2"
  if command -v curl >/dev/null 2>&1; then
    curl -fsSL "$url" -o "$dest"
    return 0
  fi
  if command -v wget >/dev/null 2>&1; then
    wget -qO "$dest" "$url"
    return 0
  fi
  echo "Error: Neither curl nor wget is available." >&2
  exit 1
}

fetch_latest_tag() {
  if command -v curl >/dev/null 2>&1; then
    curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" 2>/dev/null | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/'
    return 0
  fi
  if command -v wget >/dev/null 2>&1; then
    wget -qO- "https://api.github.com/repos/${REPO}/releases/latest" 2>/dev/null | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/'
    return 0
  fi
  return 1
}

OS=$(detect_os)
ARCH=$(detect_arch)

TAG=$(fetch_latest_tag)
if [ -z "$TAG" ]; then
  echo "Error: Failed to fetch latest release tag for ${REPO}." >&2
  exit 1
fi

TARBALL="coder-${OS}-${ARCH}.tar.gz"
DOWNLOAD_URL="https://github.com/${REPO}/releases/download/${TAG}/${TARBALL}"

TARGET_DIR="${XDG_BIN_HOME:-$HOME/.local/bin}"
mkdir -p "$TARGET_DIR"

TMP_DIR=$(mktemp -d 2>/dev/null || mktemp -d -t 'coder-quickinstall')
trap 'rm -rf "$TMP_DIR"' EXIT

echo "Downloading Coder Suite ${TAG} for ${OS}/${ARCH}..."
download_file "$DOWNLOAD_URL" "$TMP_DIR/$TARBALL"

tar -xzf "$TMP_DIR/$TARBALL" -C "$TMP_DIR"

BINARIES="coder co itf sf pcat pti"
for bin in $BINARIES; do
  if [ -f "$TMP_DIR/$bin" ]; then
    chmod +x "$TMP_DIR/$bin"
    mv "$TMP_DIR/$bin" "$TARGET_DIR/$bin"
  fi
done

echo ""
echo "Successfully installed Coder Suite to ${TARGET_DIR}:"
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
