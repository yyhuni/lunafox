package ociartifact

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/opencontainers/go-digest"
	"github.com/opencontainers/image-spec/specs-go"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
)

const testDigest = "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"

func TestEnginePackageMediaTypesRemainVersioned(t *testing.T) {
	if EnginePackageArtifactType != "application/vnd.lunafox.engine-package.v2" {
		t.Fatalf("v2 artifact type = %q", EnginePackageArtifactType)
	}
	if EnginePackageLayerMediaType != "application/vnd.lunafox.engine-package.layer.v2.tar+gzip" {
		t.Fatalf("v2 layer media type = %q", EnginePackageLayerMediaType)
	}
}

func TestParseDigestReferenceRejectsTagOnlyReference(t *testing.T) {
	if _, err := ParseDigestReference("ghcr.io/yyhuni/lunafox-engine-subdomain-discovery:latest"); err == nil {
		t.Fatal("expected tag-only reference to be rejected")
	}
}

func TestParseDigestReferenceParsesCanonicalSHA256Reference(t *testing.T) {
	ref, err := ParseDigestReference("ghcr.io/yyhuni/lunafox-engine-subdomain-discovery@" + testDigest)
	if err != nil {
		t.Fatalf("ParseDigestReference() error = %v", err)
	}
	if ref.Registry != "ghcr.io" || ref.Repository != "yyhuni/lunafox-engine-subdomain-discovery" || ref.Digest != testDigest {
		t.Fatalf("unexpected parsed reference: %#v", ref)
	}
}

func TestValidateEnginePackageManifestAcceptsExactMediaTypes(t *testing.T) {
	err := ValidateEnginePackageManifest(EnginePackageManifest{
		ArtifactManifestDigest: ArtifactManifestDigest(testDigest),
		ArtifactType:           EnginePackageArtifactType,
		PackageLayer: EnginePackageLayer{
			MediaType:     EnginePackageLayerMediaType,
			PackageDigest: PackageDigest("sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc"),
			Size:          1,
		},
	}, ArtifactManifestDigest(testDigest))
	if err != nil {
		t.Fatalf("ValidateEnginePackageManifest() error = %v", err)
	}
}

func TestValidateEnginePackageManifestRejectsArtifactType(t *testing.T) {
	err := ValidateEnginePackageManifest(EnginePackageManifest{
		ArtifactManifestDigest: ArtifactManifestDigest(testDigest),
		ArtifactType:           "application/vnd.lunafox.engine-package.v1",
		PackageLayer: EnginePackageLayer{
			MediaType:     EnginePackageLayerMediaType,
			PackageDigest: PackageDigest(testDigest),
			Size:          1,
		},
	}, ArtifactManifestDigest(testDigest))
	if err == nil || !strings.Contains(err.Error(), "unsupported OCI artifact type") {
		t.Fatalf("expected v1 artifact type rejection, got %v", err)
	}
}

func TestValidateEnginePackageManifestRejectsLayerMediaType(t *testing.T) {
	err := ValidateEnginePackageManifest(EnginePackageManifest{
		ArtifactManifestDigest: ArtifactManifestDigest(testDigest),
		ArtifactType:           EnginePackageArtifactType,
		PackageLayer: EnginePackageLayer{
			MediaType:     "application/vnd.lunafox.engine-package.layer.v1.tar+gzip",
			PackageDigest: PackageDigest(testDigest),
			Size:          1,
		},
	}, ArtifactManifestDigest(testDigest))
	if err == nil || !strings.Contains(err.Error(), "unsupported engine package layer media type") {
		t.Fatalf("expected v1 layer media type rejection, got %v", err)
	}
}

func TestDecodeEnginePackageManifestRejectsUnknownField(t *testing.T) {
	payload := mustMarshalEnginePackageManifest(t)
	payload = append(payload[:len(payload)-1], []byte(`,"fallbackArtifactType":"application/vnd.lunafox.engine-package.v1"}`)...)

	if _, err := DecodeEnginePackageManifest(payload, "fixture"); err == nil || !strings.Contains(err.Error(), "unknown field") {
		t.Fatalf("expected unknown field rejection, got %v", err)
	}
}

func TestDecodeEnginePackageManifestRejectsTrailingJSON(t *testing.T) {
	payload := append(mustMarshalEnginePackageManifest(t), []byte(` {}`)...)

	if _, err := DecodeEnginePackageManifest(payload, "fixture"); err == nil || !strings.Contains(err.Error(), "unexpected trailing JSON content") {
		t.Fatalf("expected trailing JSON rejection, got %v", err)
	}
}

