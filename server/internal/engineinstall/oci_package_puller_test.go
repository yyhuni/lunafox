package engineinstall

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"syscall"
	"testing"
	"time"

	"github.com/opencontainers/go-digest"
	"github.com/opencontainers/image-spec/specs-go"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/yyhuni/lunafox/contracts/ociartifact"
	"oras.land/oras-go/v2/errdef"
	"oras.land/oras-go/v2/registry/remote/errcode"
)

func TestNewORASPackagePullerRequiresBoundedProcessingOnly(t *testing.T) {
	valid := ORASPackagePullerOptions{
		MaxManifestBytes:     1024,
		MaxPackageLayerBytes: 2048,
		PerCandidateTimeout:  time.Second,
	}
	tests := []struct {
		name   string
		mutate func(*ORASPackagePullerOptions)
	}{
		{name: "manifest limit", mutate: func(options *ORASPackagePullerOptions) { options.MaxManifestBytes = 0 }},
		{name: "unbounded manifest limit", mutate: func(options *ORASPackagePullerOptions) { options.MaxManifestBytes = math.MaxInt64 }},
		{name: "layer limit", mutate: func(options *ORASPackagePullerOptions) { options.MaxPackageLayerBytes = 0 }},
		{name: "unbounded layer limit", mutate: func(options *ORASPackagePullerOptions) { options.MaxPackageLayerBytes = math.MaxInt64 }},
		{name: "candidate timeout", mutate: func(options *ORASPackagePullerOptions) { options.PerCandidateTimeout = 0 }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			options := valid
			test.mutate(&options)
			if _, err := NewORASPackagePuller(options); err == nil {
				t.Fatalf("NewORASPackagePuller() accepted missing %s", test.name)
			}
		})
	}
	if _, err := NewORASPackagePuller(valid); err != nil {
		t.Fatalf("NewORASPackagePuller() error = %v", err)
	}
}

func TestORASPackagePullerConsumesVerifiedLayerAndReturnsTypedDigests(t *testing.T) {
	artifact := newPackageArtifactFixture(t, []byte("package archive"), ociartifact.EnginePackageArtifactType, ociartifact.EnginePackageLayerMediaType)
	repository := artifact.repository()
	puller := newTestORASPackagePuller(t, map[string]*fakePackageRepository{"registry.example": repository})
	candidates := parsePackageCandidates(t, "registry.example/lunafox/engine@"+artifact.manifestDigest.String())

	var consumed []byte
	result, err := puller.ConsumePackageLayer(context.Background(), candidates, func(_ context.Context, layer PulledPackageLayer) error {
		if layer.PackageDigest != ociartifact.PackageDigest(artifact.layerDigest.String()) || layer.Size != int64(len(artifact.layer)) {
			t.Fatalf("unexpected layer identity: %#v", layer)
		}
		var readErr error
		consumed, readErr = io.ReadAll(layer.Reader)
		return readErr
	})
	if err != nil {
		t.Fatalf("ConsumePackageLayer() error = %v", err)
	}
	if !bytes.Equal(consumed, artifact.layer) {
		t.Fatalf("consumed layer = %q, want %q", consumed, artifact.layer)
	}
	if result.Reference.String() != candidates.References[0].String() ||
		result.ArtifactManifestDigest != candidates.ArtifactManifestDigest ||
		result.PackageDigest != ociartifact.PackageDigest(artifact.layerDigest.String()) {
		t.Fatalf("unexpected consumed package facts: %#v", result)
	}
	if repository.layerFetches != 1 {
		t.Fatalf("layer fetches = %d, want 1", repository.layerFetches)
	}
}

func TestORASPackagePullerDefaultsToHTTPSAndRejectsPlainHTTP(t *testing.T) {
	artifact := newPackageArtifactFixture(t, []byte("local Registry package archive"), ociartifact.EnginePackageArtifactType, ociartifact.EnginePackageLayerMediaType)
	registry := newLocalRegistryFixture(t, artifact.manifest, artifact.manifestDigest, artifact.layerDigest, artifact.layer)
	defer registry.Close()
	registryHost := strings.TrimPrefix(registry.URL, "http://")
	puller, err := NewORASPackagePuller(ORASPackagePullerOptions{
		MaxManifestBytes:     2048,
		MaxPackageLayerBytes: 4096,
		PerCandidateTimeout:  time.Second,
	})
	if err != nil {
		t.Fatal(err)
	}
	candidates := parsePackageCandidates(t, registryHost+"/lunafox/test@"+artifact.manifestDigest.String())

	if _, err := puller.ConsumePackageLayer(context.Background(), candidates, readAllPackageLayer); err == nil {
		t.Fatal("ConsumePackageLayer() accepted a plain HTTP Registry")
	}
	if registry.layerRequests.Load() != 0 {
		t.Fatalf("plain HTTP Registry layer requests=%d, want 0", registry.layerRequests.Load())
	}
}

