#!/bin/bash
set -eu

REPO="DiamondLightSource/dra-usbip-driver"
INSTALL_DIR="/usr/local/bin"
SERVICE_NAME="dra-usbip-agent"
BINARY_NAME="agent"

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

DOWNLOAD_URL="https://github.com/${REPO}/releases/download/${TAG}/agent-linux-${GOARCH}"

echo "Installing ${SERVICE_NAME} ${TAG} (linux/${GOARCH})..."

# Download binary
curl -fsSL -o /tmp/${BINARY_NAME} "$DOWNLOAD_URL"
chmod +x /tmp/${BINARY_NAME}
mv /tmp/${BINARY_NAME} "${INSTALL_DIR}/${BINARY_NAME}"

echo "Installed ${INSTALL_DIR}/${BINARY_NAME}"

# Install default config
CONFIG_DIR="/etc/dra-usbip-agent"
CONFIG_FILE="${CONFIG_DIR}/config.yaml"
CONFIG_URL="https://raw.githubusercontent.com/${REPO}/${TAG}/configs/agent.yaml"

mkdir -p "${CONFIG_DIR}"
if [ ! -f "${CONFIG_FILE}" ]; then
  curl -fsSL -o "${CONFIG_FILE}" "${CONFIG_URL}"
  echo "Installed default config to ${CONFIG_FILE}"
else
  echo "Config ${CONFIG_FILE} already exists, skipping"
fi

# Create systemd service
cat > /etc/systemd/system/${SERVICE_NAME}.service <<EOF
[Unit]
Description=DRA USB/IP Agent
After=network.target

[Service]
ExecStart=${INSTALL_DIR}/${BINARY_NAME} --bind-all-devices --auto-bind --config=${CONFIG_FILE}
Restart=on-failure
RestartSec=5

[Install]
WantedBy=multi-user.target
EOF

systemctl daemon-reload
systemctl enable "${SERVICE_NAME}"
systemctl start "${SERVICE_NAME}"

echo "Service ${SERVICE_NAME} enabled and started."
