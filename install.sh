#!/bin/bash

set -e

BINARIES=("coder" "itf" "sf" "pcat" "pti")
REPO="sokinpui/coder"

if [ -d "cmd/coder" ]; then
  if ! command -v go &>/dev/null; then
    echo "Error: Go is not installed." >&2
    exit 1
  fi

  VERSION=$(git describe --tags --always --dirty 2>/dev/null || echo "devel")
  LD_FLAGS="-s -w -X github.com/sokinpui/coder/pkg/version.Version=$VERSION"

  echo "Installing Coder Suite ($VERSION)..."
  for bin in "${BINARIES[@]}"; do
    GOWORK=off go install -ldflags="$LD_FLAGS" "./cmd/$bin"
  done

  GOPATH_BIN="$(go env GOPATH)/bin"
  echo "Successfully installed to $GOPATH_BIN:"
  for bin in "${BINARIES[@]}"; do
    echo "  - $bin"
  done
  exit 0
fi

if ! command -v curl &>/dev/null; then
  echo "Error: curl is required." >&2
  exit 1
fi

detect_os() {
  local os
  os="$(uname -s | tr '[:upper:]' '[:lower:]')"
  case "$os" in
    linux*)  echo "linux" ;;
    darwin*) echo "darwin" ;;
    msys*|mingw*|cygwin*) echo "windows" ;;
    *)
      echo "Error: Unsupported operating system: $os" >&2
      exit 1
      ;;
  esac
}

detect_arch() {
  local arch
  arch="$(uname -m)"
  case "$arch" in
    x86_64|amd64) echo "amd64" ;;
    arm64|aarch64) echo "arm64" ;;
    *)
      echo "Error: Unsupported architecture: $arch" >&2
      exit 1
      ;;
  esac
}

is_in_path() {
  case ":$PATH:" in
    *":$1:"*) return 0 ;;
    *) return 1 ;;
  esac
}

resolve_dest_dir() {
  if [ -n "${INSTALL_DIR:-}" ]; then
    echo "$INSTALL_DIR"
    return
  fi

  if [ -w "/usr/local/bin" ]; then
    echo "/usr/local/bin"
    return
  fi

  if is_in_path "$HOME/.local/bin"; then
    echo "$HOME/.local/bin"
    return
  fi

  if is_in_path "$HOME/bin"; then
    echo "$HOME/bin"
    return
  fi

  if is_in_path "/usr/local/bin"; then
    echo "/usr/local/bin"
    return
  fi

  echo "$HOME/.local/bin"
}

