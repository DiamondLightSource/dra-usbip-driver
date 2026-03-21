package usbip

import (
	"errors"
	"testing"
)

func TestRemoteDeviceNotFoundError(t *testing.T) {
	if RemoteDeviceNotFoundError == nil {
		t.Fatal("RemoteDeviceNotFoundError should not be nil")
	}

	if RemoteDeviceNotFoundError.Error() != "remote device not found locally" {
		t.Errorf("unexpected error message: %q", RemoteDeviceNotFoundError.Error())
	}

	// Verify it works with errors.Is
	wrapped := errors.New("wrapper")
	_ = wrapped
	if !errors.Is(RemoteDeviceNotFoundError, RemoteDeviceNotFoundError) {
		t.Error("errors.Is should match RemoteDeviceNotFoundError")
	}
}
