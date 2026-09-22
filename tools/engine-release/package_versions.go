package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/yyhuni/lunafox/contracts/versioning"
)

const packageVersionMapSchemaVersion = "lunafox.engine-package-version-map.v1"

// PackageVersionMap is the release resolver's content-addressed version
// handoff. It is intentionally separate from the image receipt: a package
// archive must be named from the exact Runtime/Package input pair, not from
// the product release tag.
type PackageVersionMap struct {
	SchemaVersion string                   `json:"schemaVersion"`
	PlanDigest    string                   `json:"planDigest"`
	Versions      []PackageVersionMapEntry `json:"versions"`
}

type PackageVersionMapEntry struct {
	EngineID       string `json:"engineId"`
	PackageVersion string `json:"packageVersion"`
}

func decodePackageVersionMap(payload []byte, source string) (PackageVersionMap, error) {
	if err := rejectDuplicateJSONFields(bytes.NewReader(payload)); err != nil {
		return PackageVersionMap{}, fmt.Errorf("decode package version map %q: %w", source, err)
	}
	decoder := json.NewDecoder(bytes.NewReader(payload))
	decoder.DisallowUnknownFields()
	var value PackageVersionMap
	if err := decoder.Decode(&value); err != nil {
		return PackageVersionMap{}, fmt.Errorf("decode package version map %q: %w", source, err)
	}
	if err := consumeJSONEOF(decoder); err != nil {
		return PackageVersionMap{}, fmt.Errorf("decode package version map %q: %w", source, err)
	}
	if err := validatePackageVersionMapShape(value); err != nil {
		return PackageVersionMap{}, fmt.Errorf("package version map %q: %w", source, err)
	}
	return value, nil
}

func validatePackageVersionMapShape(value PackageVersionMap) error {
	if value.SchemaVersion != packageVersionMapSchemaVersion {
		return fmt.Errorf("unsupported schemaVersion %q", value.SchemaVersion)
	}
	if !sha256DigestPattern.MatchString(value.PlanDigest) {
		return fmt.Errorf("planDigest must be a canonical sha256 digest")
	}
	if len(value.Versions) == 0 {
		return fmt.Errorf("versions cannot be empty")
	}
	lastEngineID := ""
	seen := make(map[string]struct{}, len(value.Versions))
	for index, entry := range value.Versions {
		if entry.EngineID == "" || entry.EngineID != strings.TrimSpace(entry.EngineID) {
			return fmt.Errorf("versions[%d].engineId is required and must be canonical", index)
		}
		if _, exists := seen[entry.EngineID]; exists {
			return fmt.Errorf("versions contains duplicate engineId %q", entry.EngineID)
		}
		if lastEngineID != "" && entry.EngineID <= lastEngineID {
			return fmt.Errorf("versions must be ordered by canonical engineId")
		}
		if entry.PackageVersion == "" || entry.PackageVersion != strings.TrimSpace(entry.PackageVersion) || !versioning.IsValidSemVer(entry.PackageVersion) {
			return fmt.Errorf("package version for %q must be one canonical semantic version", entry.EngineID)
		}
		seen[entry.EngineID] = struct{}{}
		lastEngineID = entry.EngineID
	}
	return nil
}

func packageVersionsForDiscovery(value PackageVersionMap, discovery Discovery) (map[string]string, error) {
	if err := validatePackageVersionMapShape(value); err != nil {
		return nil, err
	}
	if len(value.Versions) != len(discovery.Engines) {
		return nil, fmt.Errorf("package version map count %d does not match selected Engine discovery count %d", len(value.Versions), len(discovery.Engines))
	}
	versions := make(map[string]string, len(value.Versions))
	for _, entry := range value.Versions {
		versions[entry.EngineID] = entry.PackageVersion
	}
	for _, source := range discovery.Engines {
		if _, ok := versions[source.EngineID]; !ok {
			return nil, fmt.Errorf("package version map is missing selected Engine %q", source.EngineID)
		}
	}
	return versions, nil
}

func readPackageVersionMap(path string) (PackageVersionMap, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return PackageVersionMap{}, fmt.Errorf("-package-version-map is required")
	}
	payload, err := readRegularFile(path)
	if err != nil {
		return PackageVersionMap{}, fmt.Errorf("read package version map: %w", err)
	}
	return decodePackageVersionMap(payload, path)
}
