package agentconfig

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"sigs.k8s.io/yaml"
)

type ExcludeDevices struct {
	DeviceIDs     []string `json:"deviceIDs"`
	DeviceClasses []string `json:"deviceClasses"`
}

type Config struct {
	ExcludeDevices ExcludeDevices `json:"excludeDevices"`
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading config %s: %w", path, err)
	}

	cfg := &Config{}
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("parsing config %s: %w", path, err)
	}

	// Normalize to lowercase for case-insensitive matching.
	for i, id := range cfg.ExcludeDevices.DeviceIDs {
		cfg.ExcludeDevices.DeviceIDs[i] = strings.ToLower(id)
	}
	for i, c := range cfg.ExcludeDevices.DeviceClasses {
		cfg.ExcludeDevices.DeviceClasses[i] = strings.ToLower(c)
	}

	return cfg, nil
}

// ShouldExclude returns true if the device matches any exclusion rule.
// vendorID and modelID are hex strings like "2e8a" and "0005".
// busID is the sysfs bus ID like "1-1.5" used to look up device class.
func (c *Config) ShouldExclude(busID, vendorID, modelID string) bool {
	if c == nil {
		return false
	}

	deviceID := strings.ToLower(vendorID + ":" + modelID)
	for _, excluded := range c.ExcludeDevices.DeviceIDs {
		if deviceID == excluded {
			return true
		}
	}

	if len(c.ExcludeDevices.DeviceClasses) > 0 {
		if matchesDeviceClass(busID, c.ExcludeDevices.DeviceClasses) {
			return true
		}
	}

	return false
}

// sysfsBase is the root path for USB device sysfs entries.
// Overridable in tests.
var sysfsBase = "/sys/bus/usb/devices"

func matchesDeviceClass(busID string, classes []string) bool {
	return matchesDeviceClassIn(sysfsBase, busID, classes)
}

// matchesDeviceClassIn checks the USB device class at both the device level
// and interface level. Many devices (especially HID) set bDeviceClass=00
// and define the actual class on their interfaces.
func matchesDeviceClassIn(base, busID string, classes []string) bool {
	// Check device-level class.
	deviceClassPath := filepath.Join(base, busID, "bDeviceClass")
	if data, err := os.ReadFile(deviceClassPath); err == nil {
		dc := strings.ToLower(strings.TrimSpace(string(data)))
		if dc != "00" {
			for _, c := range classes {
				if dc == c {
					return true
				}
			}
		}
	}

	// Check interface-level classes.
	pattern := filepath.Join(base, busID+":*", "bInterfaceClass")
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return false
	}
	for _, m := range matches {
		data, err := os.ReadFile(m)
		if err != nil {
			continue
		}
		ic := strings.ToLower(strings.TrimSpace(string(data)))
		for _, c := range classes {
			if ic == c {
				return true
			}
		}
	}

	return false
}
