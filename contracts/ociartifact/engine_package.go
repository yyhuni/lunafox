// Package ociartifact owns the OCI-level contract for LunaFox engine packages.
// Package contents remain the responsibility of contracts/enginemanifest.
package ociartifact

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"regexp"
	"strings"

	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
)

var (
	digestPattern     = regexp.MustCompile(`^sha256:[a-f0-9]{64}$`)
	registryPattern   = regexp.MustCompile(`^[a-z0-9](?:[a-z0-9.-]*[a-z0-9])?(?::[0-9]+)?$`)
	repositoryPattern = regexp.MustCompile(`^[a-z0-9]+(?:[._-][a-z0-9]+)*(?:/[a-z0-9]+(?:[._-][a-z0-9]+)*)*$`)
)

// DigestReference is the canonical OCI identity accepted by installation.
// It intentionally has no tag field: tags are mutable and are never runtime identity.
type DigestReference struct {
	Registry   string
	Repository string
	Digest     string
}

// String returns the canonical digest-qualified OCI reference.
func (r DigestReference) String() string {
	return r.Registry + "/" + r.Repository + "@" + r.Digest
}

// ParseDigestReference rejects tag-only, malformed, and non-SHA256 OCI references.
func ParseDigestReference(value string) (DigestReference, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return DigestReference{}, fmt.Errorf("oci digest reference is required")
	}
	if strings.Count(value, "@") != 1 {
		return DigestReference{}, fmt.Errorf("oci reference %q must contain exactly one digest separator", value)
	}
	parts := strings.SplitN(value, "@", 2)
	location := strings.TrimSpace(parts[0])
	digest := strings.TrimSpace(parts[1])
	if err := validateSHA256Digest(digest); err != nil {
		return DigestReference{}, fmt.Errorf("oci reference %q: %w", value, err)
	}
	path := strings.Split(location, "/")
	if len(path) < 2 {
		return DigestReference{}, fmt.Errorf("oci reference %q must include registry and repository", value)
	}
	registry := path[0]
	repository := strings.Join(path[1:], "/")
	if !registryPattern.MatchString(registry) {
		return DigestReference{}, fmt.Errorf("oci reference %q has invalid registry %q", value, registry)
	}
	if !repositoryPattern.MatchString(repository) {
		return DigestReference{}, fmt.Errorf("oci reference %q has invalid repository %q", value, repository)
	}
	return DigestReference{Registry: registry, Repository: repository, Digest: digest}, nil
}

// Descriptor represents the OCI fields required to validate the package layer.
type Descriptor struct {
	MediaType string
	Digest    string
	Size      int64
}

// Manifest is the OCI manifest projection used by the pull adapter. Digest is
// the digest computed from the received manifest bytes, not a package-layer digest.
type Manifest struct {
	Digest       string
	ArtifactType string
	Layers       []Descriptor
}

func validateEnginePackageManifest(manifest Manifest, expectedManifestDigest, expectedArtifactType, expectedLayerMediaType string) error {
	if err := validateSHA256Digest(expectedManifestDigest); err != nil {
		return fmt.Errorf("expected manifest digest: %w", err)
	}
	if manifest.Digest != expectedManifestDigest {
		return fmt.Errorf("oci manifest digest mismatch: got %q, want %q", manifest.Digest, expectedManifestDigest)
	}
	if manifest.ArtifactType != expectedArtifactType {
		return fmt.Errorf("unsupported OCI artifact type %q", manifest.ArtifactType)
	}
	if len(manifest.Layers) != 1 {
		return fmt.Errorf("lunafox engine OCI manifest must contain exactly one package layer, got %d", len(manifest.Layers))
	}
	layer := manifest.Layers[0]
	if layer.MediaType != expectedLayerMediaType {
		return fmt.Errorf("unsupported engine package layer media type %q", layer.MediaType)
	}
	if err := validateSHA256Digest(layer.Digest); err != nil {
		return fmt.Errorf("engine package layer digest: %w", err)
	}
	if layer.Size <= 0 {
		return fmt.Errorf("engine package layer size must be positive")
	}
	return nil
}

func validateSHA256Digest(value string) error {
	if !digestPattern.MatchString(value) {
		return fmt.Errorf("must be a lowercase sha256 digest")
	}
	return nil
}

const (
	// EnginePackageArtifactType is the active engine package artifact type.
	EnginePackageArtifactType = "application/vnd.lunafox.engine-package.v2"
	// EnginePackageLayerMediaType is the active engine package layer type.
	EnginePackageLayerMediaType = "application/vnd.lunafox.engine-package.layer.v2.tar+gzip"
)

// ArtifactManifestDigest identifies the exact OCI artifact manifest bytes.
// It is not the digest of the package archive layer or Engine Runtime Image.
type ArtifactManifestDigest string

// PackageDigest identifies the exact compressed Engine Package archive bytes.
// It is carried by the sole package-layer descriptor.
type PackageDigest string

