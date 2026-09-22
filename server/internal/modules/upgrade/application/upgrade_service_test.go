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

func TestUpgradeServiceFrontendOnlyPlanSkipsAgentAndCancellationLifecycle(t *testing.T) {
	planDigest := "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	baselineDigest := "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
	dispatcher := &scopePlanningDispatcherStub{plan: HostUpgradeScopePlan{
		ExecutionMode: domain.ExecutionModeFrontendOnly, PlanDigest: planDigest,
		BaselineDeploymentDigest: baselineDigest, TouchedServices: []string{"frontend"},
		ConfirmedDeploymentVersion: "1.0.0",
	}}
	service, repository := newUpgradeServiceForTest(t, dispatcher)
	agentSource := &scopeAgentSourceStub{}
	service.agentSource = agentSource
	coordinatorCalls := 0
	service.coordinator = preDispatchCoordinatorStub{prepare: func(context.Context) (PreparationResult, error) {
		coordinatorCalls++
		return PreparationResult{}, nil
	}}
	manifest, err := LoadReleaseManifest(fixturePath("release.manifest.yaml"), "")
	if err != nil {
		t.Fatal(err)
	}
	operation, created, err := service.CreateOperation(context.Background(), 7, CreateUpgradeOperationInput{
		RequestID: uuidTestRequestID, ManifestID: manifest.Upgrade.ManifestID, ManifestDigest: manifest.Digest(), Confirmed: true,
	})
	if err != nil || !created {
		t.Fatalf("CreateOperation() operation=%#v created=%t err=%v", operation, created, err)
	}
	if dispatcher.planCalls != 1 || len(dispatcher.requests) != 1 || agentSource.snapshotCalls != 0 || coordinatorCalls != 0 {
		t.Fatalf("frontend-only side effects plan=%d dispatch=%d agentSnapshots=%d coordinator=%d", dispatcher.planCalls, len(dispatcher.requests), agentSource.snapshotCalls, coordinatorCalls)
	}
	if operation.ExecutionMode != domain.ExecutionModeFrontendOnly || operation.WorkDisposition != domain.WorkDispositionNotRequired || operation.Status != domain.StatusQueued {
		t.Fatalf("frontend-only operation facts = %#v", operation)
	}
	if len(operation.AgentExpectations) != 0 || operation.AgentVerificationDeadline != nil || operation.AgentSummary != (domain.AgentSummary{}) {
		t.Fatalf("frontend-only operation contains Agent lifecycle data: %#v", operation)
	}
	if request := dispatcher.requests[0]; request.PlanDigest != planDigest || request.ExecutionMode != domain.ExecutionModeFrontendOnly || len(request.TouchedServices) != 1 || request.TouchedServices[0] != "frontend" {
		t.Fatalf("frontend-only host handoff = %#v", request)
	}
	if repository.byID[operation.OperationID].WorkDisposition != domain.WorkDispositionNotRequired {
		t.Fatalf("frontend-only work disposition was not persisted: %#v", repository.byID[operation.OperationID])
	}
}

func TestUpgradeServiceRejectsMalformedV2PlanBeforeSideEffects(t *testing.T) {
	dispatcher := &scopePlanningDispatcherStub{plan: HostUpgradeScopePlan{
		ExecutionMode: domain.ExecutionModeFrontendOnly, PlanDigest: "not-a-digest",
		BaselineDeploymentDigest: "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
		TouchedServices:          []string{"frontend"}, ConfirmedDeploymentVersion: "1.0.0",
	}}
	service, repository := newUpgradeServiceForTest(t, dispatcher)
	agentSource := &scopeAgentSourceStub{}
	service.agentSource = agentSource
	coordinatorCalls := 0
	service.coordinator = preDispatchCoordinatorStub{prepare: func(context.Context) (PreparationResult, error) {
		coordinatorCalls++
		return PreparationResult{}, nil
	}}
	manifest, err := LoadReleaseManifest(fixturePath("release.manifest.yaml"), "")
	if err != nil {
		t.Fatal(err)
	}
	operation, created, err := service.CreateOperation(context.Background(), 7, CreateUpgradeOperationInput{
		RequestID: uuidTestRequestID, ManifestID: manifest.Upgrade.ManifestID, ManifestDigest: manifest.Digest(), Confirmed: true,
	})
	if operation != nil || created || !errors.Is(err, domain.ErrUpgradeHostUnavailable) {
		t.Fatalf("malformed plan result operation=%#v created=%t err=%v", operation, created, err)
	}
	if dispatcher.planCalls != 1 || len(dispatcher.requests) != 0 || agentSource.snapshotCalls != 0 || coordinatorCalls != 0 || len(repository.byID) != 0 {
		t.Fatalf("malformed plan had side effects: plan=%d dispatch=%d agent=%d coordinator=%d rows=%d", dispatcher.planCalls, len(dispatcher.requests), agentSource.snapshotCalls, coordinatorCalls, len(repository.byID))
	}
}

