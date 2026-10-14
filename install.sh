#!/bin/sh
set -e

# BitRun — One-Line Installer
# Usage: curl -fsSL https://raw.githubusercontent.com/IshekKhal/bitrun/main/install.sh | sh

VERSION="0.1.1-alpha"
INSTALL_DIR="$HOME/.bitrun/bin"
REPO="IshekKhal/bitrun"

echo "=== BitRun Installer (v$VERSION) ==="

OS="$(uname -s | tr '[:upper:]' '[:lower:]')"
ARCH="$(uname -m)"

case "$ARCH" in
  x86_64|amd64)
    ARCH="amd64"
    ;;
  arm64|aarch64)
    ARCH="arm64"
    ;;
  *)
    echo "Unsupported architecture: $ARCH"
    exit 1
    ;;
esac

echo "Detected host: $OS/$ARCH"

mkdir -p "$INSTALL_DIR"
mkdir -p "$HOME/.bitrun/models"

TARBALL="bitrun-${VERSION}-${OS}-${ARCH}.tar.gz"
DOWNLOAD_URL="https://github.com/${REPO}/releases/download/v${VERSION}/${TARBALL}"

echo "Downloading ${TARBALL} from GitHub Releases..."
if curl -fsSL "$DOWNLOAD_URL" | tar -xz -C "$HOME/.bitrun" --strip-components=1; then
  chmod +x "$INSTALL_DIR"/* 2>/dev/null || true
  echo "Binaries installed to $INSTALL_DIR"
else
  echo "[NOTE] Release asset not yet published on GitHub Releases or network error."
  echo "You can build locally with scripts/build_*.sh"
fi

# Check shell profile
SHELL_PROFILE=""
if [ -f "$HOME/.zshrc" ]; then
  SHELL_PROFILE="$HOME/.zshrc"
elif [ -f "$HOME/.bashrc" ]; then
  SHELL_PROFILE="$HOME/.bashrc"
fi

if [ -n "$SHELL_PROFILE" ]; then
  if ! grep -q 'export PATH="$HOME/.bitrun/bin:$PATH"' "$SHELL_PROFILE"; then
    echo 'export PATH="$HOME/.bitrun/bin:$PATH"' >> "$SHELL_PROFILE"
    echo "Added $INSTALL_DIR to PATH in $SHELL_PROFILE"
  fi
fi

echo ""
echo "BitRun installed successfully!"
echo "Run 'bitrun --help' to get started."
echo "Run 'bitrun pull bitnet-70m' to download your first model."