func TestORASPackagePullerUsesExplicitPlainHTTPTransport(t *testing.T) {
	artifact := newPackageArtifactFixture(t, []byte("local Registry package archive"), ociartifact.EnginePackageArtifactType, ociartifact.EnginePackageLayerMediaType)
	registry := newLocalRegistryFixture(t, artifact.manifest, artifact.manifestDigest, artifact.layerDigest, artifact.layer)
	defer registry.Close()
	registryHost := strings.TrimPrefix(registry.URL, "http://")
	puller, err := NewORASPackagePuller(ORASPackagePullerOptions{
		MaxManifestBytes:     2048,
		MaxPackageLayerBytes: 4096,
		PerCandidateTimeout:  time.Second,
		AllowPlainHTTP:       true,
	})
	if err != nil {
		t.Fatal(err)
	}
	candidates := parsePackageCandidates(t, registryHost+"/lunafox/test@"+artifact.manifestDigest.String())

	if _, err := puller.ConsumePackageLayer(context.Background(), candidates, readAllPackageLayer); err != nil {
		t.Fatalf("ConsumePackageLayer() development error = %v", err)
	}
	if registry.layerRequests.Load() != 1 {
		t.Fatalf("plain HTTP Registry layer requests=%d, want 1", registry.layerRequests.Load())
	}
}

func TestORASPackagePullerRejectsArtifactOrLayerBeforeSignatureAndLayerFetch(t *testing.T) {
	const (
		legacyArtifactType   = "application/vnd.lunafox.engine-package.v1"
		legacyLayerMediaType = "application/vnd.lunafox.engine-package.layer.v1.tar+gzip"
	)
	tests := []struct {
		name         string
		artifactType string
		layerType    string
	}{
		{name: "v1 artifact type", artifactType: legacyArtifactType, layerType: ociartifact.EnginePackageLayerMediaType},
		{name: "v1 layer media type", artifactType: ociartifact.EnginePackageArtifactType, layerType: legacyLayerMediaType},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			artifact := newPackageArtifactFixture(t, []byte("legacy package"), test.artifactType, test.layerType)
			repository := artifact.repository()
			puller := newTestORASPackagePuller(t, map[string]*fakePackageRepository{"registry.example": repository})
			candidates := parsePackageCandidates(t, "registry.example/lunafox/engine@"+artifact.manifestDigest.String())

			_, err := puller.ConsumePackageLayer(context.Background(), candidates, readAllPackageLayer)
			if err == nil || !stringsContainAny(err.Error(), "unsupported OCI artifact type", "unsupported engine package layer media type") {
				t.Fatalf("ConsumePackageLayer() error = %v, want strict v2 media rejection", err)
			}
			if repository.layerFetches != 0 {
				t.Fatalf("layer fetches=%d, want zero", repository.layerFetches)
			}
		})
	}
}

func TestORASPackagePullerRejectsOldOCIEnvelopeBeforeSignatureLayerFetchOrFallback(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*ocispec.Manifest)
	}{
		{
			name: "missing root media type",
			mutate: func(manifest *ocispec.Manifest) {
				manifest.MediaType = ""
			},
		},
		{
			name: "OCI image config",
			mutate: func(manifest *ocispec.Manifest) {
				manifest.Config.MediaType = ocispec.MediaTypeImageConfig
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			artifact := newPackageArtifactFixture(t, []byte("package"), ociartifact.EnginePackageArtifactType, ociartifact.EnginePackageLayerMediaType)
			mutatePackageArtifactFixture(t, &artifact, test.mutate)
			first := artifact.repository()
			second := artifact.repository()
			puller := newTestORASPackagePuller(t, map[string]*fakePackageRepository{
				"first.example": first, "second.example": second,
			})
			candidates := parsePackageCandidates(t,
				"first.example/lunafox/engine@"+artifact.manifestDigest.String(),
				"second.example/lunafox/engine@"+artifact.manifestDigest.String(),
			)

			_, err := puller.ConsumePackageLayer(context.Background(), candidates, readAllPackageLayer)
			if err == nil {
				t.Fatal("ConsumePackageLayer() accepted old OCI envelope")
			}
			if first.manifestFetches != 1 {
				t.Fatalf("first manifest fetches = %d, want 1 before envelope decoding", first.manifestFetches)
			}
			if first.layerFetches != 0 || second.resolveCalls != 0 {
				t.Fatalf(
					"first layer fetches=%d second resolve calls=%d, want both zero",
					first.layerFetches,
					second.resolveCalls,
				)
			}
		})
	}
}

func TestORASPackagePullerRejectsManifestDescriptorBeforeBodyFetch(t *testing.T) {
	artifact := newPackageArtifactFixture(t, []byte("package"), ociartifact.EnginePackageArtifactType, ociartifact.EnginePackageLayerMediaType)
	tests := []struct {
		name   string
		mutate func(*ocispec.Descriptor)
	}{
		{name: "wrong media type", mutate: func(descriptor *ocispec.Descriptor) { descriptor.MediaType = ocispec.MediaTypeImageIndex }},
		{name: "wrong digest", mutate: func(descriptor *ocispec.Descriptor) { descriptor.Digest = digest.FromString("other manifest") }},
		{name: "zero size", mutate: func(descriptor *ocispec.Descriptor) { descriptor.Size = 0 }},
		{name: "over size limit", mutate: func(descriptor *ocispec.Descriptor) { descriptor.Size = 2049 }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			repository := artifact.repository()
			test.mutate(&repository.manifestDescriptor)
			puller := newTestORASPackagePuller(t, map[string]*fakePackageRepository{"registry.example": repository}, &recordingPackageSignatureVerifier{})
			candidates := parsePackageCandidates(t, "registry.example/lunafox/engine@"+artifact.manifestDigest.String())

			if _, err := puller.ConsumePackageLayer(context.Background(), candidates, readAllPackageLayer); err == nil {
				t.Fatal("ConsumePackageLayer() accepted invalid manifest descriptor")
			}
			if repository.manifestFetches != 0 || repository.layerFetches != 0 {
				t.Fatalf("fetches manifest=%d layer=%d, want zero", repository.manifestFetches, repository.layerFetches)
			}
		})
	}
}

