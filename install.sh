#!/usr/bin/env sh
set -e

REPO="hackersfun369/nexus"
BINARY="nexus"
INSTALL_DIR="/usr/local/bin"

# Detect OS and arch
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)

case "$OS" in
  linux)  GOOS="linux" ;;
  darwin) GOOS="darwin" ;;
  *)
    echo "Unsupported OS: $OS"
    exit 1
    ;;
esac

case "$ARCH" in
  x86_64|amd64) GOARCH="amd64" ;;
  arm64|aarch64) GOARCH="arm64" ;;
  *)
    echo "Unsupported architecture: $ARCH"
    exit 1
    ;;
esac

SUFFIX="${GOOS}-${GOARCH}"

# Get latest release version
echo "Fetching latest NEXUS release..."
VERSION=$(curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" \
  | grep '"tag_name"' | sed 's/.*"tag_name": *"\([^"]*\)".*/\1/')

if [ -z "$VERSION" ]; then
  echo "Could not determine latest version. Check https://github.com/${REPO}/releases"
  exit 1
fi

echo "Installing NEXUS ${VERSION} (${SUFFIX})..."

URL="https://github.com/${REPO}/releases/download/${VERSION}/nexus-${SUFFIX}"
TMP=$(mktemp)

curl -fsSL "$URL" -o "$TMP"
chmod +x "$TMP"

# Install to /usr/local/bin (try sudo if needed)
if [ -w "$INSTALL_DIR" ]; then
  mv "$TMP" "${INSTALL_DIR}/${BINARY}"
else
  echo "Installing to ${INSTALL_DIR} (requires sudo)..."
  sudo mv "$TMP" "${INSTALL_DIR}/${BINARY}"
fi

echo ""
echo "✓ NEXUS ${VERSION} installed to ${INSTALL_DIR}/${BINARY}"
echo ""
echo "Get started:"
echo "  nexus serve       # Start web UI"
echo "  nexus             # Terminal chat"
echo "  nexus version     # Show version"
echo ""
