package upgrader

import "github.com/yyhuni/lunafox/contracts/releasemanifest"

// loadManifestWithLegacyCompatibility keeps the ordinary manifest contract
// strict while preserving recovery for the policy-pinned v1 bootstrap and the
// exact alpha.164 bridge profile. The compatibility parser rejects unknown,
// partial, and unregistered shapes, so it cannot turn a newly malformed or
// composition-less release into a trusted deployment artifact.
func loadManifestWithLegacyCompatibility(path string) (*releasemanifest.Manifest, error) {
	manifest, err := releasemanifest.Load(path)
	if err == nil {
		return manifest, nil
	}
	legacy, legacyErr := releasemanifest.LoadLegacyCompatible(path)
	if legacyErr == nil {
		return legacy, nil
	}
	return nil, err
}
