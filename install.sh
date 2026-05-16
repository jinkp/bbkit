#!/bin/sh
# bbkit installer for Linux/macOS
# Usage: curl -fsSL https://raw.githubusercontent.com/jinkp/bbkit/main/install.sh | sh

set -eu

RELEASES_URL="https://github.com/jinkp/bbkit/releases"
INSTALL_DIR="$HOME/.local/bin"
TARGET="$INSTALL_DIR/bbk"
ORIGINAL_PATH="$PATH"

fail() {
    echo ""
    echo "bbkit install failed: $1" >&2
    echo "Download a release manually from: $RELEASES_URL" >&2
    exit 1
}

OS_NAME=$(uname -s)
case "$OS_NAME" in
    Linux) OS="linux" ;;
    Darwin) OS="darwin" ;;
    *) fail "unsupported operating system: $OS_NAME" ;;
esac

ARCH_NAME=$(uname -m)
case "$ARCH_NAME" in
    x86_64|amd64) ARCH="amd64" ;;
    arm64|aarch64) ARCH="arm64" ;;
    *) fail "unsupported architecture: $ARCH_NAME" ;;
esac

ASSET="bbk-$OS-$ARCH"
URL="https://github.com/jinkp/bbkit/releases/latest/download/$ASSET"

if command -v curl >/dev/null 2>&1; then
    DOWNLOAD='curl -fsSL'
elif command -v wget >/dev/null 2>&1; then
    DOWNLOAD='wget -qO-'
else
    fail "curl or wget is required"
fi

mkdir -p "$INSTALL_DIR"
TMP_FILE=$(mktemp)
trap 'rm -f "$TMP_FILE"' EXIT

echo ""
echo "bbkit installer"
echo "Downloading $ASSET..."

if [ "$DOWNLOAD" = 'curl -fsSL' ]; then
    curl -fsSL "$URL" -o "$TMP_FILE" || fail "download failed"
else
    wget -qO "$TMP_FILE" "$URL" || fail "download failed"
fi

chmod +x "$TMP_FILE" || fail "could not make binary executable"
mv "$TMP_FILE" "$TARGET" || fail "could not install binary to $TARGET"

PATH="$INSTALL_DIR:$PATH"
if ! bbk --version >/dev/null 2>&1; then
    fail "installed binary did not pass 'bbk --version'"
fi

echo ""
echo "bbkit installed successfully to $TARGET"
bbk --version

# Persist PATH if $INSTALL_DIR is not already on the user's PATH
case ":$ORIGINAL_PATH:" in
    *":$INSTALL_DIR:"*)
        ;;
    *)
        EXPORT_LINE="export PATH=\"$INSTALL_DIR:\$PATH\""
        SHELL_NAME=$(basename "${SHELL:-/bin/sh}")
        case "$SHELL_NAME" in
            zsh)  RC_FILE="$HOME/.zshrc" ;;
            bash) RC_FILE="$HOME/.bashrc" ;;
            fish) RC_FILE="" ;;
            *)    RC_FILE="$HOME/.profile" ;;
        esac

        if [ "$SHELL_NAME" = "fish" ]; then
            FISH_CONFIG="$HOME/.config/fish/config.fish"
            FISH_LINE="set -gx PATH $INSTALL_DIR \$PATH"
            if [ -f "$FISH_CONFIG" ] && grep -qF "$INSTALL_DIR" "$FISH_CONFIG" 2>/dev/null; then
                : # already present
            else
                mkdir -p "$(dirname "$FISH_CONFIG")"
                echo "$FISH_LINE" >> "$FISH_CONFIG"
                echo "Added $INSTALL_DIR to PATH in $FISH_CONFIG"
            fi
            echo "Restart your shell or run:"
            echo "  $FISH_LINE"
        elif [ -n "$RC_FILE" ]; then
            if [ -f "$RC_FILE" ] && grep -qF "$INSTALL_DIR" "$RC_FILE" 2>/dev/null; then
                : # already present
            else
                echo "$EXPORT_LINE" >> "$RC_FILE"
                echo "Added $INSTALL_DIR to PATH in $RC_FILE"
            fi
            echo "Restart your shell or run:"
            echo "  $EXPORT_LINE"
        fi
        ;;
esac

echo ""
echo "Get started:"
echo "  bbk setup"
echo "  bbk auth login"
echo "  bbk repo list"