func TestUpgradeServiceCompositionUnavailableFallsBackToLegacyFullHandoff(t *testing.T) {
	dispatcher := &scopePlanningDispatcherStub{planErr: ErrHostUpgradeScopePlanningFullOnly}
	service, repository := newUpgradeServiceForTest(t, dispatcher)
	coordinatorCalls := 0
	service.coordinator = preDispatchCoordinatorStub{prepare: func(context.Context) (PreparationResult, error) {
		coordinatorCalls++
		return PreparationResult{CancelledScanCount: 2, CancelledTaskCount: 1}, nil
	}}
	manifest, err := LoadReleaseManifest(fixturePath("release.manifest.yaml"), "")
	if err != nil {
		t.Fatal(err)
	}

	operation, created, err := service.CreateOperation(context.Background(), 7, CreateUpgradeOperationInput{
		RequestID: uuidTestRequestID, ManifestID: manifest.Upgrade.ManifestID, ManifestDigest: manifest.Digest(), Confirmed: true,
	})
	if err != nil || !created {
		t.Fatalf("CreateOperation() operation=%#v created=%t err=%v", operation, created, err)
	}
	if dispatcher.planCalls != 1 || coordinatorCalls != 1 || len(dispatcher.requests) != 1 {
		t.Fatalf("composition fallback side effects plan=%d coordinator=%d dispatches=%d", dispatcher.planCalls, coordinatorCalls, len(dispatcher.requests))
	}
	if operation.ExecutionMode != "" || operation.PlanDigest != "" || operation.WorkDisposition != domain.WorkDispositionCancelled {
		t.Fatalf("composition fallback persisted v2 evidence = %#v", operation)
	}
	if request := dispatcher.requests[0]; request.ExecutionMode != "" || request.PlanDigest != "" || len(request.TouchedServices) != 0 {
		t.Fatalf("composition fallback request = %#v, want schema-v1 full handoff", request)
	}
	if repository.byID[operation.OperationID].Status != domain.StatusStopping {
		t.Fatalf("composition fallback operation did not retain full lifecycle = %#v", repository.byID[operation.OperationID])
	}
}

