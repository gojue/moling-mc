#!/bin/bash

set -e

echo "Welcome to MoLing MCP Server initialization script."
echo "Home page: https://gojue.cc/moling"
echo "Github: https://github.com/gojue/moling"

# Determine the OS and architecture
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)

case $ARCH in
  x86_64)
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

# Determine the download URL
VERSION="v0.0.1"
BASE_URL="https://github.com/gojue/moling/releases/download/${VERSION}"
FILE_NAME="moling-${VERSION}-${OS}-${ARCH}"

if [ "$OS" = "darwin" ] || [ "$OS" = "linux" ]; then
  FILE_NAME="${FILE_NAME}.tar.gz"
else
  echo "Unsupported OS: $OS"
  exit 1
fi

DOWNLOAD_URL="${BASE_URL}/${FILE_NAME}"

# Download the installation package
echo "Downloading ${DOWNLOAD_URL}..."
curl -LO "${DOWNLOAD_URL}"

# Extract the package
tar -xzf "${FILE_NAME}" -C moling

# Move the binary to /usr/local/bin
sudo mv moling/moling /usr/local/bin/moling
sudo chmod +x /usr/local/bin/moling

# Clean up
rm -rf moling "${FILE_NAME}"

echo "MoLing has been installed successfully!"