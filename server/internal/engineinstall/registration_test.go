package engineinstall

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/yyhuni/lunafox/contracts/enginemanifest/repositoryname"
	"github.com/yyhuni/lunafox/contracts/ociartifact"
	catalogdomain "github.com/yyhuni/lunafox/server/internal/modules/catalog/domain"
)

const (
	registrationArtifactDigestA = "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	registrationArtifactDigestB = "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
	registrationArtifactDigestC = "sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc"
	registrationPackageDigest   = "sha256:dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd"
)

func TestNewEngineRegistrationServiceRequiresDependencies(t *testing.T) {
	installer := &registrationInstallerStub{}
	repository := &registrationRepositoryStub{}
	var typedNilInstaller *registrationInstallerStub
	var typedNilRepository *registrationRepositoryStub
	tests := []struct {
		name       string
		installer  VerifiedEngineInstaller
		repository catalogdomain.InstalledEngineCommandRepository
	}{
		{name: "nil installer", repository: repository},
		{name: "nil repository", installer: installer},
		{name: "typed nil installer", installer: typedNilInstaller, repository: repository},
		{name: "typed nil repository", installer: installer, repository: typedNilRepository},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, err := NewEngineRegistrationService(test.installer, test.repository); err == nil {
				t.Fatalf("NewEngineRegistrationService() accepted %s", test.name)
			}
		})
	}
	if _, err := NewEngineRegistrationService(installer, repository); err != nil {
		t.Fatalf("NewEngineRegistrationService() error = %v", err)
	}
}

func TestEngineRegistrationServiceMapsExactFactsAndReturnsDetachedRecord(t *testing.T) {
	engineID := repositoryname.FirstPartyEngineIDPortScan
	manifest := json.RawMessage(`{"manifestVersion":"engine.v5","engineId":"engine.lunafox.port_scan"}`)
	verified := VerifiedEngineInstallation{
		EngineID:                 engineID,
		Publisher:                "lunafox",
		PackageVersion:           "1.2.3",
		ArtifactRef:              "registry.example/yyhuni/lunafox-engine-port-scan@" + registrationArtifactDigestA,
		PackageDigest:            ociartifact.PackageDigest(registrationPackageDigest),
		EngineManifestProjection: manifest,
	}
	order := []string{}
	installer := &registrationInstallerStub{
		resultsByEngineID: map[string]VerifiedEngineInstallation{engineID: verified},
		order:             &order,
	}
	repository := &registrationRepositoryStub{order: &order, mutateInput: true}
	service, err := NewEngineRegistrationService(installer, repository)
	if err != nil {
		t.Fatal(err)
	}
	candidates := registrationCandidates(t, registrationArtifactDigestA)
	type contextKey string
	ctx := context.WithValue(context.Background(), contextKey("request"), "registration-v2")

	record, err := service.Install(ctx, candidates, false)
	if err != nil {
		t.Fatalf("Install() error = %v", err)
	}
	want := catalogdomain.Engine{
		EngineID:       engineID,
		Publisher:      "lunafox",
		PackageVersion: "1.2.3",
		ArtifactRef:    verified.ArtifactRef,
		PackageDigest:  registrationPackageDigest,
		Manifest:       append(json.RawMessage(nil), manifest...),
	}
	if !reflect.DeepEqual(*record, want) {
		t.Fatalf("registered record = %#v, want %#v", *record, want)
	}
	if len(installer.calls) != 1 || installer.calls[0].ctx != ctx || installer.calls[0].engineID != engineID ||
		!reflect.DeepEqual(installer.calls[0].candidates, candidates) {
		t.Fatalf("installer calls = %#v, want exact context/identity/candidates", installer.calls)
	}
	if repository.calls != 1 || repository.contexts[0] != ctx || !reflect.DeepEqual(repository.snapshots[0], want) {
		t.Fatalf("repository calls=%d contexts=%#v snapshots=%#v", repository.calls, repository.contexts, repository.snapshots)
	}
	if !reflect.DeepEqual(order, []string{"install:" + engineID, "upsert:" + engineID}) {
		t.Fatalf("application order = %#v", order)
	}
	if repository.inputs[0] == record {
		t.Fatal("repository input and returned record must be detached pointers")
	}
	manifest[0] = 'X'
	if record.Manifest[0] == 'X' || repository.snapshots[0].Manifest[0] == 'X' {
		t.Fatal("verified manifest backing array leaked into persistence or return record")
	}
	record.Manifest[0] = 'Y'
	if repository.snapshots[0].Manifest[0] == 'Y' {
		t.Fatal("returned manifest aliases the repository record")
	}
}

