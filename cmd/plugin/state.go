package main

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"strings"
	"sync"

	resourceapi "k8s.io/api/resource/v1"
	"k8s.io/klog/v2"
	drav1 "k8s.io/kubelet/pkg/apis/dra/v1"
	"k8s.io/kubernetes/pkg/kubelet/checkpointmanager"
	cdiapi "tags.cncf.io/container-device-interface/pkg/cdi"

	"github.com/diamondlightsource/dra-usbip-driver/pkg/devicemetadata"
	"github.com/diamondlightsource/dra-usbip-driver/pkg/usbip"
)

const (
	DriverPluginCheckpointFile = "checkpoint.json"
)

type DeviceState struct {
	sync.Mutex
	cdi               *CDIHandler
	checkpointManager checkpointmanager.CheckpointManager
	driverName        string
}

// Need to persist which devices get attached for
// a particular claim, to be able to detach them
// when the container is torn down.
type AttachedDevice struct {
	drav1.Device
	ContainerEdits *cdiapi.ContainerEdits

	RemoteHost  string
	RemoteBusID string

	// Which claim/request this device comes from.
	// Not exported as doesn't need to be saved in
	// the checkpoint file, only passed to the CDI
	// spec file setup.
	claimName   string
	requestName string

	// Device minor number, to uniquely
	// identify the locally mounted device,
	// as the local bus ID may be re-used.
	DeviceNodeMinor int64
}

type AttachedDevices []*AttachedDevice

func (ads AttachedDevices) GetDevices() []*drav1.Device {
	var devices []*drav1.Device
	for _, ad := range ads {
		devices = append(devices, &ad.Device)
	}
	return devices
}

// This is what gets persisted into the checkpoint file.
// Map of claim UID -> attached devices.
type PreparedClaims map[string]AttachedDevices

func NewDeviceState(driverName string) (*DeviceState, error) {
	cdi, err := NewCDIHandler("/etc/cdi", driverName, "usbip")
	if err != nil {
		return nil, fmt.Errorf("unable to create CDI handler: %v", err)
	}

	err = cdi.CreateCommonSpecFile()
	if err != nil {
		return nil, fmt.Errorf("unable to create CDI spec file for common edits: %v", err)
	}

	checkpointDirectory := fmt.Sprintf("/var/lib/kubelet/plugins/%s", driverName)

	// Fix bad checkpoint path (didn't use full driver name).
	// Temporary, remove this section after migration.
	_, err = os.Stat("/var/lib/kubelet/plugins/usbip/checkpoint.json")
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		// Unknown error, exit out.
		return nil, fmt.Errorf("unknown error reading old checkpoint file location: %v", err)
	} else if errors.Is(err, fs.ErrNotExist) {
		// New node or old checkpoint file has already been migrated. Do nothing.
	} else {
		// Checkpoint file exists in old location.
		// Migrate from plugins/usbip to new location plugins/{driverName}.
		oldPath := "/var/lib/kubelet/plugins/usbip/checkpoint.json"
		newPath := path.Join(checkpointDirectory, "checkpoint.json")

		// First ensure the new directory exists.
		if err := os.MkdirAll(checkpointDirectory, 0755); err != nil {
			return nil, fmt.Errorf("error making new checkpoint directory: %v", err)
		}

		checkpointData, err := os.ReadFile(oldPath)
		if err != nil {
			return nil, fmt.Errorf("error reading old checkpoint file: %v", err)
		}

		err = os.WriteFile(newPath, checkpointData, 0600)
		if err != nil {
			return nil, fmt.Errorf("error writing new checkpoint file: %v", err)
		}

		// Remove the old file to complete the migration.
		err = os.Remove(oldPath)
		if err != nil {
			return nil, fmt.Errorf("error removing old checkpoint file: %v", err)
		}
	}

	checkpointManager, err := checkpointmanager.NewCheckpointManager(checkpointDirectory)
	if err != nil {
		return nil, fmt.Errorf("unable to create checkpoint manager: %v", err)
	}

	state := &DeviceState{
		cdi:               cdi,
		checkpointManager: checkpointManager,
		driverName:        driverName,
	}

	checkpoints, err := state.checkpointManager.ListCheckpoints()
	if err != nil {
		return nil, fmt.Errorf("unable to list checkpoints: %v", err)
	}

	for _, c := range checkpoints {
		if c == DriverPluginCheckpointFile {
			return state, nil
		}
	}

	checkpoint := newCheckpoint()

	if err := state.checkpointManager.CreateCheckpoint(DriverPluginCheckpointFile, checkpoint); err != nil {
		return nil, fmt.Errorf("unable to create initial checkpoint: %v", err)
	}

	return state, nil
}