func TestUpgradeServiceMigrationRequestsAFullV2PlanAfterEligibility(t *testing.T) {
	planDigest := "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	dispatcher := &scopePlanningDispatcherStub{plan: HostUpgradeScopePlan{
		ExecutionMode: domain.ExecutionModeFull, PlanDigest: planDigest,
		TouchedServices: []string{"agent", "bootstrap", "engine", "engine_package", "engine_runtime", "frontend", "migration", "nginx", "server"},
	}}
	service, _ := newUpgradeServiceForTest(t, dispatcher)
	manifest, err := LoadReleaseManifest(fixturePath("release.manifest.yaml"), "")
	if err != nil {
		t.Fatal(err)
	}
	candidate := *manifest
	candidate.Upgrade.DatabaseMigration.HasDatabaseMigration = true
	candidate.Upgrade.DatabaseMigration.MigrationType = "compatible"
	candidate.Upgrade.DatabaseMigration.MigrationID = "000002_upgrade"
	candidate.Upgrade.DatabaseMigration.Checksum = "sha256:" + strings.Repeat("b", 64)
	service.manifestSource = ManifestSourceFunc(func() (*releasemanifest.Manifest, error) { return &candidate, nil })
	policy, err := LoadMigrationPolicy(fixturePath("migration-policy.json"))
	if err != nil {
		t.Fatal(err)
	}
	policy.PreserveDataUpgrade = true
	policy.DataRetainingDeploymentAllowed = true
	service.migrationPolicySource = StaticMigrationPolicySource{Policy: policy}

	operation, created, err := service.CreateOperation(context.Background(), 7, CreateUpgradeOperationInput{
		RequestID: uuidTestRequestID, ManifestID: candidate.Upgrade.ManifestID, ManifestDigest: candidate.Digest(), Confirmed: true,
	})
	if err != nil || !created {
		t.Fatalf("CreateOperation() operation=%#v created=%t err=%v", operation, created, err)
	}
	if dispatcher.planCalls != 1 || !dispatcher.planRequest.RequireFull {
		t.Fatalf("migration scope request = %#v, calls=%d", dispatcher.planRequest, dispatcher.planCalls)
	}
	if operation.ExecutionMode != domain.ExecutionModeFull || operation.PlanDigest != planDigest {
		t.Fatalf("migration operation scope = %#v", operation)
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

func TestUpgradeServiceProjectsVerifiedReleaseNotes(t *testing.T) {
	service, _ := newUpgradeServiceForTest(t, &upgradeDispatcherStub{})
	result, err := service.CheckForUpdates(context.Background(), 7)
	if err != nil {
		t.Fatal(err)
	}
	if result.Manifest.ReleaseNotes == nil {
		t.Fatal("CheckForUpdates() omitted release notes")
	}
	if result.Manifest.ReleaseNotes.Digest != "sha256:4406112ce062dd05feacce5f519b8cb7250fd01c7237c43e0f7335da912e8188" {
		t.Fatalf("release notes digest = %q", result.Manifest.ReleaseNotes.Digest)
	}
	if result.Manifest.ReleaseNotes.Body != "## English\n\n- Test release notes.\n\n## 简体中文\n\n- 测试发布说明。\n" {
		t.Fatalf("release notes body = %q", result.Manifest.ReleaseNotes.Body)
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

type deploymentStateDispatcherStub struct {
	upgradeDispatcherStub
	state    HostDeploymentState
	stateErr error
}

func (stub *deploymentStateDispatcherStub) ConfirmedDeploymentState(context.Context) (HostDeploymentState, error) {
	return stub.state, stub.stateErr
}

type candidateAvailabilityDispatcherStub struct {
	upgradeDispatcherStub
	state             HostDeploymentState
	stateErr          error
	stateCalls        int
	availability      HostCandidateAvailability
	availabilityErr   error
	availabilityCalls int
}

func (stub *candidateAvailabilityDispatcherStub) ConfirmedDeploymentState(context.Context) (HostDeploymentState, error) {
	stub.stateCalls++
	return stub.state, stub.stateErr
}

func (stub *candidateAvailabilityDispatcherStub) CandidateAvailability(context.Context, string) (HostCandidateAvailability, error) {
	stub.availabilityCalls++
	return stub.availability, stub.availabilityErr
}

type scopePlanningDispatcherStub struct {
	upgradeDispatcherStub
	plan        HostUpgradeScopePlan
	planErr     error
	planCalls   int
	planRequest HostUpgradeScopePlanRequest
}

func (stub *scopePlanningDispatcherStub) PlanUpgradeScope(_ context.Context, request HostUpgradeScopePlanRequest) (HostUpgradeScopePlan, error) {
	stub.planCalls++
	stub.planRequest = request
	return stub.plan, stub.planErr
}

type scopeAgentSourceStub struct{ snapshotCalls int }

func (stub *scopeAgentSourceStub) Snapshot(context.Context, AgentUpgradeTarget) ([]domain.AgentExpectation, error) {
	stub.snapshotCalls++
	return []domain.AgentExpectation{}, nil
}

func (stub *scopeAgentSourceStub) NotifyUpdateRequired(context.Context, AgentUpgradeTarget, []domain.AgentExpectation) error {
	return nil
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

func TestUpgradeServiceOnlyOffersStrictlyNewerSemanticVersion(t *testing.T) {
	tests := map[string]struct {
		current   string
		hasUpdate bool
	}{
		"older candidate":     {current: "1.2.4", hasUpdate: false},
		"same candidate":      {current: "1.2.3", hasUpdate: false},
		"stable after canary": {current: "1.2.3-alpha.1", hasUpdate: true},
		"newer candidate":     {current: "1.2.2", hasUpdate: true},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			service, _ := newUpgradeServiceForTest(t, &upgradeDispatcherStub{})
			service.currentVersion = test.current
			result, err := service.CheckForUpdates(context.Background(), 7)
			if err != nil {
				t.Fatal(err)
			}
			if result.HasUpdate != test.hasUpdate {
				t.Fatalf("HasUpdate = %t, want %t", result.HasUpdate, test.hasUpdate)
			}
		})
	}
}

func TestUpgradeServiceCandidateAvailabilityUsesConfirmedReleaseFloor(t *testing.T) {
	manifest, err := LoadReleaseManifest(fixturePath("release.manifest.yaml"), "")
	if err != nil {
		t.Fatal(err)
	}
	otherDigest := "sha256:" + strings.Repeat("a", 64)
	stateDigest := "sha256:" + strings.Repeat("b", 64)
	tests := []struct {
		name       string
		state      HostDeploymentState
		stateErr   error
		wantUpdate bool
		wantError  string
	}{
		{
			name:       "confirmed deployment is newer",
			state:      HostDeploymentState{ReleaseVersion: "1.3.0", ManifestDigest: otherDigest, StateDigest: stateDigest},
			wantUpdate: false,
		},
		{
			name:       "confirmed deployment is equal with same manifest",
			state:      HostDeploymentState{ReleaseVersion: manifest.ReleaseVersion, ManifestDigest: manifest.Digest(), StateDigest: stateDigest},
			wantUpdate: false,
		},
		{
			name:      "confirmed deployment is equal with conflicting manifest",
			state:     HostDeploymentState{ReleaseVersion: manifest.ReleaseVersion, ManifestDigest: otherDigest, StateDigest: stateDigest},
			wantError: ErrHostDeploymentStateConflict.Error(),
		},
		{
			name:       "confirmed deployment is older",
			state:      HostDeploymentState{ReleaseVersion: "1.1.0", ManifestDigest: otherDigest, StateDigest: stateDigest},
			wantUpdate: true,
		},
		{
			name:       "legacy host fallback",
			stateErr:   ErrHostDeploymentStateUnsupported,
			wantUpdate: true,
		},
		{
			name:       "baseline unavailable fallback",
			stateErr:   ErrHostDeploymentStateUnavailable,
			wantUpdate: true,
		},
		{
			name:      "malformed confirmed release",
			state:     HostDeploymentState{ReleaseVersion: "not-semver", ManifestDigest: otherDigest, StateDigest: stateDigest},
			wantError: "confirmed host deployment state release version is invalid",
		},
		{
			name:      "missing confirmed manifest identity",
			state:     HostDeploymentState{ReleaseVersion: "1.1.0", StateDigest: stateDigest},
			wantError: "confirmed host deployment state manifest identity is invalid",
		},
		{
			name:      "malformed confirmed state digest",
			state:     HostDeploymentState{ReleaseVersion: "1.1.0", ManifestDigest: otherDigest, StateDigest: "not-a-digest"},
			wantError: "confirmed host deployment state digest is invalid",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			dispatcher := &deploymentStateDispatcherStub{
				upgradeDispatcherStub: upgradeDispatcherStub{},
				state:                 test.state,
				stateErr:              test.stateErr,
			}
			service, _ := newUpgradeServiceForTest(t, dispatcher)
			got, gotErr := service.candidateAvailability(context.Background(), manifest)
			if test.wantError != "" {
				if gotErr == nil || !strings.Contains(gotErr.Error(), test.wantError) {
					t.Fatalf("candidateAvailability() error = %v, want substring %q", gotErr, test.wantError)
				}
				return
			}
			if gotErr != nil {
				t.Fatalf("candidateAvailability() error = %v", gotErr)
			}
			if got.HasUpdate != test.wantUpdate {
				t.Fatalf("candidateAvailability() = %#v, want HasUpdate=%t", got, test.wantUpdate)
			}
		})
	}
}

func TestUpgradeServiceInventoryAvailabilityOverridesRunningBinary(t *testing.T) {
	manifest, err := LoadReleaseManifest(fixturePath("release.manifest.yaml"), "")
	if err != nil {
		t.Fatal(err)
	}
	baselineDigest := "sha256:" + strings.Repeat("b", 64)
	confirmed := HostDeploymentState{ReleaseVersion: "9.0.0", ManifestDigest: "sha256:" + strings.Repeat("a", 64), StateDigest: baselineDigest}
	tests := []struct {
		name         string
		current      string
		availability HostCandidateAvailability
		wantUpdate   bool
		wantCurrent  string
		wantError    string
		stateCalls   int
	}{
		{
			name:    "already applied frontend stays hidden",
			current: "1.0.0",
			availability: HostCandidateAvailability{
				Decision: HostCandidateAvailabilityAlreadyApplied, BaselineDeploymentDigest: baselineDigest, ConfirmedDeploymentVersion: manifest.ReleaseVersion,
			},
			wantCurrent: manifest.ReleaseVersion,
		},
		{
			name:    "changed inventory is offered from confirmed version",
			current: "1.0.0",
			availability: HostCandidateAvailability{
				Decision: HostCandidateAvailabilityAvailable, BaselineDeploymentDigest: baselineDigest, ConfirmedDeploymentVersion: "1.2.0",
			},
			wantUpdate:  true,
			wantCurrent: "1.2.0",
		},
		{
			name:    "stale confirmed state cannot downgrade the running server",
			current: "9.0.0",
			availability: HostCandidateAvailability{
				Decision: HostCandidateAvailabilityAvailable, BaselineDeploymentDigest: baselineDigest, ConfirmedDeploymentVersion: "1.0.0",
			},
			wantCurrent: "1.0.0",
		},
		{
			name:    "empty running version still accepts an available inventory",
			current: "",
			availability: HostCandidateAvailability{
				Decision: HostCandidateAvailabilityAvailable, BaselineDeploymentDigest: baselineDigest, ConfirmedDeploymentVersion: "1.2.0",
			},
			wantUpdate:  true,
			wantCurrent: "1.2.0",
		},
		{
			name:    "not newer hides the candidate",
			current: "1.0.0",
			availability: HostCandidateAvailability{
				Decision: HostCandidateAvailabilityNotNewer, BaselineDeploymentDigest: baselineDigest, ConfirmedDeploymentVersion: "1.3.0",
			},
			wantCurrent: "1.3.0",
		},
		{
			name:         "fallback keeps the legacy confirmed floor",
			current:      "1.0.0",
			availability: HostCandidateAvailability{Decision: HostCandidateAvailabilityFallback},
			wantCurrent:  "1.0.0",
			stateCalls:   1,
		},
		{
			name:    "fallback evidence fails closed",
			current: "1.0.0",
			availability: HostCandidateAvailability{
				Decision: HostCandidateAvailabilityFallback, BaselineDeploymentDigest: baselineDigest, ConfirmedDeploymentVersion: "1.2.3",
			},
			wantError: "fallback host candidate availability contains confirmed deployment evidence",
		},
		{
			name:    "invalid running version fails closed",
			current: "not-semver",
			availability: HostCandidateAvailability{
				Decision: HostCandidateAvailabilityAvailable, BaselineDeploymentDigest: baselineDigest, ConfirmedDeploymentVersion: "1.2.0",
			},
			wantError: "current release version is invalid",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			dispatcher := &candidateAvailabilityDispatcherStub{
				state:        confirmed,
				availability: test.availability,
			}
			service, repository := newUpgradeServiceForTest(t, dispatcher)
			service.currentVersion = test.current
			result, checkErr := service.CheckForUpdates(context.Background(), 7)
			if test.wantError != "" {
				if checkErr == nil || !strings.Contains(checkErr.Error(), test.wantError) {
					t.Fatalf("CheckForUpdates() error = %v, want substring %q", checkErr, test.wantError)
				}
				if len(repository.byID) != 0 || len(dispatcher.requests) != 0 {
					t.Fatalf("failed availability check had side effects: operations=%d dispatches=%d", len(repository.byID), len(dispatcher.requests))
				}
				return
			}
			if checkErr != nil {
				t.Fatal(checkErr)
			}
			if result.HasUpdate != test.wantUpdate || result.CurrentVersion != test.wantCurrent {
				t.Fatalf("CheckForUpdates() = %#v, want update=%t current=%s", result, test.wantUpdate, test.wantCurrent)
			}
			if dispatcher.stateCalls != test.stateCalls {
				t.Fatalf("legacy state reads = %d, want %d", dispatcher.stateCalls, test.stateCalls)
			}
			if test.wantUpdate {
				return
			}
			operation, created, createErr := service.CreateOperation(context.Background(), 7, CreateUpgradeOperationInput{
				RequestID: uuidTestRequestID, ManifestID: manifest.Upgrade.ManifestID, ManifestDigest: manifest.Digest(), Confirmed: true,
			})
			if operation != nil || created || !errors.Is(createErr, domain.ErrUpgradeNoUpdateAvailable) {
				t.Fatalf("CreateOperation() operation=%#v created=%t err=%v, want no update", operation, created, createErr)
			}
			if len(repository.byID) != 0 || len(dispatcher.requests) != 0 {
				t.Fatalf("hidden candidate had side effects: operations=%d dispatches=%d", len(repository.byID), len(dispatcher.requests))
			}
		})
	}
}