func TestEngineRegistrationServiceUpstreamFailuresNeverWriteRepository(t *testing.T) {
	tests := []struct {
		name string
		err  error
	}{
		{name: "identity", err: errors.New("identity mismatch")},
		{name: "package verification", err: errors.New("package integrity failure")},
		{name: "cache promotion", err: errors.New("cache promotion failure")},
		{name: "Runtime Image", err: errors.New("Runtime Image verification failure")},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			engineID := repositoryname.FirstPartyEngineIDPortScan
			installer := &registrationInstallerStub{errorsByEngineID: map[string]error{engineID: test.err}}
			repository := &registrationRepositoryStub{}
			service, err := NewEngineRegistrationService(installer, repository)
			if err != nil {
				t.Fatal(err)
			}

			_, err = service.Install(context.Background(), registrationCandidates(t, registrationArtifactDigestA), false)
			if !errors.Is(err, test.err) {
				t.Fatalf("Install() error = %v, want %v", err, test.err)
			}
			if repository.calls != 0 {
				t.Fatalf("repository calls = %d after upstream failure", repository.calls)
			}
		})
	}
}

func TestEngineRegistrationServiceCancellationAfterVerificationDoesNotWrite(t *testing.T) {
	engineID := repositoryname.FirstPartyEngineIDPortScan
	ctx, cancel := context.WithCancel(context.Background())
	installer := &registrationInstallerStub{
		resultsByEngineID: map[string]VerifiedEngineInstallation{engineID: registrationVerifiedInstallation(engineID, registrationArtifactDigestA)},
		cancelAfterCall:   cancel,
	}
	repository := &registrationRepositoryStub{}
	service, err := NewEngineRegistrationService(installer, repository)
	if err != nil {
		t.Fatal(err)
	}

	_, err = service.Install(ctx, registrationCandidates(t, registrationArtifactDigestA), false)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("Install() error = %v, want context.Canceled", err)
	}
	if repository.calls != 0 {
		t.Fatalf("repository calls = %d after cancellation", repository.calls)
	}
}

func TestEngineRegistrationServiceWrapsRepositoryFailure(t *testing.T) {
	engineID := repositoryname.FirstPartyEngineIDPortScan
	databaseErr := errors.New("database unavailable")
	installer := &registrationInstallerStub{
		resultsByEngineID: map[string]VerifiedEngineInstallation{engineID: registrationVerifiedInstallation(engineID, registrationArtifactDigestA)},
	}
	repository := &registrationRepositoryStub{err: databaseErr}
	service, err := NewEngineRegistrationService(installer, repository)
	if err != nil {
		t.Fatal(err)
	}

	record, err := service.Install(context.Background(), registrationCandidates(t, registrationArtifactDigestA), false)
	if record != nil || !errors.Is(err, databaseErr) || !strings.Contains(err.Error(), "register verified Engine Package v2") {
		t.Fatalf("Install() record=%#v error=%v, want wrapped database failure", record, err)
	}
	if repository.calls != 1 {
		t.Fatalf("repository calls = %d, want 1", repository.calls)
	}
}

func TestEngineRegistrationServiceRegistersInventoryInDeclarationOrder(t *testing.T) {
	engineIDs := []string{
		repositoryname.FirstPartyEngineIDWebsiteDiscovery,
		repositoryname.FirstPartyEngineIDSubdomainDiscovery,
		repositoryname.FirstPartyEngineIDPortScan,
	}
	digests := []string{registrationArtifactDigestA, registrationArtifactDigestB, registrationArtifactDigestC}
	results := make(map[string]VerifiedEngineInstallation, len(engineIDs))
	inventory := &Inventory{EnginePackages: make([]InventoryPackage, 0, len(engineIDs))}
	for index, engineID := range engineIDs {
		results[engineID] = registrationVerifiedInstallation(engineID, digests[index])
		inventory.EnginePackages = append(inventory.EnginePackages, InventoryPackage{Candidates: registrationCandidates(t, digests[index])})
	}
	order := []string{}
	installer := &registrationInstallerStub{resultsByEngineID: results, order: &order}
	repository := &registrationRepositoryStub{order: &order}
	service, err := NewEngineRegistrationService(installer, repository)
	if err != nil {
		t.Fatal(err)
	}

	records, err := service.InstallInventory(context.Background(), inventory, false)
	if err != nil {
		t.Fatalf("InstallInventory() error = %v", err)
	}
	gotEngineIDs := make([]string, 0, len(records))
	for _, record := range records {
		gotEngineIDs = append(gotEngineIDs, record.EngineID)
	}
	if !reflect.DeepEqual(gotEngineIDs, engineIDs) {
		t.Fatalf("registered engine order = %#v, want %#v", gotEngineIDs, engineIDs)
	}
	wantOrder := make([]string, 0, len(engineIDs)*2)
	for _, engineID := range engineIDs {
		wantOrder = append(wantOrder, "install:"+engineID, "upsert:"+engineID)
	}
	if !reflect.DeepEqual(order, wantOrder) {
		t.Fatalf("application order = %#v, want %#v", order, wantOrder)
	}
}