func TestORASPackagePullerRejectsManifestPayloadSizeAndDigestWithoutFallback(t *testing.T) {
	artifact := newPackageArtifactFixture(t, []byte("package"), ociartifact.EnginePackageArtifactType, ociartifact.EnginePackageLayerMediaType)
	tests := []struct {
		name   string
		mutate func(*fakePackageRepository, *ociartifact.ArtifactCandidates)
		want   string
	}{
		{
			name: "payload size",
			mutate: func(repository *fakePackageRepository, _ *ociartifact.ArtifactCandidates) {
				repository.manifestDescriptor.Size++
			},
			want: "payload size mismatch",
		},
		{
			name: "payload digest",
			mutate: func(repository *fakePackageRepository, candidates *ociartifact.ArtifactCandidates) {
				wrongDigest := ociartifact.ArtifactManifestDigest("sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")
				candidates.ArtifactManifestDigest = wrongDigest
				for index := range candidates.References {
					candidates.References[index].ArtifactManifestDigest = wrongDigest
				}
				repository.manifestDescriptor.Digest = digest.Digest(wrongDigest)
			},
			want: "payload digest mismatch",
		},
		{
			name: "bounded payload",
			mutate: func(repository *fakePackageRepository, _ *ociartifact.ArtifactCandidates) {
				repository.manifest = bytes.Repeat([]byte("x"), 2049)
				repository.manifestDescriptor.Size = 2048
			},
			want: "exceeds limit",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			first := artifact.repository()
			second := artifact.repository()
			candidates := parsePackageCandidates(t,
				"first.example/lunafox/engine@"+artifact.manifestDigest.String(),
				"second.example/lunafox/engine@"+artifact.manifestDigest.String(),
			)
			test.mutate(first, &candidates)
			puller := newTestORASPackagePuller(t, map[string]*fakePackageRepository{
				"first.example": first, "second.example": second,
			}, &recordingPackageSignatureVerifier{})

			_, err := puller.ConsumePackageLayer(context.Background(), candidates, readAllPackageLayer)
			if err == nil || !bytes.Contains([]byte(err.Error()), []byte(test.want)) {
				t.Fatalf("ConsumePackageLayer() error = %v, want %q", err, test.want)
			}
			if second.resolveCalls != 0 {
				t.Fatalf("second candidate resolve calls = %d, integrity failure must not fallback", second.resolveCalls)
			}
		})
	}
}

func TestORASPackagePullerManifestReadAndCloseFailureFailsClosed(t *testing.T) {
	artifact := newPackageArtifactFixture(t, []byte("package"), ociartifact.EnginePackageArtifactType, ociartifact.EnginePackageLayerMediaType)
	first := artifact.repository()
	first.manifestReader = func() io.ReadCloser {
		return &failingPackageLayerReader{
			payload:  artifact.manifest[:5],
			err:      &net.OpError{Op: "read", Net: "tcp", Err: syscall.ECONNRESET},
			closeErr: errors.New("unknown manifest close failure"),
		}
	}
	second := artifact.repository()
	puller := newTestORASPackagePuller(t, map[string]*fakePackageRepository{
		"first.example": first, "second.example": second,
	}, &recordingPackageSignatureVerifier{})
	candidates := parsePackageCandidates(t,
		"first.example/lunafox/engine@"+artifact.manifestDigest.String(),
		"second.example/lunafox/engine@"+artifact.manifestDigest.String(),
	)

	_, err := puller.ConsumePackageLayer(context.Background(), candidates, readAllPackageLayer)
	if err == nil || !bytes.Contains([]byte(err.Error()), []byte("unknown manifest close failure")) || ociartifact.IsCandidateUnavailable(err) {
		t.Fatalf("ConsumePackageLayer() error = %v, want aggregate manifest failure", err)
	}
	if second.resolveCalls != 0 {
		t.Fatalf("second candidate resolve calls = %d, close failure must not fallback", second.resolveCalls)
	}
}

