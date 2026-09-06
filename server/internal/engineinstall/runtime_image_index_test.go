package engineinstall

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strconv"
	"strings"
	"sync/atomic"
	"syscall"
	"testing"
	"time"

	"github.com/opencontainers/go-digest"
	"github.com/opencontainers/image-spec/specs-go"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/yyhuni/lunafox/contracts/enginemanifest/repositoryname"
	"github.com/yyhuni/lunafox/contracts/enginemanifest/runtimeimage"
	"github.com/yyhuni/lunafox/contracts/ociartifact"
	"oras.land/oras-go/v2/registry/remote"
	"oras.land/oras-go/v2/registry/remote/errcode"
)

const runtimeImageRepository = "yyhuni/lunafox-engine-runtime-subdomain-discovery"

func TestNewRuntimeImageIndexVerifierRequiresBoundedProcessingPolicy(t *testing.T) {
	valid := RuntimeImageIndexVerifierOptions{
		MaxIndexBytes:       1,
		PerCandidateTimeout: time.Second,
	}
	tests := []struct {
		name    string
		options RuntimeImageIndexVerifierOptions
	}{
		{name: "missing maximum size", options: withRuntimeImageVerifierOption(valid, func(options *RuntimeImageIndexVerifierOptions) {
			options.MaxIndexBytes = 0
		})},
		{name: "missing attempt timeout", options: withRuntimeImageVerifierOption(valid, func(options *RuntimeImageIndexVerifierOptions) {
			options.PerCandidateTimeout = 0
		})},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, err := NewRuntimeImageIndexVerifier(test.options); err == nil {
				t.Fatal("expected invalid verifier policy to fail")
			}
		})
	}

	_, err := NewRuntimeImageIndexVerifier(valid)
	if err != nil {
		t.Fatalf("NewRuntimeImageIndexVerifier() error = %v", err)
	}
}

func TestNewRuntimeImageIndexVerifierRequiresCompleteDevelopmentTransport(t *testing.T) {
	transport, err := NewDevelopmentRuntimeImageRegistryTransport("localhost:5000", "registry:5000")
	if err != nil {
		t.Fatal(err)
	}
	valid := RuntimeImageIndexVerifierOptions{
		MaxIndexBytes:                  1,
		PerCandidateTimeout:            time.Second,
		AllowPlainHTTP:                 true,
		DevelopmentRegistryTransport:   transport,
		AllowDevelopmentSinglePlatform: true,
	}
	if _, err := NewRuntimeImageIndexVerifier(valid); err != nil {
		t.Fatalf("complete development transport error = %v", err)
	}

	for _, test := range []struct {
		name   string
		mutate func(*RuntimeImageIndexVerifierOptions)
	}{
		{name: "missing plain HTTP", mutate: func(options *RuntimeImageIndexVerifierOptions) { options.AllowPlainHTTP = false }},
		{name: "missing Registry mapping", mutate: func(options *RuntimeImageIndexVerifierOptions) { options.DevelopmentRegistryTransport = nil }},
		{name: "missing single-platform policy", mutate: func(options *RuntimeImageIndexVerifierOptions) { options.AllowDevelopmentSinglePlatform = false }},
	} {
		t.Run(test.name, func(t *testing.T) {
			options := withRuntimeImageVerifierOption(valid, test.mutate)
			if _, err := NewRuntimeImageIndexVerifier(options); err == nil || !strings.Contains(err.Error(), "development transport requires") {
				t.Fatalf("NewRuntimeImageIndexVerifier() error = %v", err)
			}
		})
	}
}

func TestNewRuntimeImageIndexVerifierCloudflareAccelerationRequiresSignatureVerifier(t *testing.T) {
	options := RuntimeImageIndexVerifierOptions{
		MaxIndexBytes:          1024,
		PerCandidateTimeout:    time.Second,
		CloudflareAcceleration: true,
	}
	if _, err := NewRuntimeImageIndexVerifier(options); err == nil || !strings.Contains(err.Error(), "requires a GHCR signature verifier") {
		t.Fatalf("missing Cloudflare signature verifier error = %v", err)
	}

	options.SignatureVerifier = &recordingDigestReferenceSignatureVerifier{}
	if _, err := NewRuntimeImageIndexVerifier(options); err != nil {
		t.Fatalf("Cloudflare acceleration verifier initialization error = %v", err)
	}
}

