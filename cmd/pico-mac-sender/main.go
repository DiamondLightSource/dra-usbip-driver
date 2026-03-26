package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"golang.org/x/sys/unix"
	"k8s.io/klog/v2"
)

const (
	picoVID  = "2e8a"
	picoPID  = "0005"
	baudRate = 115200
)

func getMACAddress() string {
	netPath := "/sys/class/net"
	entries, err := os.ReadDir(netPath)
	if err != nil {
		return "00:00:00:00:00:00"
	}

	var interfaces []string
	for _, e := range entries {
		name := e.Name()
		if name != "lo" && !strings.HasPrefix(name, "docker") {
			interfaces = append(interfaces, name)
		}
	}
	sort.Strings(interfaces)

	if len(interfaces) == 0 {
		return "00:00:00:00:00:00"
	}

	macFile := filepath.Join(netPath, interfaces[0], "address")
	data, err := os.ReadFile(macFile)
	if err != nil {
		return "00:00:00:00:00:00"
	}
	return strings.ToUpper(strings.TrimSpace(string(data)))
}

func getHostname() string {
	name, err := os.Hostname()
	if err != nil {
		return "unknown"
	}
	return name
}

// findPico checks /dev/ttyACM* devices for a Raspberry Pi Pico
// by reading the USB VID/PID from sysfs.
func findPico() string {
	matches, err := filepath.Glob("/dev/ttyACM*")
	if err != nil {
		return ""
	}

	for _, dev := range matches {
		// The sysfs path for a tty device's USB parent has idVendor/idProduct.
		// We must resolve the "device" symlink before navigating to its parent,
		// because filepath.Join would collapse "device/.." lexically.
		ttyName := filepath.Base(dev)
		realDevice, err := filepath.EvalSymlinks(filepath.Join("/sys/class/tty", ttyName, "device"))
		if err != nil {
			continue
		}
		devicePath := filepath.Dir(realDevice)

		vid, err := os.ReadFile(filepath.Join(devicePath, "idVendor"))
		if err != nil {
			continue
		}
		pid, err := os.ReadFile(filepath.Join(devicePath, "idProduct"))
		if err != nil {
			continue
		}

		if strings.TrimSpace(string(vid)) == picoVID && strings.TrimSpace(string(pid)) == picoPID {
			return dev
		}
	}
	return ""
}

// configureSerial sets the tty to raw mode at 115200 baud.
func configureSerial(fd int) error {
	termios, err := unix.IoctlGetTermios(fd, unix.TCGETS)
	if err != nil {
		return fmt.Errorf("TCGETS: %w", err)
	}

	// Set raw mode
	termios.Iflag &^= unix.IGNBRK | unix.BRKINT | unix.PARMRK | unix.ISTRIP | unix.INLCR | unix.IGNCR | unix.ICRNL | unix.IXON
	termios.Oflag &^= unix.OPOST
	termios.Lflag &^= unix.ECHO | unix.ECHONL | unix.ICANON | unix.ISIG | unix.IEXTEN
	termios.Cflag &^= unix.CSIZE | unix.PARENB
	termios.Cflag |= unix.CS8 | unix.CLOCAL | unix.CREAD

	// Set baud rate to 115200
	termios.Ispeed = unix.B115200
	termios.Ospeed = unix.B115200

	// Min bytes = 0, timeout = 10 (1 second)
	termios.Cc[unix.VMIN] = 0
	termios.Cc[unix.VTIME] = 10

	if err := unix.IoctlSetTermios(fd, unix.TCSETS, termios); err != nil {
		return fmt.Errorf("TCSETS: %w", err)
	}
	return nil
}

func sendMessage(devPath, message string) error {
	fd, err := unix.Open(devPath, unix.O_RDWR|unix.O_NOCTTY, 0)
	if err != nil {
		return fmt.Errorf("opening %s: %w", devPath, err)
	}
	defer unix.Close(fd)

	if err := configureSerial(fd); err != nil {
		return fmt.Errorf("configuring serial: %w", err)
	}

	_, err = unix.Write(fd, []byte(message))
	if err != nil {
		return fmt.Errorf("writing to %s: %w", devPath, err)
	}
	return nil
}

func main() {
	msg := fmt.Sprintf("MAC address:\n%s\n---------------\nhost name:\n%s\n", getMACAddress(), getHostname())

	// Check if a Pico is already connected.
	if dev := findPico(); dev != "" {
		klog.Infof("Pico found at %s, sending message", dev)
		time.Sleep(1 * time.Second) // wait for device to be ready
		if err := sendMessage(dev, msg); err != nil {
			klog.Errorf("Failed to send message: %v", err)
		}
	}

	// Poll for new Pico connections.
	klog.Info("Waiting for Pico device to be connected...")
	lastSeen := ""
	for {
		time.Sleep(2 * time.Second)
		dev := findPico()
		if dev != "" && dev != lastSeen {
			klog.Infof("Pico found at %s, sending message", dev)
			time.Sleep(1 * time.Second)
			if err := sendMessage(dev, msg); err != nil {
				klog.Errorf("Failed to send message: %v", err)
			}
			lastSeen = dev
		} else if dev == "" {
			lastSeen = ""
		}
	}
}
