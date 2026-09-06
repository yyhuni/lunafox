// Package ocidistribution owns first-party OCI download transport mappings.
package ocidistribution

import (
	"errors"
	"fmt"
	"strings"

	"github.com/yyhuni/lunafox/contracts/ociartifact"
)

const (
	CloudflareRegistry = "docker.lunafox.cc.cd"
	dockerHubRegistry  = "docker.io"
	ghcrRegistry       = "ghcr.io"
	firstPartyPrefix   = "yyhuni/lunafox-"
)

// CloudflareAcceleration keeps signature identity separate from download
// transport. SignatureReference always remains the original GHCR reference.
type CloudflareAcceleration struct {
	SignatureReference ociartifact.DigestReference
	DownloadReferences []ociartifact.DigestReference
}

// BuildCloudflareAcceleration accepts only the exact first-party release pair:
// Docker Hub followed by GHCR, with one repository and immutable digest. This
// prevents the accelerator from becoming an alternate source selector.
func BuildCloudflareAcceleration(references []string) (CloudflareAcceleration, error) {
	if len(references) != 2 {
		return CloudflareAcceleration{}, fmt.Errorf("cloudflare acceleration requires exactly Docker Hub and GHCR release candidates")
	}

	parsed := make([]ociartifact.DigestReference, len(references))
	for index, rawReference := range references {
		if rawReference == "" || rawReference != strings.TrimSpace(rawReference) {
			return CloudflareAcceleration{}, fmt.Errorf("cloudflare acceleration candidate %d must be a canonical OCI digest reference", index)
		}
		reference, err := ociartifact.ParseDigestReference(rawReference)
		if err != nil || reference.String() != rawReference {
			if err == nil {
				err = errors.New("reference is not canonical")
			}
			return CloudflareAcceleration{}, fmt.Errorf("cloudflare acceleration candidate %d: %w", index, err)
		}
		parsed[index] = reference
	}

	dockerHub := parsed[0]
	ghcr := parsed[1]
	if dockerHub.Registry != dockerHubRegistry || ghcr.Registry != ghcrRegistry {
		return CloudflareAcceleration{}, fmt.Errorf("cloudflare acceleration requires Docker Hub followed by GHCR release candidates")
	}
	if dockerHub.Repository != ghcr.Repository || dockerHub.Digest != ghcr.Digest {
		return CloudflareAcceleration{}, fmt.Errorf("cloudflare acceleration release candidates must use one repository and digest")
	}
	if !strings.HasPrefix(dockerHub.Repository, firstPartyPrefix) || len(dockerHub.Repository) == len(firstPartyPrefix) {
		return CloudflareAcceleration{}, fmt.Errorf("cloudflare acceleration repository %q is not a first-party LunaFox repository", dockerHub.Repository)
	}

	cloudflare := ociartifact.DigestReference{
		Registry:   CloudflareRegistry,
		Repository: ghcr.Repository,
		Digest:     ghcr.Digest,
	}
	return CloudflareAcceleration{
		SignatureReference: ghcr,
		DownloadReferences: []ociartifact.DigestReference{cloudflare, dockerHub, ghcr},
	}, nil
}

// ParseCloudflareAcceleration validates the exact CF-first transport sequence
// stored in an accelerated Engine Package bootstrap inventory.
func ParseCloudflareAcceleration(references []string) (CloudflareAcceleration, error) {
	if len(references) != 3 {
		return CloudflareAcceleration{}, fmt.Errorf("cloudflare acceleration requires exactly CF, Docker Hub, and GHCR candidates")
	}

	acceleration, err := BuildCloudflareAcceleration(references[1:])
	if err != nil {
		return CloudflareAcceleration{}, err
	}
	if references[0] != acceleration.DownloadReferences[0].String() ||
		references[1] != acceleration.DownloadReferences[1].String() ||
		references[2] != acceleration.DownloadReferences[2].String() {
		return CloudflareAcceleration{}, fmt.Errorf("cloudflare acceleration candidates must be ordered CF, Docker Hub, GHCR with one repository and digest")
	}
	return acceleration, nil
}

// DownloadReferenceStrings returns independently owned, canonical transport
// references so callers cannot mutate the mapping's internal slice.
func (acceleration CloudflareAcceleration) DownloadReferenceStrings() []string {
	values := make([]string, len(acceleration.DownloadReferences))
	for index, reference := range acceleration.DownloadReferences {
		values[index] = reference.String()
	}
	return values
}

// CanAdvanceAfterFailure preserves normal candidate behavior for official
// registries. CF is narrower: a policy response or content-integrity signal
// must fail closed instead of hiding a bad proxy response behind another host.
func CanAdvanceAfterFailure(reference ociartifact.DigestReference, err error) bool {
	if reference.Registry != CloudflareRegistry {
		return ociartifact.IsCandidateUnavailable(err)
	}

	var failure *ociartifact.CandidateFailure
	if !errors.As(err, &failure) {
		return false
	}
	switch failure.Reason() {
	case ociartifact.CandidateFailureReasonDNS,
		ociartifact.CandidateFailureReasonTCP,
		ociartifact.CandidateFailureReasonTLS,
		ociartifact.CandidateFailureReasonAttemptTimeout,
		ociartifact.CandidateFailureReasonRateLimited,
		ociartifact.CandidateFailureReasonTransientServer:
		return true
	default:
		return false
	}
}
