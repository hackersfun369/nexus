#!/usr/bin/env sh
set -e

BINARY="nexus"
GREEN='\033[0;32m'
RED='\033[0;31m'
BLUE='\033[0;34m'
RESET='\033[0m'

info()    { printf "${BLUE}[nexus]${RESET} %s\n" "$1"; }
success() { printf "${GREEN}[nexus]${RESET} %s\n" "$1"; }
error()   { printf "${RED}[nexus]${RESET} %s\n" "$1" >&2; exit 1; }

main() {
    BINARY_PATH=$(command -v "$BINARY" 2>/dev/null || echo "")
    if [ -z "$BINARY_PATH" ]; then
        error "nexus is not installed or not in PATH"
    fi

    info "Removing $BINARY_PATH..."
    if ! rm "$BINARY_PATH" 2>/dev/null; then
        if command -v sudo >/dev/null 2>&1; then
            sudo rm "$BINARY_PATH"
        else
            error "Could not remove $BINARY_PATH — permission denied"
        fi
    fi

    # Remove nexus data directory if requested
    NEXUS_DATA="$HOME/.nexus"
    if [ -d "$NEXUS_DATA" ]; then
        printf "Remove nexus data directory %s? [y/N] " "$NEXUS_DATA"
        read -r ANSWER
        if [ "$ANSWER" = "y" ] || [ "$ANSWER" = "Y" ]; then
            rm -rf "$NEXUS_DATA"
            success "Removed $NEXUS_DATA"
        else
            info "Kept $NEXUS_DATA"
        fi
    fi

    success "nexus uninstalled successfully"
}

main "$@"