func TestRuntimeImageIndexVerifierCloudflareAccelerationVerifiesGHCRAndClassifiesCFFailures(t *testing.T) {
	payload := mustMarshalRuntimeImageIndex(t, nil)
	digestValue := digest.FromBytes(payload).String()
	dockerHub := "docker.io/yyhuni/lunafox-engine-runtime-subdomain-discovery@" + digestValue
	ghcr := "ghcr.io/yyhuni/lunafox-engine-runtime-subdomain-discovery@" + digestValue
	cloudflare := "docker.lunafox.cc.cd/yyhuni/lunafox-engine-runtime-subdomain-discovery@" + digestValue
	descriptor := ocispec.Descriptor{MediaType: ocispec.MediaTypeImageIndex, Digest: digest.Digest(digestValue), Size: int64(len(payload))}

	t.Run("transport failure advances to Docker Hub after GHCR verification", func(t *testing.T) {
		signatureVerifier := &recordingDigestReferenceSignatureVerifier{}
		verifier, err := NewRuntimeImageIndexVerifier(RuntimeImageIndexVerifierOptions{
			MaxIndexBytes:          int64(len(payload)),
			PerCandidateTimeout:    time.Second,
			CloudflareAcceleration: true,
			SignatureVerifier:      signatureVerifier,
		})
		if err != nil {
			t.Fatal(err)
		}
		var attempted []string
		verifier.newRepository = func(reference ociartifact.DigestReference, _ string, _ bool) (runtimeImageOCIRepository, error) {
			attempted = append(attempted, reference.String())
			if reference.Registry == "docker.lunafox.cc.cd" {
				return runtimeImageOCIRepositoryStub{resolveErr: ociartifact.NewCandidateFailure(ociartifact.CandidateFailureReasonTCP, errors.New("connection refused"))}, nil
			}
			return runtimeImageOCIRepositoryStub{descriptor: descriptor, payload: payload}, nil
		}

		verified, err := verifier.Verify(context.Background(), repositoryname.FirstPartyEngineIDSubdomainDiscovery, []string{dockerHub, ghcr})
		if err != nil {
			t.Fatalf("Verify() error = %v", err)
		}
		if verified.Reference.String() != dockerHub {
			t.Fatalf("selected candidate = %q, want Docker Hub fallback %q", verified.Reference.String(), dockerHub)
		}
		if len(signatureVerifier.references) != 1 || signatureVerifier.references[0].String() != ghcr {
			t.Fatalf("signature references = %#v, want original GHCR %q", signatureVerifier.references, ghcr)
		}
		if want := []string{cloudflare, dockerHub}; !reflect.DeepEqual(attempted, want) {
			t.Fatalf("download attempts = %#v, want %#v", attempted, want)
		}
	})

	t.Run("policy failure does not advance after GHCR verification", func(t *testing.T) {
		signatureVerifier := &recordingDigestReferenceSignatureVerifier{}
		verifier, err := NewRuntimeImageIndexVerifier(RuntimeImageIndexVerifierOptions{
			MaxIndexBytes:          int64(len(payload)),
			PerCandidateTimeout:    time.Second,
			CloudflareAcceleration: true,
			SignatureVerifier:      signatureVerifier,
		})
		if err != nil {
			t.Fatal(err)
		}
		var attempted []string
		verifier.newRepository = func(reference ociartifact.DigestReference, _ string, _ bool) (runtimeImageOCIRepository, error) {
			attempted = append(attempted, reference.String())
			return runtimeImageOCIRepositoryStub{resolveErr: ociartifact.NewCandidateFailure(ociartifact.CandidateFailureReasonForbidden, errors.New("forbidden"))}, nil
		}

		_, err = verifier.Verify(context.Background(), repositoryname.FirstPartyEngineIDSubdomainDiscovery, []string{dockerHub, ghcr})
		if err == nil {
			t.Fatal("policy failure unexpectedly succeeded")
		}
		if want := []string{cloudflare}; !reflect.DeepEqual(attempted, want) {
			t.Fatalf("policy failure must not fall back: attempts=%#v want=%#v", attempted, want)
		}
	})
}

func TestNewDevelopmentRuntimeImageRegistryTransportRejectsInvalidAuthorities(t *testing.T) {
	for _, test := range []struct {
		name      string
		identity  string
		transport string
		want      string
	}{
		{name: "identity scheme", identity: "http://localhost:5000", transport: "registry:5000", want: "identity Registry"},
		{name: "identity whitespace", identity: " localhost:5000", transport: "registry:5000", want: "identity Registry"},
		{name: "transport path", identity: "localhost:5000", transport: "registry:5000/path", want: "transport Registry"},
	} {
		t.Run(test.name, func(t *testing.T) {
			_, err := NewDevelopmentRuntimeImageRegistryTransport(test.identity, test.transport)
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("transport error = %v, want %q", err, test.want)
			}
		})
	}
}

