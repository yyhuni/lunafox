package application

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/yyhuni/lunafox/contracts/releasemanifest"
	"github.com/yyhuni/lunafox/server/internal/modules/upgrade/domain"
)

func TestCreateOperationAppliesMigrationGateBeforePersistence(t *testing.T) {
	manifest, err := LoadReleaseManifest(fixturePath("release.manifest.yaml"), "")
	if err != nil {
		t.Fatal(err)
	}
	manifestWithMigration := *manifest
	manifestWithMigration.Upgrade.DatabaseMigration.HasDatabaseMigration = true
	manifestWithMigration.Upgrade.DatabaseMigration.MigrationType = "compatible"
	manifestWithMigration.Upgrade.DatabaseMigration.MigrationID = "000002_upgrade"
	manifestWithMigration.Upgrade.DatabaseMigration.Checksum = "sha256:" + strings.Repeat("b", 64)

	dispatcher := &upgradeDispatcherStub{}
	service, repository := newUpgradeServiceForTest(t, dispatcher)
	service.manifestSource = ManifestSourceFunc(func() (*releasemanifest.Manifest, error) {
		return &manifestWithMigration, nil
	})
	_, _, err = service.CreateOperation(context.Background(), 7, CreateUpgradeOperationInput{
		RequestID:      uuidTestRequestID,
		ManifestID:     manifestWithMigration.Upgrade.ManifestID,
		ManifestDigest: manifestWithMigration.Digest(),
		Confirmed:      true,
	})
	if !errors.Is(err, domain.ErrMigrationPolicyDisallowsDataRetain) {
		t.Fatalf("CreateOperation migration gate error=%v", err)
	}
	if len(repository.byID) != 0 || len(dispatcher.requests) != 0 {
		t.Fatalf("migration-gated request mutated state: operations=%d dispatches=%d", len(repository.byID), len(dispatcher.requests))
	}
}

func TestCreateOperationReplayUsesDurableRequestBindingBeforeManifestReload(t *testing.T) {
	service, repository := newUpgradeServiceForTest(t, &upgradeDispatcherStub{})
	manifest, err := LoadReleaseManifest(fixturePath("release.manifest.yaml"), "")
	if err != nil {
		t.Fatal(err)
	}
	input := CreateUpgradeOperationInput{
		RequestID:      uuidTestRequestID,
		ManifestID:     manifest.Upgrade.ManifestID,
		ManifestDigest: manifest.Digest(),
		Confirmed:      true,
	}
	created, createdNew, err := service.CreateOperation(context.Background(), 7, input)
	if err != nil || !createdNew {
		t.Fatalf("initial create=%#v created=%v err=%v", created, createdNew, err)
	}

	manifestLoads := 0
	service.manifestSource = ManifestSourceFunc(func() (*releasemanifest.Manifest, error) {
		manifestLoads++
		return nil, errors.New("manifest should not be loaded for a durable replay")
	})
	replayed, createdNew, err := service.CreateOperation(context.Background(), 7, input)
	if err != nil || createdNew || replayed == nil || replayed.OperationID != created.OperationID {
		t.Fatalf("replay=%#v created=%v err=%v", replayed, createdNew, err)
	}
	if manifestLoads != 0 || len(repository.byID) != 1 {
		t.Fatalf("replay reloaded manifest or created a row: loads=%d rows=%d", manifestLoads, len(repository.byID))
	}
}

func TestCreateOperationHostHandoffFailurePersistsRetryableFailure(t *testing.T) {
	dispatcher := &upgradeDispatcherStub{err: errors.New("socket unavailable")}
	service, repository := newUpgradeServiceForTest(t, dispatcher)
	manifest, err := LoadReleaseManifest(fixturePath("release.manifest.yaml"), "")
	if err != nil {
		t.Fatal(err)
	}
	operation, created, err := service.CreateOperation(context.Background(), 7, CreateUpgradeOperationInput{
		RequestID:      uuidTestRequestID,
		ManifestID:     manifest.Upgrade.ManifestID,
		ManifestDigest: manifest.Digest(),
		Confirmed:      true,
	})
	if !created || operation == nil || !errors.Is(err, domain.ErrUpgradeHostUnavailable) {
		t.Fatalf("handoff failure operation=%#v created=%v err=%v", operation, created, err)
	}
	if operation.Status != domain.StatusFailed || operation.CompletedAt == nil || operation.Diagnostic == "" {
		t.Fatalf("handoff failure was not persisted as failed: %#v", operation)
	}
	persisted := repository.byID[operation.OperationID]
	if persisted == nil || persisted.Status != domain.StatusFailed {
		t.Fatalf("persisted handoff failure=%#v", persisted)
	}
}

