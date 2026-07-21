# Architecture

Unlike a typical Kubernetes device driver where devices are locally available
on a Node and everything can be handled by a single plugin DaemonSet, this
driver is split into three components to support remote USB devices via USB/IP.

```{image} ../architecture.png
:alt: Architecture diagram
:align: center
```

## Agent

The Agent runs on a machine outside the cluster -- for example a Raspberry Pi
-- that has physical USB devices connected. It:

- Discovers locally connected USB devices using `usbip list --local`
- Optionally binds devices to the `usbip-host` driver
- Serves device metadata as JSON over HTTP on port 13240

The Agent is a standalone Go binary, not a container, since it needs direct
access to host USB devices and kernel modules.

## Manager

The Manager runs as a single pod inside the cluster. It:

- Polls one or more Agents to discover available USB devices
- Creates and updates Kubernetes `ResourceSlice` objects representing those
  devices, with attributes like vendor ID, product ID, and serial number
- Uses the DRA resource slice controller to keep slices in sync

## Plugin

The Plugin runs as a DaemonSet on each node in the cluster. When the Kubelet
assigns a USB device to a pod, the Plugin:

1. Attaches the remote device using `usbip attach`
2. Waits for the device to appear locally via the VHCI driver
3. Creates a CDI (Container Device Interface) spec file so the container
   runtime mounts the device node into the pod
4. Persists state to a checkpoint file so it can detach devices when the pod
   terminates

## Data flow

1. Agent exposes device metadata via HTTP
2. Manager polls Agents and publishes ResourceSlices to the API server
3. User creates a ResourceClaim; the scheduler matches it against ResourceSlices
4. Kubelet calls the Plugin to prepare the claim
5. Plugin attaches the device via USB/IP and creates CDI specs
6. Container runtime uses CDI specs to mount the device into the pod
