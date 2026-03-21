# Install the agent on a USB/IP host

The agent runs on a machine with USB devices (typically a Raspberry Pi) and
exposes them over the network via USB/IP. The Kubernetes manager polls agents
to discover available devices.

## Prerequisites

- A Linux machine (Raspberry Pi OS, Debian, Ubuntu, etc.) with USB devices
  attached
- The `usbip` kernel module and userspace tools installed
- `curl` available for the install script
- Root access (the installer creates a systemd service)

### Install USB/IP tools

On Debian/Ubuntu/Raspberry Pi OS:

```bash
$ sudo apt-get update && sudo apt-get install -y linux-tools-generic
```

On Raspberry Pi OS the package may be `usbip` instead:

```bash
$ sudo apt-get update && sudo apt-get install -y usbip
```

### Load the kernel module

```bash
$ sudo modprobe usbip_host
```

To load the module automatically on boot:

```bash
$ echo usbip_host | sudo tee /etc/modules-load.d/usbip_host.conf
```

## Install the agent

The install script downloads the correct binary for your architecture,
installs it to `/usr/local/bin/agent`, and creates a systemd service:

```bash
$ curl -fsSLO https://raw.githubusercontent.com/DiamondLightSource/dra-usbip-driver/main/scripts/install-agent.sh
$ sudo sh install-agent.sh
```

To install a specific version:

```bash
$ sudo sh install-agent.sh 0.1.0
```

## Verify

Check that the service is running:

```bash
$ sudo systemctl status dra-usbip-agent
```

The agent serves a device list on port 13240. Test it with:

```bash
$ curl http://localhost:13240/devices
```

## Uninstall

```bash
$ sudo systemctl stop dra-usbip-agent
$ sudo systemctl disable dra-usbip-agent
$ sudo rm /etc/systemd/system/dra-usbip-agent.service
$ sudo systemctl daemon-reload
$ sudo rm /usr/local/bin/agent
```
