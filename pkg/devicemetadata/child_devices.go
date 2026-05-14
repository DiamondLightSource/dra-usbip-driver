package devicemetadata

import (
	"bufio"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/diamondlightsource/dra-usbip-driver/pkg/utils"
)

// Find all child devices of a given device.
func (d *UdevadmInfo) FindChildren() ([]*UdevadmInfo, error) {
	baseSysPath := filepath.Join("/sys", d.DevPath)
	baseFS := os.DirFS(baseSysPath)

	relativePaths, err := findChildDevicePaths(baseFS)
	if err != nil {
		return nil, fmt.Errorf("failed to scan for child devices: %s", err)
	}

	var children []*UdevadmInfo

	for _, childPath := range relativePaths {
		absPath := filepath.Join(baseSysPath, childPath)
		childDevice, err := ReadDeviceInfo(absPath)
		if err != nil {
			return nil, fmt.Errorf("error reading data for child device %s: %s", childPath, err)
		}

		// The search shouldn't return any without,
		// but double check that there is actually
		// a /dev device that can be mounted.
		if childDevice.DevName == "" {
			// If not, ignore it.
			continue
		}

		children = append(children, childDevice)
	}

	return children, nil
}

// Scan a filesystem under /sys corresponding to the root of
// a device e.g /sys/devices/platform/vhci_hcd.0/usb3/3-1.
//
// The returned values should be the relative paths to any
// child devices of that USB device, e.g a serial device at
// the relative path 3-1:1.0/ttyUSB0/tty/ttyUSB0.
func findChildDevicePaths(sysFilesystem fs.FS) ([]string, error) {
	var foundDevices []string

	err := fs.WalkDir(sysFilesystem, ".", func(path string, d fs.DirEntry, err error) error {
		// Exit out of the search on any error.
		if err != nil {
			return fmt.Errorf("error processing %q: %s", path, err)
		}

		// If the FULL path is "uevent" then it's
		// not a sub device, it's the top level one.
		if path == "uevent" {
			return nil
		}

		// A file called "uevent" corresponds to a
		// sub device of the USB device.
		if d.Name() == "uevent" {
			isDevice, err := hasDevName(sysFilesystem, path)
			if err != nil {
				return err
			}

			// Ignore the device if it has no entry in /dev.
			if !isDevice {
				return nil
			}

			// The real device path is the directory
			// that contains the "uevent" file.
			dir := filepath.Dir(path)
			foundDevices = append(foundDevices, dir)
		}

		// Skip, keep searching.
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("filesystem scan failed: %s", err)
	}

	return foundDevices, nil
}

// Read a uevent file and check if it has
// a DEVNAME property, indicating a device
// path in /dev.
func hasDevName(sysFilesystem fs.FS, path string) (bool, error) {
	file, err := sysFilesystem.Open(path)
	if err != nil {
		return false, fmt.Errorf("error opening uevent file: %v", err)
	}

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "DEVNAME=") {
			return true, nil
		}
	}

	return false, nil
}

type aliasedDevice struct {
	// Device alias, path in /dev.
	Alias  string
	Env    string
	Device *UdevadmInfo
}

// Construct a set of unique device aliases for
// this set of child devices, with the subsystem
// of the device as the main identifier.
// E.g /dev/usbip-<req>-<dev>-tty (-> /dev/ttyUSB0).
func ToAliases(claim, request string, children []*UdevadmInfo) []*aliasedDevice {
	var aliasedDevices []*aliasedDevice

	claim = utils.SanitiseAliasName(claim)
	request = utils.SanitiseAliasName(request)

	// Count how many children there are of each subsystem type.
	totalDevicesPerSubsystem := make(map[string]int)
	for _, child := range children {
		n, _ := totalDevicesPerSubsystem[child.Subsystem]
		totalDevicesPerSubsystem[child.Subsystem] = n + 1
	}

	// While creating the child device aliases, how
	// many of each subsystem have already been seen.
	seenDevicesPerSubsystem := make(map[string]int)

	// Construct the aliases.
	for _, child := range children {
		// Is this the first, second, etc device of this subsystem type?
		subsystemIndex, _ := seenDevicesPerSubsystem[child.Subsystem]
		seenDevicesPerSubsystem[child.Subsystem] = subsystemIndex + 1

		deviceAlias := fmt.Sprintf("/dev/usbip-%s-%s-%s", claim, request, child.Subsystem)
		if total, _ := totalDevicesPerSubsystem[child.Subsystem]; total > 1 {
			// There are more than one children of this subsystem, have to index them.
			// E.g /dev/usbip-c-d-tty -> /dev/usbip-c-d-tty0.
			deviceAlias = fmt.Sprintf("%s%d", deviceAlias, subsystemIndex)
		}

		envVar := fmt.Sprintf("USBIP_DEVICE_%s_%s_%s", claim, request, child.Subsystem)
		if total, _ := totalDevicesPerSubsystem[child.Subsystem]; total > 1 {
			envVar = fmt.Sprintf("%s%d", envVar, subsystemIndex)
		}
		envVar = utils.SanitiseEnvName(envVar)

		a := &aliasedDevice{
			Alias:  deviceAlias,
			Env:    envVar,
			Device: child,
		}

		aliasedDevices = append(aliasedDevices, a)
	}

	return aliasedDevices
}
