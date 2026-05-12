package devicemetadata

import (
	"encoding/json"
	"fmt"
	"os/exec"
)

// Representing the required fields from the
// output of a the udevadm info command e.g:
// "udevadm info --json=pretty <dev>".
type UdevadmInfo struct {
	// E.g "1-4", with different numbers to the bus/dev num.
	// This is how usbip represents the device.
	SysName string `json:"SYSNAME"`

	// Device driver, should be "usb" or "usbip-host" if bound.
	Driver string `json:"DRIVER"`

	// Path of the device under /sys.
	DevPath string `json:"DEVPATH"`

	// Path of device under /dev, and the bus/dev numbers.
	DevName string `json:"DEVNAME"`
	BusNum  string `json:"BUSNUM"`
	DevNum  string `json:"DEVNUM"`

	// Device subsystem.
	Subsystem string `json:"SUBSYSTEM"`

	// Device node major and minor numbers.
	Major int64 `json:"MAJOR,string"`
	Minor int64 `json:"MINOR,string"`

	// Device serial number representations.
	// Short seems to be the better option.
	Serial      string `json:"ID_USB_SERIAL"`
	SerialShort string `json:"ID_USB_SERIAL_SHORT"`

	// Vendor and product IDs e.g 0403:6015.
	VendorID string `json:"ID_VENDOR_ID"`
	ModelID  string `json:"ID_MODEL_ID"`

	// Vendor and product names.
	VendorName string `json:"ID_VENDOR"`
	VendorDB   string `json:"ID_VENDOR_FROM_DATABASE"`
	ModelName  string `json:"ID_MODEL"`
	ModelDB    string `json:"ID_MODEL_FROM_DATABASE"`
}

// Use udevadm info to read metadata about a device.
// Can be any path that udevadm accepts e.g
// - /dev/bus/usb/003/004
// - /sys/bus/usb/devices/3-1/3-1.6
func ReadDeviceInfo(devicePath string) (*UdevadmInfo, error) {
	udevInfoJson, err := exec.Command("udevadm", "info", "--json=short", devicePath).Output()
	if err != nil {
		return nil, fmt.Errorf("could not get udev device info for %s: %w", devicePath, err)
	}

	udevInfo := &UdevadmInfo{}
	if err := json.Unmarshal(udevInfoJson, &udevInfo); err != nil {
		return nil, fmt.Errorf("could not parse udev info for %s: %w", devicePath, err)
	}

	return udevInfo, nil
}
