// Package repositoryname owns the canonical first-party engine OCI repository derivation.
package repositoryname

import (
	"fmt"
	"regexp"
	"strings"
)

const (
	firstPartyEngineIDPrefix = "engine.lunafox."

	// Existing first-party Engine IDs remain useful for fixtures and default
	// workflow declarations. They are not a closed allowlist: discovery and
	// release receipts establish the active first-party Engine set.
	FirstPartyEngineIDSubdomainDiscovery = "engine.lunafox.subdomain_discovery"
	FirstPartyEngineIDPortScan           = "engine.lunafox.port_scan"
	FirstPartyEngineIDWebsiteDiscovery   = "engine.lunafox.website_discovery"
)

var (
	firstPartyEngineLocalIDPattern = regexp.MustCompile(`^[a-z][a-z0-9_]*$`)
	firstPartyEngineReleaseSuffix  = regexp.MustCompile(`(?:_v[0-9]+(?:_[0-9]+)*|_sha256_[a-f0-9]{64}|_[a-f0-9]{7,64}|_release_[a-z0-9]+|_[0-9]+)$`)
)

// ValidateFirstPartyBuiltinEngineID validates the canonical namespace and
// stable local-name grammar for a first-party builtin Engine. It deliberately
// does not enumerate source identities: the discovered source/inventory and
// signed release receipts are the authority for which Engines are active.
func ValidateFirstPartyBuiltinEngineID(engineID string) error {
	_, err := firstPartyEngineLocalName(engineID)
	return err
}

// FirstPartyOCIRepositoryName derives the first-party Engine Package OCI
// repository for one canonical builtin Engine identity. Release publication
// intentionally uses the same repository family as the Runtime Image so the
// package, image, and release manifest cannot drift to different identities.
func FirstPartyOCIRepositoryName(engineID string) (string, error) {
	localID, err := firstPartyEngineLocalName(engineID)
	if err != nil {
		return "", err
	}
	return "lunafox-engine-runtime-" + strings.ReplaceAll(localID, "_", "-"), nil
}

// FirstPartyRuntimeImageRepositoryName derives the repository name used by a
// runnable Engine Runtime Image. It intentionally agrees with
// FirstPartyOCIRepositoryName for first-party release artifacts.
func FirstPartyRuntimeImageRepositoryName(engineID string) (string, error) {
	localID, err := firstPartyEngineLocalName(engineID)
	if err != nil {
		return "", err
	}
	return "lunafox-engine-runtime-" + strings.ReplaceAll(localID, "_", "-"), nil
}

func firstPartyEngineLocalName(engineID string) (string, error) {
	localID, found := strings.CutPrefix(engineID, firstPartyEngineIDPrefix)
	if !found || len(engineID) > 128 || !firstPartyEngineLocalIDPattern.MatchString(localID) {
		return "", fmt.Errorf(
			"first-party builtin engineId %q must use %s<lowercase_underscore_name>",
			engineID,
			firstPartyEngineIDPrefix,
		)
	}
	if firstPartyEngineReleaseSuffix.MatchString(localID) {
		return "", fmt.Errorf("first-party builtin engineId %q must not carry a release-scoped suffix", engineID)
	}
	return localID, nil
}