func TestUpgradeServiceInventoryAvailabilityDoesNotFallBackAfterHostError(t *testing.T) {
	dispatcher := &candidateAvailabilityDispatcherStub{availabilityErr: errors.New("inventory comparison failed")}
	service, repository := newUpgradeServiceForTest(t, dispatcher)
	if _, err := service.CheckForUpdates(context.Background(), 7); err == nil || !strings.Contains(err.Error(), "read host candidate availability") || errors.Is(err, ErrHostCandidateAvailabilityUnsupported) {
		t.Fatalf("CheckForUpdates() error = %v, want hard availability failure", err)
	}
	if dispatcher.stateCalls != 0 || len(repository.byID) != 0 || len(dispatcher.requests) != 0 {
		t.Fatalf("hard availability failure fell back: stateCalls=%d operations=%d dispatches=%d", dispatcher.stateCalls, len(repository.byID), len(dispatcher.requests))
	}
}

func TestUpgradeServiceUsesDefinitiveCandidateInventoryAvailability(t *testing.T) {
	manifest, err := LoadReleaseManifest(fixturePath("release.manifest.yaml"), "")
	if err != nil {
		t.Fatal(err)
	}
	baselineDigest := "sha256:" + strings.Repeat("b", 64)
	tests := []struct {
		name            string
		current         string
		availability    HostCandidateAvailability
		availabilityErr error
		stateErr        error
		wantUpdate      bool
		wantCurrent     string
		wantError       string
	}{
		{
			name:    "metadata-only release is already applied",
			current: "1.0.0",
			availability: HostCandidateAvailability{
				Decision: HostCandidateAvailabilityAlreadyApplied, BaselineDeploymentDigest: baselineDigest, ConfirmedDeploymentVersion: manifest.ReleaseVersion,
			},
			wantCurrent: manifest.ReleaseVersion,
		},
		{
			name:    "same server version can repair missing frontend deployment",
			current: manifest.ReleaseVersion,
			availability: HostCandidateAvailability{
				Decision: HostCandidateAvailabilityAvailable, BaselineDeploymentDigest: baselineDigest, ConfirmedDeploymentVersion: "1.2.2",
			},
			wantUpdate:  true,
			wantCurrent: "1.2.2",
		},
		{
			name:    "host availability cannot downgrade newer running server",
			current: "1.3.0",
			availability: HostCandidateAvailability{
				Decision: HostCandidateAvailabilityAvailable, BaselineDeploymentDigest: baselineDigest, ConfirmedDeploymentVersion: "1.0.0",
			},
			wantCurrent: "1.0.0",
		},
		{
			name:         "host fallback retains running-server behavior",
			current:      "1.0.0",
			availability: HostCandidateAvailability{Decision: HostCandidateAvailabilityFallback},
			stateErr:     ErrHostDeploymentStateUnsupported,
			wantUpdate:   true,
			wantCurrent:  "1.0.0",
		},
		{
			name:            "unsupported host retains running-server behavior",
			current:         "1.0.0",
			availabilityErr: ErrHostCandidateAvailabilityUnsupported,
			stateErr:        ErrHostDeploymentStateUnsupported,
			wantUpdate:      true,
			wantCurrent:     "1.0.0",
		},
		{
			name:    "fallback with evidence is rejected",
			current: "1.0.0",
			availability: HostCandidateAvailability{
				Decision: HostCandidateAvailabilityFallback, BaselineDeploymentDigest: baselineDigest,
			},
			wantError: "fallback host candidate availability contains confirmed deployment evidence",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			dispatcher := &candidateAvailabilityDispatcherStub{availability: test.availability, availabilityErr: test.availabilityErr, stateErr: test.stateErr}
			service, _ := newUpgradeServiceForTest(t, dispatcher)
			service.currentVersion = test.current
			availability, gotErr := service.candidateAvailability(context.Background(), manifest)
			if test.wantError != "" {
				if gotErr == nil || !strings.Contains(gotErr.Error(), test.wantError) {
					t.Fatalf("candidateAvailability() error = %v, want substring %q", gotErr, test.wantError)
				}
				return
			}
			if gotErr != nil {
				t.Fatal(gotErr)
			}
			if availability.HasUpdate != test.wantUpdate || availability.CurrentVersion != test.wantCurrent {
				t.Fatalf("candidateAvailability() = %#v, want update=%t current=%q", availability, test.wantUpdate, test.wantCurrent)
			}
		})
	}
}

