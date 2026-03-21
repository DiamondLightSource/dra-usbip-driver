#!/bin/bash
set -eu

# Create a run-once service that enables read-only overlay on first boot
cat > /etc/systemd/system/runonce.service <<EOF
[Unit]
Description=Run script once on next boot
ConditionPathExists=/var/local/runonce.sh
After=multi-user.target

[Service]
Type=oneshot
ExecStart=/bin/bash /var/local/runonce.sh

[Install]
WantedBy=multi-user.target
EOF

systemctl enable runonce.service

# Create the runonce script
cat > /var/local/runonce.sh <<'SCRIPT'
#!/bin/bash
set -x

# Disable this script from running again
mv /var/local/runonce.sh /var/local/runonce.sh.done

# Disable services that will report errors when in RO mode
systemctl mask dphys-swapfile.service
systemctl mask systemd-zram-setup@zram0.service

# Enable read-only overlay mode
raspi-config nonint do_overlayfs 0

# Reboot to pick up the change
sync; reboot
SCRIPT

chmod +x /var/local/runonce.sh

# Disable WiFi and Bluetooth for production
cat >> /boot/firmware/config.txt <<EOF

dtoverlay=disable-wifi
dtoverlay=disable-bt
EOF
