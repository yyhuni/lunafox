package upgrader

import "github.com/yyhuni/lunafox/contracts/releasemanifest"

// loadManifestWithLegacyCompatibility keeps the ordinary manifest contract
// strict while preserving recovery for the one policy-pinned v1 bootstrap
// release. ParseLegacyAlpha114 verifies the exact historical bytes and digest,
// so this fallback cannot turn a newly malformed or composition-less release
// into a trusted deployment artifact.
func loadManifestWithLegacyCompatibility(path string) (*releasemanifest.Manifest, error) {
	manifest, err := releasemanifest.Load(path)
	if err == nil {
		return manifest, nil
	}
	legacy, legacyErr := releasemanifest.LoadLegacyAlpha114(path)
	if legacyErr == nil {
		return legacy, nil
	}
	return nil, err
}
