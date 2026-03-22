#!/usr/bin/env bash
# cowork-setup.sh — Download nab binary for Claude Cowork's Linux VM
#
# Usage (run inside Cowork workspace):
#   bash scripts/cowork-setup.sh
#   # or from a GitHub URL:
#   curl -fsSL https://raw.githubusercontent.com/kfriede/nab/main/scripts/cowork-setup.sh | bash
#
# This script detects the VM architecture, downloads the correct nab binary
# from GitHub releases, and places it in the current working directory.

set -euo pipefail

REPO="kfriede/nab"
BINARY_NAME="nab"

# Detect architecture
ARCH=$(uname -m)
case "$ARCH" in
  x86_64)  GOARCH="amd64" ;;
  aarch64) GOARCH="arm64" ;;
  arm64)   GOARCH="arm64" ;;
  *)
    echo "Error: Unsupported architecture: $ARCH" >&2
    exit 1
    ;;
esac

OS=$(uname -s | tr '[:upper:]' '[:lower:]')
if [ "$OS" != "linux" ]; then
  echo "Warning: This script is designed for Claude Cowork's Linux VM." >&2
  echo "Detected OS: $OS. Proceeding anyway..." >&2
fi

# Determine version (latest release or user-specified)
VERSION="${NAB_VERSION:-latest}"

if [ "$VERSION" = "latest" ]; then
  echo "Fetching latest release..." >&2
  VERSION=$(curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" | grep '"tag_name"' | sed -E 's/.*"([^"]+)".*/\1/')
  if [ -z "$VERSION" ]; then
    echo "Error: Could not determine latest version" >&2
    exit 1
  fi
fi

# Strip leading 'v' for the archive name
VERSION_NUM="${VERSION#v}"

ARCHIVE_NAME="${BINARY_NAME}_${VERSION_NUM}_${OS}_${GOARCH}.tar.gz"
DOWNLOAD_URL="https://github.com/${REPO}/releases/download/${VERSION}/${ARCHIVE_NAME}"

echo "Downloading nab ${VERSION} for ${OS}/${GOARCH}..." >&2
echo "URL: ${DOWNLOAD_URL}" >&2

# Download and extract
TMPDIR=$(mktemp -d)
trap 'rm -rf "$TMPDIR"' EXIT

if ! curl -fsSL "$DOWNLOAD_URL" -o "${TMPDIR}/${ARCHIVE_NAME}"; then
  echo "Error: Download failed. Check that version ${VERSION} exists and has a ${OS}/${GOARCH} build." >&2
  exit 1
fi

tar -xzf "${TMPDIR}/${ARCHIVE_NAME}" -C "$TMPDIR"

# Move binary to current directory
mv "${TMPDIR}/${BINARY_NAME}" "./${BINARY_NAME}"
chmod +x "./${BINARY_NAME}"

echo "" >&2
echo "✓ nab ${VERSION} installed to $(pwd)/${BINARY_NAME}" >&2
echo "" >&2
echo "Test it:" >&2
echo "  ./nab version" >&2
echo "  ./nab schema" >&2
echo "" >&2
echo "Configure:" >&2
echo "  export NAB_TOKEN=<your-ynab-personal-access-token>" >&2
echo "  export NAB_BUDGET=last-used" >&2
