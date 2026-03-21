#!/bin/bash
set -eu

REPO="DiamondLightSource/dra-usbip-driver"
INSTALL_DIR="/usr/local/bin"
SERVICE_NAME="pico-mac-sender"
BINARY_NAME="pico-mac-sender"

# Detect architecture
ARCH=$(uname -m)
case "$ARCH" in
  x86_64)  GOARCH="amd64" ;;
  aarch64) GOARCH="arm64" ;;
  *)
    echo "Error: unsupported architecture: $ARCH" >&2
    exit 1
    ;;
esac

# Determine version — use first argument, VERSION env var, or fetch latest release
if [ -n "${1:-}" ]; then
  TAG="$1"
elif [ -n "${VERSION:-}" ]; then
  TAG="$VERSION"
else
  TAG=$(curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" | grep '"tag_name"' | cut -d'"' -f4)
  if [ -z "$TAG" ]; then
    echo "Error: could not determine latest release" >&2
    exit 1
  fi
fi

DOWNLOAD_URL="https://github.com/${REPO}/releases/download/${TAG}/${BINARY_NAME}-linux-${GOARCH}"

echo "Installing ${SERVICE_NAME} ${TAG} (linux/${GOARCH})..."

# Download binary
curl -fsSL -o /tmp/${BINARY_NAME} "$DOWNLOAD_URL"
chmod +x /tmp/${BINARY_NAME}
mv /tmp/${BINARY_NAME} "${INSTALL_DIR}/${BINARY_NAME}"

echo "Installed ${INSTALL_DIR}/${BINARY_NAME}"

# Create systemd service
cat > /etc/systemd/system/${SERVICE_NAME}.service <<EOF
[Unit]
Description=Send MAC address to Raspberry Pi Pico OLED display
After=multi-user.target

[Service]
ExecStart=${INSTALL_DIR}/${BINARY_NAME}
Restart=on-failure
RestartSec=5

[Install]
WantedBy=multi-user.target
EOF

systemctl daemon-reload
systemctl enable "${SERVICE_NAME}"
systemctl start "${SERVICE_NAME}"

echo "Service ${SERVICE_NAME} enabled and started."
