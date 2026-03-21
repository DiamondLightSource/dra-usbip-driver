# Updating an Existing Raspberry Pi Boot Image

To upgrade the agent or other software on an existing Raspberry Pi image,
follow these steps.

:::{tip}
If you just need a new image with an updated agent version, you can build
one automatically using the **Build SD Card Image** GitHub Actions
workflow instead of following the manual steps below. See the tip in
{doc}`new-raspi-image` for details.
:::

1. Boot a Raspberry Pi using the existing microSD card image. Make sure
   you have network connectivity so you can SSH in.

1. SSH into the Raspberry Pi:
   ```bash
   ssh local@<raspberry_pi_ip_address>
   # password is "local"
   ```

1. Restore the root filesystem to writeable mode:
   ```bash
   sudo raspi-config nonint do_overlayfs 1
   # on reboot the root fs will be writeable
   sudo reboot
   ```

1. Re-run the agent install script with the desired version:
   ```bash
   curl -fsSLO https://raw.githubusercontent.com/DiamondLightSource/dra-usbip-driver/main/scripts/install-agent.sh
   sudo bash install-agent.sh <version>
   ```

1. Optionally update the Pico MAC sender:
   ```bash
   curl -fsSLO https://raw.githubusercontent.com/DiamondLightSource/dra-usbip-driver/main/scripts/install-pico-mac-sender.sh
   sudo bash install-pico-mac-sender.sh <version>
   ```

1. Make any other desired changes.

1. Restore `runonce.sh` to re-enable read-only filesystem mode on next
   boot:
   ```bash
   cp /var/local/runonce.sh.done /var/local/runonce.sh
   ```

1. Create a new image file using these instructions:
   {ref}`create-a-backup-image`.
