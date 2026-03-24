package main

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/spf13/pflag"
	"k8s.io/klog/v2"

	"github.com/diamondlightsource/dra-usbip-driver/pkg/devicemetadata"
	"github.com/diamondlightsource/dra-usbip-driver/pkg/kmod"
	"github.com/diamondlightsource/dra-usbip-driver/pkg/usbip"
)

var (
	bindAllDevices bool
)

func init() {
	pflag.BoolVar(&bindAllDevices, "bind-all-devices", false, "bind all valid devices to usbip")
}

func listDevices(w http.ResponseWriter, req *http.Request) {
	localDevices, err := usbip.GetLocalDevices()
	if err != nil {
		http.Error(w, fmt.Sprintf("could not list local usb devices: %s", err), http.StatusInternalServerError)
		return
	}

	var devicesMetadata []*devicemetadata.UdevadmInfo
	for _, device := range localDevices {
		udevInfo, err := usbip.GetLocalDeviceInfo(device)
		if err != nil {
			http.Error(w, fmt.Sprintf("failed to get device info for %s: %s", device, err), http.StatusInternalServerError)
			continue
		}

		if udevInfo.Driver != "usbip-host" && bindAllDevices {
			// Device is not bound to usbip driver, do that now.
			err := usbip.BindDevice(device)
			if err != nil {
				// Log but don't error - need to continue on to
				// return list of other devices.
				klog.Errorf("Failed to bind device %s: %s", device, err)
				continue
			}
			klog.Infof("Bound device %s", device)
		}

		devicesMetadata = append(devicesMetadata, udevInfo)
	}

	b, err := json.Marshal(devicesMetadata)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to encode devices as json: %s", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if _, err := w.Write(b); err != nil {
		klog.Errorf("Failed to write response: %s", err)
	}
}

func main() {
	pflag.Parse()

	moduleLoaded, err := kmod.IsLoaded("usbip_host")
	if err != nil {
		klog.Fatalf("Unable to check kernel modules: %s", err)
	}
	if !moduleLoaded {
		klog.Fatal("usbip_host kernel module is not loaded")
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /devices", listDevices)

	klog.Fatal(http.ListenAndServe(":13240", mux))
}
