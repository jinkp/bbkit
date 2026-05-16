#!/bin/sh
# bbkit-cli installer for Linux/macOS
# Usage: curl -fsSL https://raw.githubusercontent.com/jinkp/bbkit/main/install.sh | sh

set -e

echo ""
echo "  bbkit — Bitbucket Cloud CLI"
echo ""

# Check Node.js
if ! command -v node >/dev/null 2>&1; then
    echo "  [ERROR] Node.js is required but not found."
    echo "  Install it from https://nodejs.org (v20+)"
    echo ""
    exit 1
fi

NODE_VERSION=$(node --version | sed 's/^v//')
MAJOR=$(echo "$NODE_VERSION" | cut -d. -f1)

if [ "$MAJOR" -lt 20 ]; then
    echo "  [ERROR] Node.js v20+ is required. Found v$NODE_VERSION"
    echo "  Update from https://nodejs.org"
    echo ""
    exit 1
fi

echo "  Node.js v$NODE_VERSION detected"

# Install bbkit-cli globally via npm
echo "  Installing bbkit-cli..."

npm install -g bbkit-cli >/dev/null 2>&1

# Verify installation
if ! command -v bbk >/dev/null 2>&1; then
    echo "  [ERROR] bbk command not found after install."
    echo "  Try running: npm install -g bbkit-cli"
    exit 1
fi

BBK_VERSION=$(bbk --version)

echo ""
echo "  bbkit v$BBK_VERSION installed successfully!"
echo ""
echo "  Get started:"
echo "    bbk setup        Configure credentials and workspace"
echo "    bbk auth login   Authenticate with Bitbucket"
echo "    bbk repo list    List your repositories"
echo "    bbk --help       See all commands"
echo ""
