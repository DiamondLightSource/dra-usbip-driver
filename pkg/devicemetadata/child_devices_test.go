package devicemetadata

import (
	"io/fs"
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/require"
)

// Test representing a given device type and a mocked
// filesystem under its /sys/bus/usb/devices/x-y/ path
// where the plugin will look for children of that device.
// Tests should include some extra files e.g the vendor
// and product IDs that should be ignored by the search.
// The expected output should be the child device paths
// relative to the parent.
type childrenTest struct {
	device   string
	sysfs    fstest.MapFS
	expected []string
}

// Shortcut for defining a mocked file.
func file(content string) *fstest.MapFile {
	return &fstest.MapFile{Data: []byte(content)}
}

// Shortcut for defining a mocked symlink.
func link(content string) *fstest.MapFile {
	return &fstest.MapFile{Data: []byte(content), Mode: fs.ModeSymlink}
}

var childrenTests = []childrenTest{
	{
		// Basic device with no children.
		"basic",
		fstest.MapFS{
			"idVendor":  file("0011"),
			"idProduct": file("7788"),
		},
		nil,
	},
	{
		// FTDI Chipi-X with serial device child.
		"chipi-x",
		fstest.MapFS{
			"idVendor":                           file("0403"),
			"idProduct":                          file("6015"),
			"uevent":                             file("MAJOR=189\nMINOR=260\nDEVNAME=bus/usb/003/005\nDEVTYPE=usb_device"),
			"3-1:1.0/ttyUSB0/tty/ttyUSB0/dev":    file("188:0"),
			"3-1:1.0/ttyUSB0/tty/ttyUSB0/device": link("../../../ttyUSB0"),
			"3-1:1.0/ttyUSB0/tty/ttyUSB0/uevent": file("MAJOR=188\nMINOR=0\nDEVNAME=ttyUSB0"),
		},
		[]string{"3-1:1.0/ttyUSB0/tty/ttyUSB0"},
	},
	{
		// Thorlabs flipper.
		"flipper",
		fstest.MapFS{
			"idVendor":  file("0403"),
			"idProduct": file("faf0"),
			"uevent":    file("MAJOR=189\nMINOR=258\nDEVNAME=bus/usb/003/003\nDEVTYPE=usb_device"),
			// Serial child.
			"3-1:1.0/ttyUSB0/tty/ttyUSB0/dev":    file("188:0"),
			"3-1:1.0/ttyUSB0/tty/ttyUSB0/device": link("../../../ttyUSB0"),
			"3-1:1.0/ttyUSB0/tty/ttyUSB0/uevent": file("MAJOR=188\nMINOR=0\nDEVNAME=ttyUSB0"),
			// GPIO child.
			"3-1:1.0/gpiochip0/dev":    file("254:0"),
			"3-1:1.0/gpiochip0/uevent": file("MAJOR=254\nMINOR=0\nDEVNAME=gpiochip0"),
			// Has a "device" file but not an actual child device.
			"3-1:1.0/gpio/gpiochip1020/device": link("../../../3-1:1.0"),
			"3-1:1.0/gpio/gpiochip1020/uevent": file(""),
		},
		[]string{"3-1:1.0/ttyUSB0/tty/ttyUSB0", "3-1:1.0/gpiochip0"},
	},
}

func TestFindChildren(t *testing.T) {
	for _, tt := range childrenTests {
		t.Run(tt.device, func(t *testing.T) {
			children, err := findChildDevicePaths(tt.sysfs)
			require.NoError(t, err)
			require.ElementsMatch(t, tt.expected, children)
		})
	}
}

var childTTY0 = &UdevadmInfo{
	DevName:   "/dev/ttyUSB0",
	Subsystem: "tty",
}

var childTTY1 = &UdevadmInfo{
	DevName:   "/dev/ttyUSB1",
	Subsystem: "tty",
}

var childGPIO0 = &UdevadmInfo{
	DevName:   "/dev/gpiochip0",
	Subsystem: "gpio",
}

type aliasTest struct {
	name            string
	children        []*UdevadmInfo
	expectedAliases []string
}

var aliasTests = []aliasTest{
	{name: "onetty", children: []*UdevadmInfo{childTTY0}, expectedAliases: []string{"/dev/usbip-c-d-tty"}},
	{name: "twotty", children: []*UdevadmInfo{childTTY0, childTTY1}, expectedAliases: []string{"/dev/usbip-c-d-tty0", "/dev/usbip-c-d-tty1"}},
	{name: "onegpio", children: []*UdevadmInfo{childGPIO0}, expectedAliases: []string{"/dev/usbip-c-d-gpio"}},
	{name: "mix", children: []*UdevadmInfo{childTTY0, childGPIO0}, expectedAliases: []string{"/dev/usbip-c-d-tty", "/dev/usbip-c-d-gpio"}},
	{name: "mixmulti", children: []*UdevadmInfo{childTTY0, childTTY1, childGPIO0}, expectedAliases: []string{"/dev/usbip-c-d-tty0", "/dev/usbip-c-d-tty1", "/dev/usbip-c-d-gpio"}},
}

func TestToAliases(t *testing.T) {
	for _, tt := range aliasTests {
		t.Run(tt.name, func(t *testing.T) {
			aliases := ToAliases("c", "d", tt.children)

			var aliasNames []string
			for _, child := range aliases {
				aliasNames = append(aliasNames, child.Alias)
			}

			require.ElementsMatch(t, tt.expectedAliases, aliasNames)
		})
	}
}
