package releasemanifest

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"fmt"
	"io"
)

// ReleaseCompatibilityProfile identifies the exact emitted Manifest shape for
// one release version.
type ReleaseCompatibilityProfile string

const (
	// ReleaseCompatibilityProfileModern is the normal Manifest contract.
	ReleaseCompatibilityProfileModern ReleaseCompatibilityProfile = "modern"
	// ReleaseCompatibilityProfileAlpha164Bridge omits only the two fields that
	// alpha.164's strict decoder predates.
	ReleaseCompatibilityProfileAlpha164Bridge ReleaseCompatibilityProfile = "alpha164-bridge"
)

type releaseCompatibilityProfileRegistry struct {
	SchemaVersion int                                `json:"schemaVersion"`
	Profiles      []releaseCompatibilityProfileEntry `json:"profiles"`
}

type releaseCompatibilityProfileEntry struct {
	ReleaseVersion string                      `json:"releaseVersion"`
	Profile        ReleaseCompatibilityProfile `json:"profile"`
}

//go:embed release_compatibility_profiles.json
var releaseCompatibilityProfilesJSON []byte

var releaseCompatibilityProfiles = mustLoadReleaseCompatibilityProfiles(releaseCompatibilityProfilesJSON)

// ReleaseCompatibilityProfileForVersion returns the checked-in profile for an
// exact bare release version. Unregistered versions always use the modern
// contract so the bridge cannot silently become a future default.
func ReleaseCompatibilityProfileForVersion(releaseVersion string) ReleaseCompatibilityProfile {
	if profile, found := releaseCompatibilityProfiles[releaseVersion]; found {
		return profile
	}
	return ReleaseCompatibilityProfileModern
}

func mustLoadReleaseCompatibilityProfiles(raw []byte) map[string]ReleaseCompatibilityProfile {
	profiles, err := parseReleaseCompatibilityProfiles(raw)
	if err != nil {
		panic(fmt.Sprintf("invalid embedded release compatibility profile registry: %v", err))
	}
	return profiles
}

func parseReleaseCompatibilityProfiles(raw []byte) (map[string]ReleaseCompatibilityProfile, error) {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	var registry releaseCompatibilityProfileRegistry
	if err := decoder.Decode(&registry); err != nil {
		return nil, err
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		if err != nil {
			return nil, err
		}
		return nil, fmt.Errorf("release compatibility profile registry must contain exactly one JSON document")
	}
	if registry.SchemaVersion != 1 {
		return nil, fmt.Errorf("release compatibility profile registry schemaVersion must be 1")
	}
	profiles := make(map[string]ReleaseCompatibilityProfile, len(registry.Profiles))
	for index, entry := range registry.Profiles {
		if !semVerPattern.MatchString(entry.ReleaseVersion) {
			return nil, fmt.Errorf("profiles[%d].releaseVersion must be a canonical semantic version", index)
		}
		if entry.Profile != ReleaseCompatibilityProfileAlpha164Bridge {
			return nil, fmt.Errorf("profiles[%d].profile is not supported", index)
		}
		if _, duplicate := profiles[entry.ReleaseVersion]; duplicate {
			return nil, fmt.Errorf("release compatibility profile registry contains duplicate version %q", entry.ReleaseVersion)
		}
		profiles[entry.ReleaseVersion] = entry.Profile
	}
	return profiles, nil
}
