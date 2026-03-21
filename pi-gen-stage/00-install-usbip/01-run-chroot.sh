#!/bin/bash
set -eu

# Configure USB/IP kernel modules to load at boot
cat > /etc/modules-load.d/usbip.conf <<EOF
usbip-core
usbip-host
EOF

# Create systemd service for usbipd
cat > /etc/systemd/system/usbipd.service <<EOF
[Unit]
Description=USB/IP Daemon
After=network.target

[Service]
Type=forking
ExecStart=/usr/sbin/usbipd -D
Restart=on-failure

[Install]
WantedBy=multi-user.target
EOF

systemctl enable usbipd.service