func TestRetryOperationDoesNotReopenSucceededOperation(t *testing.T) {
	dispatcher := &upgradeDispatcherStub{}
	service, repository := newUpgradeServiceForTest(t, dispatcher)
	manifest, err := LoadReleaseManifest(fixturePath("release.manifest.yaml"), "")
	if err != nil {
		t.Fatal(err)
	}
	operation, _, err := service.CreateOperation(context.Background(), 7, CreateUpgradeOperationInput{
		RequestID:      uuidTestRequestID,
		ManifestID:     manifest.Upgrade.ManifestID,
		ManifestDigest: manifest.Digest(),
		Confirmed:      true,
	})
	if err != nil {
		t.Fatal(err)
	}
	now := operation.UpdatedAt.Add(2 * time.Minute)
	operation.Status = domain.StatusSucceeded
	operation.MigrationStatus = domain.MigrationStatusSucceeded
	operation.CompletedAt = &now
	operation.UpdatedAt = now
	if err := repository.Update(context.Background(), operation); err != nil {
		t.Fatal(err)
	}
	if _, err := service.RetryOperation(context.Background(), 7, operation.OperationID, true); !errors.Is(err, domain.ErrUpgradeRetryNotAllowed) {
		t.Fatalf("retry succeeded operation error=%v", err)
	}
	if len(dispatcher.requests) != 1 {
		t.Fatalf("retry dispatched after success: %d requests", len(dispatcher.requests))
	}
}

func TestCheckForUpdatesRejectsMissingManifestWithStableDiagnostic(t *testing.T) {
	service, _ := newUpgradeServiceForTest(t, &upgradeDispatcherStub{})
	service.manifestSource = ManifestSourceFunc(func() (*releasemanifest.Manifest, error) { return nil, nil })
	result, err := service.CheckForUpdates(context.Background(), 7)
	if err == nil || domain.CodeOf(err) != domain.ErrorCodeReleaseManifestInvalid {
		t.Fatalf("missing manifest result=%#v err=%v", result, err)
	}
}

func TestStopOperationConvergesWhenHostDoesNotAcceptStop(t *testing.T) {
	dispatcher := &upgradeDispatcherStub{}
	service, repository := newUpgradeServiceForTest(t, dispatcher)
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
	dispatcher.err = errors.New("socket unavailable")
	updated, stopErr := service.StopOperation(context.Background(), 7, operation.OperationID, true)
	if !errors.Is(stopErr, domain.ErrUpgradeHostUnavailable) {
		t.Fatalf("stop error = %v, want host unavailable", stopErr)
	}
	if updated == nil || updated.Status != domain.StatusNeedsAttention || updated.CompletedAt == nil {
		t.Fatalf("stop result = %#v, want needs_attention terminal state", updated)
	}
	if repository.byID[operation.OperationID].Status != domain.StatusNeedsAttention {
		t.Fatalf("persisted stop result = %#v", repository.byID[operation.OperationID])
	}

	// A repeated stop is a read of the same terminal result and must not dispatch
	// a second host request.
	requests := len(dispatcher.requests)
	replayed, err := service.StopOperation(context.Background(), 7, operation.OperationID, true)
	if err != nil || replayed.Status != domain.StatusNeedsAttention || len(dispatcher.requests) != requests {
		t.Fatalf("repeated stop = %#v err=%v requests=%d want=%d", replayed, err, len(dispatcher.requests), requests)
	}
}

func TestStopOperationAfterMigrationRequiresRecoveryWhenHostUnavailable(t *testing.T) {
	dispatcher := &upgradeDispatcherStub{}
	service, repository := newUpgradeServiceForTest(t, dispatcher)
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
	operation.Status = domain.StatusMigrating
	operation.MigrationStatus = domain.MigrationStatusRunning
	if err := repository.Update(context.Background(), operation); err != nil {
		t.Fatal(err)
	}
	dispatcher.err = errors.New("socket unavailable")
	updated, stopErr := service.StopOperation(context.Background(), 7, operation.OperationID, true)
	if !errors.Is(stopErr, domain.ErrUpgradeHostUnavailable) || updated.Status != domain.StatusNeedsRecovery {
		t.Fatalf("post-migration stop = %#v err=%v, want needs_recovery and host error", updated, stopErr)
	}
}

func TestFindActiveOperationDoesNotExposeTerminalAdapterResult(t *testing.T) {
	service, repository := newUpgradeServiceForTest(t, &upgradeDispatcherStub{})
	manifest, err := LoadReleaseManifest(fixturePath("release.manifest.yaml"), "")
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	operation := &domain.Operation{
		OperationID: "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa", RequestID: "bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb", OperatorID: 7,
		ManifestID: manifest.Upgrade.ManifestID, ManifestDigest: manifest.Digest(), ReleaseVersion: "1.2.3",
		CompatibilityRange: "*", Status: domain.StatusSucceeded, MigrationStatus: domain.MigrationStatusNotStarted,
		MigrationType: "none", CreatedAt: now, UpdatedAt: now,
	}
	repository.active = operation
	repository.byID[operation.OperationID] = operation
	service.repository = &terminalActiveRepository{upgradeRepositoryStub: repository, operation: operation}
	if _, err := service.FindActiveOperation(context.Background(), 7); !errors.Is(err, domain.ErrUpgradeNotFound) {
		t.Fatalf("terminal active result error = %v, want not found", err)
	}
}

type terminalActiveRepository struct {
	*upgradeRepositoryStub
	operation *domain.Operation
}

func (repository *terminalActiveRepository) FindActive(context.Context) (*domain.Operation, error) {
	return repository.operation, nil
}
