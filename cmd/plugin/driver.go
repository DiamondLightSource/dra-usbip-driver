package main

import (
	"context"
	"fmt"
	"os"

	resourceapi "k8s.io/api/resource/v1"
	"k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	coreclientset "k8s.io/client-go/kubernetes"
	"k8s.io/dynamic-resource-allocation/kubeletplugin"
	"k8s.io/klog/v2"
)

type driver struct {
	client coreclientset.Interface
	helper *kubeletplugin.Helper
	state  *DeviceState
}

func NewDriver(ctx context.Context, kubeClient coreclientset.Interface) (*driver, error) {
	state, err := NewDeviceState()
	if err != nil {
		return nil, fmt.Errorf("error initialising state: %w", err)
	}

	driver := &driver{
		client: kubeClient,
		state:  state,
	}

	nodeName := os.Getenv("NODE_NAME")
	if nodeName == "" {
		return nil, fmt.Errorf("env var NODE_NAME not set")
	}

	helper, err := kubeletplugin.Start(
		ctx,
		driver,
		kubeletplugin.KubeClient(kubeClient),
		kubeletplugin.NodeName(nodeName),
		kubeletplugin.DriverName("usbip"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create driver helper: %w", err)
	}

	driver.helper = helper

	return driver, nil
}

func (d *driver) Shutdown() error {
	d.helper.Stop()
	return nil
}

func (d *driver) PrepareResourceClaims(ctx context.Context, claims []*resourceapi.ResourceClaim) (map[types.UID]kubeletplugin.PrepareResult, error) {
	klog.Infof("PrepareResourceClaims is called: number of claims: %d", len(claims))
	result := make(map[types.UID]kubeletplugin.PrepareResult)

	for _, claim := range claims {
		result[claim.UID] = d.prepareResourceClaim(ctx, claim)
	}

	return result, nil
}

func (d *driver) prepareResourceClaim(_ context.Context, claim *resourceapi.ResourceClaim) kubeletplugin.PrepareResult {
	preparedDevices, err := d.state.Prepare(claim)
	if err != nil {
		return kubeletplugin.PrepareResult{
			Err: fmt.Errorf("error preparing claim %v: %w", claim.UID, err),
		}
	}

	var prepared []kubeletplugin.Device
	for _, preparedDev := range preparedDevices {
		prepared = append(prepared, kubeletplugin.Device{
			Requests:     preparedDev.GetRequestNames(),
			PoolName:     preparedDev.GetPoolName(),
			DeviceName:   preparedDev.GetDeviceName(),
			CDIDeviceIDs: preparedDev.GetCDIDeviceIDs(),
		})
	}

	klog.Infof("Returning newly prepared devices for claim '%v': %v", claim.UID, prepared)

	return kubeletplugin.PrepareResult{Devices: prepared}
}

func (d *driver) UnprepareResourceClaims(ctx context.Context, claims []kubeletplugin.NamespacedObject) (map[types.UID]error, error) {
	klog.Infof("UnprepareResourceClaims is called: number of claims: %d", len(claims))
	result := make(map[types.UID]error)

	for _, claim := range claims {
		klog.Infof("Unprepare claim %s", claim.UID)
		result[claim.UID] = d.state.Unprepare(string(claim.UID))
	}

	return result, nil
}

func (d *driver) HandleError(ctx context.Context, err error, msg string) {
	utilruntime.HandleErrorWithContext(ctx, err, msg)
}
