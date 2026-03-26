package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"image"
	"image/color"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
	"golang.org/x/image/math/fixed"
	"golang.org/x/sys/unix"
	"k8s.io/klog/v2"
)

const (
	picoVID  = "2e8a"
	picoPID  = "0005"
	baudRate = 115200

	brotherVID = "04f9"
	brotherPID = "2042"

	// QL-700 print parameters for 62mm continuous tape.
	printWidth    = 720
	bytesPerLine  = printWidth / 8 // 90
	fontScale     = 4
	smallWidth    = printWidth / fontScale // 180
	lineHeight    = 16                     // pixels in small image
	labelPadding  = 8                      // top/bottom padding in small image
	feedMargin    = 35                     // dots of margin before/after label
	mediaWidthMM  = 62
	mediaTypeCont = 0x0a // continuous roll
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

// findQL700 scans /dev/usb/lp* for a Brother QL-700 by checking
// the USB VID/PID via sysfs.
func findQL700() string {
	matches, err := filepath.Glob("/dev/usb/lp*")
	if err != nil {
		return ""
	}

	for _, dev := range matches {
		name := filepath.Base(dev)
		for _, class := range []string{"/sys/class/usbmisc", "/sys/class/usb"} {
			realDevice, err := filepath.EvalSymlinks(filepath.Join(class, name, "device"))
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

			if strings.TrimSpace(string(vid)) == brotherVID && strings.TrimSpace(string(pid)) == brotherPID {
				return dev
			}
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

func sendSerialMessage(devPath, message string) error {
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

// renderLabel creates a 720-pixel-wide monochrome image with the MAC address
// and hostname rendered using a scaled-up bitmap font.
func renderLabel(mac, hostname string) *image.Gray {
	lines := []string{
		"MAC Address:",
		mac,
		"",
		"Hostname:",
		hostname,
	}

	smallH := len(lines)*lineHeight + 2*labelPadding

	small := image.NewGray(image.Rect(0, 0, smallWidth, smallH))
	for i := range small.Pix {
		small.Pix[i] = 255
	}

	d := &font.Drawer{
		Dst:  small,
		Src:  image.NewUniform(color.Black),
		Face: basicfont.Face7x13,
	}
	for i, line := range lines {
		d.Dot = fixed.P(5, labelPadding+(i+1)*lineHeight-3)
		d.DrawString(line)
	}

	bigH := smallH * fontScale
	big := image.NewGray(image.Rect(0, 0, printWidth, bigH))
	for i := range big.Pix {
		big.Pix[i] = 255
	}
	for y := 0; y < smallH; y++ {
		for x := 0; x < smallWidth; x++ {
			c := small.GrayAt(x, y)
			if c.Y == 255 {
				continue
			}
			for dy := 0; dy < fontScale; dy++ {
				for dx := 0; dx < fontScale; dx++ {
					big.SetGray(x*fontScale+dx, y*fontScale+dy, c)
				}
			}
		}
	}

	return big
}

// imageToRaster converts a grayscale image to packed 1-bit raster lines
// suitable for the Brother QL protocol (dark pixel = 1).
func imageToRaster(img *image.Gray) [][]byte {
	bounds := img.Bounds()
	height := bounds.Dy()
	lines := make([][]byte, height)

	for y := 0; y < height; y++ {
		line := make([]byte, bytesPerLine)
		for x := 0; x < printWidth; x++ {
			if img.GrayAt(x, y).Y < 128 {
				line[x/8] |= 1 << (7 - uint(x%8))
			}
		}
		lines[y] = line
	}
	return lines
}

// ql700Status reads and logs the 32-byte status response from the printer.
func ql700Status(f *os.File) {
	// Request status information.
	if _, err := f.Write([]byte{0x1b, 0x69, 0x53}); err != nil {
		klog.Warningf("Failed to request status: %v", err)
		return
	}
	status := make([]byte, 32)
	n, err := f.Read(status)
	if err != nil {
		klog.Warningf("Failed to read status: %v", err)
		return
	}
	status = status[:n]
	klog.Infof("Printer status (%d bytes): % 02x", n, status)
	if n >= 19 {
		klog.Infof("  Error info 1: 0x%02x  Error info 2: 0x%02x", status[8], status[9])
		klog.Infof("  Media width: %d mm  Media type: 0x%02x", status[10], status[11])
		klog.Infof("  Status type: 0x%02x  Phase: 0x%02x", status[18], status[19])
	}
}

// printQL700Label sends a raster image to the Brother QL-700 using its
// raster command protocol.
func printQL700Label(devPath string, img *image.Gray) error {
	f, err := os.OpenFile(devPath, os.O_WRONLY, 0)
	if err != nil {
		return fmt.Errorf("opening %s: %w", devPath, err)
	}
	defer f.Close()

	rasterLines := imageToRaster(img)
	numLines := uint32(len(rasterLines))
	klog.Infof("Preparing %d raster lines for QL-700", numLines)

	// Invalidate: clear any partial command in the printer's buffer.
	if _, err := f.Write(bytes.Repeat([]byte{0x00}, 200)); err != nil {
		return fmt.Errorf("invalidate: %w", err)
	}

	// Initialize.
	if _, err := f.Write([]byte{0x1b, 0x40}); err != nil {
		return fmt.Errorf("initialize: %w", err)
	}

	// Print information command.
	printInfo := make([]byte, 13)
	printInfo[0] = 0x1b
	printInfo[1] = 0x69
	printInfo[2] = 0x7a
	printInfo[3] = 0xce // validity: recover | quality | length | width | type
	printInfo[4] = mediaTypeCont
	printInfo[5] = mediaWidthMM
	printInfo[6] = 0x00 // length = 0 (continuous)
	binary.LittleEndian.PutUint32(printInfo[7:11], numLines)
	printInfo[11] = 0x00 // starting page
	printInfo[12] = 0x00
	if _, err := f.Write(printInfo); err != nil {
		return fmt.Errorf("print info: %w", err)
	}

	// Auto-cut on.
	if _, err := f.Write([]byte{0x1b, 0x69, 0x4d, 0x40}); err != nil {
		return fmt.Errorf("auto cut: %w", err)
	}

	// Margins.
	if _, err := f.Write([]byte{0x1b, 0x69, 0x64, byte(feedMargin), byte(feedMargin >> 8)}); err != nil {
		return fmt.Errorf("margins: %w", err)
	}

	// Raster data — write each line individually.
	for i, line := range rasterLines {
		blank := true
		for _, b := range line {
			if b != 0 {
				blank = false
				break
			}
		}
		var err error
		if blank {
			_, err = f.Write([]byte{0x5a})
		} else {
			_, err = f.Write(append([]byte{0x67, 0x00, byte(bytesPerLine)}, line...))
		}
		if err != nil {
			return fmt.Errorf("raster line %d: %w", i, err)
		}
	}

	// Print with feeding.
	if _, err := f.Write([]byte{0x1a}); err != nil {
		return fmt.Errorf("print command: %w", err)
	}

	klog.Infof("Printed %d raster lines to %s", numLines, devPath)
	return nil
}

func main() {
	mac := getMACAddress()
	hostname := getHostname()
	serialMsg := fmt.Sprintf("MAC address:\n%s\n---------------\nhost name:\n%s\n", mac, hostname)

	// Check for devices already connected.
	lastPico := ""
	lastQL700 := ""
	if dev := findPico(); dev != "" {
		klog.Infof("Pico found at %s, sending message", dev)
		time.Sleep(1 * time.Second)
		if err := sendSerialMessage(dev, serialMsg); err != nil {
			klog.Errorf("Failed to send message: %v", err)
		}
		lastPico = dev
	}
	if dev := findQL700(); dev != "" {
		klog.Infof("QL-700 found at %s, printing label", dev)
		time.Sleep(1 * time.Second)
		if err := printQL700Label(dev, renderLabel(mac, hostname)); err != nil {
			klog.Errorf("Failed to print label: %v", err)
		}
		lastQL700 = dev
	}

	// Poll for new device connections.
	klog.Info("Waiting for devices to be connected...")
	for {
		time.Sleep(2 * time.Second)

		if dev := findPico(); dev != "" && dev != lastPico {
			klog.Infof("Pico found at %s, sending message", dev)
			time.Sleep(1 * time.Second)
			if err := sendSerialMessage(dev, serialMsg); err != nil {
				klog.Errorf("Failed to send message: %v", err)
			}
			lastPico = dev
		} else if dev == "" {
			lastPico = ""
		}

		if dev := findQL700(); dev != "" && dev != lastQL700 {
			klog.Infof("QL-700 found at %s, printing label", dev)
			time.Sleep(1 * time.Second)
			if err := printQL700Label(dev, renderLabel(mac, hostname)); err != nil {
				klog.Errorf("Failed to print label: %v", err)
			}
			lastQL700 = dev
		} else if dev == "" {
			lastQL700 = ""
		}
	}
}
