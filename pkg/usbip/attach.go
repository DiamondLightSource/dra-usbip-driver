package usbip

import (
	"errors"
	"fmt"
	"os/exec"
	"time"

	"k8s.io/klog/v2"
)

// Attach a device by its remote host/bus,
// and return the matching local bus ID.
func AttachDevice(remoteHost, remoteBusID string) (string, error) {
	var exitError *exec.ExitError
	output, err := exec.Command("usbip", "attach", "-r", remoteHost, "-b", remoteBusID).Output()
	if errors.As(err, &exitError) {
		return "", fmt.Errorf("usbip attach failed: %s", string(exitError.Stderr))
	} else if err != nil {
		return "", fmt.Errorf("attach command failed: %w", err)
	}
	fmt.Println(output)

	// Local device can't be found immediately.
	// TODO: check in loop?
	time.Sleep(500 * time.Millisecond)

	_, localBus, err := GetLocalFromRemote(remoteHost, remoteBusID)
	if err != nil {
		return "", fmt.Errorf("could not find local bus of newly mounted device")
	}

	return localBus, nil
}

func DetachDevice(remoteHost, remoteBusID string, expectedMinor int64) error {
	// Detach is done by passing the local port.
	localPort, localBusID, err := GetLocalFromRemote(remoteHost, remoteBusID)
	if errors.Is(err, RemoteDeviceNotFoundError) {
		// If expected device is not present locally, then log as a warning
		// but return no error to allow the pod termination to continue.
		klog.Warningf("detach called for remote device %s/%s, but device not present locally", remoteHost, remoteBusID)
		return nil
	} else if err != nil {
		return fmt.Errorf("error finding device to detach: %w", err)
	}

	devInfo, err := GetLocalDeviceInfo(localBusID)
	if err != nil {
		return fmt.Errorf("could not get local device info for %s/%s: %w", remoteHost, remoteBusID, err)
	}

	if devInfo.Minor != expectedMinor {
		// If the device was detached externally to the plugin,
		// then another device may have been attached with the
		// same local bus ID, and that device may be mounted to
		// another pod.
		// If that device node minor number does not match, then
		// it must not be the same device as was expected.
		// TODO: maybe log a warning and return nil instead?
		return fmt.Errorf("device has minor number %d, not expected number %d", devInfo.Minor, expectedMinor)
	}

	var exitError *exec.ExitError
	output, err := exec.Command("usbip", "detach", fmt.Sprintf("--port=%d", localPort)).Output()
	if errors.As(err, &exitError) {
		return fmt.Errorf("usbip detach failed: %s", string(exitError.Stderr))
	} else if err != nil {
		return fmt.Errorf("detach command failed: %w", err)
	}
	fmt.Println(output)

	return nil
}