func TestORASPackagePullerFallsBackOnlyForTypedCandidateUnavailable(t *testing.T) {
	artifact := newPackageArtifactFixture(t, []byte("package"), ociartifact.EnginePackageArtifactType, ociartifact.EnginePackageLayerMediaType)
	t.Run("resolve unavailable", func(t *testing.T) {
		first := artifact.repository()
		first.resolveErr = fmt.Errorf("registry miss: %w", errdef.ErrNotFound)
		second := artifact.repository()
		puller := newTestORASPackagePuller(t, map[string]*fakePackageRepository{
			"first.example": first, "second.example": second,
		}, &recordingPackageSignatureVerifier{})
		candidates := parsePackageCandidates(t,
			"first.example/lunafox/engine@"+artifact.manifestDigest.String(),
			"second.example/lunafox/engine@"+artifact.manifestDigest.String(),
		)

		result, err := puller.ConsumePackageLayer(context.Background(), candidates, readAllPackageLayer)
		if err != nil {
			t.Fatalf("ConsumePackageLayer() error = %v", err)
		}
		if result.Reference.Registry != "second.example" || second.layerFetches != 1 {
			t.Fatalf("unexpected fallback result=%#v second layer fetches=%d", result, second.layerFetches)
		}
	})

	t.Run("unknown failure", func(t *testing.T) {
		first := artifact.repository()
		first.resolveErr = errors.New("unclassified registry failure")
		second := artifact.repository()
		puller := newTestORASPackagePuller(t, map[string]*fakePackageRepository{
			"first.example": first, "second.example": second,
		}, &recordingPackageSignatureVerifier{})
		candidates := parsePackageCandidates(t,
			"first.example/lunafox/engine@"+artifact.manifestDigest.String(),
			"second.example/lunafox/engine@"+artifact.manifestDigest.String(),
		)

		if _, err := puller.ConsumePackageLayer(context.Background(), candidates, readAllPackageLayer); err == nil {
			t.Fatal("ConsumePackageLayer() accepted unknown failure")
		}
		if second.resolveCalls != 0 {
			t.Fatalf("second candidate resolve calls = %d, unknown failure must not fallback", second.resolveCalls)
		}
	})
}

func TestORASPackagePullerKeepsLayerStreamInsideCandidateFailover(t *testing.T) {
	artifact := newPackageArtifactFixture(t, []byte("complete package body"), ociartifact.EnginePackageArtifactType, ociartifact.EnginePackageLayerMediaType)
	first := artifact.repository()
	first.layerReader = func() io.ReadCloser {
		return &failingPackageLayerReader{
			payload: artifact.layer[:5],
			err:     &net.OpError{Op: "read", Net: "tcp", Err: syscall.ECONNRESET},
		}
	}
	second := artifact.repository()
	puller := newTestORASPackagePuller(t, map[string]*fakePackageRepository{
		"first.example": first, "second.example": second,
	}, &recordingPackageSignatureVerifier{})
	candidates := parsePackageCandidates(t,
		"first.example/lunafox/engine@"+artifact.manifestDigest.String(),
		"second.example/lunafox/engine@"+artifact.manifestDigest.String(),
	)

	consumerCalls := 0
	result, err := puller.ConsumePackageLayer(context.Background(), candidates, func(_ context.Context, layer PulledPackageLayer) error {
		consumerCalls++
		_, readErr := io.Copy(io.Discard, layer.Reader)
		if readErr != nil {
			return fmt.Errorf("stage candidate archive: %w", readErr)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("ConsumePackageLayer() error = %v", err)
	}
	if consumerCalls != 2 || result.Reference.Registry != "second.example" {
		t.Fatalf("consumer calls=%d result=%#v, want full-attempt fallback", consumerCalls, result)
	}
}

func TestORASPackagePullerLayerIntegrityAndLocalStorageFailuresNeverFallback(t *testing.T) {
	artifact := newPackageArtifactFixture(t, []byte("expected package"), ociartifact.EnginePackageArtifactType, ociartifact.EnginePackageLayerMediaType)
	t.Run("layer digest mismatch", func(t *testing.T) {
		first := artifact.repository()
		first.layerReader = func() io.ReadCloser { return io.NopCloser(bytes.NewReader([]byte("tampered package"))) }
		second := artifact.repository()
		puller := newTestORASPackagePuller(t, map[string]*fakePackageRepository{
			"first.example": first, "second.example": second,
		}, &recordingPackageSignatureVerifier{})
		candidates := parsePackageCandidates(t,
			"first.example/lunafox/engine@"+artifact.manifestDigest.String(),
			"second.example/lunafox/engine@"+artifact.manifestDigest.String(),
		)

		_, err := puller.ConsumePackageLayer(context.Background(), candidates, readAllPackageLayer)
		if err == nil || !bytes.Contains([]byte(err.Error()), []byte("layer digest mismatch")) {
			t.Fatalf("ConsumePackageLayer() error = %v, want layer digest mismatch", err)
		}
		if second.resolveCalls != 0 {
			t.Fatalf("second candidate resolve calls = %d, integrity failure must not fallback", second.resolveCalls)
		}
	})

	t.Run("local staging error", func(t *testing.T) {
		first := artifact.repository()
		second := artifact.repository()
		puller := newTestORASPackagePuller(t, map[string]*fakePackageRepository{
			"first.example": first, "second.example": second,
		}, &recordingPackageSignatureVerifier{})
		candidates := parsePackageCandidates(t,
			"first.example/lunafox/engine@"+artifact.manifestDigest.String(),
			"second.example/lunafox/engine@"+artifact.manifestDigest.String(),
		)

		_, err := puller.ConsumePackageLayer(context.Background(), candidates, func(context.Context, PulledPackageLayer) error {
			return syscall.ENOSPC
		})
		if !errors.Is(err, syscall.ENOSPC) {
			t.Fatalf("ConsumePackageLayer() error = %v, want ENOSPC", err)
		}
		if second.resolveCalls != 0 {
			t.Fatalf("second candidate resolve calls = %d, local failure must not fallback", second.resolveCalls)
		}
	})
}

func TestORASPackagePullerRejectsCallbackAuthoredCandidateFailures(t *testing.T) {
	artifact := newPackageArtifactFixture(t, []byte("expected package"), ociartifact.EnginePackageArtifactType, ociartifact.EnginePackageLayerMediaType)
	tests := []struct {
		name        string
		callbackErr error
	}{
		{
			name: "self-authored typed unavailable",
			callbackErr: ociartifact.NewCandidateFailure(
				ociartifact.CandidateFailureReasonNotFound,
				errors.New("local stage invented Registry miss"),
			),
		},
		{name: "local deadline", callbackErr: context.DeadlineExceeded},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			first := artifact.repository()
			second := artifact.repository()
			puller := newTestORASPackagePuller(t, map[string]*fakePackageRepository{
				"first.example": first, "second.example": second,
			}, &recordingPackageSignatureVerifier{})
			candidates := parsePackageCandidates(t,
				"first.example/lunafox/engine@"+artifact.manifestDigest.String(),
				"second.example/lunafox/engine@"+artifact.manifestDigest.String(),
			)

			_, err := puller.ConsumePackageLayer(context.Background(), candidates, func(_ context.Context, layer PulledPackageLayer) error {
				if _, readErr := io.Copy(io.Discard, layer.Reader); readErr != nil {
					return readErr
				}
				return test.callbackErr
			})
			if err == nil || ociartifact.IsCandidateUnavailable(err) {
				t.Fatalf("ConsumePackageLayer() error = %v, want immediate callback failure", err)
			}
			if second.resolveCalls != 0 {
				t.Fatalf("second candidate resolve calls = %d, callback failure must not fallback", second.resolveCalls)
			}
		})
	}
}

