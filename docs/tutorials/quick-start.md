# Quick start

This tutorial walks through deploying the dra-usbip-driver to make a remote
USB device available to a pod in your Kubernetes cluster.

## Prerequisites

- A Kubernetes cluster (v1.34+) with DRA enabled
- `kubectl` and [Helm](https://helm.sh/docs/intro/install/) 3.x configured
  to access the cluster
- A machine with USB devices to share (the agent host, typically a
  Raspberry Pi or similar)
- The `usbip` tools installed on both the agent host and cluster nodes
- The `vhci-hcd` and `usbip-host` kernel modules loaded

## 1. Install the agent

On the machine with USB devices attached, run the install script:

```bash
curl -fsSL https://raw.githubusercontent.com/DiamondLightSource/dra-usbip-driver/main/scripts/install-agent.sh | sudo sh -s -- 0.1.0
```

This downloads the correct binary for your architecture, installs it as a
systemd service, and starts it. The agent serves device metadata on port
13240.

Verify it is working:

```bash
curl http://<agent-host>:13240/devices
```

You should see a JSON array describing the connected USB devices.

For more detail see {doc}`/how-to/setup-agent`.

## 2. Deploy the manager and plugin

Install the Helm chart, passing the address of your agent host:

```bash
helm -n kube-system upgrade --install dra-usbip-driver \
    oci://ghcr.io/diamondlightsource/charts/dra-usbip-driver \
    --version 0.1.0 \
    --set 'manager.agents={<agent-host>}'
```

Check that ResourceSlices have been created:

```bash
kubectl get resourceslices
```

For more detail see {doc}`/how-to/deploy`.

## 3. Request a USB device in a pod

Create a `ResourceClaimTemplate` to match a specific USB device, then
reference it from a pod:

```yaml
apiVersion: resource.k8s.io/v1
kind: ResourceClaimTemplate
metadata:
  name: serial-adapter
spec:
  spec:
    devices:
      requests:
      - name: req-0
        firstAvailable:
        - name: adapter
          deviceClassName: usbip
          selectors:
          - cel:
              expression: |-
                device.attributes["usbip.diamond.ac.uk"].vendor == "0403" &&
                device.attributes["usbip.diamond.ac.uk"].product == "6015" &&
                device.attributes["usbip.diamond.ac.uk"].serial == "FTA954OZ"
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
      resourceClaimTemplateName: serial-adapter
```

```bash
kubectl apply -f my-pod.yaml
```

Once the pod is running, the USB device will be available inside the container.

## Next steps

- {doc}`/explanations/architecture` -- understand the three-component design
