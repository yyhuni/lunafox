package application

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/yyhuni/lunafox/contracts/releasemanifest"
	"github.com/yyhuni/lunafox/server/internal/modules/upgrade/domain"
)

type upgradeAuthorizerStub struct{ allowed bool }

func (stub upgradeAuthorizerStub) IsActiveSuperuser(context.Context, int) (bool, error) {
	return stub.allowed, nil
}

func TestUpgradeServiceRequiresActiveSuperuserForEveryBoundary(t *testing.T) {
	service, _ := newUpgradeServiceForTest(t, &upgradeDispatcherStub{})
	service.authorizer = upgradeAuthorizerStub{allowed: false}
	if _, err := service.CheckForUpdates(context.Background(), 7); !errors.Is(err, domain.ErrUpgradeUnauthorized) {
		t.Fatalf("check authorization error = %v", err)
	}
	if _, _, err := service.CreateOperation(context.Background(), 7, CreateUpgradeOperationInput{RequestID: uuidTestRequestID, Confirmed: true}); !errors.Is(err, domain.ErrUpgradeUnauthorized) {
		t.Fatalf("create authorization error = %v", err)
	}
	if _, err := service.GetOperation(context.Background(), 7, "11111111-1111-4111-8111-111111111111"); !errors.Is(err, domain.ErrUpgradeUnauthorized) {
		t.Fatalf("get authorization error = %v", err)
	}
}

func TestUpgradeServicePreDispatchCancellationCompletesBeforeHostHandoff(t *testing.T) {
	dispatcher := &upgradeDispatcherStub{}
	service, repository := newUpgradeServiceForTest(t, dispatcher)
	order := []string{}
	service.coordinator = preDispatchCoordinatorStub{prepare: func(context.Context) (PreparationResult, error) {
		order = append(order, "cancel")
		return PreparationResult{CancelledScanCount: 3, CancelledTaskCount: 4}, nil
	}}
	dispatcher.onDispatch = func(HostUpgradeRequest) { order = append(order, "dispatch") }
	manifest, _ := LoadReleaseManifest(fixturePath("release.manifest.yaml"), "")
	operation, _, err := service.CreateOperation(context.Background(), 7, CreateUpgradeOperationInput{RequestID: uuidTestRequestID, ManifestID: manifest.Upgrade.ManifestID, ManifestDigest: manifest.Digest(), Confirmed: true})
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.Join(order, ","); got != "cancel,dispatch" {
		t.Fatalf("handoff order = %q", got)
	}
	if operation.CancelledScanCount != 3 || operation.CancelledTaskCount != 4 || repository.updates < 2 {
		t.Fatalf("cancellation evidence = %#v updates=%d", operation, repository.updates)
	}
}

func TestUpgradeServiceDoesNotWaitForNaturalTaskCompletionBeforeDispatch(t *testing.T) {
	dispatcher := &upgradeDispatcherStub{}
	naturalCompletion := make(chan struct{})
	service, _ := newUpgradeServiceForTest(t, dispatcher)
	service.coordinator = preDispatchCoordinatorStub{prepare: func(context.Context) (PreparationResult, error) {
		// The coordinator represents the committed cancellation fence. The
		// underlying Agent process is intentionally still running; handoff must
		// proceed without waiting for that process to drain naturally.
		select {
		case <-naturalCompletion:
			t.Fatalf("test fixture unexpectedly completed the natural task")
		default:
		}
		return PreparationResult{CancelledScanCount: 1, CancelledTaskCount: 1}, nil
	}}
	dispatcher.onDispatch = func(HostUpgradeRequest) {
		select {
		case <-naturalCompletion:
			t.Fatalf("host dispatch waited for natural task completion")
		default:
		}
	}
	manifest, err := LoadReleaseManifest(fixturePath("release.manifest.yaml"), "")
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := service.CreateOperation(context.Background(), 7, CreateUpgradeOperationInput{
		RequestID: uuidTestRequestID, ManifestID: manifest.Upgrade.ManifestID, ManifestDigest: manifest.Digest(), Confirmed: true,
	}); err != nil {
		t.Fatal(err)
	}
	close(naturalCompletion)
}

