#!/bin/bash
set -eu

REPO="DiamondLightSource/dra-usbip-driver"
INSTALL_DIR="/usr/local/bin"
SERVICE_NAME="pico-mac-sender"
BINARY_NAME="pico-mac-sender"
GOARCH="arm64"

if [ -z "${AGENT_VERSION:-}" ]; then
  echo "Error: AGENT_VERSION must be set" >&2
  exit 1
fi

DOWNLOAD_URL="https://github.com/${REPO}/releases/download/${AGENT_VERSION}/${BINARY_NAME}-linux-${GOARCH}"

echo "Installing ${SERVICE_NAME} ${AGENT_VERSION} (linux/${GOARCH})..."

curl -fsSL -o "${INSTALL_DIR}/${BINARY_NAME}" "$DOWNLOAD_URL"
chmod +x "${INSTALL_DIR}/${BINARY_NAME}"

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

systemctl enable "${SERVICE_NAME}"
