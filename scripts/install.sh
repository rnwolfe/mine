#!/usr/bin/env bash
set -euo pipefail

# mine installer — fast, safe, opinionated.
#
# Usage:
#   curl -fsSL https://mine.rwolfe.io/install | bash
#
# Or with a specific version:
#   curl -fsSL https://mine.rwolfe.io/install | bash -s -- v0.1.0

VERSION="${1:-latest}"
REPO="rnwolfe/mine"
BINARY="mine"
INSTALL_DIR="${HOME}/.local/bin"

main() {
    echo "⛏  Installing mine..."
    echo ""

    # Detect OS and arch
    OS="$(uname -s | tr '[:upper:]' '[:lower:]')"
    ARCH="$(uname -m)"

    case "$ARCH" in
        x86_64|amd64) ARCH="amd64" ;;
        aarch64|arm64) ARCH="arm64" ;;
        *) echo "  ✗ Unsupported architecture: $ARCH"; exit 1 ;;
    esac

    case "$OS" in
        linux|darwin) ;;
        *) echo "  ✗ Unsupported OS: $OS"; exit 1 ;;
    esac

    # Resolve version
    if [ "$VERSION" = "latest" ]; then
        VERSION=$(curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" 2>/dev/null \
            | grep '"tag_name"' | cut -d'"' -f4) || true
        if [ -z "$VERSION" ]; then
            echo "  ✗ No releases found. mine may not have a release yet."
            echo ""
            echo "  To build from source instead:"
            echo "    git clone https://github.com/${REPO}.git"
            echo "    cd mine && make install"
            echo ""
            exit 1
        fi
    fi

    echo "  Version:  $VERSION"
    echo "  OS:       $OS"
    echo "  Arch:     $ARCH"
    echo ""

    # Download
    URL="https://github.com/${REPO}/releases/download/${VERSION}/${BINARY}_${OS}_${ARCH}.tar.gz"
    TMPDIR=$(mktemp -d)
    trap "rm -rf $TMPDIR" EXIT

    echo "  Downloading..."
    if ! curl -fsSL "$URL" -o "$TMPDIR/mine.tar.gz"; then
        echo ""
        echo "  ✗ Download failed. Release ${VERSION} may not exist for ${OS}/${ARCH}."
        echo "  Check available releases: https://github.com/${REPO}/releases"
        exit 1
    fi
    tar -xzf "$TMPDIR/mine.tar.gz" -C "$TMPDIR"

    # Install
    mkdir -p "$INSTALL_DIR"
    mv "$TMPDIR/$BINARY" "$INSTALL_DIR/$BINARY"
    chmod +x "$INSTALL_DIR/$BINARY"

    echo ""
    echo "  ✓ Installed to $INSTALL_DIR/$BINARY"

    # Check PATH
    if ! echo "$PATH" | tr ':' '\n' | grep -qx "$INSTALL_DIR"; then
        echo ""
        echo "  ⚠  $INSTALL_DIR is not in your PATH."
        echo "  Add this to your shell config:"
        echo ""
        echo "    export PATH=\"\$HOME/.local/bin:\$PATH\""
    fi

    echo ""
    echo "  Run 'mine init' to get started!"
    echo ""
}

main "$@"
