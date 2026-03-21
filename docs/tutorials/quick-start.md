# Quick start

This tutorial walks through deploying the dra-usbip-driver to make a remote
USB device available to a pod in your Kubernetes cluster.

## Prerequisites

- A Kubernetes cluster (v1.34+) with DRA enabled
- `kubectl` configured to access the cluster
- A machine with USB devices to share (the Agent host)
- The `usbip` tools installed on both the Agent host and cluster nodes
- The `vhci-hcd` and `usbip-host` kernel modules loaded

## 1. Run the Agent

On the machine with USB devices attached, build and run the Agent binary:

```bash
make agent
./agent --bind-all-devices
```

The Agent serves device metadata on port 13240. Verify it is working:

```bash
curl http://<agent-host>:13240/devices
```

You should see a JSON array describing the connected USB devices.

## 2. Deploy the Manager

The Manager runs inside the cluster and polls Agents for device information.

```bash
kubectl apply -f examples/manager.yaml
```

Or run the container image directly:

```bash
kubectl run usbip-manager \
  --image=ghcr.io/diamondlightsource/dra-usbip-driver-manager:latest \
  -- --agent=<agent-host>
```

Check that ResourceSlices have been created:

```bash
kubectl get resourceslices
```

## 3. Deploy the Plugin DaemonSet

The Plugin runs on each node and handles attaching USB/IP devices when pods
request them:

```bash
kubectl apply -f examples/plugin-daemonset.yaml
```

## 4. Create a ResourceClaim and Pod

Create a ResourceClaim to request a USB device, then a Pod that references it:

```yaml
apiVersion: resource.k8s.io/v1
kind: ResourceClaim
metadata:
  name: my-usb-device
spec:
  devices:
    requests:
      - name: usb
        deviceClassName: usbip
---
apiVersion: v1
kind: Pod
metadata:
  name: usb-consumer
spec:
  containers:
    - name: app
      image: ubuntu
      command: ["sleep", "infinity"]
  resourceClaims:
    - name: usb-claim
      resourceClaimName: my-usb-device
```

```bash
kubectl apply -f my-pod.yaml
```

Once the pod is running, the USB device will be available inside the container.

## Next steps

- {doc}`/explanations/architecture` -- understand the three-component design