func (s *DeviceState) Prepare(claim *resourceapi.ResourceClaim) ([]*drav1.Device, error) {
	s.Lock()
	defer s.Unlock()

	claimUID := string(claim.UID)

	// Fetch the current attached devices state from the checkpoint file.
	checkpoint := newCheckpoint()
	if err := s.checkpointManager.GetCheckpoint(DriverPluginCheckpointFile, checkpoint); err != nil {
		return nil, fmt.Errorf("unable to sync from checkpoint: %v", err)
	}
	preparedClaims := checkpoint.V1.PreparedClaims

	if preparedClaims[claimUID] != nil {
		// Claim has already been prepared, the kubelet must be calling the
		// prepare function a second time. Return the previous list of devices.
		return preparedClaims[claimUID].GetDevices(), nil
	}

	preparedDevices, err := s.prepareDevices(claim)
	if err != nil {
		return nil, fmt.Errorf("prepared failed: %v", err)
	}

	// Must wait for devices to settle, or the CDI
	// creation may fail to discover child devices.
	if err = devicemetadata.UdevadmSettle(); err != nil {
		if err2 := s.unprepareDevices(claimUID, preparedDevices); err2 != nil {
			return nil, fmt.Errorf("failed to detach devices (%v) while recovering from devices failing to settle (%v)", err2, err)
		}
		return nil, fmt.Errorf("error waiting for devices to settle: %v", err)
	}

	if err = s.cdi.CreateClaimSpecFile(claimUID, preparedDevices); err != nil {
		// Need to revert attaching the device or kubernetes will retry
		// the Prepare and fail because the device is already attached.
		if err2 := s.unprepareDevices(claimUID, preparedDevices); err2 != nil {
			return nil, fmt.Errorf("failed to detach devices (%v) while recovering from CDI creation error (%v)", err2, err)
		}
		return nil, fmt.Errorf("unable to create CDI spec file for claim: %v", err)
	}

	// Sync the prepared claims back to the checkpoint file on disk.
	preparedClaims[claimUID] = preparedDevices
	if err := s.checkpointManager.CreateCheckpoint(DriverPluginCheckpointFile, checkpoint); err != nil {
		return nil, fmt.Errorf("unable to sync back to checkpoint: %v", err)
	}

	return preparedClaims[claimUID].GetDevices(), nil
}

func (s *DeviceState) prepareDevices(claim *resourceapi.ResourceClaim) (AttachedDevices, error) {
	if claim.Status.Allocation == nil {
		return nil, fmt.Errorf("claim not yet allocated")
	}

	var attachedDevices AttachedDevices
	for _, result := range claim.Status.Allocation.Devices.Results {
		// The claim may include allocations meant for other drivers.
		if result.Driver != s.driverName {
			continue
		}

		remoteHost := result.Pool

		// Remote bus ID is stored as the device name, but device names
		// don't support having a dot character in which is needed for
		// devices connected to USB hubs.
		// The dot is converted to a "d" by the manager, change it back
		// here to get the actual remote bus ID.
		remoteBusID := strings.ReplaceAll(result.Device, "d", ".")

		localBus, err := usbip.AttachDevice(remoteHost, remoteBusID)
		if err != nil {
			return nil, fmt.Errorf("error attaching device %s/%s: %w", remoteHost, remoteBusID, err)
		}

		devInfo, err := usbip.GetLocalDeviceInfo(localBus)
		if err != nil {
			return nil, fmt.Errorf("could not get local device info for %s/%s: %w", remoteHost, remoteBusID, err)
		}

		klog.Infof("Attached remote device %s/%s with local bus ID %s", remoteHost, remoteBusID, localBus)

		// The driver needs to be able to construct a unique env var name per device.
		// For templated claims, the name of the  ResourceClaim object contains the
		// name of the pod, so it is not deterministic. Use this annotation to get the
		// (user given) name of the claim from the Pod's resourceClaim definitions.
		claimName, ok := claim.ObjectMeta.Annotations["resource.kubernetes.io/pod-claim-name"]
		if !ok {
			// Annotation is only set for templated claims.
			// Fall back to the name of the claim object itself.
			// If not a templated claim, should be deterministic.
			claimName = claim.ObjectMeta.Name
		}

		d := &AttachedDevice{
			RemoteHost:      remoteHost,
			RemoteBusID:     remoteBusID,
			claimName:       claimName,
			requestName:     result.Request,
			DeviceNodeMinor: devInfo.Minor,
			Device: drav1.Device{
				RequestNames: []string{result.Request},
				PoolName:     result.Pool,
				DeviceName:   result.Device,
				CDIDeviceIDs: s.cdi.GetClaimDevices(string(claim.UID), []string{result.Device}),
			},
		}
		attachedDevices = append(attachedDevices, d)
	}
	return attachedDevices, nil
}

func (s *DeviceState) Unprepare(claimUID string) error {
	s.Lock()
	defer s.Unlock()

	// Fetch the current attached devices state from the checkpoint file.
	checkpoint := newCheckpoint()
	if err := s.checkpointManager.GetCheckpoint(DriverPluginCheckpointFile, checkpoint); err != nil {
		return fmt.Errorf("unable to sync from checkpoint: %v", err)
	}
	preparedClaims := checkpoint.V1.PreparedClaims

	if preparedClaims[claimUID] == nil {
		// Claim is already gone.
		return nil
	}

	if err := s.unprepareDevices(claimUID, preparedClaims[claimUID]); err != nil {
		return fmt.Errorf("unprepare failed: %v", err)
	}

	err := s.cdi.DeleteClaimSpecFile(claimUID)
	if err != nil {
		return fmt.Errorf("unable to delete CDI spec file for claim: %v", err)
	}

	// Remove the claim from the checkpoint file on disk.
	delete(preparedClaims, claimUID)
	if err := s.checkpointManager.CreateCheckpoint(DriverPluginCheckpointFile, checkpoint); err != nil {
		return fmt.Errorf("unable to sync back to checkpoint: %v", err)
	}

	return nil
}

func (s *DeviceState) unprepareDevices(claimUID string, devices AttachedDevices) error {
	for _, device := range devices {
		err := usbip.DetachDevice(device.RemoteHost, device.RemoteBusID, device.DeviceNodeMinor)
		if err != nil {
			return fmt.Errorf("error detaching device %s/%s: %v", device.RemoteHost, device.RemoteBusID, err)
		}
	}

	return nil
}
