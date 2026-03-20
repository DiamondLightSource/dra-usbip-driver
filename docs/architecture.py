from diagrams import Diagram, Cluster
from diagrams.k8s.compute import Pod, RS
from diagrams.k8s.controlplane import Kubelet
from diagrams.custom import Custom

# https://en.wikipedia.org/wiki/File:USB_icon.svg
usb_icon = "icons/USB_icon.png"

claim_icon = "icons/claim.png"

rpi_icon = "icons/rpi.png"

graph_attr = {
    "pad": "0.5",
}

with Diagram(graph_attr=graph_attr, filename="architecture", show=False):
    with Cluster("Cluster"):
        manager = Pod("Manager")

        # Repurpose the replicaset icon.
        resource_slice = RS("ResourceSlice")
        manager >> resource_slice

        claim = Custom("ResourceClaim", claim_icon)

        with Cluster("Node"):
            driver_plugin = Pod("Driver plugin")
            kubelet = Kubelet("Kubelet")
            user_pod = Pod("User pod")

            driver_plugin >> kubelet >> user_pod

        resource_slice << claim << user_pod

    agent = Custom("Agent", rpi_icon, height="1.4")
    usb_1 = Custom("Device 1-1", usb_icon, height="1.2")
    agent >> usb_1

    driver_plugin << claim

    manager >> agent

    usb_1 << user_pod