func TestEngineRegistrationServiceInventoryPropagatesReplacementPolicy(t *testing.T) {
	engineID := repositoryname.FirstPartyEngineIDPortScan
	inventory := &Inventory{EnginePackages: []InventoryPackage{{
		Candidates: registrationCandidates(t, registrationArtifactDigestA),
	}}}

	for _, allowReplacement := range []bool{false, true} {
		t.Run(fmt.Sprintf("allow replacement %t", allowReplacement), func(t *testing.T) {
			installer := &registrationInstallerStub{resultsByEngineID: map[string]VerifiedEngineInstallation{
				engineID: registrationVerifiedInstallation(engineID, registrationArtifactDigestA),
			}}
			repository := &registrationRepositoryStub{}
			service, err := NewEngineRegistrationService(installer, repository)
			if err != nil {
				t.Fatal(err)
			}

			if _, err := service.InstallInventory(context.Background(), inventory, allowReplacement); err != nil {
				t.Fatalf("InstallInventory() error = %v", err)
			}
			if !reflect.DeepEqual(repository.allowReplacements, []bool{allowReplacement}) {
				t.Fatalf("replacement policies = %#v, want %#v", repository.allowReplacements, []bool{allowReplacement})
			}
		})
	}
}

type registrationInstallerCall struct {
	ctx        context.Context
	engineID   string
	candidates ociartifact.ArtifactCandidates
}

type registrationInstallerStub struct {
	resultsByEngineID map[string]VerifiedEngineInstallation
	errorsByEngineID  map[string]error
	calls             []registrationInstallerCall
	order             *[]string
	cancelAfterCall   context.CancelFunc
}

func (installer *registrationInstallerStub) Install(
	ctx context.Context,
	candidates ociartifact.ArtifactCandidates,
) (VerifiedEngineInstallation, error) {
	for engineID, result := range installer.resultsByEngineID {
		if len(candidates.References) > 0 && strings.HasSuffix(result.ArtifactRef, string(candidates.References[0].ArtifactManifestDigest)) {
			return installer.record(ctx, engineID, candidates)
		}
	}
	if len(installer.resultsByEngineID) == 1 {
		for engineID := range installer.resultsByEngineID {
			return installer.record(ctx, engineID, candidates)
		}
	}
	if len(installer.errorsByEngineID) == 1 {
		for engineID := range installer.errorsByEngineID {
			return installer.record(ctx, engineID, candidates)
		}
	}
	return VerifiedEngineInstallation{}, errors.New("missing stub installation result")
}

func (installer *registrationInstallerStub) record(
	ctx context.Context,
	engineID string,
	candidates ociartifact.ArtifactCandidates,
) (VerifiedEngineInstallation, error) {
	installer.calls = append(installer.calls, registrationInstallerCall{ctx: ctx, engineID: engineID, candidates: candidates})
	if installer.order != nil {
		*installer.order = append(*installer.order, "install:"+engineID)
	}
	if installer.cancelAfterCall != nil {
		installer.cancelAfterCall()
	}
	if err := installer.errorsByEngineID[engineID]; err != nil {
		return VerifiedEngineInstallation{}, err
	}
	return installer.resultsByEngineID[engineID], nil
}

type registrationRepositoryStub struct {
	err               error
	calls             int
	contexts          []context.Context
	inputs            []*catalogdomain.Engine
	snapshots         []catalogdomain.Engine
	allowReplacements []bool
	order             *[]string
	mutateInput       bool
}

func (repository *registrationRepositoryStub) UpsertInstalledEngine(ctx context.Context, record *catalogdomain.Engine, allowReplacement bool) error {
	repository.calls++
	repository.contexts = append(repository.contexts, ctx)
	repository.inputs = append(repository.inputs, record)
	repository.snapshots = append(repository.snapshots, cloneEngineRegistration(*record))
	repository.allowReplacements = append(repository.allowReplacements, allowReplacement)
	if repository.order != nil {
		*repository.order = append(*repository.order, "upsert:"+record.EngineID)
	}
	if repository.mutateInput {
		record.EngineID = "mutated.by.repository"
		if len(record.Manifest) > 0 {
			record.Manifest[0] = 'R'
		}
	}
	return repository.err
}

func registrationVerifiedInstallation(engineID, artifactDigest string) VerifiedEngineInstallation {
	return VerifiedEngineInstallation{
		EngineID:                 engineID,
		Publisher:                "lunafox",
		PackageVersion:           "1.2.3",
		ArtifactRef:              "registry.example/yyhuni/lunafox-engine-package@" + artifactDigest,
		PackageDigest:            ociartifact.PackageDigest(registrationPackageDigest),
		EngineManifestProjection: json.RawMessage(`{"manifestVersion":"engine.v5"}`),
	}
}

func registrationCandidates(t *testing.T, artifactDigest string) ociartifact.ArtifactCandidates {
	t.Helper()
	candidates, err := ociartifact.ParseArtifactCandidates([]string{
		"registry.example/yyhuni/lunafox-engine-package@" + artifactDigest,
	})
	if err != nil {
		t.Fatal(err)
	}
	return candidates
}
