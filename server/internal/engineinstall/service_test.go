package engineinstall

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/yyhuni/lunafox/contracts/enginemanifest/packagemanifest"
	"github.com/yyhuni/lunafox/contracts/enginemanifest/repositoryname"
	"github.com/yyhuni/lunafox/contracts/ociartifact"
)

const (
	serviceArtifactDigestA = "sha256:1111111111111111111111111111111111111111111111111111111111111111"
	serviceArtifactDigestB = "sha256:2222222222222222222222222222222222222222222222222222222222222222"
	serviceArtifactDigestC = "sha256:3333333333333333333333333333333333333333333333333333333333333333"
)

func TestNewEnginePackageInstallerRequiresEveryDependency(t *testing.T) {
	puller := &servicePackagePullerStub{}
	cache := CacheInstaller{}
	runtimeVerifier := &serviceRuntimeImageVerifierStub{}
	tests := []struct {
		name            string
		puller          packageLayerConsumerPuller
		cache           packageCacheStager
		runtimeVerifier runtimeImageIndexVerifier
	}{
		{name: "puller", cache: cache, runtimeVerifier: runtimeVerifier},
		{name: "cache", puller: puller, runtimeVerifier: runtimeVerifier},
		{name: "Runtime Image verifier", puller: puller, cache: cache},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, err := NewEnginePackageInstaller(test.puller, test.cache, test.runtimeVerifier); err == nil {
				t.Fatalf("NewEnginePackageInstaller() accepted nil %s", test.name)
			}
		})
	}
	if _, err := NewEnginePackageInstaller(puller, cache, runtimeVerifier); err != nil {
		t.Fatalf("NewEnginePackageInstaller() error = %v", err)
	}
}

func TestEnginePackageInstallerReturnsOnlyVerifiedRegistrationFacts(t *testing.T) {
	engineID := repositoryname.FirstPartyEngineIDPortScan
	fixture := newServicePackageFixture(t, engineID)
	candidates := servicePackageCandidates(t, engineID, serviceArtifactDigestA, "registry.example")
	puller := &servicePackagePullerStub{
		fixtures: map[ociartifact.ArtifactManifestDigest]servicePackageFixture{
			candidates.ArtifactManifestDigest: fixture,
		},
	}
	runtimeVerifier := &serviceRuntimeImageVerifierStub{}
	cache, _ := newServicePackageCache(t)
	installer, err := NewEnginePackageInstaller(puller, cache, runtimeVerifier)
	if err != nil {
		t.Fatal(err)
	}

	installed, err := installer.Install(context.Background(), candidates)
	if err != nil {
		t.Fatalf("Install() error = %v", err)
	}
	if installed.EngineID != engineID ||
		installed.Publisher != "lunafox" ||
		installed.PackageVersion != "1.2.3" ||
		installed.ArtifactRef != candidates.References[0].String() ||
		installed.PackageDigest != fixture.packageDigest {
		t.Fatalf("unexpected registration facts: %#v", installed)
	}
	var projection map[string]any
	if err := json.Unmarshal(installed.EngineManifestProjection, &projection); err != nil {
		t.Fatalf("decode EngineManifestProjection: %v", err)
	}
	if projection["engineId"] != engineID || projection["publisher"] != "lunafox" {
		t.Fatalf("unexpected engine manifest projection: %#v", projection)
	}
	if !reflect.DeepEqual(puller.attemptedRefs, []string{candidates.References[0].String()}) {
		t.Fatalf("package candidate attempts = %#v", puller.attemptedRefs)
	}
	if len(runtimeVerifier.calls) != 1 ||
		runtimeVerifier.calls[0].engineID != engineID ||
		!reflect.DeepEqual(runtimeVerifier.calls[0].refs, fixture.runtimeImageRefs) {
		t.Fatalf("Runtime Image verifier calls = %#v, want engineId %q refs %#v", runtimeVerifier.calls, engineID, fixture.runtimeImageRefs)
	}
}

