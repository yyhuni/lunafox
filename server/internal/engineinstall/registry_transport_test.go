package engineinstall

import (
	"context"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/yyhuni/lunafox/contracts/ociartifact"
)

func TestPackageMirrorTransportRetainsCanonicalArtifactIdentity(t *testing.T) {
	artifact := newPackageArtifactFixture(t, []byte("package archive"), ociartifact.EnginePackageArtifactType, ociartifact.EnginePackageLayerMediaType)
	puller := newTestORASPackagePuller(t, map[string]*fakePackageRepository{"engine.company.cn": artifact.repository()})
	puller.options.TransportRegistry = "engine.company.cn"
	identity := "ghcr.io/lunafox/engine@" + artifact.manifestDigest.String()
	result, err := puller.ConsumePackageLayer(context.Background(), parsePackageCandidates(t, identity), func(_ context.Context, layer PulledPackageLayer) error {
		if layer.Reference.String() != identity {
			t.Fatal(layer.Reference.String())
		}
		_, err := io.Copy(io.Discard, layer.Reader)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Reference.String() != identity {
		t.Fatal(result.Reference.String())
	}
}

func TestRuntimeMirrorTransportRetainsCanonicalIdentity(t *testing.T) {
	payload := mustMarshalRuntimeImageIndex(t, nil)
	fixture := newRuntimeImageRegistryFixture(t, runtimeImageRegistryResponse{payload: payload})
	defer fixture.Close()
	verifier := mustNewRuntimeImageIndexVerifier(t, &runtimeImageSignatureVerifierStub{}, time.Second, int64(len(payload)), fixture.registry())
	verifier.options.TransportRegistry = fixture.registry()
	verifier.newRepository = func(reference ociartifact.DigestReference, transport string, _ bool) (runtimeImageOCIRepository, error) {
		return newRuntimeImageRepository(reference, transport, true)
	}
	identity := fixture.reference(payload)
	parsed, err := ociartifact.ParseDigestReference(identity)
	if err != nil {
		t.Fatal(err)
	}
	parsed.Registry = "ghcr.io"
	verified, err := verifier.Verify(context.Background(), "", []string{parsed.String()})
	if err != nil {
		t.Fatal(err)
	}
	if verified.Reference != parsed {
		t.Fatal(verified.Reference)
	}
	if fixture.requests.Load() != 2 {
		t.Fatal(fixture.requests.Load())
	}
}

func TestSignatureMirrorTransportRetainsSignedIdentity(t *testing.T) {
	delegate := &recordingDigestReferenceSignatureVerifier{}
	verifier, err := WithSignatureRegistry(delegate, "engine.company.cn")
	if err != nil {
		t.Fatal(err)
	}
	identity, err := ociartifact.ParseDigestReference("ghcr.io/yyhuni/lunafox-engine@" + inventoryDigestA)
	if err != nil {
		t.Fatal(err)
	}
	if err := verifier.VerifyReference(context.Background(), identity); err != nil {
		t.Fatal(err)
	}
	if len(delegate.references) != 1 || delegate.references[0] != identity {
		t.Fatal(delegate.references)
	}
	transport := delegate.transportReferences[0]
	if transport.Registry != "engine.company.cn" || transport.Repository != identity.Repository || transport.Digest != identity.Digest {
		t.Fatal(transport)
	}
	if _, err := WithSignatureRegistry(&legacyDigestReferenceSignatureVerifier{}, "engine.company.cn"); err == nil {
		t.Fatal("non-transport verifier accepted")
	}
}

func TestSignatureSelectedCanonicalRegistryRetainsNativeReferrerDiscovery(t *testing.T) {
	delegate := &recordingDigestReferenceSignatureVerifier{}
	verifier, err := WithSignatureRegistry(delegate, "docker.io")
	if err != nil {
		t.Fatal(err)
	}
	identity, err := ociartifact.ParseDigestReference("docker.io/yyhuni/lunafox-engine@" + inventoryDigestA)
	if err != nil {
		t.Fatal(err)
	}
	if err := verifier.VerifyReference(context.Background(), identity); err != nil {
		t.Fatal(err)
	}
	if len(delegate.references) != 1 || len(delegate.transportReferences) != 0 {
		t.Fatal("native signature discovery changed")
	}
}

func TestPackageMirrorStillRejectsTamperedLayerWithoutOfficialFallback(t *testing.T) {
	artifact := newPackageArtifactFixture(t, []byte("expected package"), ociartifact.EnginePackageArtifactType, ociartifact.EnginePackageLayerMediaType)
	mirror := artifact.repository()
	mirror.layerReader = func() io.ReadCloser { return io.NopCloser(strings.NewReader("tampered package")) }
	puller := newTestORASPackagePuller(t, map[string]*fakePackageRepository{"engine.company.cn": mirror})
	puller.options.TransportRegistry = "engine.company.cn"
	candidates := parsePackageCandidates(t, "docker.io/lunafox/engine@"+artifact.manifestDigest.String(), "ghcr.io/lunafox/engine@"+artifact.manifestDigest.String())
	_, err := puller.ConsumePackageLayer(context.Background(), candidates, readAllPackageLayer)
	if err == nil || !strings.Contains(err.Error(), "layer digest mismatch") {
		t.Fatalf("tampered mirror: %v", err)
	}
	if mirror.resolveCalls != 1 {
		t.Fatal("integrity failure advanced to another candidate", mirror.resolveCalls)
	}
}
