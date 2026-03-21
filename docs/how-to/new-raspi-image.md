# Create a New Raspberry Pi Boot Image From Scratch

## Introduction

Normal operation for setting up a new Raspberry Pi is to use the provided
pre-built image. See {doc}`/tutorials/commission-raspi`.

This page is only needed if you want to create a completely new SD card
image, perhaps because a new version of Raspberry Pi OS has been released.

If you just want to update the agent version on an existing image, see
{doc}`update-raspi-image`.

## Prerequisites

You will need:

- A computer running Linux with sudo privileges
- A microSD card of at least 16GB capacity
- A microSD card reader connected to your computer
- A Raspberry Pi 4 or 5 (see {doc}`/reference/recommended-hardware`)
- A USB stick of at least 8GB capacity to store the backup image

## Step 1: Image the microSD Card with Raspberry Pi OS

1. Download the latest 'Raspberry Pi OS Lite' image from the
   [Raspberry Pi website](https://www.raspberrypi.com/software/operating-systems/).
   The Lite version is the last option on the linked page.

1. Use `lsblk` to get a list of block devices before inserting the
   microSD card.

1. Insert the microSD card and use `lsblk` again to identify the device
   name (e.g. `/dev/sdb`).

1. Uncompress the downloaded image:
    ```bash
    cd ~/Downloads
    unxz ./2025-12-04-raspios-trixie-arm64-lite.img.xz
    ```

1. Write the image to the microSD card. Replace `/dev/sdX` with the
   actual device name. **Be careful** as this will overwrite the
   specified device.
    ```bash
    sudo dd if=./2025-12-04-raspios-trixie-arm64-lite.img of=/dev/sdX bs=4M status=progress conv=fsync
    ```

## Step 2: Configure the Raspberry Pi OS Image

These steps must be done before the first boot.

1. Mount the microSD card boot partition:
    ```bash
    mkdir sdcard-bootfs
    sudo mount /dev/sdX1 sdcard-bootfs
    cd sdcard-bootfs
    ```

1. Enable SSH:
    ```bash
    sudo touch ssh
    ```

1. Create a user `local` with password `local`:
    ```bash
    echo "local:$(echo local | openssl passwd -6 -stdin)" | sudo tee userconf.txt
    ```

1. If you need a static IP address, edit `cmdline.txt`:
    ```bash
    sudo vim cmdline.txt
    # add " ip=<your_static_ip_address>" at the end of the single line
    ```

1. Unmount the boot partition:
    ```bash
    cd ..
    sudo umount sdcard-bootfs
    sudo umount /dev/sdX  # there may be a second mount point
    rmdir sdcard-bootfs
    ```

## Step 3: First Boot and Connect to Internet

1. Insert the microSD card into your Raspberry Pi and power it on.

1. SSH into the Raspberry Pi. The username is `local` and the password is
   `local`. You may need to check your router for the assigned IP address
   or use the static IP set above.

1. If you do not have internet access, temporarily connect to WiFi:
    ```bash
    sudo raspi-config
    # Select "System Options" -> "Wireless LAN"
    # Enter your SSID and password
    ```

1. Update the system:
    ```bash
    sudo apt update
    sudo apt upgrade -y
    sudo apt install -y git vim
    ```

1. Record the Raspberry Pi's MAC address:
    ```bash
    ip link show eth0
    # look for "link/ether xx:xx:xx:xx:xx:xx"
    ```

## Step 4: Install and Configure USB/IP

1. Load the kernel modules and configure them to load at boot:
    ```bash
    sudo modprobe usbip_core
    sudo modprobe usbip_host
    echo -e "usbip-core\nusbip-host" | sudo tee /etc/modules-load.d/usbip.conf
    ```

1. Install the `usbip` package:
    ```bash
    sudo apt install -y usbip
    ```

1. Create a systemd service for `usbipd`:
    ```bash
    echo "[Unit]
    Description=USB/IP Daemon
    After=network.target

    [Service]
    Type=forking
    ExecStart=/usr/sbin/usbipd -D
    Restart=on-failure

    [Install]
    WantedBy=multi-user.target" | sudo tee /etc/systemd/system/usbipd.service

    sudo systemctl enable --now usbipd.service
    ```

## Step 5: Install the Agent

```bash
curl -fsSLO https://raw.githubusercontent.com/DiamondLightSource/dra-usbip-driver/main/scripts/install-agent.sh
sudo bash install-agent.sh
```

Verify:
```bash
sudo systemctl status dra-usbip-agent
curl http://localhost:13240/devices
```

## Step 6: Install the Pico MAC Sender (optional)

This service monitors USB ports for a Raspberry Pi Pico and sends the
Pi's MAC address to its OLED display. Useful for commissioning without a
monitor or keyboard.

```bash
curl -fsSLO https://raw.githubusercontent.com/DiamondLightSource/dra-usbip-driver/main/scripts/install-pico-mac-sender.sh
sudo bash install-pico-mac-sender.sh
```

## Step 7: Install image-backup

`image-backup` creates compressed backup images of Raspberry Pi SD cards
that auto-expand on first boot. See
<https://forums.raspberrypi.com/viewtopic.php?t=332000> for details.

```bash
cd ~
git clone https://github.com/seamusdemora/RonR-RPi-image-utils.git
sudo install --mode=755 ~/RonR-RPi-image-utils/image-* /usr/local/sbin
```

## Step 8: Clean Up Settings for Production

Undo any temporary configuration so the image can be reused on multiple
Pis with DHCP.

1. Disable the static IP if you set one:
    ```bash
    sudo vim /boot/firmware/cmdline.txt
    # remove " ip=<your_static_ip_address>"
    ```

1. Disable WiFi if you enabled it:
    ```bash
    sudo vim /boot/firmware/config.txt
    # add at the end:
    dtoverlay=disable-wifi
    dtoverlay=disable-bt
    ```

## Step 9: Prepare the Backup Image for Distribution

Set up a run-once service so copies of the image enable read-only mode
on first boot:

```bash
echo '[Unit]
Description=Run script once on next boot
ConditionPathExists=/var/local/runonce.sh
After=multi-user.target

[Service]
Type=oneshot
ExecStart=/bin/bash /var/local/runonce.sh

[Install]
WantedBy=multi-user.target
' | sudo tee /etc/systemd/system/runonce.service
sudo systemctl enable runonce.service
```

Create the `runonce.sh` script:

```bash
echo '#!/bin/bash
set -x

# Disable this script from running again
mv /var/local/runonce.sh /var/local/runonce.sh.done

# disable services that will report errors when in RO mode
systemctl mask dphys-swapfile.service
systemctl mask systemd-zram-setup@zram0.service

# enable read only overlay mode
raspi-config nonint do_overlayfs 0

# reboot to pick up the change
sync; reboot
' | sudo tee /var/local/runonce.sh

# Add packages required for read-only mode
sudo apt-get -y install cryptsetup cryptsetup-bin overlayroot
```

(create-a-backup-image)=
## Step 10: Create a Backup Image of the microSD Card

1. Insert a USB stick into the Raspberry Pi.

1. Mount the USB stick:
    ```bash
    sudo mkdir -p /media/local/usb
    sudo mount /dev/sda1 /media/local/usb
    ```

1. Run `image-backup`:
    ```bash
    sudo image-backup
    # when prompted for output file, use something like:
    /media/local/usb/raspi-agent-server-0.3.0.img
    ```

1. Sync and unmount:
    ```bash
    sync
    sudo umount /media/local/usb
    ```

Your image can now be used to commission additional Raspberry Pi agent
servers as described in {doc}`/tutorials/commission-raspi`.