func TestUpgradeServiceDoesNotCreateOperationForAlreadyAppliedInventory(t *testing.T) {
	manifest, err := LoadReleaseManifest(fixturePath("release.manifest.yaml"), "")
	if err != nil {
		t.Fatal(err)
	}
	dispatcher := &candidateAvailabilityDispatcherStub{availability: HostCandidateAvailability{
		Decision:                   HostCandidateAvailabilityAlreadyApplied,
		BaselineDeploymentDigest:   "sha256:" + strings.Repeat("b", 64),
		ConfirmedDeploymentVersion: manifest.ReleaseVersion,
	}}
	service, repository := newUpgradeServiceForTest(t, dispatcher)
	result, err := service.CheckForUpdates(context.Background(), 7)
	if err != nil {
		t.Fatal(err)
	}
	if result.HasUpdate || result.CurrentVersion != manifest.ReleaseVersion {
		t.Fatalf("CheckForUpdates() = %#v, want confirmed no-update result", result)
	}
	operation, created, err := service.CreateOperation(context.Background(), 7, CreateUpgradeOperationInput{
		RequestID: uuidTestRequestID, ManifestID: manifest.Upgrade.ManifestID, ManifestDigest: manifest.Digest(), Confirmed: true,
	})
	if operation != nil || created || !errors.Is(err, domain.ErrUpgradeNoUpdateAvailable) {
		t.Fatalf("CreateOperation() operation=%#v created=%t err=%v, want no update", operation, created, err)
	}
	if len(repository.byID) != 0 || len(dispatcher.requests) != 0 {
		t.Fatalf("already-applied candidate had side effects: operations=%d dispatches=%d", len(repository.byID), len(dispatcher.requests))
	}
}

