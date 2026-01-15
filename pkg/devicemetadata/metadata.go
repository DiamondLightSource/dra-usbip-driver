package devicemetadata

import (
	"github.com/google/gousb"
)

type Metadata struct {
	Bus     int
	Address int

	Vendor  gousb.ID
	Product gousb.ID

	Class gousb.Class

	Serial string

	// What udev info calls the SysName,
	// usbip calls the BusID.
	BusID string
	Model string
}

// Representing the required fields from the
// output of a the udevadm info command e.g:
// "udevadm info --json=pretty <dev>".
type UdevadmInfo struct {
	// E.g "1-4", with different numbers to the bus/dev num.
	// This is how usbip represents the device.
	SysName string `json::SYSNAME:`

	// Device model. From looking at a few, seems to
	// be more descriptive than the libusb product string.
	Model string `json:"ID_MODEL"`

	// Path of the device under /sys.
	DevPath string `json:"DEVPATH"`
}