func TestRuntimeImageIndexVerifierUsesDevelopmentTransportAndPreservesIdentity(t *testing.T) {
	payload := mustMarshalRuntimeImageIndex(t, func(index *ocispec.Index) {
		index.Manifests = index.Manifests[:1]
	})
	fixture := newRuntimeImageRegistryFixture(t, runtimeImageRegistryResponse{payload: payload})
	defer fixture.Close()
	transport, err := NewDevelopmentRuntimeImageRegistryTransport("localhost:5000", fixture.registry())
	if err != nil {
		t.Fatal(err)
	}
	verifier, err := NewRuntimeImageIndexVerifier(RuntimeImageIndexVerifierOptions{
		MaxIndexBytes:                  int64(len(payload)),
		PerCandidateTimeout:            time.Second,
		AllowPlainHTTP:                 true,
		DevelopmentRegistryTransport:   transport,
		AllowDevelopmentSinglePlatform: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	identityRef := "localhost:5000/" + runtimeImageRepository + "@" + digest.FromBytes(payload).String()

	verified, err := verifier.Verify(context.Background(), repositoryname.FirstPartyEngineIDSubdomainDiscovery, []string{identityRef})
	if err != nil {
		t.Fatalf("Verify() development error = %v", err)
	}
	if verified.Reference.String() != identityRef {
		t.Fatalf("verified identity = %q, want %q", verified.Reference.String(), identityRef)
	}
	if fixture.requests.Load() != 2 {
		t.Fatalf("development transport requests = %d, want 2", fixture.requests.Load())
	}
}

func TestRuntimeImageIndexVerifierRejectsUnmappedDevelopmentIdentity(t *testing.T) {
	transport, err := NewDevelopmentRuntimeImageRegistryTransport("localhost:5000", "registry:5000")
	if err != nil {
		t.Fatal(err)
	}
	verifier, err := NewRuntimeImageIndexVerifier(RuntimeImageIndexVerifierOptions{
		MaxIndexBytes:                  1024,
		PerCandidateTimeout:            time.Second,
		AllowPlainHTTP:                 true,
		DevelopmentRegistryTransport:   transport,
		AllowDevelopmentSinglePlatform: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	ref := "other.example/" + runtimeImageRepository + "@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"

	_, err = verifier.Verify(context.Background(), repositoryname.FirstPartyEngineIDSubdomainDiscovery, []string{ref})
	if err == nil || !strings.Contains(err.Error(), "does not match configured development identity Registry") || ociartifact.IsCandidateUnavailable(err) {
		t.Fatalf("Verify() error = %v, want fail-closed mapping rejection", err)
	}
}

func TestRuntimeImageIndexVerifierDoesNotApplyRegistryAllowlist(t *testing.T) {
	payload := mustMarshalRuntimeImageIndex(t, nil)
	fixture := newRuntimeImageRegistryFixture(t, runtimeImageRegistryResponse{payload: payload})
	defer fixture.Close()

	verifier := mustNewRuntimeImageIndexVerifier(t, &runtimeImageSignatureVerifierStub{}, time.Second, int64(len(payload)), fixture.registry())
	ref := ociartifact.DigestReference{
		Registry:   "localhost:55433",
		Repository: runtimeImageRepository,
		Digest:     digest.FromBytes(payload).String(),
	}
	_, err := verifier.Verify(context.Background(), repositoryname.FirstPartyEngineIDSubdomainDiscovery, []string{ref.String()})
	if err == nil || strings.Contains(err.Error(), "not allowlisted") {
		t.Fatalf("Verify() error = %v, want normal transport failure without allowlist rejection", err)
	}
	if fixture.requests.Load() != 0 {
		t.Fatalf("unexpected Registry request count: %d", fixture.requests.Load())
	}
}

func TestRuntimeImageIndexVerifierResolvesFetchesAndVerifiesRealOCIIndex(t *testing.T) {
	payload := mustMarshalRuntimeImageIndex(t, nil)
	fixture := newRuntimeImageRegistryFixture(t, runtimeImageRegistryResponse{payload: payload})
	defer fixture.Close()

	signatureVerifier := &runtimeImageSignatureVerifierStub{}
	verifier := mustNewRuntimeImageIndexVerifier(t, signatureVerifier, time.Second, int64(len(payload)), fixture.registry())
	ref := fixture.reference(payload)

	verified, err := verifier.Verify(context.Background(), repositoryname.FirstPartyEngineIDSubdomainDiscovery, []string{ref})
	if err != nil {
		t.Fatalf("Verify() error = %v", err)
	}
	wantDigest := digest.FromBytes(payload).String()
	if verified.Reference.String() != ref {
		t.Fatalf("verified reference = %q, want %q", verified.Reference.String(), ref)
	}
	if verified.RuntimeImageDigest != runtimeimage.RuntimeImageDigest(wantDigest) {
		t.Fatalf("runtimeImageDigest = %q, want %q", verified.RuntimeImageDigest, wantDigest)
	}
	if verified.Descriptor.Digest.String() != wantDigest ||
		verified.Descriptor.MediaType != ocispec.MediaTypeImageIndex ||
		verified.Descriptor.Size != int64(len(payload)) {
		t.Fatalf("unexpected verified descriptor: %#v", verified.Descriptor)
	}
	if len(verified.Index.Manifests) != 2 {
		t.Fatalf("verified index manifest count = %d, want 2", len(verified.Index.Manifests))
	}
	if fixture.requests.Load() != 2 {
		t.Fatalf("Registry requests = %d, want one HEAD and one GET", fixture.requests.Load())
	}
}

func TestRuntimeImageIndexVerifierDefaultsToHTTPSAndRejectsPlainHTTP(t *testing.T) {
	payload := mustMarshalRuntimeImageIndex(t, nil)
	fixture := newRuntimeImageRegistryFixture(t, runtimeImageRegistryResponse{payload: payload})
	defer fixture.Close()
	verifier, err := NewRuntimeImageIndexVerifier(RuntimeImageIndexVerifierOptions{
		MaxIndexBytes:       int64(len(payload)),
		PerCandidateTimeout: time.Second,
	})
	if err != nil {
		t.Fatal(err)
	}

	_, err = verifier.Verify(context.Background(), repositoryname.FirstPartyEngineIDSubdomainDiscovery, []string{fixture.reference(payload)})
	if err == nil {
		t.Fatal("Verify() accepted a plain HTTP Registry without development transport")
	}
	if fixture.requests.Load() != 0 {
		t.Fatalf("plain HTTP Registry requests = %d, want 0", fixture.requests.Load())
	}
}

func TestRuntimeImageIndexVerifierFallsBackOnlyForTypedUnavailableCandidates(t *testing.T) {
	payload := mustMarshalRuntimeImageIndex(t, nil)
	unavailable := newRuntimeImageRegistryFixture(t, runtimeImageRegistryResponse{status: http.StatusNotFound})
	defer unavailable.Close()
	available := newRuntimeImageRegistryFixture(t, runtimeImageRegistryResponse{payload: payload})
	defer available.Close()

	signatureVerifier := &runtimeImageSignatureVerifierStub{}
	verifier := mustNewRuntimeImageIndexVerifier(
		t,
		signatureVerifier,
		time.Second,
		int64(len(payload)),
		unavailable.registry(),
		available.registry(),
	)
	refs := []string{unavailable.reference(payload), available.reference(payload)}

	verified, err := verifier.Verify(context.Background(), repositoryname.FirstPartyEngineIDSubdomainDiscovery, refs)
	if err != nil {
		t.Fatalf("Verify() error = %v", err)
	}
	if verified.Reference.String() != refs[1] {
		t.Fatalf("selected reference = %q, want second candidate %q", verified.Reference.String(), refs[1])
	}
	if unavailable.requests.Load() != 1 || available.requests.Load() != 2 {
		t.Fatalf("candidate requests = unavailable:%d available:%d, want 1 and 2", unavailable.requests.Load(), available.requests.Load())
	}
}

func TestRuntimeImageIndexVerifierUsesPerCandidateTimeoutWithinCallerBudget(t *testing.T) {
	payload := mustMarshalRuntimeImageIndex(t, nil)
	timedOut := newRuntimeImageRegistryFixture(t, runtimeImageRegistryResponse{waitForCancellation: true})
	defer timedOut.Close()
	available := newRuntimeImageRegistryFixture(t, runtimeImageRegistryResponse{payload: payload})
	defer available.Close()

	verifier := mustNewRuntimeImageIndexVerifier(
		t,
		&runtimeImageSignatureVerifierStub{},
		30*time.Millisecond,
		int64(len(payload)),
		timedOut.registry(),
		available.registry(),
	)
	refs := []string{timedOut.reference(payload), available.reference(payload)}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	verified, err := verifier.Verify(ctx, repositoryname.FirstPartyEngineIDSubdomainDiscovery, refs)
	if err != nil {
		t.Fatalf("Verify() error = %v", err)
	}
	if verified.Reference.String() != refs[1] {
		t.Fatalf("selected reference = %q, want second candidate %q", verified.Reference.String(), refs[1])
	}
}

func TestRuntimeImageIndexVerifierDoesNotFallbackAfterIntegrityFailure(t *testing.T) {
	payload := mustMarshalRuntimeImageIndex(t, nil)
	corruptPayload := append([]byte(nil), payload...)
	corruptPayload[len(corruptPayload)-2] ^= 1
	corrupt := newRuntimeImageRegistryFixture(t, runtimeImageRegistryResponse{
		payload:          corruptPayload,
		advertisedDigest: digest.FromBytes(payload).String(),
	})
	defer corrupt.Close()
	unused := newRuntimeImageRegistryFixture(t, runtimeImageRegistryResponse{payload: payload})
	defer unused.Close()

	signatureVerifier := &runtimeImageSignatureVerifierStub{}
	verifier := mustNewRuntimeImageIndexVerifier(
		t,
		signatureVerifier,
		time.Second,
		int64(len(payload)),
		corrupt.registry(),
		unused.registry(),
	)
	refs := []string{corrupt.reference(payload), unused.reference(payload)}

	_, err := verifier.Verify(context.Background(), repositoryname.FirstPartyEngineIDSubdomainDiscovery, refs)
	if err == nil || !strings.Contains(err.Error(), "payload digest mismatch") {
		t.Fatalf("Verify() error = %v, want payload digest mismatch", err)
	}
	if unused.requests.Load() != 0 {
		t.Fatalf("integrity failure incorrectly advanced to second candidate: requests=%d", unused.requests.Load())
	}
}

func TestRuntimeImageIndexVerifierDoesNotFallbackAfterIndexFailure(t *testing.T) {
	missingPlatformPayload := mustMarshalRuntimeImageIndex(t, func(index *ocispec.Index) {
		index.Manifests = index.Manifests[:1]
	})

	t.Run("malformed platform set", func(t *testing.T) {
		invalid := newRuntimeImageRegistryFixture(t, runtimeImageRegistryResponse{payload: missingPlatformPayload})
		defer invalid.Close()
		unused := newRuntimeImageRegistryFixture(t, runtimeImageRegistryResponse{payload: missingPlatformPayload})
		defer unused.Close()
		verifier := mustNewRuntimeImageIndexVerifier(
			t,
			&runtimeImageSignatureVerifierStub{},
			time.Second,
			int64(len(missingPlatformPayload)),
			invalid.registry(),
			unused.registry(),
		)

		_, err := verifier.Verify(context.Background(), repositoryname.FirstPartyEngineIDSubdomainDiscovery, []string{
			invalid.reference(missingPlatformPayload),
			unused.reference(missingPlatformPayload),
		})
		if err == nil || !strings.Contains(err.Error(), "linux/arm64") {
			t.Fatalf("Verify() error = %v, want missing platform failure", err)
		}
		if unused.requests.Load() != 0 {
			t.Fatalf("index failure incorrectly advanced to second candidate: requests=%d", unused.requests.Load())
		}
	})

}

func TestRuntimeImageIndexVerifierDoesNotFallbackForUnknownOrCallerFailure(t *testing.T) {
	payload := mustMarshalRuntimeImageIndex(t, nil)

	t.Run("unknown Registry response", func(t *testing.T) {
		unknown := newRuntimeImageRegistryFixture(t, runtimeImageRegistryResponse{status: http.StatusTeapot})
		defer unknown.Close()
		unused := newRuntimeImageRegistryFixture(t, runtimeImageRegistryResponse{payload: payload})
		defer unused.Close()
		verifier := mustNewRuntimeImageIndexVerifier(
			t,
			&runtimeImageSignatureVerifierStub{},
			time.Second,
			int64(len(payload)),
			unknown.registry(),
			unused.registry(),
		)

		_, err := verifier.Verify(context.Background(), repositoryname.FirstPartyEngineIDSubdomainDiscovery, []string{
			unknown.reference(payload),
			unused.reference(payload),
		})
		if err == nil {
			t.Fatal("expected unknown Registry failure")
		}
		if unused.requests.Load() != 0 {
			t.Fatalf("unknown failure incorrectly advanced to second candidate: requests=%d", unused.requests.Load())
		}
	})

	t.Run("caller cancellation", func(t *testing.T) {
		first := newRuntimeImageRegistryFixture(t, runtimeImageRegistryResponse{payload: payload})
		defer first.Close()
		unused := newRuntimeImageRegistryFixture(t, runtimeImageRegistryResponse{payload: payload})
		defer unused.Close()
		verifier := mustNewRuntimeImageIndexVerifier(
			t,
			&runtimeImageSignatureVerifierStub{},
			time.Second,
			int64(len(payload)),
			first.registry(),
			unused.registry(),
		)
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		_, err := verifier.Verify(ctx, repositoryname.FirstPartyEngineIDSubdomainDiscovery, []string{
			first.reference(payload),
			unused.reference(payload),
		})
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("Verify() error = %v, want caller cancellation", err)
		}
		if first.requests.Load() != 0 || unused.requests.Load() != 0 {
			t.Fatalf("cancelled caller reached Registry: first=%d second=%d", first.requests.Load(), unused.requests.Load())
		}
	})
}

func TestDecodeRuntimeImageIndexRejectsNonCanonicalIndexAndChildren(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*ocispec.Index)
		want   string
	}{
		{name: "wrong schema", mutate: func(index *ocispec.Index) { index.SchemaVersion = 1 }, want: "schemaVersion"},
		{name: "wrong root media", mutate: func(index *ocispec.Index) { index.MediaType = ocispec.MediaTypeImageManifest }, want: "mediaType"},
		{name: "artifact type", mutate: func(index *ocispec.Index) { index.ArtifactType = "application/vnd.example.artifact" }, want: "artifactType"},
		{name: "subject", mutate: func(index *ocispec.Index) { index.Subject = &index.Manifests[0] }, want: "subject"},
		{name: "missing amd64", mutate: func(index *ocispec.Index) { index.Manifests = index.Manifests[1:] }, want: "linux/amd64"},
		{name: "duplicate arm64", mutate: func(index *ocispec.Index) {
			duplicate := index.Manifests[1]
			duplicate.Digest = digest.Digest("sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc")
			index.Manifests = append(index.Manifests, duplicate)
		}, want: "linux/arm64"},
		{name: "wrong child media", mutate: func(index *ocispec.Index) { index.Manifests[0].MediaType = ocispec.MediaTypeImageIndex }, want: "manifests[0] media type"},
		{name: "invalid child digest", mutate: func(index *ocispec.Index) { index.Manifests[0].Digest = "sha256:bad" }, want: "canonical SHA-256"},
		{name: "zero child size", mutate: func(index *ocispec.Index) { index.Manifests[0].Size = 0 }, want: "size must be positive"},
		{name: "missing child platform", mutate: func(index *ocispec.Index) { index.Manifests[0].Platform = nil }, want: "non-empty platform"},
		{name: "duplicate child digest", mutate: func(index *ocispec.Index) { index.Manifests[1].Digest = index.Manifests[0].Digest }, want: "repeats child digest"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			payload := mustMarshalRuntimeImageIndex(t, test.mutate)
			_, err := decodeRuntimeImageIndex(payload, false)
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("decodeRuntimeImageIndex() error = %v, want %q", err, test.want)
			}
		})
	}

	valid := mustMarshalRuntimeImageIndex(t, nil)
	withUnknownBase := append([]byte(nil), valid...)
	withUnknown := append(withUnknownBase[:len(withUnknownBase)-1], []byte(",\"fallback\":\"latest\"}")...)
	if _, err := decodeRuntimeImageIndex(withUnknown, false); err == nil || !strings.Contains(err.Error(), "unknown field") {
		t.Fatalf("unknown field error = %v", err)
	}
	withTrailing := append(append([]byte(nil), valid...), []byte(" {}")...)
	if _, err := decodeRuntimeImageIndex(withTrailing, false); err == nil || !strings.Contains(err.Error(), "trailing") {
		t.Fatalf("trailing JSON error = %v", err)
	}
}

