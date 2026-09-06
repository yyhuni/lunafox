package ociartifact

import (
	"reflect"
	"strings"
	"testing"
)

const artifactReferenceTestDigestB = "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"

func TestParseArtifactManifestDigestAcceptsCanonicalSHA256(t *testing.T) {
	digest, err := ParseArtifactManifestDigest(testDigest)
	if err != nil {
		t.Fatalf("ParseArtifactManifestDigest() error = %v", err)
	}
	if digest != ArtifactManifestDigest(testDigest) {
		t.Fatalf("artifactManifestDigest = %q, want %q", digest, testDigest)
	}
}

func TestParseArtifactManifestDigestRejectsNonCanonicalValues(t *testing.T) {
	for _, value := range []string{
		"",
		" " + testDigest,
		testDigest + " ",
		"sha256:AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA",
		"sha256:abc",
		"sha512:" + strings.Repeat("a", 128),
	} {
		t.Run(value, func(t *testing.T) {
			if _, err := ParseArtifactManifestDigest(value); err == nil {
				t.Fatalf("ParseArtifactManifestDigest(%q) succeeded, want rejection", value)
			}
		})
	}
}

func TestParseEnginePackageArtifactReferencePreservesCanonicalIdentity(t *testing.T) {
	value := "ghcr.io/yyhuni/lunafox-engine-port-scan@" + testDigest

	reference, err := ParseEnginePackageArtifactReference(value)
	if err != nil {
		t.Fatalf("ParseEnginePackageArtifactReference() error = %v", err)
	}
	if reference.Registry != "ghcr.io" || reference.Repository != "yyhuni/lunafox-engine-port-scan" {
		t.Fatalf("unexpected artifact reference location: %#v", reference)
	}
	if reference.ArtifactManifestDigest != ArtifactManifestDigest(testDigest) || reference.String() != value {
		t.Fatalf("unexpected artifact reference identity: %#v", reference)
	}
	wantProjection := DigestReference{Registry: reference.Registry, Repository: reference.Repository, Digest: testDigest}
	if got := reference.DigestReference(); got != wantProjection {
		t.Fatalf("DigestReference() = %#v, want %#v", got, wantProjection)
	}
}

func TestParseEnginePackageArtifactReferenceRejectsNonCanonicalReferences(t *testing.T) {
	valid := "ghcr.io/yyhuni/lunafox-engine-port-scan@" + testDigest
	tests := []struct {
		name  string
		value string
	}{
		{name: "empty"},
		{name: "leading whitespace", value: " " + valid},
		{name: "trailing whitespace", value: valid + " "},
		{name: "separator whitespace", value: "ghcr.io/yyhuni/lunafox-engine-port-scan @ " + testDigest},
		{name: "tag only", value: "ghcr.io/yyhuni/lunafox-engine-port-scan:latest"},
		{name: "tag plus digest", value: "ghcr.io/yyhuni/lunafox-engine-port-scan:latest@" + testDigest},
		{name: "short digest", value: "ghcr.io/yyhuni/lunafox-engine-port-scan@sha256:abc"},
		{name: "non sha256", value: "ghcr.io/yyhuni/lunafox-engine-port-scan@sha512:" + strings.Repeat("a", 128)},
		{name: "uppercase digest", value: "ghcr.io/yyhuni/lunafox-engine-port-scan@sha256:" + strings.Repeat("A", 64)},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, err := ParseEnginePackageArtifactReference(test.value); err == nil {
				t.Fatalf("ParseEnginePackageArtifactReference(%q) succeeded, want rejection", test.value)
			}
		})
	}
}

func TestParseArtifactCandidatesPreservesOrderAndSharedDigest(t *testing.T) {
	values := []string{
		"docker.io/yyhuni/lunafox-engine-port-scan@" + testDigest,
		"ghcr.io/yyhuni/lunafox-engine-port-scan@" + testDigest,
	}

	candidates, err := ParseArtifactCandidates(values)
	if err != nil {
		t.Fatalf("ParseArtifactCandidates() error = %v", err)
	}
	if candidates.ArtifactManifestDigest != ArtifactManifestDigest(testDigest) {
		t.Fatalf("artifactManifestDigest = %q, want %q", candidates.ArtifactManifestDigest, testDigest)
	}
	got := make([]string, 0, len(candidates.References))
	for _, reference := range candidates.References {
		got = append(got, reference.String())
	}
	if !reflect.DeepEqual(got, values) {
		t.Fatalf("candidate order = %#v, want %#v", got, values)
	}
}

func TestParseArtifactCandidatesRejectsInvalidCandidateGroups(t *testing.T) {
	first := "docker.io/yyhuni/lunafox-engine-port-scan@" + testDigest
	tests := []struct {
		name   string
		values []string
		want   string
	}{
		{name: "empty", want: "candidates are required"},
		{name: "non canonical candidate", values: []string{" " + first}, want: "canonical OCI digest reference"},
		{name: "duplicate location", values: []string{first, first}, want: "duplicate"},
		{
			name: "digest drift",
			values: []string{
				first,
				"ghcr.io/yyhuni/lunafox-engine-port-scan@" + artifactReferenceTestDigestB,
			},
			want: "same OCI artifact manifest digest",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := ParseArtifactCandidates(test.values)
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("ParseArtifactCandidates() error = %v, want error containing %q", err, test.want)
			}
		})
	}
}
