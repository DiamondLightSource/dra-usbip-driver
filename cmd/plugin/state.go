package main

import (
	"fmt"
	"sync"

	resourceapi "k8s.io/api/resource/v1"
	"k8s.io/klog/v2"
	drav1 "k8s.io/kubelet/pkg/apis/dra/v1"
	"k8s.io/kubernetes/pkg/kubelet/checkpointmanager"
	cdiapi "tags.cncf.io/container-device-interface/pkg/cdi"

	"gitlab.diamond.ac.uk/sysadmin/container-tools/dra-usbip-driver/pkg/usbip"
)

const (
	DriverPluginCheckpointFile = "checkpoint.json"
)

type DeviceState struct {
	sync.Mutex
	cdi               *CDIHandler
	checkpointManager checkpointmanager.CheckpointManager
}

// Need to persist which devices get attached for
// a particular claim, to be able to detach them
// when the container is torn down.
type AttachedDevice struct {
	drav1.Device
	ContainerEdits *cdiapi.ContainerEdits

	RemoteHost  string
	RemoteBusID string

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

func NewDeviceState() (*DeviceState, error) {
	cdi, err := NewCDIHandler("/etc/cdi", "usbip", "usbip")
	if err != nil {
		return nil, fmt.Errorf("unable to create CDI handler: %v", err)
	}

	err = cdi.CreateCommonSpecFile()
	if err != nil {
		return nil, fmt.Errorf("unable to create CDI spec file for common edits: %v", err)
	}

	checkpointManager, err := checkpointmanager.NewCheckpointManager("/var/lib/kubelet/plugins/usbip")
	if err != nil {
		return nil, fmt.Errorf("unable to create checkpoint manager: %v")
	}

	state := &DeviceState{
		cdi:               cdi,
		checkpointManager: checkpointManager,
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

	if err = s.cdi.CreateClaimSpecFile(claimUID, preparedDevices); err != nil {
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
		if result.Driver != "usbip" {
			continue
		}

		remoteHost := result.Pool
		remoteBusID := result.Device

		localBus, err := usbip.AttachDevice(remoteHost, remoteBusID)
		if err != nil {
			return nil, fmt.Errorf("error attaching device %s/%s: %w", remoteHost, remoteBusID, err)
		}

		devInfo, err := usbip.GetLocalDeviceInfo(localBus)
		if err != nil {
			return nil, fmt.Errorf("could not get local device info for %s/%s: %w", remoteHost, remoteBusID, err)
		}

		klog.Infof("Attached remote device %s/%s with local bus ID %s", remoteHost, remoteBusID, localBus)

		d := &AttachedDevice{
			RemoteHost:      remoteHost,
			RemoteBusID:     remoteBusID,
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
		klog.Info("Detaching device", device)
		err := usbip.DetachDevice(device.RemoteHost, device.RemoteBusID, device.DeviceNodeMinor)
		if err != nil {
			klog.Infof("error detaching device %s/%s: %v", device.RemoteHost, device.RemoteBusID, err)
			return fmt.Errorf("error detaching device %s/%s: %v", device.RemoteHost, device.RemoteBusID, err)
		}
	}

	return nil
}
