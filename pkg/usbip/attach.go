package usbip

import (
	"errors"
	"fmt"
	"os/exec"
	"time"
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
	time.Sleep(100 * time.Millisecond)

	_, localBus, err := GetLocalFromRemote(remoteHost, remoteBusID)
	if err != nil {
		return "", fmt.Errorf("could not find local bus of newly mounted device")
	}

	return localBus, nil
}

func DetachDevice(remoteHost, remoteBusID string) error {
	// Detach is done by passing the local port.
	localPort, _, err := GetLocalFromRemote(remoteHost, remoteBusID)
	if err != nil {
		return fmt.Errorf("error detaching: %w", err)
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