func TestVerifiedEngineInstallationHasNoDerivedIdentityOrCacheSiblings(t *testing.T) {
	outputType := reflect.TypeOf(VerifiedEngineInstallation{})
	got := make([]string, 0, outputType.NumField())
	for index := 0; index < outputType.NumField(); index++ {
		got = append(got, outputType.Field(index).Name)
	}
	want := []string{
		"EngineID",
		"Publisher",
		"PackageVersion",
		"ArtifactRef",
		"PackageDigest",
		"EngineManifestProjection",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("VerifiedEngineInstallation fields = %#v, want exact registration facts %#v", got, want)
	}
}

func TestEnginePackageInstallerRuntimeFailureCannotPromoteOrFallbackPackageCandidate(t *testing.T) {
	engineID := repositoryname.FirstPartyEngineIDPortScan
	fixture := newServicePackageFixture(t, engineID)
	candidates := servicePackageCandidates(
		t,
		engineID,
		serviceArtifactDigestA,
		"first.example",
		"second.example",
	)
	puller := &servicePackagePullerStub{
		fixtures: map[ociartifact.ArtifactManifestDigest]servicePackageFixture{
			candidates.ArtifactManifestDigest: fixture,
		},
	}
	runtimeFailure := ociartifact.NewCandidateFailure(
		ociartifact.CandidateFailureReasonNotFound,
		errors.New("Runtime Image candidate unavailable"),
	)
	runtimeVerifier := &serviceRuntimeImageVerifierStub{
		errorsByEngineID: map[string]error{engineID: runtimeFailure},
	}
	cache, cacheRoot := newServicePackageCache(t)
	installer, err := NewEnginePackageInstaller(puller, cache, runtimeVerifier)
	if err != nil {
		t.Fatal(err)
	}

	_, err = installer.Install(context.Background(), candidates)
	if err == nil || !strings.Contains(err.Error(), "Runtime Image candidate unavailable") {
		t.Fatalf("Install() error = %v, want Runtime Image verification failure", err)
	}
	if ociartifact.IsCandidateUnavailable(err) {
		t.Fatalf("Runtime Image candidate failure escaped package-dependent validation: %v", err)
	}
	if !reflect.DeepEqual(puller.attemptedRefs, []string{candidates.References[0].String()}) {
		t.Fatalf("package puller advanced after Runtime Image failure: %#v", puller.attemptedRefs)
	}
	assertPackageCanonicalCacheAbsent(t, cacheRoot, fixture.packageDigest)
}

func TestEnginePackageInstallerVerifiesGHCRBeforeCloudflarePackageDownload(t *testing.T) {
	refs := []string{
		"docker.lunafox.cc.cd/yyhuni/lunafox-engine-port-scan@" + serviceArtifactDigestA,
		"docker.io/yyhuni/lunafox-engine-port-scan@" + serviceArtifactDigestA,
		"ghcr.io/yyhuni/lunafox-engine-port-scan@" + serviceArtifactDigestA,
	}
	candidates, err := ociartifact.ParseArtifactCandidates(refs)
	if err != nil {
		t.Fatalf("parse accelerated candidates: %v", err)
	}
	puller := &servicePackagePullerStub{}
	cache, _ := newServicePackageCache(t)
	runtimeVerifier := &serviceRuntimeImageVerifierStub{}
	signatureVerifier := &recordingDigestReferenceSignatureVerifier{err: errors.New("signature unavailable")}
	installer, err := NewEnginePackageInstaller(puller, cache, runtimeVerifier, signatureVerifier)
	if err != nil {
		t.Fatal(err)
	}

	_, err = installer.Install(context.Background(), candidates)
	if err == nil || !strings.Contains(err.Error(), "verify Engine Package GHCR signature") {
		t.Fatalf("Install() signature failure = %v", err)
	}
	wantSignature := "ghcr.io/yyhuni/lunafox-engine-port-scan@" + serviceArtifactDigestA
	if len(signatureVerifier.references) != 1 || signatureVerifier.references[0].String() != wantSignature {
		t.Fatalf("signature references = %#v, want %q", signatureVerifier.references, wantSignature)
	}
	if len(puller.attemptedRefs) != 0 || len(runtimeVerifier.calls) != 0 {
		t.Fatalf("failed GHCR verification must stop before downloads: pulls=%#v runtime=%#v", puller.attemptedRefs, runtimeVerifier.calls)
	}
}