func TestUpgradeServiceCandidateInventoryConflictFailsBeforeOperationCreation(t *testing.T) {
	manifest, err := LoadReleaseManifest(fixturePath("release.manifest.yaml"), "")
	if err != nil {
		t.Fatal(err)
	}
	dispatcher := &candidateAvailabilityDispatcherStub{availabilityErr: ErrHostDeploymentStateConflict}
	service, repository := newUpgradeServiceForTest(t, dispatcher)
	if _, err := service.CheckForUpdates(context.Background(), 7); !errors.Is(err, ErrHostDeploymentStateConflict) {
		t.Fatalf("CheckForUpdates() error = %v, want ErrHostDeploymentStateConflict", err)
	}
	operation, created, err := service.CreateOperation(context.Background(), 7, CreateUpgradeOperationInput{
		RequestID: uuidTestRequestID, ManifestID: manifest.Upgrade.ManifestID, ManifestDigest: manifest.Digest(), Confirmed: true,
	})
	if operation != nil || created || !errors.Is(err, ErrHostDeploymentStateConflict) {
		t.Fatalf("CreateOperation() operation=%#v created=%t err=%v, want conflict", operation, created, err)
	}
	if len(repository.byID) != 0 || len(dispatcher.requests) != 0 {
		t.Fatalf("conflicting candidate had side effects: operations=%d dispatches=%d", len(repository.byID), len(dispatcher.requests))
	}
}

