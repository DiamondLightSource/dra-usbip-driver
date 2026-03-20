package kmod

import (
	"fmt"
	"os"
	"strings"
)

func IsLoaded(moduleName string) (bool, error) {
	modules, err := os.ReadFile("/proc/modules")
	if err != nil {
		return false, fmt.Errorf("error reading modules proc file: %w", err)
	}

	for line := range strings.Lines(string(modules)) {
		parts := strings.Fields(line)
		if len(parts) == 0 {
			continue
		}
		if parts[0] == moduleName {
			return true, nil
		}
	}

	return false, nil
}