func TestORASPackagePullerRejectsMixedReaderAndLocalFailureWithoutFallback(t *testing.T) {
	artifact := newPackageArtifactFixture(t, []byte("complete package body"), ociartifact.EnginePackageArtifactType, ociartifact.EnginePackageLayerMediaType)
	first := artifact.repository()
	first.layerReader = func() io.ReadCloser {
		return &failingPackageLayerReader{
			payload: artifact.layer[:5],
			err:     &net.OpError{Op: "read", Net: "tcp", Err: syscall.ECONNRESET},
		}
	}
	second := artifact.repository()
	puller := newTestORASPackagePuller(t, map[string]*fakePackageRepository{
		"first.example": first, "second.example": second,
	}, &recordingPackageSignatureVerifier{})
	candidates := parsePackageCandidates(t,
		"first.example/lunafox/engine@"+artifact.manifestDigest.String(),
		"second.example/lunafox/engine@"+artifact.manifestDigest.String(),
	)

	_, err := puller.ConsumePackageLayer(context.Background(), candidates, func(_ context.Context, layer PulledPackageLayer) error {
		_, readErr := io.Copy(io.Discard, layer.Reader)
		return errors.Join(readErr, syscall.ENOSPC)
	})
	if err == nil || ociartifact.IsCandidateUnavailable(err) {
		t.Fatalf("ConsumePackageLayer() error = %v, mixed local failure must fail closed", err)
	}
	if second.resolveCalls != 0 {
		t.Fatalf("second candidate resolve calls = %d, mixed failure must not fallback", second.resolveCalls)
	}
}

func TestORASPackagePullerCloseFailureOverridesReaderCandidateFailure(t *testing.T) {
	artifact := newPackageArtifactFixture(t, []byte("complete package body"), ociartifact.EnginePackageArtifactType, ociartifact.EnginePackageLayerMediaType)
	first := artifact.repository()
	first.layerReader = func() io.ReadCloser {
		return &failingPackageLayerReader{
			payload:  artifact.layer[:5],
			err:      &net.OpError{Op: "read", Net: "tcp", Err: syscall.ECONNRESET},
			closeErr: errors.New("unknown close failure"),
		}
	}
	second := artifact.repository()
	puller := newTestORASPackagePuller(t, map[string]*fakePackageRepository{
		"first.example": first, "second.example": second,
	}, &recordingPackageSignatureVerifier{})
	candidates := parsePackageCandidates(t,
		"first.example/lunafox/engine@"+artifact.manifestDigest.String(),
		"second.example/lunafox/engine@"+artifact.manifestDigest.String(),
	)

	_, err := puller.ConsumePackageLayer(context.Background(), candidates, readAllPackageLayer)
	if err == nil || !bytes.Contains([]byte(err.Error()), []byte("unknown close failure")) || ociartifact.IsCandidateUnavailable(err) {
		t.Fatalf("ConsumePackageLayer() error = %v, want immediate close failure", err)
	}
	if second.resolveCalls != 0 {
		t.Fatalf("second candidate resolve calls = %d, close failure must not fallback", second.resolveCalls)
	}
}