func TestDecodeRuntimeImageIndexAllowsSingleDevelopmentPlatform(t *testing.T) {
	payload := mustMarshalRuntimeImageIndex(t, func(index *ocispec.Index) {
		index.Manifests = index.Manifests[:1]
	})
	if _, err := decodeRuntimeImageIndex(payload, true); err != nil {
		t.Fatalf("decodeRuntimeImageIndex() development error = %v", err)
	}
	if _, err := decodeRuntimeImageIndex(payload, false); err == nil || !strings.Contains(err.Error(), "linux/arm64") {
		t.Fatalf("decodeRuntimeImageIndex() production error = %v, want missing platform rejection", err)
	}
}

func TestRuntimeImageCandidateFailureClassificationIsClosed(t *testing.T) {
	tests := []struct {
		name   string
		err    error
		reason ociartifact.CandidateFailureReason
	}{
		{name: "manifest unknown", err: runtimeImageRegistryError(http.StatusBadRequest, errcode.ErrorCodeManifestUnknown), reason: ociartifact.CandidateFailureReasonManifestUnknown},
		{name: "blob unknown", err: runtimeImageRegistryError(http.StatusBadRequest, errcode.ErrorCodeBlobUnknown), reason: ociartifact.CandidateFailureReasonBlobUnknown},
		{name: "unauthorized", err: runtimeImageRegistryError(http.StatusUnauthorized, ""), reason: ociartifact.CandidateFailureReasonUnauthorized},
		{name: "forbidden", err: runtimeImageRegistryError(http.StatusForbidden, ""), reason: ociartifact.CandidateFailureReasonForbidden},
		{name: "not found", err: runtimeImageRegistryError(http.StatusNotFound, ""), reason: ociartifact.CandidateFailureReasonNotFound},
		{name: "rate limited", err: runtimeImageRegistryError(http.StatusTooManyRequests, ""), reason: ociartifact.CandidateFailureReasonRateLimited},
		{name: "transient server", err: runtimeImageRegistryError(http.StatusServiceUnavailable, ""), reason: ociartifact.CandidateFailureReasonTransientServer},
		{name: "dns", err: &net.DNSError{Err: "no such host", Name: "registry.invalid"}, reason: ociartifact.CandidateFailureReasonDNS},
		{name: "tcp", err: &net.OpError{Op: "dial", Net: "tcp", Err: syscall.ECONNREFUSED}, reason: ociartifact.CandidateFailureReasonTCP},
		{name: "tls", err: tls.RecordHeaderError{Msg: "invalid TLS header"}, reason: ociartifact.CandidateFailureReasonTLS},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			classified := classifyRuntimeImageCandidateError(context.Background(), context.Background(), test.err)
			var failure *ociartifact.CandidateFailure
			if !errors.As(classified, &failure) || failure.Reason() != test.reason || !ociartifact.IsCandidateUnavailable(classified) {
				t.Fatalf("classified error = %#v, want unavailable reason %v", classified, test.reason)
			}
		})
	}

	attemptCtx, cancelAttempt := context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
	defer cancelAttempt()
	classified := classifyRuntimeImageCandidateError(context.Background(), attemptCtx, attemptCtx.Err())
	var attemptFailure *ociartifact.CandidateFailure
	if !errors.As(classified, &attemptFailure) || attemptFailure.Reason() != ociartifact.CandidateFailureReasonAttemptTimeout {
		t.Fatalf("attempt timeout classification = %#v", classified)
	}

	unknown := errors.New("unknown")
	if classified := classifyRuntimeImageCandidateError(context.Background(), context.Background(), unknown); !errors.Is(classified, unknown) || ociartifact.IsCandidateUnavailable(classified) {
		t.Fatalf("unknown error enabled fallback: %#v", classified)
	}
	mixedIntegrity := &errcode.ErrorResponse{
		StatusCode: http.StatusInternalServerError,
		Errors: errcode.Errors{
			{Code: errcode.ErrorCodeManifestUnknown, Message: "unavailable"},
			{Code: errcode.ErrorCodeDigestInvalid, Message: "integrity failure"},
		},
	}
	if classified := classifyRuntimeImageCandidateError(context.Background(), context.Background(), mixedIntegrity); ociartifact.IsCandidateUnavailable(classified) {
		t.Fatalf("mixed Registry integrity response enabled fallback: %#v", classified)
	}
	if classified := classifyRuntimeImageCandidateError(context.Background(), attemptCtx, mixedIntegrity); ociartifact.IsCandidateUnavailable(classified) {
		t.Fatalf("expired candidate masked mixed Registry integrity response: %#v", classified)
	}
	allowedResponse := runtimeImageRegistryError(http.StatusNotFound, errcode.ErrorCodeManifestUnknown)
	aggregates := []error{
		errors.Join(
			ociartifact.NewCandidateFailure(ociartifact.CandidateFailureReasonNotFound, errors.New("unavailable")),
			errors.New("local integrity failure"),
		),
		errors.Join(allowedResponse, errors.New("unknown adapter failure")),
	}
	for _, aggregate := range aggregates {
		classified := classifyRuntimeImageCandidateError(context.Background(), context.Background(), aggregate)
		if ociartifact.IsCandidateUnavailable(classified) {
			t.Fatalf("aggregate raw error enabled Runtime Image fallback: %#v", classified)
		}
	}

	parentCtx, cancelParent := context.WithCancel(context.Background())
	cancelParent()
	if classified := classifyRuntimeImageCandidateError(parentCtx, attemptCtx, errors.New("masked")); !errors.Is(classified, context.Canceled) {
		t.Fatalf("caller cancellation classification = %v, want context.Canceled", classified)
	}
}