// ParsePackageDigest accepts only the canonical lowercase SHA-256 identity of
// exact compressed Engine Package archive bytes.
func ParsePackageDigest(value string) (PackageDigest, error) {
	if err := validateSHA256Digest(value); err != nil {
		return "", fmt.Errorf("packageDigest: %w", err)
	}
	return PackageDigest(value), nil
}

// EnginePackageLayer is the one archive layer bound by the active OCI artifact.
type EnginePackageLayer struct {
	MediaType     string
	PackageDigest PackageDigest
	Size          int64
}

// EnginePackageManifest keeps the artifact-manifest and archive-byte digest
// roles explicit so callers cannot accidentally substitute one for the other.
type EnginePackageManifest struct {
	ArtifactManifestDigest ArtifactManifestDigest
	ArtifactType           string
	PackageLayer           EnginePackageLayer
}

// DecodeEnginePackageManifest strictly decodes and validates one active OCI
// manifest. It does not resolve references, fetch layers, or activate a release
// path; those responsibilities remain with the later installer cutover tasks.
func DecodeEnginePackageManifest(payload []byte, source string) (EnginePackageManifest, error) {
	decoder := json.NewDecoder(bytes.NewReader(payload))
	decoder.DisallowUnknownFields()

	var decoded ocispec.Manifest
	if err := decoder.Decode(&decoded); err != nil {
		return EnginePackageManifest{}, fmt.Errorf("decode OCI engine package v2 manifest %q: %w", source, err)
	}
	if err := consumeEnginePackageJSONEOF(decoder); err != nil {
		return EnginePackageManifest{}, fmt.Errorf("decode OCI engine package v2 manifest %q: %w", source, err)
	}
	if err := validateEnginePackageOCIEnvelope(decoded); err != nil {
		return EnginePackageManifest{}, fmt.Errorf("validate OCI engine package v2 envelope %q: %w", source, err)
	}

	manifestDigest := sha256.Sum256(payload)
	layer := decoded.Layers[0]
	manifest := EnginePackageManifest{
		ArtifactManifestDigest: ArtifactManifestDigest("sha256:" + hex.EncodeToString(manifestDigest[:])),
		ArtifactType:           decoded.ArtifactType,
		PackageLayer: EnginePackageLayer{
			MediaType:     layer.MediaType,
			PackageDigest: PackageDigest(layer.Digest.String()),
			Size:          layer.Size,
		},
	}
	if err := ValidateEnginePackageManifest(manifest, manifest.ArtifactManifestDigest); err != nil {
		return EnginePackageManifest{}, fmt.Errorf("validate OCI engine package v2 manifest %q: %w", source, err)
	}
	return manifest, nil
}

// ValidateEnginePackageManifest validates the active OCI identity and one-layer
// shape. It deliberately has no v1 compatibility fallback.
func ValidateEnginePackageManifest(manifest EnginePackageManifest, expectedManifestDigest ArtifactManifestDigest) error {
	return validateEnginePackageManifest(Manifest{
		Digest:       string(manifest.ArtifactManifestDigest),
		ArtifactType: manifest.ArtifactType,
		Layers: []Descriptor{{
			MediaType: manifest.PackageLayer.MediaType,
			Digest:    string(manifest.PackageLayer.PackageDigest),
			Size:      manifest.PackageLayer.Size,
		}},
	}, string(expectedManifestDigest), EnginePackageArtifactType, EnginePackageLayerMediaType)
}

func validateEnginePackageOCIEnvelope(manifest ocispec.Manifest) error {
	if manifest.SchemaVersion != 2 {
		return fmt.Errorf("oci manifest schemaVersion must be 2, got %d", manifest.SchemaVersion)
	}
	if manifest.MediaType != ocispec.MediaTypeImageManifest {
		return fmt.Errorf("oci manifest media type must be %q, got %q", ocispec.MediaTypeImageManifest, manifest.MediaType)
	}
	emptyConfig := ocispec.DescriptorEmptyJSON
	if manifest.Config.MediaType != emptyConfig.MediaType {
		return fmt.Errorf("oci manifest config media type must be %q, got %q", emptyConfig.MediaType, manifest.Config.MediaType)
	}
	if manifest.Config.Digest != emptyConfig.Digest {
		return fmt.Errorf("oci manifest config digest must be %q, got %q", emptyConfig.Digest, manifest.Config.Digest)
	}
	if manifest.Config.Size != emptyConfig.Size {
		return fmt.Errorf("oci manifest config size must be %d, got %d", emptyConfig.Size, manifest.Config.Size)
	}
	if len(manifest.Layers) != 1 {
		return fmt.Errorf("oci engine package v2 manifest must contain exactly one package layer, got %d", len(manifest.Layers))
	}
	return nil
}

func consumeEnginePackageJSONEOF(decoder *json.Decoder) error {
	if decoder == nil {
		return nil
	}
	if _, err := decoder.Token(); err != nil {
		if err == io.EOF {
			return nil
		}
		return err
	}
	return fmt.Errorf("unexpected trailing JSON content")
}
