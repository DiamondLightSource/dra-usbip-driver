#!/usr/bin/env bash
# Build a customised Raspberry Pi OS SD card image with the DRA USB/IP agent
# pre-installed. Intended to be called from the sdcard.yml GitHub Actions
# workflow.
#
# Required environment variables:
#   AGENT_VERSION  – release tag to install (e.g. 0.3.0)

set -euo pipefail

: "${AGENT_VERSION:?AGENT_VERSION must be set}"

REPO="DiamondLightSource/dra-usbip-driver"

# --- Download base image ------------------------------------------------

curl -fSL -o raspi.img.xz \
  https://downloads.raspberrypi.com/raspios_lite_arm64_latest
unxz raspi.img.xz

# --- Expand image to fit additional packages ----------------------------

truncate -s +500M raspi.img

LOOP=$(sudo losetup -fP --show raspi.img)
# Export for the cleanup trap and the workflow's unmount step
echo "LOOP=$LOOP" >> "$GITHUB_ENV"

sudo parted -s "$LOOP" resizepart 2 100%
sudo e2fsck -fy "${LOOP}p2" || true
sudo resize2fs "${LOOP}p2"

# --- Mount filesystems --------------------------------------------------

sudo mount "${LOOP}p2" /mnt
sudo mount "${LOOP}p1" /mnt/boot/firmware

sudo mount -t proc proc /mnt/proc
sudo mount -t sysfs sys /mnt/sys
sudo mount --bind /dev /mnt/dev
sudo mount --bind /dev/pts /mnt/dev/pts
sudo cp /etc/resolv.conf /mnt/etc/resolv.conf

# --- First-boot user (local/local) and SSH ------------------------------

echo "local:$(echo local | openssl passwd -6 -stdin)" | \
  sudo tee /mnt/boot/firmware/userconf.txt
sudo touch /mnt/boot/firmware/ssh

# --- Install USB/IP inside chroot --------------------------------------

sudo chroot /mnt bash -e <<'CHROOT'
export DEBIAN_FRONTEND=noninteractive

apt-get update
cd /tmp && apt-get download usbip
dpkg-deb -x usbip_*.deb /
rm usbip_*.deb
apt-get clean

cat > /etc/modules-load.d/usbip.conf <<MOD
usbip-core
usbip-host
MOD

cat > /etc/systemd/system/usbipd.service <<SVC
[Unit]
Description=USB/IP Daemon
After=network.target

[Service]
Type=forking
ExecStart=/usr/sbin/usbipd -D
Restart=on-failure

[Install]
WantedBy=multi-user.target
SVC
systemctl enable usbipd.service
CHROOT

# --- Install agent binary -----------------------------------------------

sudo curl -fsSL -o /mnt/usr/local/bin/agent \
  "https://github.com/${REPO}/releases/download/${AGENT_VERSION}/agent-linux-arm64"
sudo chmod +x /mnt/usr/local/bin/agent

sudo mkdir -p /mnt/etc/dra-usbip-agent
sudo curl -fsSL -o /mnt/etc/dra-usbip-agent/config.yaml \
  "https://raw.githubusercontent.com/${REPO}/${AGENT_VERSION}/configs/agent.yaml"

sudo tee /mnt/etc/systemd/system/dra-usbip-agent.service <<SVC
[Unit]
Description=DRA USB/IP Agent
After=network.target

[Service]
ExecStart=/usr/local/bin/agent --bind-all-devices --auto-bind --config=/etc/dra-usbip-agent/config.yaml
Restart=on-failure
RestartSec=5

[Install]
WantedBy=multi-user.target
SVC
sudo chroot /mnt systemctl enable dra-usbip-agent

# --- Install pico-mac-sender binary ------------------------------------

sudo curl -fsSL -o /mnt/usr/local/bin/pico-mac-sender \
  "https://github.com/${REPO}/releases/download/${AGENT_VERSION}/pico-mac-sender-linux-arm64"
sudo chmod +x /mnt/usr/local/bin/pico-mac-sender

sudo tee /mnt/etc/systemd/system/pico-mac-sender.service <<SVC
[Unit]
Description=Send MAC address to Raspberry Pi Pico OLED display
After=multi-user.target

[Service]
ExecStart=/usr/local/bin/pico-mac-sender
Restart=on-failure
RestartSec=5

[Install]
WantedBy=multi-user.target
SVC
sudo chroot /mnt systemctl enable pico-mac-sender

# --- Disable WiFi and Bluetooth ----------------------------------------

sudo tee -a /mnt/boot/firmware/config.txt <<CFG

dtoverlay=disable-wifi
dtoverlay=disable-bt
CFG

# --- Read-only overlay mode ---------------------------------------------

sudo chroot /mnt bash -e <<'CHROOT'
export DEBIAN_FRONTEND=noninteractive
apt-get install -y cryptsetup cryptsetup-bin overlayroot
apt-get clean
CHROOT

sudo tee /mnt/etc/systemd/system/runonce.service <<SVC
[Unit]
Description=Run script once on next boot
ConditionPathExists=/var/local/runonce.sh
After=multi-user.target

[Service]
Type=oneshot
ExecStart=/bin/bash /var/local/runonce.sh

[Install]
WantedBy=multi-user.target
SVC
sudo chroot /mnt systemctl enable runonce.service

sudo tee /mnt/var/local/runonce.sh <<'SCRIPT'
#!/bin/bash
set -x

# Disable this script from running again
mv /var/local/runonce.sh /var/local/runonce.sh.done

# Disable services that report errors in RO mode
systemctl mask dphys-swapfile.service
systemctl mask systemd-zram-setup@zram0.service

# Enable read-only overlay mode
raspi-config nonint do_overlayfs 0

# Reboot to pick up the change
sync; reboot
SCRIPT
sudo chmod +x /mnt/var/local/runonce.sh

# --- Clean up inside image ---------------------------------------------

sudo rm /mnt/etc/resolv.conf
