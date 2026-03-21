# Troubleshooting

## Plugin fails to attach devices: `usbip: error: import device`

The plugin container needs to run as privileged. The `usbip attach` command
writes to `/sys/devices/platform/vhci_hcd.0/attach`, which is gated by a
`capable(CAP_SYS_ADMIN)` check in the kernel's `vhci_hcd` driver. This
check requires the capability in the **init user namespace**, so adding
`CAP_SYS_ADMIN` to a non-privileged container is not sufficient — the
container must be fully privileged.

The Helm chart sets `securityContext.privileged: true` on the plugin
DaemonSet. If you have modified the chart or are running the plugin
manually, ensure this is set.

## Pod stuck in Pending: `device class usbip does not exist`

The `usbip` `DeviceClass` is created by the Helm chart. If you see this
error, the chart may not have been installed or was installed with an
older version that did not include the `DeviceClass`. Upgrade the chart
or create it manually:

```yaml
apiVersion: resource.k8s.io/v1
kind: DeviceClass
metadata:
  name: usbip
spec:
  selectors:
  - cel:
      expression: device.driver == "usbip.diamond.ac.uk"
```

## No ResourceSlices created

The manager must be able to reach the agents over HTTP on port 13240.
Check the manager logs for connection errors:

```bash
kubectl logs -n kube-system -l app.kubernetes.io/component=manager
```

Verify the agent addresses in your Helm values are correct and reachable
from inside the cluster.

## USB device not visible inside the pod

The pod's container must reference the resource claim in its `resources.claims`
field. Without this, the Kubelet will not inject the CDI devices into the
container:

```yaml
containers:
  - name: app
    image: ubuntu
    resources:
      claims:
        - name: usb-claim
resourceClaims:
  - name: usb-claim
    resourceClaimTemplateName: serial-adapter
```

## `vhci-hcd` kernel module not loaded

The `vhci-hcd` module must be loaded on every cluster node where the
plugin runs. Without it, `usbip attach` will fail. Load it with:

```bash
sudo modprobe vhci-hcd
```

To persist across reboots, add `vhci-hcd` to `/etc/modules-load.d/`.
