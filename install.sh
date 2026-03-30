#!/bin/sh
set -e

REPO="sonarbay/news"
BINARY="sonarbay"
INSTALL_DIR="/usr/local/bin"
BASE_URL="https://github.com/${REPO}/releases/latest/download"

detect_platform() {
  OS=$(uname -s | tr '[:upper:]' '[:lower:]')
  ARCH=$(uname -m)

  case "$OS" in
    linux) OS="linux" ;;
    darwin) OS="darwin" ;;
    *) echo "Unsupported OS: $OS" && exit 1 ;;
  esac

  case "$ARCH" in
    x86_64|amd64) ARCH="x64" ;;
    aarch64|arm64) ARCH="arm64" ;;
    *) echo "Unsupported architecture: $ARCH" && exit 1 ;;
  esac

  echo "${OS}-${ARCH}"
}

main() {
  PLATFORM=$(detect_platform)
  URL="${BASE_URL}/${BINARY}-${PLATFORM}"

  echo ""
  echo "  SonarBay CLI Installer"
  echo "  ─────────────────────"
  echo ""
  echo "  Platform:  ${PLATFORM}"
  echo "  Binary:    ${INSTALL_DIR}/${BINARY}"
  echo ""

  TMPFILE=$(mktemp)
  echo "  Downloading..."
  if command -v curl > /dev/null 2>&1; then
    curl -fsSL "$URL" -o "$TMPFILE"
  elif command -v wget > /dev/null 2>&1; then
    wget -qO "$TMPFILE" "$URL"
  else
    echo "  Error: curl or wget is required" && exit 1
  fi

  chmod +x "$TMPFILE"

  if [ -w "$INSTALL_DIR" ]; then
    mv "$TMPFILE" "${INSTALL_DIR}/${BINARY}"
  else
    echo "  Installing to ${INSTALL_DIR} (requires sudo)..."
    sudo mv "$TMPFILE" "${INSTALL_DIR}/${BINARY}"
  fi

  echo "  ✓ Installed ${BINARY} to ${INSTALL_DIR}/${BINARY}"
  echo ""
  echo "  Run: sonarbay --help"
  echo ""
}

main
