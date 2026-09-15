// Package loggingplugin supplies installer helpers, not a runtime protocol.
package loggingplugin

import (
	_ "embed"
	"fmt"
	"strings"
)

//go:embed policy.sh
var policy string

//go:embed manager.sh
var manager string

// Script is sourced by installers. The caller supplies lunafox_plugin_docker
// so credentials, daemon selection and privilege handling remain caller-owned.
func Script() string { return policy + "\n" + manager }

func Reference(architecture string) (string, error) {
	var version string
	for _, line := range strings.Split(policy, "\n") {
		if value, ok := strings.CutPrefix(line, "LUNAFOX_LOKI_VERSION="); ok {
			version = value
		}
	}
	if version == "" {
		return "", fmt.Errorf("Loki plugin policy has no version")
	}
	switch architecture {
	case "amd64", "x86_64":
		architecture = "amd64"
	case "arm64", "aarch64":
		architecture = "arm64"
	default:
		return "", fmt.Errorf("unsupported Docker architecture %q", architecture)
	}
	return "grafana/loki-docker-driver:" + version + "-" + architecture, nil
}
