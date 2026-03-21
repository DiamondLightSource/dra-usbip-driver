---
html_theme.sidebar_secondary.remove: true
---

# dra-usbip-driver

Kubernetes DRA driver for accessing USB devices outside of the cluster via USB/IP.

## Architecture

Unlike a typical device driver where devices are locally available on a Node
and everything can be handled by a single plugin DaemonSet, this driver is
split into three components:

- An **Agent** runs on a machine (for example a Raspberry Pi) outside the
  cluster, exposing attributes of USB devices connected to it.
- A **Manager** runs as one pod in the cluster, fetching data from one or more
  Agents and creating ResourceSlice objects to represent the USB devices.
- The **Plugin** runs as a DaemonSet, handling requests from the Kubelet to
  prepare resource claims by attaching remote devices with USB/IP and telling
  the Kubelet how to mount the device to the pod.

```{image} architecture.png
:alt: Architecture diagram
:align: center
```

How the documentation is structured
------------------------------------

Documentation is split into [four categories](https://diataxis.fr), also
accessible from links in the top bar.

::::{grid} 2
:gutter: 4

:::{grid-item-card} {material-regular}`directions_walk;2em`
```{toctree}
:maxdepth: 2
tutorials
```
+++
Tutorials for installation and typical usage. New users start here.
:::

:::{grid-item-card} {material-regular}`directions;2em`
```{toctree}
:maxdepth: 2
how-to
```
+++
Practical step-by-step guides for the more experienced user.
:::

:::{grid-item-card} {material-regular}`info;2em`
```{toctree}
:maxdepth: 2
explanations
```
+++
Explanations of how it works and why it works that way.
:::

:::{grid-item-card} {material-regular}`menu_book;2em`
```{toctree}
:maxdepth: 2
reference
```
+++
Technical reference material.
:::

::::
