#!/bin/sh
# install.sh — download the nab binary for the current platform
# Called by the PostInstall hook when the plugin is installed.
set -e

REPO="kfriede/nab"
VERSION="0.1.0"
INSTALL_DIR="${CLAUDE_PLUGIN_ROOT:-$(dirname "$(dirname "$0")")}/bin"

# Skip if nab is already in PATH
if command -v nab >/dev/null 2>&1; then
  echo "nab already installed: $(command -v nab)" >&2
  exit 0
fi

OS="$(uname -s | tr '[:upper:]' '[:lower:]')"
ARCH="$(uname -m)"

case "$ARCH" in
  x86_64|amd64) ARCH="amd64" ;;
  aarch64|arm64) ARCH="arm64" ;;
  *)
    echo "error: unsupported architecture: $ARCH" >&2
    exit 1
    ;;
esac

case "$OS" in
  linux|darwin) ;;
  *)
    echo "error: unsupported OS: $OS" >&2
    exit 1
    ;;
esac

ARCHIVE="nab_${VERSION}_${OS}_${ARCH}.tar.gz"
URL="https://github.com/${REPO}/releases/download/v${VERSION}/${ARCHIVE}"

mkdir -p "$INSTALL_DIR"

echo "Downloading nab v${VERSION} (${OS}/${ARCH})..." >&2
if command -v curl >/dev/null 2>&1; then
  curl -fsSL "$URL" | tar xz -C "$INSTALL_DIR" nab
elif command -v wget >/dev/null 2>&1; then
  wget -qO- "$URL" | tar xz -C "$INSTALL_DIR" nab
else
  echo "error: curl or wget required" >&2
  exit 1
fi

chmod +x "$INSTALL_DIR/nab"
echo "Installed nab to $INSTALL_DIR/nab" >&2

# Add to PATH for the current session if not already there
case ":$PATH:" in
  *":$INSTALL_DIR:"*) ;;
  *) export PATH="$INSTALL_DIR:$PATH" ;;
esac