func TestEnginePackageInstallerPreservesCallerCancellationFromRuntimeVerification(t *testing.T) {
	engineID := repositoryname.FirstPartyEngineIDPortScan
	fixture := newServicePackageFixture(t, engineID)
	candidates := servicePackageCandidates(t, engineID, serviceArtifactDigestA, "registry.example")
	puller := &servicePackagePullerStub{
		fixtures: map[ociartifact.ArtifactManifestDigest]servicePackageFixture{
			candidates.ArtifactManifestDigest: fixture,
		},
	}
	ctx, cancel := context.WithCancel(context.Background())
	runtimeVerifier := &cancelingServiceRuntimeImageVerifier{cancel: cancel}
	cache, cacheRoot := newServicePackageCache(t)
	installer, err := NewEnginePackageInstaller(puller, cache, runtimeVerifier)
	if err != nil {
		t.Fatal(err)
	}

	_, err = installer.Install(ctx, candidates)
	if !errors.Is(err, context.Canceled) || ociartifact.IsCandidateUnavailable(err) {
		t.Fatalf("Install() error = %v, want caller cancellation without package fallback", err)
	}
	assertPackageCanonicalCacheAbsent(t, cacheRoot, fixture.packageDigest)
}

func TestEnginePackageInstallerInstallInventoryPreservesDeclarationOrder(t *testing.T) {
	engineIDs := []string{
		repositoryname.FirstPartyEngineIDWebsiteDiscovery,
		repositoryname.FirstPartyEngineIDSubdomainDiscovery,
		repositoryname.FirstPartyEngineIDPortScan,
	}
	artifactDigests := []string{
		serviceArtifactDigestA,
		serviceArtifactDigestB,
		serviceArtifactDigestC,
	}
	inventory := &Inventory{EnginePackages: make([]InventoryPackage, 0, len(engineIDs))}
	fixtures := make(map[ociartifact.ArtifactManifestDigest]servicePackageFixture, len(engineIDs))
	wantPackageRefs := make([]string, 0, len(engineIDs))
	for index, engineID := range engineIDs {
		candidates := servicePackageCandidates(t, engineID, artifactDigests[index], "registry.example")
		inventory.EnginePackages = append(inventory.EnginePackages, InventoryPackage{Candidates: candidates})
		fixtures[candidates.ArtifactManifestDigest] = newServicePackageFixture(t, engineID)
		wantPackageRefs = append(wantPackageRefs, candidates.References[0].String())
	}

	puller := &servicePackagePullerStub{fixtures: fixtures}
	runtimeVerifier := &serviceRuntimeImageVerifierStub{}
	cache, _ := newServicePackageCache(t)
	installer, err := NewEnginePackageInstaller(puller, cache, runtimeVerifier)
	if err != nil {
		t.Fatal(err)
	}

	installed, err := installer.InstallInventory(context.Background(), inventory)
	if err != nil {
		t.Fatalf("InstallInventory() error = %v", err)
	}
	gotEngineIDs := make([]string, 0, len(installed))
	for _, item := range installed {
		gotEngineIDs = append(gotEngineIDs, item.EngineID)
	}
	if !reflect.DeepEqual(gotEngineIDs, engineIDs) {
		t.Fatalf("installed engine order = %#v, want declaration order %#v", gotEngineIDs, engineIDs)
	}
	if !reflect.DeepEqual(puller.attemptedRefs, wantPackageRefs) {
		t.Fatalf("package attempt order = %#v, want %#v", puller.attemptedRefs, wantPackageRefs)
	}
	gotRuntimeEngineIDs := make([]string, 0, len(runtimeVerifier.calls))
	for _, call := range runtimeVerifier.calls {
		gotRuntimeEngineIDs = append(gotRuntimeEngineIDs, call.engineID)
	}
	if !reflect.DeepEqual(gotRuntimeEngineIDs, engineIDs) {
		t.Fatalf("Runtime Image verification order = %#v, want %#v", gotRuntimeEngineIDs, engineIDs)
	}
}

