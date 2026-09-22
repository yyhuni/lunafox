package ocisignature

import (
	"errors"
	"strings"
	"testing"

	sigstorebundle "github.com/sigstore/sigstore-go/pkg/bundle"
	"github.com/yyhuni/lunafox/contracts/ociartifact"
)

func TestNewKeylessVerifierRejectsInvalidPolicyBeforeNetworkAccess(t *testing.T) {
	if _, err := NewKeylessVerifier(KeylessPolicy{Repository: "invalid", Workflow: ".github/workflows/release.yml"}, nil); err == nil {
		t.Fatal("expected invalid repository policy to fail")
	}
	if _, err := NewKeylessVerifier(KeylessPolicy{Repository: "yyhuni/lunafox", Workflow: "release.yml"}, nil); err == nil {
		t.Fatal("expected invalid workflow policy to fail")
	}
	if _, err := NewKeylessVerifier(KeylessPolicy{Repository: "yyhuni/lunafox", Workflow: ".github/workflows/release.yml"}, nil); err == nil {
		t.Fatal("expected missing trust root to fail")
	}
	if _, err := NewKeylessVerifier(KeylessPolicy{Repository: "yyhuni/lunafox", Workflow: ".github/workflows/release.yml", RefPattern: "refs/heads/dev"}, nil); err == nil {
		t.Fatal("expected arbitrary ref policy to fail")
	}
}

func TestTransportErrorIsDistinctFromIntegrityError(t *testing.T) {
	if !IsTransportError(NewTransportError(errors.New("registry unavailable"))) {
		t.Fatal("expected transport error classification")
	}
	if IsTransportError(errors.New("signature invalid")) {
		t.Fatal("signature integrity error must not be transport classified")
	}
}

func TestBundleMediaTypesMatchSigstorePublisher(t *testing.T) {
	canonical, err := sigstorebundle.MediaTypeString("0.3")
	if err != nil {
		t.Fatal(err)
	}
	for role, actual := range map[string]string{"artifact": BundleArtifactType, "layer": BundleLayerType} {
		if actual != canonical {
			t.Errorf("%s media type %q must match Sigstore publisher %q", role, actual, canonical)
		}
	}
}

func TestValidateSignatureReferencePairRequiresCanonicalMatchingRepositoryAndDigest(t *testing.T) {
	const digest = "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	identity := ociartifact.DigestReference{Registry: "ghcr.io", Repository: "yyhuni/lunafox-agent", Digest: digest}

	tests := []struct {
		name      string
		transport ociartifact.DigestReference
		want      string
	}{
		{
			name:      "different repository",
			transport: ociartifact.DigestReference{Registry: "docker.lunafox.cc.cd", Repository: "yyhuni/lunafox-server", Digest: digest},
			want:      "same repository",
		},
		{
			name:      "different digest",
			transport: ociartifact.DigestReference{Registry: "docker.lunafox.cc.cd", Repository: identity.Repository, Digest: "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"},
			want:      "same digest",
		},
		{
			name:      "malformed digest",
			transport: ociartifact.DigestReference{Registry: "docker.lunafox.cc.cd", Repository: identity.Repository, Digest: "sha256:ABC"},
			want:      "transport reference is invalid",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := validateSignatureReferencePair(identity, test.transport)
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("validateSignatureReferencePair() error = %v, want %q", err, test.want)
			}
		})
	}

	if err := validateSignatureReferencePair(identity, ociartifact.DigestReference{
		Registry: "docker.lunafox.cc.cd", Repository: identity.Repository, Digest: identity.Digest,
	}); err != nil {
		t.Fatalf("matching identity/transport pair rejected: %v", err)
	}
}
