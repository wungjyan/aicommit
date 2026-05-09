#!/bin/sh
set -e

REPO="wungjyan/aicommit"
BINARY="aicommit"
INSTALL_DIR=""

detect_platform() {
  os="$(uname -s | tr '[:upper:]' '[:lower:]')"
  arch="$(uname -m)"
  case "$os" in
    darwin) os="darwin" ;;
    linux)  os="linux" ;;
    *) echo "Unsupported OS: $os"; exit 1 ;;
  esac
  case "$arch" in
    x86_64|amd64) arch="amd64" ;;
    aarch64|arm64) arch="arm64" ;;
    *) echo "Unsupported architecture: $arch"; exit 1 ;;
  esac
  OS="$os"
  ARCH="$arch"
}

detect_install_dir() {
  if [ -w "/usr/local/bin" ]; then
    INSTALL_DIR="/usr/local/bin"
    SUDO=""
  elif [ -d "/usr/local/bin" ]; then
    if ! command -v sudo >/dev/null 2>&1; then
      echo "sudo is required to install to /usr/local/bin but was not found."
      exit 1
    fi
    INSTALL_DIR="/usr/local/bin"
    SUDO="sudo"
  else
    mkdir -p "$HOME/.local/bin"
    INSTALL_DIR="$HOME/.local/bin"
    SUDO=""
  fi
}

get_latest_version() {
  VERSION=$(curl -fsSL -H "User-Agent: ${REPO}-installer" \
    "https://api.github.com/repos/${REPO}/releases/latest" \
    | grep '"tag_name"' | sed -E 's/.*"v([^"]+)".*/\1/')
  if [ -z "$VERSION" ]; then
    echo "Failed to fetch latest version"
    exit 1
  fi
}

main() {
  detect_platform
  detect_install_dir
  get_latest_version

  FILENAME="${BINARY}-${OS}-${ARCH}"
  URL="https://github.com/${REPO}/releases/download/v${VERSION}/${FILENAME}"

  echo "Installing aicommit v${VERSION}..."
  echo "  Platform: ${OS}/${ARCH}"
  echo "  Install to: ${INSTALL_DIR}/${BINARY}"

  TMPDIR="$(mktemp -d)"
  trap "rm -rf $TMPDIR" EXIT

  curl -fsSL "$URL" -o "${TMPDIR}/${BINARY}"
  chmod +x "${TMPDIR}/${BINARY}"

  if ! "${TMPDIR}/${BINARY}" version >/dev/null 2>&1; then
    echo "Downloaded binary is invalid. Aborting."
    exit 1
  fi

  $SUDO mv "${TMPDIR}/${BINARY}" "${INSTALL_DIR}/${BINARY}"

  echo "Installed: $("${INSTALL_DIR}/${BINARY}" version)"
  echo ""
  echo "Make sure ${INSTALL_DIR} is in your PATH."
}

main