func TestEnginePackageInstallerInstallInventoryStopsAtFirstError(t *testing.T) {
	engineIDs := []string{
		repositoryname.FirstPartyEngineIDWebsiteDiscovery,
		repositoryname.FirstPartyEngineIDSubdomainDiscovery,
		repositoryname.FirstPartyEngineIDPortScan,
	}
	artifactDigests := []string{
		serviceArtifactDigestA,
		serviceArtifactDigestB,
		serviceArtifactDigestC,
	}
	inventory := &Inventory{EnginePackages: make([]InventoryPackage, 0, len(engineIDs))}
	fixtures := make(map[ociartifact.ArtifactManifestDigest]servicePackageFixture, len(engineIDs))
	wantAttemptedRefs := make([]string, 0, 2)
	var failedFixture servicePackageFixture
	for index, engineID := range engineIDs {
		candidates := servicePackageCandidates(t, engineID, artifactDigests[index], "registry.example")
		fixture := newServicePackageFixture(t, engineID)
		inventory.EnginePackages = append(inventory.EnginePackages, InventoryPackage{Candidates: candidates})
		fixtures[candidates.ArtifactManifestDigest] = fixture
		if index < 2 {
			wantAttemptedRefs = append(wantAttemptedRefs, candidates.References[0].String())
		}
		if index == 1 {
			failedFixture = fixture
		}
	}

	puller := &servicePackagePullerStub{fixtures: fixtures}
	runtimeVerifier := &serviceRuntimeImageVerifierStub{
		errorsByEngineID: map[string]error{
			engineIDs[1]: errors.New("second package Runtime Image rejected"),
		},
	}
	cache, cacheRoot := newServicePackageCache(t)
	installer, err := NewEnginePackageInstaller(puller, cache, runtimeVerifier)
	if err != nil {
		t.Fatal(err)
	}

	installed, err := installer.InstallInventory(context.Background(), inventory)
	if err == nil || !strings.Contains(err.Error(), "second package Runtime Image rejected") {
		t.Fatalf("InstallInventory() error = %v, want second-item failure", err)
	}
	if installed != nil {
		t.Fatalf("InstallInventory() returned partial registration facts: %#v", installed)
	}
	if !reflect.DeepEqual(puller.attemptedRefs, wantAttemptedRefs) {
		t.Fatalf("package attempts after failure = %#v, want %#v", puller.attemptedRefs, wantAttemptedRefs)
	}
	if len(runtimeVerifier.calls) != 2 {
		t.Fatalf("Runtime Image verifier calls = %d, want stop after second item", len(runtimeVerifier.calls))
	}
	assertPackageCanonicalCacheAbsent(t, cacheRoot, failedFixture.packageDigest)
}

type servicePackageFixture struct {
	archive          []byte
	packageDigest    ociartifact.PackageDigest
	size             int64
	runtimeImageRefs []string
}

func newServicePackageFixture(t *testing.T, engineID string) servicePackageFixture {
	t.Helper()
	runtimeRepository, err := repositoryname.FirstPartyRuntimeImageRepositoryName(engineID)
	if err != nil {
		t.Fatal(err)
	}
	entries := validPackageCacheEntries()
	for index := range entries {
		payload := string(entries[index].Payload)
		payload = strings.ReplaceAll(payload, repositoryname.FirstPartyEngineIDPortScan, engineID)
		payload = strings.ReplaceAll(payload, "lunafox-engine-runtime-port-scan", runtimeRepository)
		entries[index].Payload = []byte(payload)
	}
	archive := packageArchiveBytesForTest(t, entries, time.Time{})
	packageDigest, size, err := packagemanifest.ComputePackageDigest(bytes.NewReader(archive))
	if err != nil {
		t.Fatal(err)
	}
	return servicePackageFixture{
		archive:          archive,
		packageDigest:    packageDigest,
		size:             size,
		runtimeImageRefs: []string{"docker.io/lunafox/" + runtimeRepository + "@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"},
	}
}