func TestUpgradeServiceCheckReturnsMigrationGateDiagnostic(t *testing.T) {
	manifest, err := LoadReleaseManifest(fixturePath("release.manifest.yaml"), "")
	if err != nil {
		t.Fatal(err)
	}
	manifestWithMigration := *manifest
	manifestWithMigration.Upgrade.DatabaseMigration.HasDatabaseMigration = true
	manifestWithMigration.Upgrade.DatabaseMigration.MigrationType = "compatible"
	manifestWithMigration.Upgrade.DatabaseMigration.MigrationID = "000002_upgrade"
	manifestWithMigration.Upgrade.DatabaseMigration.Checksum = "sha256:" + strings.Repeat("b", 64)
	policy, err := LoadMigrationPolicy(fixturePath("migration-policy.json"))
	if err != nil {
		t.Fatal(err)
	}
	repository := &upgradeRepositoryStub{byRequest: map[string]*domain.Operation{}, byID: map[string]*domain.Operation{}}
	service, err := NewService(ServiceConfig{
		ManifestSource:        ManifestSourceFunc(func() (*releasemanifest.Manifest, error) { return &manifestWithMigration, nil }),
		MigrationPolicySource: StaticMigrationPolicySource{Policy: policy},
		Repository:            repository,
		Authorizer:            upgradeAuthorizerStub{allowed: true},
		Dispatcher:            &upgradeDispatcherStub{},
		CurrentVersion:        "1.0.0",
	})
	if err != nil {
		t.Fatal(err)
	}
	result, err := service.CheckForUpdates(context.Background(), 7)
	if err != nil {
		t.Fatal(err)
	}
	if result.Eligible || result.Diagnostic == nil {
		t.Fatalf("migration gate result = %#v, want ineligible diagnostic", result)
	}
	if result.Diagnostic.Code != domain.ErrorCodeMigrationPolicyDisallowsDataRetention || result.Diagnostic.Stage != "migration" {
		t.Fatalf("migration diagnostic = %#v", result.Diagnostic)
	}
	if !strings.Contains(result.Diagnostic.Reason, "phase-transition") {
		t.Fatalf("migration diagnostic reason = %q", result.Diagnostic.Reason)
	}
}

