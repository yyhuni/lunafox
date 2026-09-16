package ocisignature

import (
	"errors"
	"testing"

	sigstorebundle "github.com/sigstore/sigstore-go/pkg/bundle"
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