func TestRuntimeImageIndexReadAndCloseFailureFailsClosed(t *testing.T) {
	reference := ociartifact.DigestReference{
		Registry:   "registry.example",
		Repository: runtimeImageRepository,
		Digest:     "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
	}
	reader := &runtimeImageFailingReadCloser{
		readErr:  &net.OpError{Op: "read", Net: "tcp", Err: syscall.ECONNRESET},
		closeErr: errors.New("unknown index close failure"),
	}

	_, err := readRuntimeImageIndexPayload(reader, 1024, reference, context.Background(), context.Background())
	if err == nil || !strings.Contains(err.Error(), "unknown index close failure") || ociartifact.IsCandidateUnavailable(err) {
		t.Fatalf("readRuntimeImageIndexPayload() error = %v, want aggregate fail-closed error", err)
	}
}

func withRuntimeImageVerifierOption(
	base RuntimeImageIndexVerifierOptions,
	mutate func(*RuntimeImageIndexVerifierOptions),
) RuntimeImageIndexVerifierOptions {
	clone := base
	mutate(&clone)
	return clone
}

func mustNewRuntimeImageIndexVerifier(
	t *testing.T,
	_ any,
	timeout time.Duration,
	maxIndexBytes int64,
	registries ...string,
) *RuntimeImageIndexVerifier {
	t.Helper()
	verifier, err := NewRuntimeImageIndexVerifier(RuntimeImageIndexVerifierOptions{
		MaxIndexBytes:       maxIndexBytes,
		PerCandidateTimeout: timeout,
	})
	if err != nil {
		t.Fatalf("NewRuntimeImageIndexVerifier() error = %v", err)
	}
	verifier.newRepository = func(reference ociartifact.DigestReference, _ string, _ bool) (runtimeImageOCIRepository, error) {
		repository, err := remote.NewRepository(reference.Registry + "/" + reference.Repository)
		if err != nil {
			return nil, err
		}
		repository.PlainHTTP = true
		repository.ManifestMediaTypes = []string{ocispec.MediaTypeImageIndex}
		return repository, nil
	}
	return verifier
}