func TestUpgradeServiceCheckAndCreateShareConfirmedAvailabilityDecision(t *testing.T) {
	manifest, err := LoadReleaseManifest(fixturePath("release.manifest.yaml"), "")
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name  string
		state HostDeploymentState
	}{
		{name: "confirmed newer", state: HostDeploymentState{ReleaseVersion: "1.3.0", ManifestDigest: "sha256:" + strings.Repeat("a", 64), StateDigest: "sha256:" + strings.Repeat("b", 64)}},
		{name: "confirmed equal", state: HostDeploymentState{ReleaseVersion: manifest.ReleaseVersion, ManifestDigest: manifest.Digest(), StateDigest: "sha256:" + strings.Repeat("b", 64)}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			dispatcher := &deploymentStateDispatcherStub{
				upgradeDispatcherStub: upgradeDispatcherStub{},
				state:                 test.state,
			}
			service, repository := newUpgradeServiceForTest(t, dispatcher)
			result, checkErr := service.CheckForUpdates(context.Background(), 7)
			if checkErr != nil || result.HasUpdate {
				t.Fatalf("CheckForUpdates() result=%#v err=%v, want no update", result, checkErr)
			}
			operation, created, createErr := service.CreateOperation(context.Background(), 7, CreateUpgradeOperationInput{
				RequestID: uuidTestRequestID, ManifestID: manifest.Upgrade.ManifestID, ManifestDigest: manifest.Digest(), Confirmed: true,
			})
			if operation != nil || created || !errors.Is(createErr, domain.ErrUpgradeNoUpdateAvailable) {
				t.Fatalf("CreateOperation() operation=%#v created=%t err=%v, want no update", operation, created, createErr)
			}
			if len(repository.byID) != 0 || len(dispatcher.requests) != 0 {
				t.Fatalf("no-update decision had side effects: operations=%d dispatches=%d", len(repository.byID), len(dispatcher.requests))
			}
		})
	}
}

func TestUpgradeServiceDigestConflictFailsClosed(t *testing.T) {
	manifest, err := LoadReleaseManifest(fixturePath("release.manifest.yaml"), "")
	if err != nil {
		t.Fatal(err)
	}
	dispatcher := &deploymentStateDispatcherStub{
		upgradeDispatcherStub: upgradeDispatcherStub{},
		state:                 HostDeploymentState{ReleaseVersion: manifest.ReleaseVersion, ManifestDigest: "sha256:" + strings.Repeat("a", 64), StateDigest: "sha256:" + strings.Repeat("b", 64)},
	}
	service, repository := newUpgradeServiceForTest(t, dispatcher)
	if _, err := service.CheckForUpdates(context.Background(), 7); !errors.Is(err, ErrHostDeploymentStateConflict) {
		t.Fatalf("CheckForUpdates() error = %v, want ErrHostDeploymentStateConflict", err)
	}
	operation, created, err := service.CreateOperation(context.Background(), 7, CreateUpgradeOperationInput{
		RequestID: uuidTestRequestID, ManifestID: manifest.Upgrade.ManifestID, ManifestDigest: manifest.Digest(), Confirmed: true,
	})
	if operation != nil || created || !errors.Is(err, ErrHostDeploymentStateConflict) {
		t.Fatalf("CreateOperation() operation=%#v created=%t err=%v, want conflict", operation, created, err)
	}
	if len(repository.byID) != 0 || len(dispatcher.requests) != 0 {
		t.Fatalf("conflicting candidate had side effects: operations=%d dispatches=%d", len(repository.byID), len(dispatcher.requests))
	}
}

func TestUpgradeServiceReportsIncompatibleReleaseAsIneligible(t *testing.T) {
	manifest := manifestWithCompatibilityRange(t, ">=2.0.0 <3.0.0")
	service, _ := newUpgradeServiceForTest(t, &upgradeDispatcherStub{})
	service.manifestSource = ManifestSourceFunc(func() (*releasemanifest.Manifest, error) { return manifest, nil })

	result, err := service.CheckForUpdates(context.Background(), 7)
	if err != nil {
		t.Fatal(err)
	}
	if !result.HasUpdate || result.Eligible || result.Diagnostic == nil {
		t.Fatalf("compatibility gate result = %#v, want visible but ineligible update", result)
	}
	if result.Diagnostic.Code != domain.ErrorCodeReleaseCompatibilityUnsupported || result.Diagnostic.Field != "upgrade.compatibilityRange" {
		t.Fatalf("compatibility diagnostic = %#v", result.Diagnostic)
	}
}

func TestCreateOperationRejectsIncompatibleRelease(t *testing.T) {
	manifest := manifestWithCompatibilityRange(t, ">=2.0.0 <3.0.0")
	service, repository := newUpgradeServiceForTest(t, &upgradeDispatcherStub{})
	service.manifestSource = ManifestSourceFunc(func() (*releasemanifest.Manifest, error) { return manifest, nil })

	operation, created, err := service.CreateOperation(context.Background(), 7, CreateUpgradeOperationInput{
		RequestID: uuidTestRequestID, ManifestID: manifest.Upgrade.ManifestID, ManifestDigest: manifest.Digest(), Confirmed: true,
	})
	if operation != nil || created || !errors.Is(err, domain.ErrReleaseCompatibilityUnsupported) {
		t.Fatalf("incompatible target create = operation=%#v created=%t err=%v", operation, created, err)
	}
	if len(repository.byID) != 0 {
		t.Fatalf("incompatible target persisted an operation: %#v", repository.byID)
	}
}

func TestUpgradeServiceRejectsMalformedCompatibilityRange(t *testing.T) {
	manifest := manifestWithCompatibilityRange(t, ">=1.0.0 <2.0.0")
	manifest.Upgrade.CompatibilityRange = "not-a-range"
	service, _ := newUpgradeServiceForTest(t, &upgradeDispatcherStub{})
	service.manifestSource = ManifestSourceFunc(func() (*releasemanifest.Manifest, error) { return manifest, nil })

	if _, err := service.CheckForUpdates(context.Background(), 7); !errors.Is(err, domain.ErrReleaseManifestInvalid) {
		t.Fatalf("malformed compatibility range error = %v", err)
	}
}

