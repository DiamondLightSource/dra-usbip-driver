package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os/exec"
	"strings"

	"github.com/google/gousb"
	"github.com/google/gousb/usbid"

	"gitlab.diamond.ac.uk/sysadmin/container-tools/dra-usbip-driver/pkg/devicemetadata"
)

// Match function returns true for any
// device that should be opened by gousb.
func matchDevice(desc *gousb.DeviceDesc) bool {
	if desc.Class == gousb.ClassHub {
		// Skip hub devices, as usbip does:
		// https://github.com/torvalds/linux/blob/v6.18/tools/usb/usbip/src/usbip_list.c#L193
		return false
	}

	return true
}

type Server struct {
	deviceContext *gousb.Context
}

func (s *Server) listDevices(w http.ResponseWriter, req *http.Request) {
	devices, err := s.deviceContext.OpenDevices(matchDevice)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to open devices: %s", err), http.StatusInternalServerError)
		return
	}

	defer func() {
		for _, d := range devices {
			d.Close()
		}
	}()

	var devicesMetadata []devicemetadata.Metadata

	for _, device := range devices {
		desc := device.Desc

		busPath := fmt.Sprintf("/dev/bus/usb/%03d/%03d", desc.Bus, desc.Address)
		udevInfoJson, err := exec.Command("udevadm", "info", "--json=short", busPath).Output()
		if err != nil {
			http.Error(w, fmt.Sprintf("could not get udev info for device %s: %s", busPath, err), http.StatusInternalServerError)
			return
		}
		udevInfo := &devicemetadata.UdevadmInfo{}
		json.Unmarshal(udevInfoJson, &udevInfo)

		if strings.Contains(udevInfo.DevPath, "vhci_hcd") {
			// Device is already a virtual device attached
			// by usbip, don't re-export it.
			continue
		}

		fmt.Printf("%#v\n", udevInfo)

		var vendorName, productName string
		if vendorInfo, ok := usbid.Vendors[desc.Vendor]; ok {
			vendorName = vendorInfo.Name
			if productInfo, ok := vendorInfo.Product[desc.Product]; ok {
				productName = productInfo.Name
			}
		}

		d := devicemetadata.Metadata{
			Bus:         desc.Bus,
			Address:     desc.Address,
			Vendor:      uint16(desc.Vendor),
			Product:     uint16(desc.Product),
			Class:       uint8(desc.Class),
			VendorName:  vendorName,
			ProductName: productName,
			BusID:       udevInfo.SysName,
			Model:       udevInfo.Model,
		}

		serial, err := device.SerialNumber()
		if err == nil {
			// No error means serial number present.
			d.Serial = serial
		}

		devicesMetadata = append(devicesMetadata, d)
	}

	b, err := json.Marshal(devicesMetadata)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to encode devices as json: %s", err), http.StatusInternalServerError)
		return
	}

	w.Write(b)
}

func main() {
	server := &Server{
		deviceContext: gousb.NewContext(),
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /devices", server.listDevices)

	http.ListenAndServe(":8105", mux)
}