func mustMarshalRuntimeImageIndex(t *testing.T, mutate func(*ocispec.Index)) []byte {
	t.Helper()
	index := ocispec.Index{
		Versioned: specs.Versioned{SchemaVersion: 2},
		MediaType: ocispec.MediaTypeImageIndex,
		Manifests: []ocispec.Descriptor{
			{
				MediaType: ocispec.MediaTypeImageManifest,
				Digest:    digest.Digest("sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"),
				Size:      100,
				Platform:  &ocispec.Platform{OS: "linux", Architecture: "amd64"},
			},
			{
				MediaType: ocispec.MediaTypeImageManifest,
				Digest:    digest.Digest("sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"),
				Size:      101,
				Platform:  &ocispec.Platform{OS: "linux", Architecture: "arm64"},
			},
		},
	}
	if mutate != nil {
		mutate(&index)
	}
	payload, err := json.Marshal(index)
	if err != nil {
		t.Fatalf("marshal Runtime Image OCI index fixture: %v", err)
	}
	return payload
}

type runtimeImageFailingReadCloser struct {
	readErr  error
	closeErr error
}

func (reader *runtimeImageFailingReadCloser) Read([]byte) (int, error) { return 0, reader.readErr }
func (reader *runtimeImageFailingReadCloser) Close() error             { return reader.closeErr }