func TestORASPackagePullerRejectsLayerSizeMismatchWithoutFallback(t *testing.T) {
	artifact := newPackageArtifactFixture(t, []byte("expected package"), ociartifact.EnginePackageArtifactType, ociartifact.EnginePackageLayerMediaType)
	tests := []struct {
		name    string
		payload []byte
		want    string
	}{
		{name: "short", payload: []byte("short"), want: "layer size mismatch"},
		{name: "long", payload: append(append([]byte(nil), artifact.layer...), []byte("extra")...), want: "layer size exceeds descriptor"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			first := artifact.repository()
			first.layerReader = func() io.ReadCloser { return io.NopCloser(bytes.NewReader(test.payload)) }
			second := artifact.repository()
			puller := newTestORASPackagePuller(t, map[string]*fakePackageRepository{
				"first.example": first, "second.example": second,
			}, &recordingPackageSignatureVerifier{})
			candidates := parsePackageCandidates(t,
				"first.example/lunafox/engine@"+artifact.manifestDigest.String(),
				"second.example/lunafox/engine@"+artifact.manifestDigest.String(),
			)

			_, err := puller.ConsumePackageLayer(context.Background(), candidates, readAllPackageLayer)
			if err == nil || !bytes.Contains([]byte(err.Error()), []byte(test.want)) {
				t.Fatalf("ConsumePackageLayer() error = %v, want %q", err, test.want)
			}
			if second.resolveCalls != 0 {
				t.Fatalf("second candidate resolve calls = %d, size failure must not fallback", second.resolveCalls)
			}
		})
	}
}

func TestORASPackagePullerRejectsConsumerThatDoesNotVerifyCompleteLayer(t *testing.T) {
	artifact := newPackageArtifactFixture(t, []byte("expected package"), ociartifact.EnginePackageArtifactType, ociartifact.EnginePackageLayerMediaType)
	first := artifact.repository()
	second := artifact.repository()
	puller := newTestORASPackagePuller(t, map[string]*fakePackageRepository{
		"first.example": first, "second.example": second,
	}, &recordingPackageSignatureVerifier{})
	candidates := parsePackageCandidates(t,
		"first.example/lunafox/engine@"+artifact.manifestDigest.String(),
		"second.example/lunafox/engine@"+artifact.manifestDigest.String(),
	)

	_, err := puller.ConsumePackageLayer(context.Background(), candidates, func(context.Context, PulledPackageLayer) error {
		return nil
	})
	if err == nil || !bytes.Contains([]byte(err.Error()), []byte("closed before integrity verification")) {
		t.Fatalf("ConsumePackageLayer() error = %v, want incomplete-consumption failure", err)
	}
	if second.resolveCalls != 0 {
		t.Fatalf("second candidate resolve calls = %d, incomplete consumption must not fallback", second.resolveCalls)
	}
}

func TestPackageCandidateTimeoutClassificationRequiresLiveParentBudget(t *testing.T) {
	parent := context.Background()
	attempt, cancelAttempt := context.WithTimeout(parent, time.Nanosecond)
	defer cancelAttempt()
	<-attempt.Done()
	timedOut := classifyPackageCandidateError(parent, attempt, attempt.Err())
	if !ociartifact.IsCandidateUnavailable(timedOut) {
		t.Fatalf("single-candidate timeout = %v, want typed unavailable", timedOut)
	}

	cancelledParent, cancelParent := context.WithCancel(context.Background())
	cancelParent()
	attempt, cancelAttempt = context.WithTimeout(cancelledParent, time.Second)
	defer cancelAttempt()
	cancelled := classifyPackageCandidateError(cancelledParent, attempt, attempt.Err())
	if !errors.Is(cancelled, context.Canceled) || ociartifact.IsCandidateUnavailable(cancelled) {
		t.Fatalf("caller cancellation = %v, want immediate non-fallback cancellation", cancelled)
	}

	mixedIntegrity := &errcode.ErrorResponse{
		StatusCode: http.StatusInternalServerError,
		Errors: errcode.Errors{
			{Code: errcode.ErrorCodeManifestUnknown, Message: "unavailable"},
			{Code: errcode.ErrorCodeDigestInvalid, Message: "integrity failure"},
		},
	}
	expiredAttempt, cancelExpiredAttempt := context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
	defer cancelExpiredAttempt()
	if classified := classifyPackageCandidateError(context.Background(), expiredAttempt, mixedIntegrity); ociartifact.IsCandidateUnavailable(classified) {
		t.Fatalf("expired candidate masked mixed Registry integrity response: %#v", classified)
	}
	allowedOnly := &errcode.ErrorResponse{
		StatusCode: http.StatusBadRequest,
		Errors: errcode.Errors{
			{Code: errcode.ErrorCodeManifestUnknown, Message: "unavailable"},
			{Code: errcode.ErrorCodeBlobUnknown, Message: "unavailable"},
		},
	}
	if classified := classifyPackageCandidateError(context.Background(), context.Background(), allowedOnly); !ociartifact.IsCandidateUnavailable(classified) {
		t.Fatalf("closed availability response = %#v, want typed unavailable", classified)
	}
	aggregates := []error{
		errors.Join(
			ociartifact.NewCandidateFailure(ociartifact.CandidateFailureReasonNotFound, errors.New("unavailable")),
			errors.New("local integrity failure"),
		),
		errors.Join(allowedOnly, errors.New("unknown adapter failure")),
	}
	for _, aggregate := range aggregates {
		classified := classifyPackageCandidateError(context.Background(), context.Background(), aggregate)
		if ociartifact.IsCandidateUnavailable(classified) {
			t.Fatalf("aggregate raw error enabled package fallback: %#v", classified)
		}
	}
}

