#!/usr/bin/env sh
set -e

# ── CONFIG ────────────────────────────────────────────
REPO="hackersfun369/nexus"
BINARY="nexus"
INSTALL_DIR=""
VERSION="${NEXUS_VERSION:-latest}"

# ── COLORS ────────────────────────────────────────────
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
RESET='\033[0m'

info()    { printf "${BLUE}[nexus]${RESET} %s\n" "$1"; }
success() { printf "${GREEN}[nexus]${RESET} %s\n" "$1"; }
warn()    { printf "${YELLOW}[nexus]${RESET} %s\n" "$1"; }
error()   { printf "${RED}[nexus]${RESET} %s\n" "$1" >&2; exit 1; }

# ── DETECT OS ─────────────────────────────────────────
detect_os() {
    OS="$(uname -s)"
    case "$OS" in
        Linux)  OS="linux" ;;
        Darwin) OS="darwin" ;;
        MINGW*|MSYS*|CYGWIN*) OS="windows" ;;
        *) error "Unsupported OS: $OS" ;;
    esac
}

# ── DETECT ARCH ───────────────────────────────────────
detect_arch() {
    ARCH="$(uname -m)"
    case "$ARCH" in
        x86_64|amd64)   ARCH="amd64" ;;
        aarch64|arm64)  ARCH="arm64" ;;
        *) error "Unsupported architecture: $ARCH" ;;
    esac
}

# ── DETECT INSTALL DIR ────────────────────────────────
detect_install_dir() {
    if [ -n "$NEXUS_INSTALL_DIR" ]; then
        INSTALL_DIR="$NEXUS_INSTALL_DIR"
    elif [ -w "/usr/local/bin" ]; then
        INSTALL_DIR="/usr/local/bin"
    elif [ -d "$HOME/.local/bin" ]; then
        INSTALL_DIR="$HOME/.local/bin"
    else
        INSTALL_DIR="$HOME/.local/bin"
        mkdir -p "$INSTALL_DIR"
        warn "Created $INSTALL_DIR — make sure it's in your PATH"
    fi
}

# ── CHECK DEPENDENCIES ────────────────────────────────
check_deps() {
    for cmd in curl tar; do
        if ! command -v "$cmd" >/dev/null 2>&1; then
            error "Required tool not found: $cmd"
        fi
    done
}

# ── GET LATEST VERSION ────────────────────────────────
get_latest_version() {
    if [ "$VERSION" = "latest" ]; then
        info "Fetching latest release version..."
        VERSION=$(curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" \
            | grep '"tag_name"' \
            | sed 's/.*"tag_name": *"\([^"]*\)".*/\1/')
        if [ -z "$VERSION" ]; then
            error "Could not determine latest version. Set NEXUS_VERSION explicitly."
        fi
    fi
    info "Installing nexus $VERSION"
}

# ── BUILD DOWNLOAD URL ────────────────────────────────
build_url() {
    if [ "$OS" = "windows" ]; then
        FILENAME="${BINARY}-${OS}-${ARCH}.exe"
    else
        FILENAME="${BINARY}-${OS}-${ARCH}"
    fi
    DOWNLOAD_URL="https://github.com/${REPO}/releases/download/${VERSION}/${FILENAME}"
}

# ── DOWNLOAD ──────────────────────────────────────────
download() {
    TMP_DIR=$(mktemp -d)
    TMP_BIN="${TMP_DIR}/${BINARY}"

    info "Downloading $FILENAME..."
    if ! curl -fsSL "$DOWNLOAD_URL" -o "$TMP_BIN"; then
        rm -rf "$TMP_DIR"
        error "Download failed: $DOWNLOAD_URL"
    fi

    # Download checksum if available
    CHECKSUM_URL="${DOWNLOAD_URL}.sha256"
    if curl -fsSL "$CHECKSUM_URL" -o "${TMP_BIN}.sha256" 2>/dev/null; then
        info "Verifying checksum..."
        if command -v sha256sum >/dev/null 2>&1; then
            EXPECTED=$(cat "${TMP_BIN}.sha256" | awk '{print $1}')
            ACTUAL=$(sha256sum "$TMP_BIN" | awk '{print $1}')
        elif command -v shasum >/dev/null 2>&1; then
            EXPECTED=$(cat "${TMP_BIN}.sha256" | awk '{print $1}')
            ACTUAL=$(shasum -a 256 "$TMP_BIN" | awk '{print $1}')
        else
            warn "No checksum tool found — skipping verification"
            EXPECTED=""
            ACTUAL=""
        fi
        if [ -n "$EXPECTED" ] && [ "$EXPECTED" != "$ACTUAL" ]; then
            rm -rf "$TMP_DIR"
            error "Checksum mismatch! Expected: $EXPECTED  Got: $ACTUAL"
        fi
        success "Checksum verified"
    else
        warn "No checksum file found — skipping verification"
    fi

    chmod +x "$TMP_BIN"
    echo "$TMP_BIN"
}

# ── INSTALL ───────────────────────────────────────────
install_binary() {
    TMP_BIN="$1"
    DEST="${INSTALL_DIR}/${BINARY}"

    info "Installing to $DEST..."
    if ! mv "$TMP_BIN" "$DEST" 2>/dev/null; then
        # Try with sudo if mv failed
        if command -v sudo >/dev/null 2>&1; then
            warn "Permission denied — trying with sudo..."
            sudo mv "$TMP_BIN" "$DEST"
        else
            rm -rf "$(dirname $TMP_BIN)"
            error "Could not install to $DEST — permission denied"
        fi
    fi
    rm -rf "$(dirname $TMP_BIN)"
}

# ── VERIFY ────────────────────────────────────────────
verify() {
    if command -v "$BINARY" >/dev/null 2>&1; then
        INSTALLED_VERSION=$("$BINARY" version 2>/dev/null || echo "unknown")
        success "nexus installed successfully!"
        success "  Binary:  $(command -v $BINARY)"
        success "  Version: $INSTALLED_VERSION"
    else
        warn "nexus installed to $INSTALL_DIR but not found in PATH"
        warn "Add this to your shell profile:"
        warn "  export PATH=\"\$PATH:$INSTALL_DIR\""
    fi
}

# ── MAIN ──────────────────────────────────────────────
main() {
    printf "\n${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${RESET}\n"
    printf "${BLUE}  nexus — intelligent development system   ${RESET}\n"
    printf "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${RESET}\n\n"

    check_deps
    detect_os
    detect_arch
    detect_install_dir
    get_latest_version
    build_url
    TMP_BIN=$(download)
    install_binary "$TMP_BIN"
    verify

    printf "\n${GREEN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${RESET}\n"
    printf "${GREEN}  Installation complete! Run: nexus version  ${RESET}\n"
    printf "${GREEN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${RESET}\n\n"
}

main "$@"