type runtimeImageSignatureVerifierStub struct{}

type recordingDigestReferenceSignatureVerifier struct {
	references []ociartifact.DigestReference
	err        error
}

func (verifier *recordingDigestReferenceSignatureVerifier) VerifyReference(_ context.Context, reference ociartifact.DigestReference) error {
	verifier.references = append(verifier.references, reference)
	return verifier.err
}

type runtimeImageOCIRepositoryStub struct {
	descriptor ocispec.Descriptor
	payload    []byte
	resolveErr error
	fetchErr   error
}

func (repository runtimeImageOCIRepositoryStub) Resolve(context.Context, string) (ocispec.Descriptor, error) {
	return repository.descriptor, repository.resolveErr
}

func (repository runtimeImageOCIRepositoryStub) Fetch(context.Context, ocispec.Descriptor) (io.ReadCloser, error) {
	if repository.fetchErr != nil {
		return nil, repository.fetchErr
	}
	return io.NopCloser(bytes.NewReader(repository.payload)), nil
}

type runtimeImageRegistryResponse struct {
	payload             []byte
	advertisedDigest    string
	status              int
	waitForCancellation bool
}

type runtimeImageRegistryFixture struct {
	*httptest.Server
	requests atomic.Int32
}

func newRuntimeImageRegistryFixture(t *testing.T, response runtimeImageRegistryResponse) *runtimeImageRegistryFixture {
	t.Helper()
	fixture := &runtimeImageRegistryFixture{}
	fixture.Server = httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if !strings.Contains(request.URL.Path, "/manifests/") {
			http.NotFound(writer, request)
			return
		}
		fixture.requests.Add(1)
		if response.waitForCancellation {
			<-request.Context().Done()
			return
		}
		if response.status != 0 && response.status != http.StatusOK {
			writer.WriteHeader(response.status)
			return
		}
		advertisedDigest := response.advertisedDigest
		if advertisedDigest == "" {
			advertisedDigest = digest.FromBytes(response.payload).String()
		}
		writer.Header().Set("Content-Type", ocispec.MediaTypeImageIndex)
		writer.Header().Set("Docker-Content-Digest", advertisedDigest)
		writer.Header().Set("Content-Length", strconv.Itoa(len(response.payload)))
		writer.WriteHeader(http.StatusOK)
		if request.Method != http.MethodHead {
			_, _ = writer.Write(response.payload)
		}
	}))
	fixture.Config.ErrorLog = nil
	return fixture
}

func (fixture *runtimeImageRegistryFixture) registry() string {
	return strings.TrimPrefix(fixture.URL, "http://")
}

func (fixture *runtimeImageRegistryFixture) reference(payload []byte) string {
	return fixture.registry() + "/" + runtimeImageRepository + "@" + digest.FromBytes(payload).String()
}

func runtimeImageRegistryError(statusCode int, code string) error {
	response := &errcode.ErrorResponse{StatusCode: statusCode}
	if code != "" {
		response.Errors = errcode.Errors{{Code: code, Message: "fixture"}}
	}
	return fmt.Errorf("Registry response: %w", response)
}
