package main

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/spf13/pflag"

	"gitlab.diamond.ac.uk/sysadmin/container-tools/dra-usbip-driver/pkg/devicemetadata"
	"gitlab.diamond.ac.uk/sysadmin/container-tools/dra-usbip-driver/pkg/usbip"
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
				fmt.Printf("Failed to bind device %s: %v\n", device, err)
				continue
			}
			fmt.Printf("Bound device %s\n", device)
		}

		devicesMetadata = append(devicesMetadata, udevInfo)
	}

	b, err := json.Marshal(devicesMetadata)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to encode devices as json: %s", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(b)
}

func main() {
	pflag.Parse()

	mux := http.NewServeMux()
	mux.HandleFunc("GET /devices", listDevices)

	http.ListenAndServe(":8105", mux)
}
