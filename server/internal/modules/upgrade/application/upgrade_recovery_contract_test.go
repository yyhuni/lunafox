package application

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/yyhuni/lunafox/server/internal/modules/upgrade/domain"
)

func TestReconcileHostEventPreservesMigrationRecoveryFence(t *testing.T) {
	service, repository := newUpgradeServiceForTest(t, &upgradeDispatcherStub{})
	manifest, err := LoadReleaseManifest(fixturePath("release.manifest.yaml"), "")
	if err != nil {
		t.Fatal(err)
	}
	operation := &domain.Operation{
		OperationID: "11111111-1111-4111-8111-111111111111", RequestID: uuidTestRequestID, OperatorID: 7,
		ManifestID: manifest.Upgrade.ManifestID, ManifestDigest: manifest.Digest(),
		ReleaseVersion: "1.2.3", CompatibilityRange: ">=1.0.0 <2.0.0", Status: domain.StatusMigrating,
		MigrationStatus: domain.MigrationStatusRunning, MigrationType: "compatible", CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC(),
		StageTimes: map[domain.Status]time.Time{domain.StatusMigrating: time.Now().UTC()}, ObservedDigests: map[string]string{},
	}
	repository.byID[operation.OperationID] = operation
	repository.byRequest[operation.RequestID] = operation
	repository.active = operation
	failed, err := service.ReconcileHostEvent(context.Background(), HostUpgradeEvent{OperationID: operation.OperationID, ManifestDigest: operation.ManifestDigest, Stage: string(domain.StatusFailed), Migration: string(domain.MigrationStatusFailed), Diagnostic: "migration failed"})
	if err != nil {
		t.Fatal(err)
	}
	if failed.Status != domain.StatusNeedsRecovery || failed.MigrationStatus != domain.MigrationStatusFailed {
		t.Fatalf("failed event result=%#v", failed)
	}
	late, err := service.ReconcileHostEvent(context.Background(), HostUpgradeEvent{OperationID: operation.OperationID, ManifestDigest: operation.ManifestDigest, Stage: string(domain.StatusSucceeded), Migration: string(domain.MigrationStatusSucceeded)})
	if err != nil {
		t.Fatal(err)
	}
	if late.Status != domain.StatusNeedsRecovery || late.MigrationStatus != domain.MigrationStatusFailed {
		t.Fatalf("late success rewrote recovery fence=%#v", late)
	}
}

func TestReconcileJournalUnavailableClassifiesMigrationBoundary(t *testing.T) {
	service, repository := newUpgradeServiceForTest(t, &upgradeDispatcherStub{})
	manifest, err := LoadReleaseManifest(fixturePath("release.manifest.yaml"), "")
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	operation := &domain.Operation{OperationID: "33333333-3333-4333-8333-333333333333", RequestID: "44444444-4444-4444-8444-444444444444", OperatorID: 7, ManifestID: manifest.Upgrade.ManifestID, ManifestDigest: manifest.Digest(), ReleaseVersion: "1.2.3", CompatibilityRange: "*", Status: domain.StatusMigrating, MigrationStatus: domain.MigrationStatusRunning, MigrationType: "compatible", CreatedAt: now, UpdatedAt: now, StageTimes: map[domain.Status]time.Time{}, ObservedDigests: map[string]string{}}
	repository.byID[operation.OperationID] = operation
	repository.active = operation
	updated, err := service.ReconcileJournalUnavailable(context.Background(), "journal unavailable")
	if err != nil {
		t.Fatal(err)
	}
	if updated.Status != domain.StatusNeedsRecovery || updated.CompletedAt == nil {
		t.Fatalf("journal recovery result=%#v", updated)
	}
	if updated.Diagnostic != "journal unavailable" {
		t.Fatalf("diagnostic=%q", updated.Diagnostic)
	}
}

func TestReconcileHostEventRejectsDigestMismatchAndUnknownStage(t *testing.T) {
	service, repository := newUpgradeServiceForTest(t, &upgradeDispatcherStub{})
	manifest, err := LoadReleaseManifest(fixturePath("release.manifest.yaml"), "")
	if err != nil {
		t.Fatal(err)
	}
	operation := &domain.Operation{OperationID: "55555555-5555-4555-8555-555555555555", RequestID: "66666666-6666-4666-8666-666666666666", OperatorID: 7, ManifestID: manifest.Upgrade.ManifestID, ManifestDigest: manifest.Digest(), ReleaseVersion: "1.2.3", CompatibilityRange: "*", Status: domain.StatusQueued, MigrationStatus: domain.MigrationStatusNotStarted, MigrationType: "none", CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC(), StageTimes: map[domain.Status]time.Time{}, ObservedDigests: map[string]string{}}
	repository.byID[operation.OperationID] = operation
	if _, err := service.ReconcileHostEvent(context.Background(), HostUpgradeEvent{OperationID: operation.OperationID, ManifestDigest: "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb", Stage: string(domain.StatusStopping)}); !errors.Is(err, domain.ErrReleaseManifestTargetMismatch) {
		t.Fatalf("digest mismatch error=%v", err)
	}
	if _, err := service.ReconcileHostEvent(context.Background(), HostUpgradeEvent{OperationID: operation.OperationID, ManifestDigest: operation.ManifestDigest, Stage: "shell_command"}); err == nil {
		t.Fatal("unknown host stage was accepted")
	}
}
