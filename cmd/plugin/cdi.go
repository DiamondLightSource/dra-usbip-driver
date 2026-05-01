package main

import (
	"fmt"
	"strings"

	cdiapi "tags.cncf.io/container-device-interface/pkg/cdi"
	cdiparser "tags.cncf.io/container-device-interface/pkg/parser"
	cdispec "tags.cncf.io/container-device-interface/specs-go"

	"github.com/diamondlightsource/dra-usbip-driver/pkg/usbip"
)

const cdiCommonDeviceName = "common"

type CDIHandler struct {
	cache      *cdiapi.Cache
	driverName string
	class      string
}

func NewCDIHandler(root, driverName, class string) (*CDIHandler, error) {
	cache, err := cdiapi.NewCache(
		cdiapi.WithSpecDirs(root),
	)
	if err != nil {
		return nil, fmt.Errorf("unable to create a new CDI cache: %w", err)
	}

	handler := &CDIHandler{
		cache:      cache,
		driverName: driverName,
		class:      class,
	}

	return handler, nil
}

// Create a CDI device config with common adjustments that
// will be applied to all containers with usbip devices.
func (cdi *CDIHandler) CreateCommonSpecFile() error {
	spec := &cdispec.Spec{
		Kind: cdi.kind(),
		Devices: []cdispec.Device{
			{
				Name: cdiCommonDeviceName,
				ContainerEdits: cdispec.ContainerEdits{
					Env: []string{
						"USBIP_ACTIVE=true",
					},
				},
			},
		},
	}

	minVersion, err := cdiapi.MinimumRequiredVersion(spec)
	if err != nil {
		return fmt.Errorf("failed to get minimum required CDI spec version: %v", err)
	}
	spec.Version = minVersion

	specName, err := cdiapi.GenerateNameForTransientSpec(spec, cdiCommonDeviceName)
	if err != nil {
		return fmt.Errorf("failed to generate Spec name: %w", err)
	}

	return cdi.cache.WriteSpec(spec, specName)
}

func (cdi *CDIHandler) CreateClaimSpecFile(claimUID string, devices AttachedDevices) error {
	specName := cdiapi.GenerateTransientSpecName(cdi.vendor(), cdi.class, claimUID)

	spec := &cdispec.Spec{
		Kind:    cdi.kind(),
		Devices: []cdispec.Device{},
	}

	for _, device := range devices {
		_, localBusID, err := usbip.GetLocalFromRemote(device.RemoteHost, device.RemoteBusID)
		if err != nil {
			return fmt.Errorf("could not get local bus ID for %s/%s: %w", device.RemoteHost, device.RemoteBusID, err)
		}

		devInfo, err := usbip.GetLocalDeviceInfo(localBusID)
		if err != nil {
			return fmt.Errorf("could not get local device info for %s/%s: %w", device.RemoteHost, device.RemoteBusID, err)
		}

		// Insert an environment variable mapping the claim/request
		// to the device path of the USB device.
		envName := sanitiseEnvName(fmt.Sprintf("USBIP_DEVICE_%s_%s", device.claimName, device.requestName))
		envVars := []string{
			fmt.Sprintf("%s=%s", envName, devInfo.DevName),
		}

		// Main device node is the USB device itself.
		deviceNodes := []*cdispec.DeviceNode{
			&cdispec.DeviceNode{
				Path:  devInfo.DevName,
				Major: devInfo.Major,
				Minor: devInfo.Minor,
			},
		}

		// Add mounts for any child devices e.g serial converters.
		children, err := devInfo.FindChildren()
		if err != nil {
			return fmt.Errorf("error finding children: %s", err)
		}
		for n, child := range children {
			node := &cdispec.DeviceNode{
				Path:  child.DevName,
				Major: child.Major,
				Minor: child.Minor,
			}
			deviceNodes = append(deviceNodes, node)

			// Each child gets an individual env var.
			childEnv := fmt.Sprintf("%s_CHILD_%d=%s", envName, n, child.DevName)
			envVars = append(envVars, childEnv)
		}

		containerEdits := cdispec.ContainerEdits{
			Env:         envVars,
			DeviceNodes: deviceNodes,
		}

		cdiDevice := cdispec.Device{
			Name:           fmt.Sprintf("%s-%s", claimUID, device.DeviceName),
			ContainerEdits: containerEdits,
		}

		spec.Devices = append(spec.Devices, cdiDevice)
	}

	minVersion, err := cdiapi.MinimumRequiredVersion(spec)
	if err != nil {
		return fmt.Errorf("failed to get minimum required CDI spec version: %v", err)
	}
	spec.Version = minVersion

	return cdi.cache.WriteSpec(spec, specName)
}

func (cdi *CDIHandler) DeleteClaimSpecFile(claimUID string) error {
	specName := cdiapi.GenerateTransientSpecName(cdi.vendor(), cdi.class, claimUID)
	return cdi.cache.RemoveSpec(specName)
}

func (cdi *CDIHandler) GetClaimDevices(claimUID string, devices []string) []string {
	// Insert the common CDI device for all claims.
	cdiDevices := []string{
		cdiparser.QualifiedName(cdi.vendor(), cdi.class, cdiCommonDeviceName),
	}

	for _, device := range devices {
		cdiDevice := cdiparser.QualifiedName(cdi.vendor(), cdi.class, fmt.Sprintf("%s-%s", claimUID, device))
		cdiDevices = append(cdiDevices, cdiDevice)
	}

	return cdiDevices
}

func (cdi *CDIHandler) kind() string {
	return cdi.vendor() + "/" + cdi.class
}

func (cdi *CDIHandler) vendor() string {
	return "k8s." + cdi.driverName
}

// Make input string into a valid environment variable name.
// Replace non alphanumeric characters with underscores.
func sanitiseEnvName(input string) string {
	// Alphanumeric characters and underscore.
	valid := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789_"

	var out strings.Builder
	for _, char := range strings.Split(input, "") {
		if strings.Contains(valid, char) {
			out.WriteString(strings.ToUpper(char))
		} else {
			out.WriteString("_")
		}
	}

	return out.String()
}