func TestORASPackagePullerCannotSucceedAfterContextBudgetExpires(t *testing.T) {
	artifact := newPackageArtifactFixture(t, []byte("expected package"), ociartifact.EnginePackageArtifactType, ociartifact.EnginePackageLayerMediaType)
	t.Run("caller cancellation", func(t *testing.T) {
		first := artifact.repository()
		second := artifact.repository()
		puller := newTestORASPackagePuller(t, map[string]*fakePackageRepository{
			"first.example": first, "second.example": second,
		}, &recordingPackageSignatureVerifier{})
		candidates := parsePackageCandidates(t,
			"first.example/lunafox/engine@"+artifact.manifestDigest.String(),
			"second.example/lunafox/engine@"+artifact.manifestDigest.String(),
		)
		ctx, cancel := context.WithCancel(context.Background())

		_, err := puller.ConsumePackageLayer(ctx, candidates, func(_ context.Context, layer PulledPackageLayer) error {
			if _, readErr := io.Copy(io.Discard, layer.Reader); readErr != nil {
				return readErr
			}
			cancel()
			return nil
		})
		if !errors.Is(err, context.Canceled) || ociartifact.IsCandidateUnavailable(err) {
			t.Fatalf("ConsumePackageLayer() error = %v, want caller cancellation", err)
		}
		if second.resolveCalls != 0 {
			t.Fatalf("second candidate resolve calls = %d, caller cancellation must not fallback", second.resolveCalls)
		}
	})

	t.Run("candidate deadline after verified stream", func(t *testing.T) {
		first := artifact.repository()
		second := artifact.repository()
		puller := newTestORASPackagePuller(t, map[string]*fakePackageRepository{
			"first.example": first, "second.example": second,
		}, &recordingPackageSignatureVerifier{})
		puller.options.PerCandidateTimeout = time.Millisecond
		candidates := parsePackageCandidates(t,
			"first.example/lunafox/engine@"+artifact.manifestDigest.String(),
			"second.example/lunafox/engine@"+artifact.manifestDigest.String(),
		)

		_, err := puller.ConsumePackageLayer(context.Background(), candidates, func(attemptCtx context.Context, layer PulledPackageLayer) error {
			<-attemptCtx.Done()
			_, readErr := io.Copy(io.Discard, layer.Reader)
			return readErr
		})
		if !errors.Is(err, context.DeadlineExceeded) || ociartifact.IsCandidateUnavailable(err) {
			t.Fatalf("ConsumePackageLayer() error = %v, want non-candidate deadline failure", err)
		}
		if second.resolveCalls != 0 {
			t.Fatalf("second candidate resolve calls = %d, post-stream deadline must not fallback", second.resolveCalls)
		}
	})
}

func TestORASPackagePullerCallerCancellationNeverFallsBack(t *testing.T) {
	artifact := newPackageArtifactFixture(t, []byte("package"), ociartifact.EnginePackageArtifactType, ociartifact.EnginePackageLayerMediaType)
	first := artifact.repository()
	second := artifact.repository()
	puller := newTestORASPackagePuller(t, map[string]*fakePackageRepository{
		"first.example": first, "second.example": second,
	}, &recordingPackageSignatureVerifier{})
	candidates := parsePackageCandidates(t,
		"first.example/lunafox/engine@"+artifact.manifestDigest.String(),
		"second.example/lunafox/engine@"+artifact.manifestDigest.String(),
	)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := puller.ConsumePackageLayer(ctx, candidates, readAllPackageLayer)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("ConsumePackageLayer() error = %v, want context.Canceled", err)
	}
	if first.resolveCalls != 0 || second.resolveCalls != 0 {
		t.Fatalf("resolve calls first=%d second=%d, want zero", first.resolveCalls, second.resolveCalls)
	}
}

func readAllPackageLayer(_ context.Context, layer PulledPackageLayer) error {
	_, err := io.Copy(io.Discard, layer.Reader)
	return err
}

type packageArtifactFixture struct {
	manifest       []byte
	manifestDigest digest.Digest
	layer          []byte
	layerDigest    digest.Digest
}

type localRegistryFixture struct {
	*httptest.Server
	layerRequests atomic.Int32
}

func newLocalRegistryFixture(t *testing.T, manifest []byte, manifestDigest, layerDigest digest.Digest, layer []byte) *localRegistryFixture {
	t.Helper()
	fixture := &localRegistryFixture{}
	fixture.Server = httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		switch {
		case request.URL.Path == "/v2/":
			writer.WriteHeader(http.StatusOK)
		case strings.Contains(request.URL.Path, "/manifests/"):
			writer.Header().Set("Content-Type", ocispec.MediaTypeImageManifest)
			writer.Header().Set("Docker-Content-Digest", manifestDigest.String())
			_, _ = writer.Write(manifest)
		case strings.Contains(request.URL.Path, "/blobs/"):
			if request.URL.Path == "/v2/lunafox/test/blobs/"+layerDigest.String() {
				fixture.layerRequests.Add(1)
			}
			writer.Header().Set("Docker-Content-Digest", layerDigest.String())
			_, _ = writer.Write(layer)
		default:
			http.NotFound(writer, request)
		}
	}))
	fixture.Config.ErrorLog = nil
	return fixture
}

