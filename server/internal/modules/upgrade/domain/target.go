package domain

import (
	"fmt"
	"strings"

	"github.com/yyhuni/lunafox/contracts/releasemanifest"
)

// Target is the only release identity an Upgrade Operation may carry. Image
// references are intentionally absent; they are projected from the validated
// manifest and cannot be supplied by a browser request.
type Target struct {
	ManifestID     string `json:"manifestId"`
	ManifestDigest string `json:"manifestDigest"`
}

// TargetRequest is the narrow request-side identity accepted by the Server.
// ImageRefs is retained only to make accidental client injection explicit and
// reject it at the boundary; it is never used to select an image.
type TargetRequest struct {
	ManifestID     string   `json:"manifestId"`
	ManifestDigest string   `json:"manifestDigest"`
	ImageRefs      []string `json:"imageRefs,omitempty"`
	ImageRef       string   `json:"imageRef,omitempty"`
}

func TargetFromManifest(manifest *releasemanifest.Manifest) (Target, error) {
	if manifest == nil {
		return Target{}, newPolicyError(ErrorCodeReleaseManifestInvalid, ErrReleaseManifestInvalid, "manifest", "", "manifest is required")
	}
	digest := strings.TrimSpace(manifest.Digest())
	if digest == "" {
		return Target{}, newPolicyError(ErrorCodeReleaseManifestInvalid, ErrReleaseManifestInvalid, "manifest", "digest", "manifest digest is required")
	}
	if !isSHA256Digest(digest) {
		return Target{}, newPolicyError(ErrorCodeReleaseManifestInvalid, ErrReleaseManifestInvalid, "manifest", "digest", "manifest digest must be a sha256 digest")
	}
	if strings.TrimSpace(manifest.Upgrade.ManifestID) == "" {
		return Target{}, newPolicyError(ErrorCodeReleaseManifestInvalid, ErrReleaseManifestInvalid, "manifest", "upgrade.manifestId", "manifest identity is required")
	}
	return Target{ManifestID: manifest.Upgrade.ManifestID, ManifestDigest: digest}, nil
}

// ValidateTargetRequest verifies only server-produced target identity. It
// never parses or accepts image references from the request.
func ValidateTargetRequest(request TargetRequest, manifest *releasemanifest.Manifest) error {
	if len(request.ImageRefs) != 0 || strings.TrimSpace(request.ImageRef) != "" {
		return newPolicyError(ErrorCodeReleaseManifestTargetInvalid, ErrReleaseManifestTargetInvalid, "request", "imageRefs", "image references must be selected by the validated release manifest")
	}
	want, err := TargetFromManifest(manifest)
	if err != nil {
		return err
	}
	if strings.TrimSpace(request.ManifestID) == "" {
		return newPolicyError(ErrorCodeReleaseManifestTargetInvalid, ErrReleaseManifestTargetInvalid, "request", "manifestId", "manifest identity is required")
	}
	if strings.TrimSpace(request.ManifestDigest) == "" {
		return newPolicyError(ErrorCodeReleaseManifestTargetInvalid, ErrReleaseManifestTargetInvalid, "request", "manifestDigest", "manifest digest is required")
	}
	if !isSHA256Digest(strings.TrimSpace(request.ManifestDigest)) {
		return newPolicyError(ErrorCodeReleaseManifestTargetInvalid, ErrReleaseManifestTargetInvalid, "request", "manifestDigest", "manifest digest must be a sha256 digest")
	}
	if request.ManifestID != want.ManifestID || request.ManifestDigest != want.ManifestDigest {
		return newPolicyError(ErrorCodeReleaseManifestTargetMismatch, ErrReleaseManifestTargetMismatch, "request", "manifestDigest", "request target does not match the validated manifest")
	}
	return nil
}

// EnsureSameTarget prevents a retry from silently changing release identity.
func EnsureSameTarget(original, retry Target) error {
	if strings.TrimSpace(original.ManifestID) == "" || strings.TrimSpace(retry.ManifestID) == "" {
		return newPolicyError(ErrorCodeReleaseManifestTargetInvalid, ErrReleaseManifestTargetInvalid, "retry", "manifestId", "retry target identity is required")
	}
	if strings.TrimSpace(original.ManifestDigest) == "" || strings.TrimSpace(retry.ManifestDigest) == "" {
		return newPolicyError(ErrorCodeReleaseManifestTargetInvalid, ErrReleaseManifestTargetInvalid, "retry", "manifestDigest", "retry target digest is required")
	}
	if !isSHA256Digest(strings.TrimSpace(original.ManifestDigest)) || !isSHA256Digest(strings.TrimSpace(retry.ManifestDigest)) {
		return newPolicyError(ErrorCodeReleaseManifestTargetInvalid, ErrReleaseManifestTargetInvalid, "retry", "manifestDigest", "retry target digest must be a sha256 digest")
	}
	if original != retry {
		return newPolicyError(ErrorCodeReleaseManifestTargetMismatch, ErrReleaseManifestTargetMismatch, "retry", "manifestDigest", fmt.Sprintf("retry must remain bound to manifest %q and its digest", original.ManifestID))
	}
	return nil
}

func isSHA256Digest(value string) bool {
	const prefix = "sha256:"
	if len(value) != len(prefix)+64 || !strings.HasPrefix(value, prefix) {
		return false
	}
	for index := len(prefix); index < len(value); index++ {
		character := value[index]
		if (character < '0' || character > '9') && (character < 'a' || character > 'f') {
			return false
		}
	}
	return true
}
