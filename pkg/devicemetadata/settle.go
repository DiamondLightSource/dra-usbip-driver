package devicemetadata

import (
	"os/exec"
	"time"

	"k8s.io/klog/v2"
)

func UdevadmSettle() error {
	startTime := time.Now()
	_, err := exec.Command("udevadm", "settle").Output()
	if err != nil {
		return err
	}

	klog.Infof("Device settle took %s", time.Since(startTime))

	return nil
}
