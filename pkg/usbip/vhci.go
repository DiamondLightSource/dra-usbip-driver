package usbip

import (
	"bufio"
	"fmt"
	"io"
	"os"

	"github.com/diamondlightsource/dra-usbip-driver/pkg/devicemetadata"
)

const (
	VHCI_DEVICE = "/sys/devices/platform/vhci_hcd.0"
	VHCI_STATUS = VHCI_DEVICE + "/status"
)

const (
	StatusNull        = 4
	StatusNotAssigned = 5
	StatusUsed        = 6
	StatusError       = 7
)

type importedDevice struct {
	port   int
	status int

	localBusID string
}

// List all imported devices using the vhci status file.
// The important fields to get are the vhci port number
// and the local busID.
func ListImportedDevices() ([]*importedDevice, error) {
	// Run "cat /sys/devices/platform/vhci_hcd.0/status"
	// to see the raw content of the file.
	statusFile, err := os.Open(VHCI_STATUS)
	if err != nil {
		return nil, fmt.Errorf("unable to open vhci status file: %w", err)
	}
	defer statusFile.Close()

	reader := bufio.NewReader(statusFile)

	// Discard the column headers line.
	_, _, err = reader.ReadLine()
	if err != nil {
		return nil, fmt.Errorf("error reading header line: %w", err)
	}

	var devices []*importedDevice

	for {
		var hub, busID string
		var port, status, speed, dev, sock int

		// Have to provide the format string, otherwise the
		// leading zeroes in the fields cause e.g port "0008"
		// to be interpreted as an invalid octal number.
		_, err := fmt.Fscanf(reader, "%s  %d %d %d %x %d %s\n",
			&hub, &port, &status, &speed, &dev, &sock, &busID)

		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("error reading vhci status line: %w", err)
		}

		// The status file will list all ports, including unused ones.
		// Ports with status 6 have attached devices.
		if status == StatusUsed {
			d := &importedDevice{
				port:       port,
				status:     status,
				localBusID: busID,
			}

			devices = append(devices, d)
		}
	}

	return devices, nil
}

// Read the port file the usbip writes after attaching a device.
// https://github.com/torvalds/linux/blob/v6.18/tools/usb/usbip/src/usbip_attach.c#L39-L79
// This contains a string in the form "<host> <port> <busID>".
func readPortFile(portID int) (string, string, error) {
	portFilePath := fmt.Sprintf("/var/run/vhci_hcd/port%d", portID)

	portFile, err := os.Open(portFilePath)
	if err != nil {
		return "", "", fmt.Errorf("unable to open port file: %w", err)
	}
	defer portFile.Close()

	var remoteHost, remotePort, remoteBusID string
	_, err = fmt.Fscanln(portFile, &remoteHost, &remotePort, &remoteBusID)
	if err != nil {
		return "", "", fmt.Errorf("unable to parse port description: %w", err)
	}

	return remoteHost, remoteBusID, nil
}

// For a given remote host and remote bus ID, find the local vhci port number
// and local bus ID of that imported device.
func GetLocalFromRemote(remoteHost, remoteBusID string) (int, string, error) {
	importedDevices, err := ListImportedDevices()
	if err != nil {
		return 0, "", fmt.Errorf("could not list imported devices: %w", err)
	}

	for _, device := range importedDevices {
		deviceHost, deviceBus, err := readPortFile(device.port)
		if err != nil {
			return 0, "", fmt.Errorf("unable to check port %d: %w", device.port, err)
		}
		if deviceHost == remoteHost && deviceBus == remoteBusID {
			// Found the matching remote host/bus, return the local bus ID.
			return device.port, device.localBusID, nil
		}
	}

	return 0, "", RemoteDeviceNotFoundError
}

func GetLocalDeviceInfo(localBusID string) (*devicemetadata.UdevadmInfo, error) {
	sysPath := fmt.Sprintf("/sys/bus/usb/devices/%s", localBusID)

	udevInfo, err := devicemetadata.ReadDeviceInfo(sysPath)
	if err != nil {
		return nil, err
	}

	return udevInfo, nil
}
