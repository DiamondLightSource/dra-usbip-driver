package usbip

import (
	"errors"
	"fmt"
	"os/exec"
	"strings"
)

// Parse output of "usbip list --local"
// into list of bus IDs of local devices.
func GetLocalDevices() ([]string, error) {
	usbipLocalDevicesOutput, err := exec.Command("usbip", "list", "-p", "-l").Output()
	if err != nil {
		return nil, fmt.Errorf("error running usbip list: %w", err)
	}

	data := string(usbipLocalDevicesOutput)

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

// Bind a local device to the usbip server.
func BindDevice(busID string) error {
	var exitError *exec.ExitError
	_, err := exec.Command("usbip", "bind", "-b", busID).Output()
	if errors.As(err, &exitError) {
		return fmt.Errorf("usbip bind failed: %s", string(exitError.Stderr))
	} else if err != nil {
		return fmt.Errorf("running usbip bind failed: %w", err)
	}

	return nil
}
