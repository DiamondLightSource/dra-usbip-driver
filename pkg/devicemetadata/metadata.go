package devicemetadata

import (
	"github.com/google/gousb"
)

type Metadata struct {
	Bus     int
	Address int

	Vendor  gousb.ID
	Product gousb.ID

	Class gousb.Class

	Serial string
}