func TestDecodeEnginePackageManifestRejectsInvalidOCIEnvelope(t *testing.T) {
	tests := []struct {
		name      string
		mutate    func(*ocispec.Manifest)
		wantError string
	}{
		{
			name: "wrong schema version",
			mutate: func(manifest *ocispec.Manifest) {
				manifest.SchemaVersion = 1
			},
			wantError: "schemaVersion must be 2",
		},
		{
			name: "missing root manifest media type",
			mutate: func(manifest *ocispec.Manifest) {
				manifest.MediaType = ""
			},
			wantError: "oci manifest media type must be",
		},
		{
			name: "wrong manifest media type",
			mutate: func(manifest *ocispec.Manifest) {
				manifest.MediaType = ocispec.MediaTypeImageIndex
			},
			wantError: "oci manifest media type must be",
		},
		{
			name: "legacy image config media type",
			mutate: func(manifest *ocispec.Manifest) {
				manifest.Config.MediaType = ocispec.MediaTypeImageConfig
			},
			wantError: "oci manifest config media type must be",
		},
		{
			name: "missing config descriptor",
			mutate: func(manifest *ocispec.Manifest) {
				manifest.Config = ocispec.Descriptor{}
			},
			wantError: "oci manifest config media type must be",
		},
		{
			name: "non-canonical empty config digest",
			mutate: func(manifest *ocispec.Manifest) {
				manifest.Config.Digest = digest.Digest(testDigest)
			},
			wantError: "oci manifest config digest must be",
		},
		{
			name: "non-canonical empty config size",
			mutate: func(manifest *ocispec.Manifest) {
				manifest.Config.Size = ocispec.DescriptorEmptyJSON.Size + 1
			},
			wantError: "oci manifest config size must be",
		},
		{
			name: "zero package layers",
			mutate: func(manifest *ocispec.Manifest) {
				manifest.Layers = nil
			},
			wantError: "exactly one package layer",
		},
		{
			name: "multiple package layers",
			mutate: func(manifest *ocispec.Manifest) {
				manifest.Layers = append(manifest.Layers, manifest.Layers[0])
			},
			wantError: "exactly one package layer",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			payload := mustMarshalEnginePackageManifestWith(t, tt.mutate)
			if _, err := DecodeEnginePackageManifest(payload, "fixture"); err == nil || !strings.Contains(err.Error(), tt.wantError) {
				t.Fatalf("expected invalid OCI envelope error containing %q, got %v", tt.wantError, err)
			}
		})
	}
}

func TestDecodeEnginePackageManifestAcceptsCanonicalOCIEnvelope(t *testing.T) {
	payload := mustMarshalEnginePackageManifest(t)
	expectedDigest := digest.FromBytes(payload).String()

	manifest, err := DecodeEnginePackageManifest(payload, "fixture")
	if err != nil {
		t.Fatalf("DecodeEnginePackageManifest() error = %v", err)
	}
	if manifest.ArtifactManifestDigest != ArtifactManifestDigest(expectedDigest) {
		t.Fatalf("artifact manifest digest = %q, want %q", manifest.ArtifactManifestDigest, expectedDigest)
	}
	if manifest.ArtifactType != EnginePackageArtifactType {
		t.Fatalf("artifact type = %q, want %q", manifest.ArtifactType, EnginePackageArtifactType)
	}
	if manifest.PackageLayer.MediaType != EnginePackageLayerMediaType {
		t.Fatalf("layer media type = %q, want %q", manifest.PackageLayer.MediaType, EnginePackageLayerMediaType)
	}
}

func TestDecodeEnginePackageManifestKeepsDigestRolesDistinct(t *testing.T) {
	payload := mustMarshalEnginePackageManifest(t)
	wantArtifactDigest := ArtifactManifestDigest(digest.FromBytes(payload).String())

	manifest, err := DecodeEnginePackageManifest(payload, "fixture")
	if err != nil {
		t.Fatalf("DecodeEnginePackageManifest() error = %v", err)
	}
	if manifest.ArtifactManifestDigest != wantArtifactDigest {
		t.Fatalf("artifactManifestDigest = %q, want %q", manifest.ArtifactManifestDigest, wantArtifactDigest)
	}
	if manifest.PackageLayer.PackageDigest != PackageDigest(testDigest) {
		t.Fatalf("packageDigest = %q, want %q", manifest.PackageLayer.PackageDigest, testDigest)
	}
	if string(manifest.ArtifactManifestDigest) == string(manifest.PackageLayer.PackageDigest) {
		t.Fatal("artifact manifest and package digests must remain separate roles")
	}
}

func TestValidateEnginePackageManifestRejectsInvalidShape(t *testing.T) {
	tests := []struct {
		name  string
		layer EnginePackageLayer
	}{
		{name: "missing layer"},
		{
			name:  "invalid layer digest",
			layer: EnginePackageLayer{MediaType: EnginePackageLayerMediaType, PackageDigest: PackageDigest("sha256:not-a-digest"), Size: 1},
		},
		{
			name:  "non-positive layer size",
			layer: EnginePackageLayer{MediaType: EnginePackageLayerMediaType, PackageDigest: PackageDigest(testDigest), Size: 0},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateEnginePackageManifest(EnginePackageManifest{
				ArtifactManifestDigest: ArtifactManifestDigest(testDigest),
				ArtifactType:           EnginePackageArtifactType,
				PackageLayer:           tt.layer,
			}, ArtifactManifestDigest(testDigest))
			if err == nil {
				t.Fatal("expected invalid v2 manifest shape to be rejected")
			}
		})
	}
}

func mustMarshalEnginePackageManifest(t *testing.T) []byte {
	t.Helper()
	return mustMarshalEnginePackageManifestWith(t, nil)
}

func mustMarshalEnginePackageManifestWith(t *testing.T, mutate func(*ocispec.Manifest)) []byte {
	t.Helper()
	layerDigest := digest.Digest(testDigest)
	emptyConfig := ocispec.DescriptorEmptyJSON
	manifest := ocispec.Manifest{
		Versioned:    specs.Versioned{SchemaVersion: 2},
		MediaType:    ocispec.MediaTypeImageManifest,
		ArtifactType: EnginePackageArtifactType,
		Config: ocispec.Descriptor{
			MediaType: emptyConfig.MediaType,
			Digest:    emptyConfig.Digest,
			Size:      emptyConfig.Size,
		},
		Layers: []ocispec.Descriptor{{
			MediaType: EnginePackageLayerMediaType,
			Digest:    layerDigest,
			Size:      1,
		}},
	}
	if mutate != nil {
		mutate(&manifest)
	}
	payload, err := json.Marshal(manifest)
	if err != nil {
		t.Fatalf("marshal OCI manifest fixture: %v", err)
	}
	return payload
}
