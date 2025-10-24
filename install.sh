#!/bin/bash
set -e

# clauderock installer script
# Usage: curl -fsSL https://raw.githubusercontent.com/OlaHulleberg/clauderock/main/install.sh | bash

REPO="OlaHulleberg/clauderock"
INSTALL_DIR="/usr/local/bin"
FALLBACK_DIR="$HOME/.local/bin"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo "Installing clauderock..."

# Detect OS and architecture
OS="$(uname -s)"
ARCH="$(uname -m)"

case "$OS" in
    Linux*)     OS="linux";;
    Darwin*)    OS="darwin";;
    MINGW*|MSYS*|CYGWIN*) OS="windows";;
    *)
        echo -e "${RED}Error: Unsupported operating system: $OS${NC}"
        exit 1
        ;;
esac

case "$ARCH" in
    x86_64|amd64)   ARCH="amd64";;
    aarch64|arm64)  ARCH="arm64";;
    *)
        echo -e "${RED}Error: Unsupported architecture: $ARCH${NC}"
        exit 1
        ;;
esac

echo "Detected: $OS/$ARCH"

# Determine archive format
if [ "$OS" = "windows" ]; then
    ARCHIVE_EXT="zip"
    BINARY_NAME="clauderock.exe"
else
    ARCHIVE_EXT="tar.gz"
    BINARY_NAME="clauderock"
fi

ARCHIVE_NAME="clauderock_${OS}_${ARCH}.${ARCHIVE_EXT}"

# Fetch latest release info
echo "Fetching latest release..."
RELEASE_URL="https://api.github.com/repos/$REPO/releases/latest"
DOWNLOAD_URL=$(curl -s "$RELEASE_URL" | grep "browser_download_url.*$ARCHIVE_NAME" | cut -d '"' -f 4)

if [ -z "$DOWNLOAD_URL" ]; then
    echo -e "${RED}Error: Could not find release for $OS/$ARCH${NC}"
    echo "Please check https://github.com/$REPO/releases for available downloads"
    exit 1
fi

# Create temp directory
TMP_DIR=$(mktemp -d)
trap "rm -rf $TMP_DIR" EXIT

# Download archive
echo "Downloading $ARCHIVE_NAME..."
cd "$TMP_DIR"
if ! curl -fsSL "$DOWNLOAD_URL" -o "$ARCHIVE_NAME"; then
    echo -e "${RED}Error: Download failed${NC}"
    exit 1
fi

# Extract binary
echo "Extracting..."
if [ "$ARCHIVE_EXT" = "zip" ]; then
    unzip -q "$ARCHIVE_NAME"
else
    tar -xzf "$ARCHIVE_NAME"
fi

# Verify binary exists
if [ ! -f "$BINARY_NAME" ]; then
    echo -e "${RED}Error: Binary not found in archive${NC}"
    exit 1
fi

# Make executable
chmod +x "$BINARY_NAME"

# Try to install to /usr/local/bin, fall back to ~/.local/bin
INSTALLED=false
FINAL_INSTALL_DIR=""

if [ -w "$INSTALL_DIR" ]; then
    mv "$BINARY_NAME" "$INSTALL_DIR/"
    INSTALLED=true
    FINAL_INSTALL_DIR="$INSTALL_DIR"
    echo -e "${GREEN}Installed to $INSTALL_DIR/$BINARY_NAME${NC}"
elif command -v sudo >/dev/null 2>&1; then
    echo "Installing to $INSTALL_DIR requires sudo..."
    if sudo mv "$BINARY_NAME" "$INSTALL_DIR/"; then
        INSTALLED=true
        FINAL_INSTALL_DIR="$INSTALL_DIR"
        echo -e "${GREEN}Installed to $INSTALL_DIR/$BINARY_NAME${NC}"
    fi
fi

# Fallback to user directory
if [ "$INSTALLED" = false ]; then
    echo -e "${YELLOW}Installing to $FALLBACK_DIR (user directory)${NC}"
    mkdir -p "$FALLBACK_DIR"
    mv "$BINARY_NAME" "$FALLBACK_DIR/"
    FINAL_INSTALL_DIR="$FALLBACK_DIR"
    echo -e "${GREEN}Installed to $FALLBACK_DIR/$BINARY_NAME${NC}"

    # Check if in PATH
    if [[ ":$PATH:" != *":$FALLBACK_DIR:"* ]]; then
        echo -e "${YELLOW}Warning: $FALLBACK_DIR is not in your PATH${NC}"
        echo "Add this line to your ~/.bashrc or ~/.zshrc:"
        echo "  export PATH=\"\$PATH:$FALLBACK_DIR\""
    fi
fi

# Verify installation
cd - > /dev/null
if command -v clauderock >/dev/null 2>&1; then
    VERSION=$(clauderock version 2>&1 || echo "unknown")
    echo ""
    echo -e "${GREEN}✓ clauderock installed successfully!${NC}"
    echo "  Version: $VERSION"
    echo ""
    echo "Get started:"
    echo "  clauderock config set profile <your-aws-profile>"
    echo "  clauderock config set region <your-aws-region>"
    echo "  clauderock config set cross-region <us|eu|global>"
    echo "  clauderock"
else
    echo ""
    echo -e "${YELLOW}Note: You may need to restart your shell or run:${NC}"
    echo "  export PATH=\"\$PATH:$FINAL_INSTALL_DIR\""
    echo ""
    echo "Then verify with: clauderock version"
fi
