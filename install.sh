#!/bin/sh
set -e

REPO="chitacloud/chita-cli"
BINARY="chita"

# Detect OS
OS="$(uname -s)"
case "$OS" in
  Linux)  OS="linux" ;;
  Darwin) OS="darwin" ;;
  *)
    echo "Unsupported OS: $OS" >&2
    exit 1
    ;;
esac

# Detect architecture
ARCH="$(uname -m)"
case "$ARCH" in
  x86_64 | amd64) ARCH="amd64" ;;
  arm64 | aarch64) ARCH="arm64" ;;
  *)
    echo "Unsupported architecture: $ARCH" >&2
    exit 1
    ;;
esac

# Resolve latest version
VERSION="${CHITA_VERSION:-}"
if [ -z "$VERSION" ]; then
  VERSION="$(curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" \
    | grep '"tag_name"' | sed 's/.*"tag_name": *"\([^"]*\)".*/\1/')"
fi

if [ -z "$VERSION" ]; then
  echo "Could not determine latest version" >&2
  exit 1
fi

ARCHIVE="${BINARY}_${OS}_${ARCH}.tar.gz"
URL="https://github.com/${REPO}/releases/download/${VERSION}/${ARCHIVE}"

# Determine install dir
INSTALL_DIR="/usr/local/bin"
if [ ! -w "$INSTALL_DIR" ]; then
  INSTALL_DIR="$HOME/.local/bin"
  mkdir -p "$INSTALL_DIR"
fi

echo "Installing chita ${VERSION} (${OS}/${ARCH}) → ${INSTALL_DIR}/${BINARY}"

TMP="$(mktemp -d)"
trap 'rm -rf "$TMP"' EXIT

curl -fsSL "$URL" -o "$TMP/$ARCHIVE"
tar -xzf "$TMP/$ARCHIVE" -C "$TMP"
install -m 755 "$TMP/$BINARY" "$INSTALL_DIR/$BINARY"

echo ""
echo "✓ chita installed successfully"
echo ""

# Warn if not in PATH
case ":$PATH:" in
  *":$INSTALL_DIR:"*) ;;
  *)
    echo "  Add to your shell profile:"
    echo "    export PATH=\"\$PATH:$INSTALL_DIR\""
    echo ""
    ;;
esac

"$INSTALL_DIR/$BINARY" --version