func TestCreateOperationRejectsOlderTargetRelease(t *testing.T) {
	manifest, err := LoadReleaseManifest(fixturePath("release.manifest.yaml"), "")
	if err != nil {
		t.Fatal(err)
	}
	service, repository := newUpgradeServiceForTest(t, &upgradeDispatcherStub{})
	service.currentVersion = "1.2.4"
	operation, created, err := service.CreateOperation(context.Background(), 7, CreateUpgradeOperationInput{
		RequestID: uuidTestRequestID, ManifestID: manifest.Upgrade.ManifestID, ManifestDigest: manifest.Digest(), Confirmed: true,
	})
	if operation != nil || created || !errors.Is(err, domain.ErrUpgradeNoUpdateAvailable) {
		t.Fatalf("older target create = operation=%#v created=%t err=%v", operation, created, err)
	}
	if len(repository.byID) != 0 {
		t.Fatalf("older target persisted an operation: %#v", repository.byID)
	}
}

func TestAgentTargetUsesDeploymentRegistry(t *testing.T) {
	manifest, err := LoadReleaseManifest(fixturePath("release.manifest.yaml"), "")
	if err != nil {
		t.Fatal(err)
	}
	service, _ := newUpgradeServiceForTest(t, &upgradeDispatcherStub{})
	service.registry = "ghcr.io"
	target, err := service.agentTarget(manifest)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(target.ImageRef, "ghcr.io/") {
		t.Fatalf("Agent image = %q, want GHCR candidate", target.ImageRef)
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

func TestRetryOperationLoadsImmutableTargetAfterChannelAdvancement(t *testing.T) {
	original, err := LoadReleaseManifest(fixturePath("../testdata/release.manifest.yaml"), "")
	if err != nil {
		t.Fatal(err)
	}
	advanced := *original
	advanced.ReleaseVersion = "1.2.4"
	advanced.Upgrade.ManifestID = "lunafox-1.2.4"
	source := &targetManifestSourceStub{
		current: &advanced,
		targets: map[string]*releasemanifest.Manifest{original.Digest(): original},
	}
	dispatcher := &upgradeDispatcherStub{}
	service, repository := newUpgradeServiceForTest(t, dispatcher)
	service.manifestSource = source
	// Operation creation represents the earlier channel state. Persist it
	// directly so the retry runs after the source has advanced.
	now := time.Now().UTC()
	operation := &domain.Operation{
		OperationID:     "33333333-3333-4333-8333-333333333333",
		RequestID:       uuidTestRequestID,
		OperatorID:      7,
		ManifestID:      original.Upgrade.ManifestID,
		ManifestDigest:  original.Digest(),
		ReleaseVersion:  original.ReleaseVersion,
		Status:          domain.StatusFailed,
		MigrationStatus: domain.MigrationStatusNotStarted,
		StageTimes:      map[domain.Status]time.Time{domain.StatusFailed: now},
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	repository.byRequest[operation.RequestID] = operation
	repository.byID[operation.OperationID] = operation
	repository.active = operation

	retried, err := service.RetryOperation(context.Background(), 7, operation.OperationID, true)
	if err != nil {
		t.Fatal(err)
	}
	if source.loadedTarget != original.Digest() {
		t.Fatalf("LoadTarget digest = %q, want %q", source.loadedTarget, original.Digest())
	}
	if retried.ManifestDigest != original.Digest() {
		t.Fatalf("retry digest = %q, want original %q", retried.ManifestDigest, original.Digest())
	}
}

type targetManifestSourceStub struct {
	current      *releasemanifest.Manifest
	targets      map[string]*releasemanifest.Manifest
	loadedTarget string
}

func (source *targetManifestSourceStub) Load() (*releasemanifest.Manifest, error) {
	return source.current, nil
}

func (source *targetManifestSourceStub) LoadTarget(digest string) (*releasemanifest.Manifest, error) {
	source.loadedTarget = digest
	manifest := source.targets[digest]
	if manifest == nil {
		return nil, errors.New("target manifest not found")
	}
	return manifest, nil
}

const uuidTestRequestID = "22222222-2222-4222-8222-222222222222"

func manifestWithCompatibilityRange(t *testing.T, compatibilityRange string) *releasemanifest.Manifest {
	t.Helper()
	raw, err := os.ReadFile(fixturePath("release.manifest.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	raw = []byte(strings.Replace(string(raw), `compatibilityRange: ">=1.0.0 <2.0.0"`, `compatibilityRange: "`+compatibilityRange+`"`, 1))
	manifest, err := releasemanifest.Parse(raw)
	if err != nil {
		t.Fatal(err)
	}
	return manifest
}

func TestUpgradeFixtureIsReadable(t *testing.T) {
	if _, err := os.Stat(fixturePath("release.manifest.yaml")); err != nil {
		t.Fatal(err)
	}
}
