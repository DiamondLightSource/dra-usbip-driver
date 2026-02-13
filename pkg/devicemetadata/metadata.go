package devicemetadata

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