func TestUpgradeServiceRetryRequiresConfirmationAndPreservesTargetOnMismatch(t *testing.T) {
	service, repository := newUpgradeServiceForTest(t, &upgradeDispatcherStub{})
	manifest, err := LoadReleaseManifest(fixturePath("release.manifest.yaml"), "")
	if err != nil {
		t.Fatal(err)
	}
	operation, _, err := service.CreateOperation(context.Background(), 7, CreateUpgradeOperationInput{
		RequestID: uuidTestRequestID, ManifestID: manifest.Upgrade.ManifestID, ManifestDigest: manifest.Digest(), Confirmed: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	operation.Status = domain.StatusFailed
	if err := repository.Update(context.Background(), operation); err != nil {
		t.Fatal(err)
	}
	if _, err := service.RetryOperation(context.Background(), 7, operation.OperationID, false); !errors.Is(err, domain.ErrUpgradeConfirmationRequired) {
		t.Fatalf("unconfirmed retry error = %v", err)
	}
	// A changed server-owned manifest cannot silently redirect a repair.
	changed := *manifest
	changed.Upgrade.ManifestID = "lunafox-1.2.4"
	service.manifestSource = ManifestSourceFunc(func() (*releasemanifest.Manifest, error) { return &changed, nil })
	if _, err := service.RetryOperation(context.Background(), 7, operation.OperationID, true); !errors.Is(err, domain.ErrReleaseManifestTargetMismatch) {
		t.Fatalf("changed-target retry error = %v", err)
	}
}

type preDispatchCoordinatorStub struct {
	prepare func(context.Context) (PreparationResult, error)
}

func (stub preDispatchCoordinatorStub) Prepare(ctx context.Context) (PreparationResult, error) {
	return stub.prepare(ctx)
}

type upgradeDispatcherStub struct {
	requests   []HostUpgradeRequest
	err        error
	onDispatch func(HostUpgradeRequest)
}

func (stub *upgradeDispatcherStub) Dispatch(_ context.Context, request HostUpgradeRequest) error {
	stub.requests = append(stub.requests, request)
	if stub.onDispatch != nil {
		stub.onDispatch(request)
	}
	return stub.err
}

type upgradeRepositoryStub struct {
	byRequest map[string]*domain.Operation
	byID      map[string]*domain.Operation
	active    *domain.Operation
	updates   int
}

func (stub *upgradeRepositoryStub) CreateOrGet(_ context.Context, operation *domain.Operation) (*domain.Operation, bool, error) {
	if existing := stub.byRequest[operation.RequestID]; existing != nil {
		if existing.ManifestDigest != operation.ManifestDigest {
			return nil, false, domain.ErrUpgradeRequestConflict
		}
		return existing, false, nil
	}
	if stub.active != nil && !stub.active.Status.IsTerminal() {
		return nil, false, domain.ErrUpgradeAlreadyRunning
	}
	stub.byRequest[operation.RequestID] = operation
	stub.byID[operation.OperationID] = operation
	stub.active = operation
	return operation, true, nil
}

func (stub *upgradeRepositoryStub) Get(_ context.Context, id string) (*domain.Operation, error) {
	operation := stub.byID[id]
	if operation == nil {
		return nil, domain.ErrUpgradeNotFound
	}
	return operation, nil
}

func (stub *upgradeRepositoryStub) GetByRequest(_ context.Context, requestID string) (*domain.Operation, error) {
	operation := stub.byRequest[requestID]
	if operation == nil {
		return nil, domain.ErrUpgradeNotFound
	}
	return operation, nil
}

func (stub *upgradeRepositoryStub) Update(_ context.Context, operation *domain.Operation) error {
	stub.updates++
	stub.byID[operation.OperationID] = operation
	stub.byRequest[operation.RequestID] = operation
	stub.active = operation
	return nil
}

func (stub *upgradeRepositoryStub) FindActive(context.Context) (*domain.Operation, error) {
	if stub.active == nil || stub.active.Status.IsTerminal() {
		return nil, errors.New("not found")
	}
	return stub.active, nil
}

func newUpgradeServiceForTest(t *testing.T, dispatcher HostUpgradeDispatcher) (*Service, *upgradeRepositoryStub) {
	t.Helper()
	manifest, err := LoadReleaseManifest(fixturePath("release.manifest.yaml"), "")
	if err != nil {
		t.Fatal(err)
	}
	policy, err := LoadMigrationPolicy(fixturePath("migration-policy.json"))
	if err != nil {
		t.Fatal(err)
	}
	repository := &upgradeRepositoryStub{byRequest: map[string]*domain.Operation{}, byID: map[string]*domain.Operation{}}
	service, err := NewService(ServiceConfig{
		ManifestSource:        ManifestSourceFunc(func() (*releasemanifest.Manifest, error) { return manifest, nil }),
		MigrationPolicySource: StaticMigrationPolicySource{Policy: policy},
		Repository:            repository,
		Authorizer:            upgradeAuthorizerStub{allowed: true},
		Dispatcher:            dispatcher,
		CurrentVersion:        "1.0.0",
		Now:                   func() time.Time { return time.Date(2026, 9, 13, 12, 0, 0, 0, time.UTC) },
	})
	if err != nil {
		t.Fatal(err)
	}
	return service, repository
}

func TestCreateOperationRequiresConfirmationAndDispatchesOnlyOnce(t *testing.T) {
	manifest, err := LoadReleaseManifest(fixturePath("release.manifest.yaml"), "")
	if err != nil {
		t.Fatal(err)
	}
	dispatcher := &upgradeDispatcherStub{}
	service, _ := newUpgradeServiceForTest(t, dispatcher)
	input := CreateUpgradeOperationInput{
		RequestID: uuidTestRequestID, ManifestID: manifest.Upgrade.ManifestID, ManifestDigest: manifest.Digest(), Confirmed: true,
	}
	operation, created, err := service.CreateOperation(context.Background(), 7, input)
	if err != nil || !created || operation == nil {
		t.Fatalf("CreateOperation() = operation=%#v created=%t err=%v", operation, created, err)
	}
	replayed, created, err := service.CreateOperation(context.Background(), 7, input)
	if err != nil || created || replayed.OperationID != operation.OperationID {
		t.Fatalf("replay = operation=%#v created=%t err=%v", replayed, created, err)
	}
	if len(dispatcher.requests) != 1 {
		t.Fatalf("host dispatch count = %d, want 1", len(dispatcher.requests))
	}
	if dispatcher.requests[0].ManifestDigest != manifest.Digest() || dispatcher.requests[0].Action != HostUpgradeActionStart {
		t.Fatalf("host handoff = %#v", dispatcher.requests[0])
	}
}

func TestCreateOperationRejectsUnconfirmedAndClientImageReference(t *testing.T) {
	manifest, err := LoadReleaseManifest(fixturePath("release.manifest.yaml"), "")
	if err != nil {
		t.Fatal(err)
	}
	service, _ := newUpgradeServiceForTest(t, &upgradeDispatcherStub{})
	base := CreateUpgradeOperationInput{RequestID: uuidTestRequestID, ManifestID: manifest.Upgrade.ManifestID, ManifestDigest: manifest.Digest()}
	if _, _, err := service.CreateOperation(context.Background(), 7, base); !errors.Is(err, domain.ErrUpgradeConfirmationRequired) {
		t.Fatalf("unconfirmed create error = %v", err)
	}
	base.Confirmed = true
	base.ImageRef = "docker.io/example/app:latest"
	if _, _, err := service.CreateOperation(context.Background(), 7, base); !errors.Is(err, domain.ErrReleaseManifestTargetInvalid) {
		t.Fatalf("image ref create error = %v", err)
	}
}

func TestCreateOperationRejectsWhenTargetReleaseIsAlreadyCurrent(t *testing.T) {
	manifest, err := LoadReleaseManifest(fixturePath("release.manifest.yaml"), "")
	if err != nil {
		t.Fatal(err)
	}
	service, repository := newUpgradeServiceForTest(t, &upgradeDispatcherStub{})
	service.currentVersion = manifest.ReleaseVersion

	operation, created, err := service.CreateOperation(context.Background(), 7, CreateUpgradeOperationInput{
		RequestID: uuidTestRequestID, ManifestID: manifest.Upgrade.ManifestID, ManifestDigest: manifest.Digest(), Confirmed: true,
	})
	if operation != nil || created || !errors.Is(err, domain.ErrUpgradeNoUpdateAvailable) || domain.CodeOf(err) != domain.ErrorCodeUpgradeNoUpdateAvailable {
		t.Fatalf("same-version create = operation=%#v created=%t err=%v code=%q", operation, created, err, domain.CodeOf(err))
	}
	if len(repository.byID) != 0 {
		t.Fatalf("same-version create persisted an operation: %#v", repository.byID)
	}
}

func TestRetryOperationKeepsManifestTarget(t *testing.T) {
	manifest, err := LoadReleaseManifest(fixturePath("../testdata/release.manifest.yaml"), "")
	if err != nil {
		t.Fatal(err)
	}
	dispatcher := &upgradeDispatcherStub{}
	service, repository := newUpgradeServiceForTest(t, dispatcher)
	operation, _, err := service.CreateOperation(context.Background(), 7, CreateUpgradeOperationInput{RequestID: uuidTestRequestID, ManifestID: manifest.Upgrade.ManifestID, ManifestDigest: manifest.Digest(), Confirmed: true})
	if err != nil {
		t.Fatal(err)
	}
	operation.Status = domain.StatusFailed
	if err := repository.Update(context.Background(), operation); err != nil {
		t.Fatal(err)
	}
	retried, err := service.RetryOperation(context.Background(), 7, operation.OperationID, true)
	if err != nil {
		t.Fatal(err)
	}
	if retried.ManifestDigest != operation.ManifestDigest || retried.Status != domain.StatusQueued {
		t.Fatalf("retried operation changed target/status: %#v", retried)
	}
	if len(dispatcher.requests) != 2 || dispatcher.requests[1].Action != HostUpgradeActionRepair {
		t.Fatalf("retry dispatches = %#v", dispatcher.requests)
	}
}

const uuidTestRequestID = "22222222-2222-4222-8222-222222222222"

func TestUpgradeFixtureIsReadable(t *testing.T) {
	if _, err := os.Stat(fixturePath("release.manifest.yaml")); err != nil {
		t.Fatal(err)
	}
}