func servicePackageCandidates(
	t *testing.T,
	engineID string,
	artifactDigest string,
	registries ...string,
) ociartifact.ArtifactCandidates {
	t.Helper()
	repository, err := repositoryname.FirstPartyOCIRepositoryName(engineID)
	if err != nil {
		t.Fatal(err)
	}
	refs := make([]string, 0, len(registries))
	for _, registry := range registries {
		refs = append(refs, registry+"/lunafox/"+repository+"@"+artifactDigest)
	}
	candidates, err := ociartifact.ParseArtifactCandidates(refs)
	if err != nil {
		t.Fatal(err)
	}
	return candidates
}

func newServicePackageCache(t *testing.T) (CacheInstaller, string) {
	t.Helper()
	root := t.TempDir()
	t.Cleanup(func() { _ = removePackageCacheTree(root) })
	return CacheInstaller{Root: root, MaxArchiveBytes: 1 << 20}, root
}

type servicePackagePullerStub struct {
	fixtures      map[ociartifact.ArtifactManifestDigest]servicePackageFixture
	attemptedRefs []string
}

func (puller *servicePackagePullerStub) ConsumePackageLayer(
	ctx context.Context,
	candidates ociartifact.ArtifactCandidates,
	consume PackageLayerConsumer,
) (ConsumedPackageLayer, error) {
	fixture, exists := puller.fixtures[candidates.ArtifactManifestDigest]
	if !exists {
		return ConsumedPackageLayer{}, fmt.Errorf(
			"missing service test package fixture for artifact digest %q",
			candidates.ArtifactManifestDigest,
		)
	}
	var lastUnavailable error
	for _, reference := range candidates.References {
		puller.attemptedRefs = append(puller.attemptedRefs, reference.String())
		layer := PulledPackageLayer{
			Reference:              reference,
			ArtifactManifestDigest: candidates.ArtifactManifestDigest,
			PackageDigest:          fixture.packageDigest,
			Size:                   fixture.size,
			Reader:                 bytes.NewReader(fixture.archive),
		}
		if err := consume(ctx, layer); err != nil {
			if !ociartifact.IsCandidateUnavailable(err) {
				return ConsumedPackageLayer{}, err
			}
			lastUnavailable = err
			continue
		}
		return ConsumedPackageLayer{
			Reference:              reference,
			ArtifactManifestDigest: candidates.ArtifactManifestDigest,
			PackageDigest:          fixture.packageDigest,
			Size:                   fixture.size,
		}, nil
	}
	return ConsumedPackageLayer{}, fmt.Errorf("all service test package candidates unavailable: %w", lastUnavailable)
}

type serviceRuntimeImageVerifierCall struct {
	engineID string
	refs     []string
}

type serviceRuntimeImageVerifierStub struct {
	calls            []serviceRuntimeImageVerifierCall
	errorsByEngineID map[string]error
}

type cancelingServiceRuntimeImageVerifier struct {
	cancel context.CancelFunc
}

func (verifier *cancelingServiceRuntimeImageVerifier) Verify(
	ctx context.Context,
	_ string,
	_ []string,
) (VerifiedRuntimeImageIndex, error) {
	verifier.cancel()
	<-ctx.Done()
	return VerifiedRuntimeImageIndex{}, ctx.Err()
}

func (verifier *serviceRuntimeImageVerifierStub) Verify(
	_ context.Context,
	engineID string,
	refs []string,
) (VerifiedRuntimeImageIndex, error) {
	verifier.calls = append(verifier.calls, serviceRuntimeImageVerifierCall{
		engineID: engineID,
		refs:     append([]string(nil), refs...),
	})
	if err := verifier.errorsByEngineID[engineID]; err != nil {
		return VerifiedRuntimeImageIndex{}, err
	}
	return VerifiedRuntimeImageIndex{}, nil
}
