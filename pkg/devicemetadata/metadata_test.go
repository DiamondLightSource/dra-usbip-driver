package devicemetadata

import (
	"encoding/json"
	"testing"
)

func TestUdevadmInfoUnmarshal(t *testing.T) {
	input := `{
		"SYSNAME": "1-4",
		"DRIVER": "usb",
		"DEVPATH": "/devices/pci0000:00/0000:00:14.0/usb1/1-4",
		"DEVNAME": "/dev/bus/usb/001/004",
		"BUSNUM": "001",
		"DEVNUM": "004",
		"MAJOR": "189",
		"MINOR": "3",
		"ID_USB_SERIAL": "FTDI_FT232R_USB_UART_A12345",
		"ID_USB_SERIAL_SHORT": "A12345",
		"ID_VENDOR_ID": "0403",
		"ID_MODEL_ID": "6015",
		"ID_VENDOR": "FTDI",
		"ID_VENDOR_FROM_DATABASE": "Future Technology Devices International",
		"ID_MODEL": "FT232R_USB_UART",
		"ID_MODEL_FROM_DATABASE": "Bridge(I2C/SPI/UART/FIFO)"
	}`

	var info UdevadmInfo
	if err := json.Unmarshal([]byte(input), &info); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	tests := []struct {
		name string
		got  any
		want any
	}{
		{"SysName", info.SysName, "1-4"},
		{"Driver", info.Driver, "usb"},
		{"DevPath", info.DevPath, "/devices/pci0000:00/0000:00:14.0/usb1/1-4"},
		{"DevName", info.DevName, "/dev/bus/usb/001/004"},
		{"BusNum", info.BusNum, "001"},
		{"DevNum", info.DevNum, "004"},
		{"Major", info.Major, int64(189)},
		{"Minor", info.Minor, int64(3)},
		{"Serial", info.Serial, "FTDI_FT232R_USB_UART_A12345"},
		{"SerialShort", info.SerialShort, "A12345"},
		{"VendorID", info.VendorID, "0403"},
		{"ModelID", info.ModelID, "6015"},
		{"VendorName", info.VendorName, "FTDI"},
		{"VendorDB", info.VendorDB, "Future Technology Devices International"},
		{"ModelName", info.ModelName, "FT232R_USB_UART"},
		{"ModelDB", info.ModelDB, "Bridge(I2C/SPI/UART/FIFO)"},
	}

	for _, tt := range tests {
		if tt.got != tt.want {
			t.Errorf("%s: got %v, want %v", tt.name, tt.got, tt.want)
		}
	}
}

func TestUdevadmInfoUnmarshalMissingFields(t *testing.T) {
	// Minimal JSON with only required numeric fields
	input := `{
		"SYSNAME": "2-1",
		"MAJOR": "189",
		"MINOR": "0",
		"ID_VENDOR_ID": "1234",
		"ID_MODEL_ID": "5678"
	}`

	var info UdevadmInfo
	if err := json.Unmarshal([]byte(input), &info); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if info.SysName != "2-1" {
		t.Errorf("SysName: got %q, want %q", info.SysName, "2-1")
	}
	if info.SerialShort != "" {
		t.Errorf("SerialShort should be empty, got %q", info.SerialShort)
	}
	if info.Driver != "" {
		t.Errorf("Driver should be empty, got %q", info.Driver)
	}
}

func TestUdevadmInfoRoundTrip(t *testing.T) {
	original := UdevadmInfo{
		SysName:     "3-1.5",
		Driver:      "usbip-host",
		DevName:     "/dev/bus/usb/003/002",
		BusNum:      "003",
		DevNum:      "002",
		Major:       189,
		Minor:       257,
		VendorID:    "0403",
		ModelID:     "6015",
		SerialShort: "B67890",
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	var decoded UdevadmInfo
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if decoded.SysName != original.SysName {
		t.Errorf("SysName: got %q, want %q", decoded.SysName, original.SysName)
	}
	if decoded.Minor != original.Minor {
		t.Errorf("Minor: got %d, want %d", decoded.Minor, original.Minor)
	}
	if decoded.SerialShort != original.SerialShort {
		t.Errorf("SerialShort: got %q, want %q", decoded.SerialShort, original.SerialShort)
	}
}
