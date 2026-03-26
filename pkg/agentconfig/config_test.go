package agentconfig

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoad(t *testing.T) {
	content := `
excludeDevices:
  deviceIDs:
    - "2e8a:0005"
  deviceClasses:
    - "03"
    - "08"
`
	tmp := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(tmp, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(tmp)
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	if len(cfg.ExcludeDevices.DeviceIDs) != 1 {
		t.Fatalf("expected 1 device ID, got %d", len(cfg.ExcludeDevices.DeviceIDs))
	}
	if cfg.ExcludeDevices.DeviceIDs[0] != "2e8a:0005" {
		t.Errorf("expected 2e8a:0005, got %s", cfg.ExcludeDevices.DeviceIDs[0])
	}
	if len(cfg.ExcludeDevices.DeviceClasses) != 2 {
		t.Fatalf("expected 2 device classes, got %d", len(cfg.ExcludeDevices.DeviceClasses))
	}
}

func TestShouldExclude_NilConfig(t *testing.T) {
	var cfg *Config
	if cfg.ShouldExclude("1-1", "2e8a", "0005") {
		t.Error("nil config should not exclude anything")
	}
}

func TestShouldExclude_ByDeviceID(t *testing.T) {
	cfg := &Config{
		ExcludeDevices: ExcludeDevices{
			DeviceIDs: []string{"2e8a:0005", "0403:6015"},
		},
	}

	tests := []struct {
		vendor, model string
		want          bool
	}{
		{"2e8a", "0005", true},
		{"0403", "6015", true},
		{"2E8A", "0005", true}, // case insensitive
		{"1234", "5678", false},
	}

	for _, tt := range tests {
		// Use a non-existent busID so class check is skipped.
		got := cfg.ShouldExclude("nonexistent-bus", tt.vendor, tt.model)
		if got != tt.want {
			t.Errorf("ShouldExclude(%s:%s) = %v, want %v", tt.vendor, tt.model, got, tt.want)
		}
	}
}

func TestShouldExclude_ByDeviceClass(t *testing.T) {
	// Set up a fake sysfs tree.
	sysfs := t.TempDir()
	busID := "1-1"

	// Create device with bDeviceClass=03 (HID).
	devDir := filepath.Join(sysfs, busID)
	if err := os.MkdirAll(devDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(devDir, "bDeviceClass"), []byte("03\n"), 0644); err != nil {
		t.Fatal(err)
	}

	cfg := &Config{
		ExcludeDevices: ExcludeDevices{
			DeviceClasses: []string{"03", "08"},
		},
	}

	// matchesDeviceClass reads from /sys/bus/usb/devices which we can't
	// easily fake without root. Test the device ID path instead and
	// verify the class matching logic via matchesDeviceClass directly.
	// The integration is verified by manual testing.
	if !matchesDeviceClassIn(sysfs, busID, cfg.ExcludeDevices.DeviceClasses) {
		t.Error("expected device class 03 to match")
	}

	// Device with class 00 and no interfaces should not match.
	busID2 := "1-2"
	devDir2 := filepath.Join(sysfs, busID2)
	if err := os.MkdirAll(devDir2, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(devDir2, "bDeviceClass"), []byte("00\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if matchesDeviceClassIn(sysfs, busID2, cfg.ExcludeDevices.DeviceClasses) {
		t.Error("device class 00 with no interfaces should not match")
	}
}

func TestShouldExclude_ByInterfaceClass(t *testing.T) {
	sysfs := t.TempDir()
	busID := "1-3"

	// Device class is 00, but interface class is 03.
	devDir := filepath.Join(sysfs, busID)
	if err := os.MkdirAll(devDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(devDir, "bDeviceClass"), []byte("00\n"), 0644); err != nil {
		t.Fatal(err)
	}

	ifDir := filepath.Join(sysfs, busID+":1.0")
	if err := os.MkdirAll(ifDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(ifDir, "bInterfaceClass"), []byte("03\n"), 0644); err != nil {
		t.Fatal(err)
	}

	classes := []string{"03", "08"}
	if !matchesDeviceClassIn(sysfs, busID, classes) {
		t.Error("expected interface class 03 to match")
	}
}
