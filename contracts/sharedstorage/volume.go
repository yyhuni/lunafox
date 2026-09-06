package sharedstorage

import (
	"errors"
	"fmt"
	"strings"
)

// SharedDataVolumeBind is the canonical daemon-visible shared volume mount.
type SharedDataVolumeBind struct {
	VolumeName string
	MountPath  string
}

// ParseSharedDataVolumeBind rejects any bind that could diverge from the
// Compose-owned volume identity or make Agent workspace writes read-only.
func ParseSharedDataVolumeBind(value string) (SharedDataVolumeBind, error) {
	if value == "" {
		return SharedDataVolumeBind{}, errors.New("shared data volume bind is required")
	}
	if value != strings.TrimSpace(value) {
		return SharedDataVolumeBind{}, errors.New("shared data volume bind must be canonical text")
	}
	parts := strings.Split(value, ":")
	if len(parts) < 2 || len(parts) > 3 {
		return SharedDataVolumeBind{}, fmt.Errorf("shared data volume bind must be '<named-volume>:%s[:rw]'", SharedDataRoot)
	}
	if parts[0] != DefaultSharedDataVolumeName {
		return SharedDataVolumeBind{}, fmt.Errorf("shared data volume name must be %s", DefaultSharedDataVolumeName)
	}
	if parts[1] != SharedDataRoot {
		return SharedDataVolumeBind{}, fmt.Errorf("shared data volume target must be %s", SharedDataRoot)
	}
	if len(parts) == 3 && parts[2] != "rw" {
		return SharedDataVolumeBind{}, errors.New("shared data volume mode must be rw")
	}
	return SharedDataVolumeBind{VolumeName: parts[0], MountPath: parts[1]}, nil
}
