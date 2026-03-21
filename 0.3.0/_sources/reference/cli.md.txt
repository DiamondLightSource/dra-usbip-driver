# CLI reference

## Agent

```
agent [flags]
```

| Flag | Default | Description |
|------|---------|-------------|
| `--bind-all-devices` | `false` | Bind all valid USB devices to the usbip-host driver |

The Agent listens on port 13240 and serves `GET /devices` returning JSON
metadata for all USB/IP-exported devices.

## Manager

```
manager [flags]
```

| Flag | Default | Description |
|------|---------|-------------|
| `--agent` | (required) | One or more Agent addresses (`host:port`) to poll for devices. Can be repeated. |
| `--kubeconfig` | `""` | Path to kubeconfig file. If empty, uses in-cluster config. |
| `--driver-name` | `usbip.diamond.ac.uk` | Unique driver name for ResourceSlice objects. |

## Plugin

```
plugin [flags]
```

| Flag | Default | Description |
|------|---------|-------------|
| `--kubeconfig` | `""` | Path to kubeconfig file. If empty, uses in-cluster config. |
| `--driver-name` | `usbip.diamond.ac.uk` | Driver name matching the Manager. |

The Plugin requires the `NODE_NAME` environment variable to be set (typically
from the downward API).
