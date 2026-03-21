# Commissioning a new Raspberry Pi as an Agent Server

## Introduction

Choosing the recommended hardware simplifies commissioning as there is a
pre-built disk image available that includes all necessary software and
configuration.

## Step 1: Obtain and Assemble Recommended Hardware

See {doc}`/reference/recommended-hardware`.

Any Raspberry Pi 4 or 5 with at least 2GB RAM and at least 16GB microSD
card is suitable.

## Step 2: Flash the Agent Server Image

1. Download the latest `raspi-agent-server-*.img.xz` from the
   [GitHub Releases](https://github.com/DiamondLightSource/dra-usbip-driver/releases)
   page and decompress it:
    ```bash
    unxz raspi-agent-server-*.img.xz
    ```

1. Insert a microSD card of at least 16GB capacity into a card reader
   connected to your computer.

1. Use `lsblk` to identify the device name of the microSD card (e.g.
   `/dev/sdb`).

1. Flash the image to the microSD card. **Be careful** -- replace
   `/dev/sdX` with the correct device name for your microSD card as this
   will overwrite the specified device.
    ```bash
    sudo dd if=./raspi-agent-server.img of=/dev/sdX bs=4M status=progress conv=fsync
    ```

## Step 3: Extract the Raspberry Pi MAC Address

- If you have a Pico with OLED screen, plug it into the Raspberry Pi USB
  port and power on the Pi. The MAC address will be displayed on the
  screen within a minute.
- Otherwise, boot the Raspberry Pi and get the MAC address from the
  command line:
  ```bash
  ip link show eth0
  ```

## Step 4: Configure an IP Address for the Raspberry Pi

Use your DHCP management tool to create a reservation for the Raspberry
Pi MAC address obtained in Step 3.

## Step 5: Connect the Raspberry Pi to the Network and Power it On

- Connect the Raspberry Pi to the network using a wired ethernet
  connection.
- Power on the Raspberry Pi using the USB-C power supply.
- Wait a few minutes as the Pi will reboot twice to expand the root
  filesystem and set up read-only filesystem mode.

## Step 6: Verify the Agent is Working

From any machine that can reach the Raspberry Pi:

```bash
curl http://<raspberry_pi_ip_address>:13240/devices
```

You should see a JSON array describing the connected USB devices.

## Troubleshooting

If the agent is not responding, SSH into the Raspberry Pi and check the
services:

```bash
ssh local@<raspberry_pi_ip_address>
# password is "local"

# check the status of the services
sudo systemctl status usbipd
sudo systemctl status dra-usbip-agent

# check their logs for errors
journalctl -u usbipd -e
journalctl -u dra-usbip-agent -e
```

For more troubleshooting tips see {doc}`/how-to/troubleshoot`.
