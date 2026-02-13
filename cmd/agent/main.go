package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os/exec"
	"strings"

	"gitlab.diamond.ac.uk/sysadmin/container-tools/dra-usbip-driver/pkg/devicemetadata"
	"gitlab.diamond.ac.uk/sysadmin/container-tools/dra-usbip-driver/pkg/usbip"
)

// Parse output of "usbip list --local"
// into list of bus IDs of local devices.
func parseLocalDevices(data string) ([]string, error) {
	if data == "" {
		return nil, nil
	}

	var localDevices []string

	// Parse lines of the form:
	// busid=3-1.5#usbid=0403:6015#
	// And extract the bus ID.
	for line := range strings.Lines(data) {
		if !strings.HasPrefix(line, "busid=") {
			return nil, fmt.Errorf("failed parsing local device %q", line)
		}

		parts := strings.Split(line, "#")
		busIDPart := parts[0]
		busID := strings.TrimPrefix(busIDPart, "busid=")
		localDevices = append(localDevices, busID)
	}

	return localDevices, nil
}

func listDevices(w http.ResponseWriter, req *http.Request) {
	usbipLocalDevicesOutput, err := exec.Command("usbip", "list", "-p", "-l").Output()
	if err != nil {
		http.Error(w, fmt.Sprintf("could not list local usbip devices: %s", err), http.StatusInternalServerError)
		return
	}

	localDevices, err := parseLocalDevices(string(usbipLocalDevicesOutput))
	if err != nil {
		http.Error(w, fmt.Sprintf("could not parse local usbip devices: %s", err), http.StatusInternalServerError)
		return
	}

	var devicesMetadata []*devicemetadata.UdevadmInfo
	for _, device := range localDevices {
		udevInfo, err := usbip.GetLocalDeviceInfo(device)
		if err != nil {
			http.Error(w, fmt.Sprintf("failed to get device info for %s: %s", device, err), http.StatusInternalServerError)
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
	mux := http.NewServeMux()
	mux.HandleFunc("GET /devices", listDevices)

	http.ListenAndServe(":8105", mux)
}
