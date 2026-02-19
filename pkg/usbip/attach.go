package usbip

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"time"

	"k8s.io/klog/v2"

	"github.com/cenkalti/backoff/v5"
)

// Attach a device by its remote host/bus,
// and return the matching local bus ID.
func AttachDevice(remoteHost, remoteBusID string) (string, error) {
	var exitError *exec.ExitError
	_, err := exec.Command("usbip", "attach", "-r", remoteHost, "-b", remoteBusID).Output()
	if errors.As(err, &exitError) {
		return "", fmt.Errorf("usbip attach failed: %s", string(exitError.Stderr))
	} else if err != nil {
		return "", fmt.Errorf("attach command failed: %w", err)
	}

	// Local device can't be found immediately.
	// Retry with backoff, waiting up to 10 seconds.
	getLocalBus := func() (string, error) {
		_, localBus, err := GetLocalFromRemote(remoteHost, remoteBusID)
		if errors.Is(err, RemoteDeviceNotFoundError) {
			// Return error to retry if device not found.
			return "", err
		} else if err != nil {
			// Don't retry for any other errors.
			return "", backoff.Permanent(fmt.Errorf("error getting local device: %w", err))
		}

		// Ensure that device is also visible with udevadm.
		_, err = GetLocalDeviceInfo(localBus)
		if err != nil {
			return "", err
		}

		return localBus, nil
	}

	expBackoff := backoff.NewExponentialBackOff()
	expBackoff.InitialInterval = 50 * time.Millisecond

	localBus, err := backoff.Retry(context.TODO(), getLocalBus, backoff.WithBackOff(expBackoff), backoff.WithMaxElapsedTime(10*time.Second))
	if err != nil {
		return "", fmt.Errorf("error finding local device after attaching: %w", err)
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
		// Skip detaching the device but log a warning.
		klog.Warningf("detach called for %s/%s, but device has minor number %d, not expected number %d", remoteHost, remoteBusID, devInfo.Minor, expectedMinor)
		return nil
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
