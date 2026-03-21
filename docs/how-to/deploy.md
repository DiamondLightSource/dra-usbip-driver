# Install the manager and plugin on Kubernetes

This guide covers deploying the manager and plugin to a Kubernetes cluster
using Helm.

## Prerequisites

- [Helm](https://helm.sh/docs/intro/install/) 3.x
- A kubeconfig file with cluster-admin access
- One or more USB/IP agents running and reachable from the cluster

## Install from the OCI registry

Published chart versions are available from the GitHub Container Registry:

```bash
$ helm --kubeconfig=./admin.conf -n kube-system upgrade --install \
    dra-usbip-driver \
    oci://ghcr.io/diamondlightsource/charts/dra-usbip-driver \
    --version 0.1.0 \
    --set 'manager.agents={192.168.0.9,192.168.0.10}'
```

Replace `0.1.0` with the desired release version. The `--version` flag is
required because the CI pipeline does not publish a `latest` tag.

## Install from a local checkout

For development, install directly from the chart source:

```bash
$ helm --kubeconfig=./admin.conf -n kube-system upgrade --install \
    dra-usbip-driver \
    Charts/dra-usbip-driver/ \
    --set 'manager.agents={192.168.0.9,192.168.0.10}'
```

## Configuration

Key values to set:

| Value               | Description                          | Default |
|---------------------|--------------------------------------|---------|
| `manager.agents`    | List of agent addresses to poll      | `[]`    |
| `manager.image.tag` | Manager image tag                    | Chart appVersion |
| `plugin.image.tag`  | Plugin image tag                     | Chart appVersion |

For the full list of configurable values, see
`Charts/dra-usbip-driver/values.yaml`.

## Uninstall

```bash
$ helm --kubeconfig=./admin.conf -n kube-system uninstall dra-usbip-driver
```
