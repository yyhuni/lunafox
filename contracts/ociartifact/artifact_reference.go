package ociartifact

import (
	"fmt"
	"strings"
)

// EnginePackageArtifactReference is the canonical immutable identity of one
// Engine Package v2 OCI artifact. Its digest identifies the artifact manifest,
// never the package archive layer or Engine Runtime Image.
type EnginePackageArtifactReference struct {
	Registry               string
	Repository             string
	ArtifactManifestDigest ArtifactManifestDigest
}

// String returns the canonical digest-qualified OCI artifact reference.
func (reference EnginePackageArtifactReference) String() string {
	return reference.Registry + "/" + reference.Repository + "@" + string(reference.ArtifactManifestDigest)
}

// DigestReference projects the v2 identity for OCI clients and signature
// verifiers that consume the shared digest-reference transport shape.
func (reference EnginePackageArtifactReference) DigestReference() DigestReference {
	return DigestReference{
		Registry:   reference.Registry,
		Repository: reference.Repository,
		Digest:     string(reference.ArtifactManifestDigest),
	}
}

// ParseArtifactManifestDigest accepts only the canonical lowercase SHA-256
// identity of exact OCI artifact manifest bytes.
func ParseArtifactManifestDigest(value string) (ArtifactManifestDigest, error) {
	if err := validateSHA256Digest(value); err != nil {
		return "", fmt.Errorf("artifactManifestDigest: %w", err)
	}
	return ArtifactManifestDigest(value), nil
}

// ParseEnginePackageArtifactReference rejects any input that would require
// trimming or normalization. Installation identity must already be canonical.
func ParseEnginePackageArtifactReference(value string) (EnginePackageArtifactReference, error) {
	if value == "" || value != strings.TrimSpace(value) {
		return EnginePackageArtifactReference{}, fmt.Errorf("engine package v2 artifactRef must be a non-empty canonical OCI digest reference")
	}
	reference, err := ParseDigestReference(value)
	if err != nil {
		return EnginePackageArtifactReference{}, fmt.Errorf("engine package v2 artifactRef: %w", err)
	}
	if reference.String() != value {
		return EnginePackageArtifactReference{}, fmt.Errorf("engine package v2 artifactRef must be a canonical OCI digest reference")
	}
	manifestDigest, err := ParseArtifactManifestDigest(reference.Digest)
	if err != nil {
		return EnginePackageArtifactReference{}, fmt.Errorf("engine package v2 artifactRef: %w", err)
	}
	return EnginePackageArtifactReference{
		Registry:               reference.Registry,
		Repository:             reference.Repository,
		ArtifactManifestDigest: manifestDigest,
	}, nil
}

// ArtifactCandidates is an ordered set of Registry locations for one exact
// Engine Package v2 artifact manifest.
type ArtifactCandidates struct {
	References             []EnginePackageArtifactReference
	ArtifactManifestDigest ArtifactManifestDigest
}

// ParseArtifactCandidates validates one immutable candidate group while
// preserving its declared failover order.
func ParseArtifactCandidates(values []string) (ArtifactCandidates, error) {
	if len(values) == 0 {
		return ArtifactCandidates{}, fmt.Errorf("engine package v2 artifact candidates are required")
	}

	parsed := ArtifactCandidates{
		References: make([]EnginePackageArtifactReference, 0, len(values)),
	}
	seenLocations := make(map[string]struct{}, len(values))
	for index, value := range values {
		reference, err := ParseEnginePackageArtifactReference(value)
		if err != nil {
			return ArtifactCandidates{}, fmt.Errorf("engine package v2 artifact candidates[%d]: %w", index, err)
		}
		location := reference.Registry + "/" + reference.Repository
		if _, duplicate := seenLocations[location]; duplicate {
			return ArtifactCandidates{}, fmt.Errorf("duplicate Engine Package v2 artifact candidate location %q", location)
		}
		seenLocations[location] = struct{}{}

		if parsed.ArtifactManifestDigest == "" {
			parsed.ArtifactManifestDigest = reference.ArtifactManifestDigest
		} else if reference.ArtifactManifestDigest != parsed.ArtifactManifestDigest {
			return ArtifactCandidates{}, fmt.Errorf(
				"engine package v2 artifact candidates must identify the same OCI artifact manifest digest: candidates[%d] has %q, want %q",
				index,
				reference.ArtifactManifestDigest,
				parsed.ArtifactManifestDigest,
			)
		}
		parsed.References = append(parsed.References, reference)
	}
	return parsed, nil
}
