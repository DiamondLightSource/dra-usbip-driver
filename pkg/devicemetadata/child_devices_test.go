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
			"3-1:1.0/ttyUSB0/tty/ttyUSB0/dev":    file("188:0"),
			"3-1:1.0/ttyUSB0/tty/ttyUSB0/device": link("../../../ttyUSB0"),
		},
		[]string{"3-1:1.0/ttyUSB0/tty/ttyUSB0"},
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
