// Package runtimeimage owns package-local Engine Runtime Image identity rules.
package runtimeimage

import (
	"fmt"
	"path"
	"strings"

	"github.com/yyhuni/lunafox/contracts/enginemanifest/repositoryname"
	"github.com/yyhuni/lunafox/contracts/ociartifact"
)

// Candidates is an ordered set of Registry locations for one immutable image
// digest. Parsing proves only package-local reference invariants; callers that
// publish or install must still resolve the remote OCI descriptor and platforms.
type Candidates struct {
	References         []ociartifact.DigestReference
	RuntimeImageDigest RuntimeImageDigest
}

// RuntimeImageDigest identifies the external Engine Runtime Image or index.
// It is distinct from Engine Package artifact-manifest and archive digests.
type RuntimeImageDigest string

// ParseCandidates validates an ordered, immutable Runtime Image candidate set.
// Repository names are publisher-controlled distribution locations, not Engine
// identity or trust signals.
func ParseCandidates(refs []string) (Candidates, error) {
	if len(refs) == 0 {
		return Candidates{}, fmt.Errorf("runtimeImage candidates are required")
	}

	parsed := Candidates{References: make([]ociartifact.DigestReference, 0, len(refs))}
	seenLocations := make(map[string]struct{}, len(refs))
	for index, value := range refs {
		if value == "" || value != strings.TrimSpace(value) {
			return Candidates{}, fmt.Errorf("runtimeImage candidates[%d] must be a non-empty canonical OCI digest reference", index)
		}
		reference, err := ociartifact.ParseDigestReference(value)
		if err != nil {
			return Candidates{}, fmt.Errorf("runtimeImage candidates[%d]: %w", index, err)
		}
		if reference.String() != value {
			return Candidates{}, fmt.Errorf("runtimeImage candidates[%d] must be a canonical OCI digest reference", index)
		}
		if parsed.RuntimeImageDigest == "" {
			parsed.RuntimeImageDigest = RuntimeImageDigest(reference.Digest)
		} else if reference.Digest != string(parsed.RuntimeImageDigest) {
			return Candidates{}, fmt.Errorf(
				"runtimeImage candidates must identify the same OCI image digest: candidates[%d] has %q, want %q",
				index,
				reference.Digest,
				parsed.RuntimeImageDigest,
			)
		}

		location := reference.Registry + "/" + reference.Repository
		if _, exists := seenLocations[location]; exists {
			return Candidates{}, fmt.Errorf("duplicate runtimeImage candidate location %q", location)
		}
		seenLocations[location] = struct{}{}
		parsed.References = append(parsed.References, reference)
	}
	return parsed, nil
}

// ParseFirstPartyCandidates validates the additional repository identity
// invariant used by first-party release tooling. Generic package validation
// and installation must use ParseCandidates instead, because third-party
// packages are allowed to choose their own repository names.
func ParseFirstPartyCandidates(engineID string, refs []string) (Candidates, error) {
	candidates, err := ParseCandidates(refs)
	if err != nil {
		return Candidates{}, err
	}
	expectedRepository, err := repositoryname.FirstPartyRuntimeImageRepositoryName(engineID)
	if err != nil {
		return Candidates{}, fmt.Errorf("derive first-party Runtime Image repository: %w", err)
	}
	for index, reference := range candidates.References {
		if path.Base(reference.Repository) != expectedRepository {
			return Candidates{}, fmt.Errorf(
				"first-party Runtime Image candidate %d repository %q must use engineId-derived repository %q",
				index,
				reference.Repository,
				expectedRepository,
			)
		}
	}
	return candidates, nil
}