func newPackageArtifactFixture(t *testing.T, layer []byte, artifactType, layerMediaType string) packageArtifactFixture {
	t.Helper()
	layer = append([]byte(nil), layer...)
	layerDigest := digest.FromBytes(layer)
	emptyConfig := ocispec.DescriptorEmptyJSON
	manifest := ocispec.Manifest{
		Versioned:    specs.Versioned{SchemaVersion: 2},
		MediaType:    ocispec.MediaTypeImageManifest,
		ArtifactType: artifactType,
		Config: ocispec.Descriptor{
			MediaType: emptyConfig.MediaType,
			Digest:    emptyConfig.Digest,
			Size:      emptyConfig.Size,
		},
		Layers: []ocispec.Descriptor{{
			MediaType: layerMediaType,
			Digest:    layerDigest,
			Size:      int64(len(layer)),
		}},
	}
	manifestPayload, err := json.Marshal(manifest)
	if err != nil {
		t.Fatal(err)
	}
	return packageArtifactFixture{
		manifest:       manifestPayload,
		manifestDigest: digest.FromBytes(manifestPayload),
		layer:          layer,
		layerDigest:    layerDigest,
	}
}

func mutatePackageArtifactFixture(t *testing.T, artifact *packageArtifactFixture, mutate func(*ocispec.Manifest)) {
	t.Helper()
	var manifest ocispec.Manifest
	if err := json.Unmarshal(artifact.manifest, &manifest); err != nil {
		t.Fatalf("unmarshal OCI manifest fixture: %v", err)
	}
	mutate(&manifest)
	payload, err := json.Marshal(manifest)
	if err != nil {
		t.Fatalf("marshal OCI manifest fixture: %v", err)
	}
	artifact.manifest = payload
	artifact.manifestDigest = digest.FromBytes(payload)
}

func (artifact packageArtifactFixture) repository() *fakePackageRepository {
	return &fakePackageRepository{
		manifestDescriptor: ocispec.Descriptor{
			MediaType: ocispec.MediaTypeImageManifest,
			Digest:    artifact.manifestDigest,
			Size:      int64(len(artifact.manifest)),
		},
		manifest: append([]byte(nil), artifact.manifest...),
		layer:    append([]byte(nil), artifact.layer...),
	}
}

type fakePackageRepository struct {
	manifestDescriptor ocispec.Descriptor
	manifest           []byte
	layer              []byte
	resolveErr         error
	manifestFetchErr   error
	layerFetchErr      error
	manifestReader     func() io.ReadCloser
	layerReader        func() io.ReadCloser
	resolveCalls       int
	manifestFetches    int
	layerFetches       int
}

// recordingPackageSignatureVerifier remains only as an ignored legacy test
// argument while callers are migrated to the signature-free installer.
type recordingPackageSignatureVerifier struct {
	calls            int
	errorsByRegistry map[string]error
}

func (repository *fakePackageRepository) Resolve(context.Context, string) (ocispec.Descriptor, error) {
	repository.resolveCalls++
	return repository.manifestDescriptor, repository.resolveErr
}

func (repository *fakePackageRepository) Fetch(_ context.Context, descriptor ocispec.Descriptor) (io.ReadCloser, error) {
	if descriptor.MediaType == ocispec.MediaTypeImageManifest {
		repository.manifestFetches++
		if repository.manifestFetchErr != nil {
			return nil, repository.manifestFetchErr
		}
		if repository.manifestReader != nil {
			return repository.manifestReader(), nil
		}
		return io.NopCloser(bytes.NewReader(repository.manifest)), nil
	}
	repository.layerFetches++
	if repository.layerFetchErr != nil {
		return nil, repository.layerFetchErr
	}
	if repository.layerReader != nil {
		return repository.layerReader(), nil
	}
	return io.NopCloser(bytes.NewReader(repository.layer)), nil
}

type failingPackageLayerReader struct {
	payload  []byte
	err      error
	closeErr error
	done     bool
}

func (reader *failingPackageLayerReader) Read(payload []byte) (int, error) {
	if len(reader.payload) > 0 {
		n := copy(payload, reader.payload)
		reader.payload = reader.payload[n:]
		return n, nil
	}
	if reader.done {
		return 0, io.EOF
	}
	reader.done = true
	return 0, reader.err
}

func (reader *failingPackageLayerReader) Close() error { return reader.closeErr }

func newTestORASPackagePuller(
	t *testing.T,
	repositories map[string]*fakePackageRepository,
	_ ...any,
) *ORASPackagePuller {
	t.Helper()
	puller, err := NewORASPackagePuller(ORASPackagePullerOptions{
		MaxManifestBytes:     2048,
		MaxPackageLayerBytes: 4096,
		PerCandidateTimeout:  time.Second,
	})
	if err != nil {
		t.Fatal(err)
	}
	puller.newRepository = func(reference ociartifact.EnginePackageArtifactReference, _ bool) (packageRepository, error) {
		repository := repositories[reference.Registry]
		if repository == nil {
			return nil, fmt.Errorf("unexpected test Registry %q", reference.Registry)
		}
		return repository, nil
	}
	return puller
}

func parsePackageCandidates(t *testing.T, refs ...string) ociartifact.ArtifactCandidates {
	t.Helper()
	candidates, err := ociartifact.ParseArtifactCandidates(refs)
	if err != nil {
		t.Fatal(err)
	}
	return candidates
}

func stringsContainAny(value string, candidates ...string) bool {
	for _, candidate := range candidates {
		if bytes.Contains([]byte(value), []byte(candidate)) {
			return true
		}
	}
	return false
}