resolve_latest_tag() {
  local tag
  tag=$(curl -fsSL -o /dev/null -w "%{url_effective}" "https://github.com/${REPO}/releases/latest" 2>/dev/null | sed 's#.*/tag/##')
  if [ -n "$tag" ] && [ "$tag" != "latest" ] && [[ "$tag" != *"/"* ]]; then
    echo "$tag"
    return 0
  fi

  tag=$(curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" 2>/dev/null | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')
  if [ -n "$tag" ]; then
    echo "$tag"
    return 0
  fi

  return 1
}

install_prebuilt() {
  local os="$1"
  local arch="$2"
  local dest_dir="$3"
  local version="${CODER_VERSION:-}"

  if [ -z "$version" ]; then
    version="$(resolve_latest_tag)" || return 1
  fi

  local ext=""
  local archive_ext="tar.gz"
  if [ "$os" = "windows" ]; then
    ext=".exe"
    archive_ext="zip"
  fi

  local archive_name="coder_${version}_${os}_${arch}.${archive_ext}"
  local download_url="https://github.com/${REPO}/releases/download/${version}/${archive_name}"
  local tmp_dir
  tmp_dir=$(mktemp -d 2>/dev/null || mktemp -d -t 'coder-install')
  trap 'rm -rf "$tmp_dir"' EXIT

  echo "==> [1/3] Downloading Coder Suite (${version} for ${os}/${arch})..."
  if [ -t 1 ] || [ -t 2 ]; then
    curl -fL --progress-bar "$download_url" -o "$tmp_dir/$archive_name" || return 1
  else
    curl -fsSL "$download_url" -o "$tmp_dir/$archive_name" || return 1
  fi

  echo "==> [2/3] Extracting archive..."
  local extract_dir="$tmp_dir/extracted"
  mkdir -p "$extract_dir"

  if [ "$archive_ext" = "zip" ]; then
    if command -v unzip &>/dev/null; then
      unzip -q "$tmp_dir/$archive_name" -d "$extract_dir"
    else
      tar -xf "$tmp_dir/$archive_name" -C "$extract_dir"
    fi
  else
    tar -xzf "$tmp_dir/$archive_name" -C "$extract_dir"
  fi

  local src_dir
  src_dir=$(find "$extract_dir" -maxdepth 2 -type f -name "coder${ext}" -exec dirname {} \; | head -n 1)
  if [ -z "$src_dir" ]; then
    src_dir="$extract_dir"
  fi

  echo "==> [3/3] Installing binaries to ${dest_dir}..."
  for bin in "${BINARIES[@]}"; do
    local bin_name="${bin}${ext}"
    if [ -f "$src_dir/$bin_name" ]; then
      chmod +x "$src_dir/$bin_name"
      $USE_SUDO cp -f "$src_dir/$bin_name" "$dest_dir/$bin_name"
      echo "  ✓ Installed ${bin_name}"
    fi
  done

  echo "Successfully installed Coder Suite (${version}) to ${dest_dir}:"
  for bin in "${BINARIES[@]}"; do
    echo "  - ${bin}${ext}"
  done
  return 0
}

install_from_source() {
  local dest_dir="$1"

  if ! command -v go &>/dev/null; then
    echo "Error: Go is required to compile from source." >&2
    exit 1
  fi

  if ! command -v git &>/dev/null; then
    echo "Error: Git is required to compile from source." >&2
    exit 1
  fi

  local tmp_dir=""
  tmp_dir=$(mktemp -d 2>/dev/null || mktemp -d -t 'coder-install')
  trap 'rm -rf "$tmp_dir"' EXIT
  echo "Cloning repository..."
  git clone "https://github.com/${REPO}.git" "$tmp_dir"
  cd "$tmp_dir"
  local latest_tag
  latest_tag=$(git tag -l --sort=-v:refname | head -n 1)
  if [ -n "$latest_tag" ]; then
    git checkout "$latest_tag" --quiet
  fi

  local version
  version=$(git describe --tags --always --dirty 2>/dev/null || echo "devel")
  local ld_flags="-s -w -X github.com/sokinpui/coder/pkg/version.Version=$version"

  echo "Building Coder Suite from source ($version)..."
  for bin in "${BINARIES[@]}"; do
    GOWORK=off go build -ldflags="$ld_flags" -o "$tmp_dir/$bin" "./cmd/$bin"
    $USE_SUDO cp -f "$tmp_dir/$bin" "$dest_dir/$bin"
  done

  echo "Successfully built and installed to ${dest_dir}:"
  for bin in "${BINARIES[@]}"; do
    echo "  - ${bin}"
  done
}

TARGET_OS="$(detect_os)"
TARGET_ARCH="$(detect_arch)"
DEST_DIR="$(resolve_dest_dir)"
USE_SUDO=""

if [ ! -d "$DEST_DIR" ]; then
  mkdir -p "$DEST_DIR" 2>/dev/null || sudo mkdir -p "$DEST_DIR"
fi

if [ ! -w "$DEST_DIR" ]; then
  if command -v sudo &>/dev/null && [ "${EUID:-$(id -u)}" -ne 0 ]; then
    USE_SUDO="sudo"
  else
    DEST_DIR="$HOME/.local/bin"
    mkdir -p "$DEST_DIR"
  fi
fi

if ! install_prebuilt "$TARGET_OS" "$TARGET_ARCH" "$DEST_DIR"; then
  echo "Prebuilt binary download unavailable. Falling back to source build..."
  install_from_source "$DEST_DIR"
fi

if ! is_in_path "$DEST_DIR"; then
  echo ""
  echo "Note: '${DEST_DIR}' is not currently in your \$PATH."
  echo "Add it to your profile (e.g. ~/.bashrc or ~/.zshrc):"
  echo "  export PATH=\"${DEST_DIR}:\$PATH\""
fi
