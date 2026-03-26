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
	picoVID = "2e8a"
	picoPID = "0005"

	brotherVID = "04f9"
	brotherPID = "2042"

	// QL-700 print parameters for DK-1204 die-cut labels (17×54mm).
	// The print head is 720 dots wide; the printable area for this label
	// starts at bit 56 from the left (byte 7), spanning 165 dots.
	headWidth    = 720
	bytesPerLine = headWidth / 8 // 90
	labelWidth   = 165           // printable dots for 17mm
	labelHeight  = 566           // printable dots for 54mm
	labelOffset  = 56 // bit offset from left edge of print head
	mediaWidthMM = 17
	mediaLengthMM   = 54
	mediaTypeDieCut = 0x0b
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

// renderLabel creates a labelWidth × labelHeight monochrome image containing
// the MAC address rotated 90° so text runs along the 54mm feed direction,
// filling as much of the DK-1204 label as possible.
func renderLabel(mac string) *image.Gray {
	// Render text at native basicfont size (7×13) into a landscape buffer
	// where width = long axis (566 px) and height = narrow axis (165 px).
	// We then scale and rotate 90° CW into the final portrait image.
	charW := 7 // basicfont advance
	charH := 13
	textW := len(mac) * charW // e.g. 17 × 7 = 119

	small := image.NewGray(image.Rect(0, 0, textW, charH))
	for i := range small.Pix {
		small.Pix[i] = 255
	}
	d := &font.Drawer{
		Dst:  small,
		Src:  image.NewUniform(color.Black),
		Face: basicfont.Face7x13,
	}
	d.Dot = fixed.P(0, charH-2) // baseline
	d.DrawString(mac)

	// Scale to fill the label.  Text runs along labelHeight (566) and
	// the scaled character height must fit within labelWidth (165).
	scaleX := labelHeight / textW   // 566/119 = 4
	scaleY := labelWidth / charH    // 165/13  = 12
	if scaleX < 1 {
		scaleX = 1
	}
	if scaleY > scaleX {
		scaleY = scaleX // keep aspect ratio uniform
	}

	scaledW := textW * scaleX
	scaledH := charH * scaleY

	// Centre offsets within label dimensions.
	offX := (labelHeight - scaledW) / 2
	offY := (labelWidth - scaledH) / 2

	// Build the final labelWidth × labelHeight image by scaling the small
	// image and rotating 90° CW in a single pass.
	// Rotation: rotated(rx, ry) = landscape(ry, labelWidth-1-rx).
	out := image.NewGray(image.Rect(0, 0, labelWidth, labelHeight))
	for i := range out.Pix {
		out.Pix[i] = 255
	}
	for sy := 0; sy < charH; sy++ {
		for sx := 0; sx < textW; sx++ {
			if small.GrayAt(sx, sy).Y >= 128 {
				continue // skip white
			}
			for dy := 0; dy < scaleY; dy++ {
				for dx := 0; dx < scaleX; dx++ {
					// Landscape position.
					lx := offX + sx*scaleX + dx
					ly := offY + sy*scaleY + dy
					// 90° CW rotation into portrait.
					rx := ly
					ry := lx
					if rx >= 0 && rx < labelWidth && ry >= 0 && ry < labelHeight {
						out.SetGray(rx, ry, color.Gray{Y: 0})
					}
				}
			}
		}
	}

	return out
}

// imageToRaster converts a labelWidth × labelHeight grayscale image to
// 90-byte raster lines.  Pixels are placed at bit offset labelOffset (56)
// within each 720-bit line, matching the DK-1204 position on the print head.
func imageToRaster(img *image.Gray) [][]byte {
	height := img.Bounds().Dy()
	lines := make([][]byte, height)

	for y := 0; y < height; y++ {
		line := make([]byte, bytesPerLine)
		for x := 0; x < labelWidth; x++ {
			if img.GrayAt(x, y).Y < 128 {
				hx := labelOffset + x
				line[hx/8] |= 1 << (7 - uint(hx%8))
			}
		}
		lines[y] = line
	}
	return lines
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

	// Request status (write only — brother_ql sends this before print info).
	if _, err := f.Write([]byte{0x1b, 0x69, 0x53}); err != nil {
		return fmt.Errorf("status request: %w", err)
	}

	// Print information command.
	printInfo := make([]byte, 13)
	printInfo[0] = 0x1b
	printInfo[1] = 0x69
	printInfo[2] = 0x7a
	printInfo[3] = 0xce // validity: recover | quality | length | width | type
	printInfo[4] = mediaTypeDieCut
	printInfo[5] = mediaWidthMM
	printInfo[6] = mediaLengthMM
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

	// Cut every 1 label.
	if _, err := f.Write([]byte{0x1b, 0x69, 0x41, 0x01}); err != nil {
		return fmt.Errorf("cut interval: %w", err)
	}

	// Expanded mode: cut at end.
	if _, err := f.Write([]byte{0x1b, 0x69, 0x4b, 0x08}); err != nil {
		return fmt.Errorf("expanded mode: %w", err)
	}

	// Margins: 0 dots.
	if _, err := f.Write([]byte{0x1b, 0x69, 0x64, 0x00, 0x00}); err != nil {
		return fmt.Errorf("margins: %w", err)
	}

	// Raster data — always use 0x67 format (matching brother_ql).
	for i, line := range rasterLines {
		if _, err := f.Write(append([]byte{0x67, 0x00, byte(bytesPerLine)}, line...)); err != nil {
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
		if err := printQL700Label(dev, renderLabel(mac)); err != nil {
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
			if err := printQL700Label(dev, renderLabel(mac)); err != nil {
				klog.Errorf("Failed to print label: %v", err)
			}
			lastQL700 = dev
		} else if dev == "" {
			lastQL700 = ""
		}
	}
}
