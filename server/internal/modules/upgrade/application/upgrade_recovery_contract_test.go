package application

import (
	"context"
	"errors"
	"strings"
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

func TestReconcileStalledOperationUsesMaintenanceWindowAndPreMigrationAttention(t *testing.T) {
	service, repository := newUpgradeServiceForTest(t, &upgradeDispatcherStub{})
	now := time.Date(2026, 9, 18, 12, 0, 0, 0, time.UTC)
	service.now = func() time.Time { return now }
	operation := &domain.Operation{
		OperationID:              "77777777-7777-4777-8777-777777777777",
		RequestID:                "88888888-8888-4888-8888-888888888888",
		OperatorID:               7,
		ManifestID:               "release-1.1.0",
		ManifestDigest:           "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		ReleaseVersion:           "1.1.0",
		Status:                   domain.StatusPreflight,
		MigrationStatus:          domain.MigrationStatusNotStarted,
		MigrationType:            "none",
		MaintenanceWindowMinutes: 60,
		CreatedAt:                now.Add(-20 * time.Minute),
		UpdatedAt:                now.Add(-1 * time.Minute),
		StageTimes:               map[domain.Status]time.Time{domain.StatusPreflight: now.Add(-20 * time.Minute)},
	}
	repository.byID[operation.OperationID] = operation
	repository.byRequest[operation.RequestID] = operation
	repository.active = operation

	// A long release window is still bounded, but must not be mistaken for a
	// short global timeout while Compose is legitimately doing work.
	updated, err := service.ReconcileStalledOperation(context.Background(), 0)
	if err != nil {
		t.Fatal(err)
	}
	if updated.Status != domain.StatusPreflight {
		t.Fatalf("long-window operation was closed early: %#v", updated)
	}

	operation.MaintenanceWindowMinutes = 1
	operation.StageTimes[domain.StatusPreflight] = now.Add(-6 * time.Minute)
	operation.UpdatedAt = now.Add(-1 * time.Minute)
	repository.active = operation
	updated, err = service.ReconcileStalledOperation(context.Background(), 0)
	if err != nil {
		t.Fatal(err)
	}
	if updated.Status != domain.StatusNeedsAttention || updated.CompletedAt == nil {
		t.Fatalf("pre-migration stall = %#v, want needs_attention terminal state", updated)
	}
}

func TestReconcileStalledOperationClassifiesPostMigrationAsRecovery(t *testing.T) {
	service, repository := newUpgradeServiceForTest(t, &upgradeDispatcherStub{})
	now := time.Date(2026, 9, 18, 12, 0, 0, 0, time.UTC)
	service.now = func() time.Time { return now }
	operation := &domain.Operation{
		OperationID:              "99999999-9999-4999-8999-999999999999",
		RequestID:                "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa",
		OperatorID:               7,
		ManifestID:               "release-1.1.0",
		ManifestDigest:           "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		ReleaseVersion:           "1.1.0",
		Status:                   domain.StatusRestarting,
		MigrationStatus:          domain.MigrationStatusRunning,
		MigrationType:            "compatible",
		MaintenanceWindowMinutes: 1,
		CreatedAt:                now.Add(-6 * time.Minute),
		UpdatedAt:                now.Add(-1 * time.Minute),
		StageTimes:               map[domain.Status]time.Time{domain.StatusRestarting: now.Add(-6 * time.Minute)},
	}
	repository.byID[operation.OperationID] = operation
	repository.byRequest[operation.RequestID] = operation
	repository.active = operation

	updated, err := service.ReconcileStalledOperation(context.Background(), 0)
	if err != nil {
		t.Fatal(err)
	}
	if updated.Status != domain.StatusNeedsRecovery || updated.CompletedAt == nil {
		t.Fatalf("post-migration stall = %#v, want needs_recovery terminal state", updated)
	}
}

func TestReconcileHostEventDoesNotRefreshStageForAnIdenticalJournalReplay(t *testing.T) {
	service, repository := newUpgradeServiceForTest(t, &upgradeDispatcherStub{})
	now := time.Date(2026, 9, 18, 12, 0, 0, 0, time.UTC)
	service.now = func() time.Time { return now.Add(10 * time.Minute) }
	operation := &domain.Operation{
		OperationID: "cccccccc-cccc-4ccc-8ccc-cccccccccccc", RequestID: "dddddddd-dddd-4ddd-8ddd-dddddddddddd", OperatorID: 7,
		ManifestID: "release-1.1.0", ManifestDigest: "sha256:" + strings.Repeat("a", 64), ReleaseVersion: "1.1.0",
		CompatibilityRange: "*", Status: domain.StatusUpdating, MigrationStatus: domain.MigrationStatusNotStarted,
		MigrationType: "none", CreatedAt: now, UpdatedAt: now,
		StageTimes: map[domain.Status]time.Time{domain.StatusUpdating: now}, ObservedDigests: map[string]string{},
	}
	repository.byID[operation.OperationID] = operation
	repository.byRequest[operation.RequestID] = operation
	repository.active = operation
	event := HostUpgradeEvent{
		OperationID: operation.OperationID, ManifestDigest: operation.ManifestDigest,
		Stage: string(domain.StatusUpdating), UpdatedAt: now, FromJournal: true,
	}
	updated, err := service.ReconcileHostEvent(context.Background(), event)
	if err != nil {
		t.Fatal(err)
	}
	if updated.UpdatedAt != now || updated.StageTimes[domain.StatusUpdating] != now || repository.updates != 0 {
		t.Fatalf("identical journal replay refreshed progress: updatedAt=%s stageAt=%s updates=%d", updated.UpdatedAt, updated.StageTimes[domain.StatusUpdating], repository.updates)
	}
}

func TestReconcileHostProgressRefreshesActivityWithoutAdvancingStageOrWatchdog(t *testing.T) {
	service, repository := newUpgradeServiceForTest(t, &upgradeDispatcherStub{})
	base := time.Date(2026, 9, 18, 12, 0, 0, 0, time.UTC)
	service.now = func() time.Time { return base.Add(5 * time.Minute) }
	operation := &domain.Operation{
		OperationID: "abababab-abab-4bab-8bab-abababababab", RequestID: "cdcdcdcd-cdcd-4dcd-8dcd-cdcdcdcdcdcd", OperatorID: 7,
		ManifestID: "release-1.1.0", ManifestDigest: "sha256:" + strings.Repeat("a", 64), ReleaseVersion: "1.1.0",
		CompatibilityRange: "*", Status: domain.StatusUpdating, MigrationStatus: domain.MigrationStatusNotStarted,
		MigrationType: "none", MaintenanceWindowMinutes: 1, CreatedAt: base, UpdatedAt: base,
		StageTimes: map[domain.Status]time.Time{domain.StatusUpdating: base}, ObservedDigests: map[string]string{},
	}
	repository.byID[operation.OperationID] = operation
	repository.byRequest[operation.RequestID] = operation
	repository.active = operation
	event := HostUpgradeEvent{
		OperationID: operation.OperationID, ManifestDigest: operation.ManifestDigest, Stage: string(domain.StatusUpdating),
		UpdatedAt: base.Add(4 * time.Minute), StageUpdatedAt: base, FromJournal: true,
		ProgressEvents: []domain.ProgressEvent{{
			Timestamp: base.Add(4 * time.Minute), Stage: domain.StatusUpdating, MessageKey: "pullImagesStarted",
			Message: "Pulling release images", Metadata: map[string]string{},
		}},
	}

	updated, err := service.ReconcileHostEvent(context.Background(), event)
	if err != nil {
		t.Fatal(err)
	}
	if updated.Status != domain.StatusUpdating || !updated.StageTimes[domain.StatusUpdating].Equal(base) {
		t.Fatalf("progress advanced stage evidence: %#v", updated)
	}
	if !updated.UpdatedAt.Equal(base.Add(5*time.Minute)) || len(updated.ProgressEvents) != 1 {
		t.Fatalf("progress did not refresh observable activity: %#v", updated)
	}
	if repository.updates != 1 {
		t.Fatalf("progress update count = %d, want 1", repository.updates)
	}

	if _, err := service.ReconcileHostEvent(context.Background(), event); err != nil {
		t.Fatal(err)
	}
	if repository.updates != 1 {
		t.Fatalf("progress replay persisted duplicate evidence: updates=%d", repository.updates)
	}

	service.now = func() time.Time { return base.Add(6 * time.Minute) }
	stalled, err := service.ReconcileStalledOperation(context.Background(), time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	if stalled.Status != domain.StatusNeedsAttention {
		t.Fatalf("progress heartbeat fed stalled watchdog: %#v", stalled)
	}
}

func TestReconcileJournalUnavailableClosesQueuedOperationWithAttention(t *testing.T) {
	service, repository := newUpgradeServiceForTest(t, &upgradeDispatcherStub{})
	now := time.Now().UTC()
	operation := &domain.Operation{
		OperationID: "eeeeeeee-eeee-4eee-8eee-eeeeeeeeeeee", RequestID: "ffffffff-ffff-4fff-8fff-ffffffffffff", OperatorID: 7,
		ManifestID: "release-1.1.0", ManifestDigest: "sha256:" + strings.Repeat("b", 64), ReleaseVersion: "1.1.0",
		CompatibilityRange: "*", Status: domain.StatusQueued, MigrationStatus: domain.MigrationStatusNotStarted,
		MigrationType: "none", CreatedAt: now, UpdatedAt: now, StageTimes: map[domain.Status]time.Time{}, ObservedDigests: map[string]string{},
	}
	repository.byID[operation.OperationID] = operation
	repository.byRequest[operation.RequestID] = operation
	repository.active = operation
	updated, err := service.ReconcileJournalUnavailable(context.Background(), "host journal unavailable")
	if err != nil {
		t.Fatal(err)
	}
	if updated.Status != domain.StatusNeedsAttention || updated.CompletedAt == nil {
		t.Fatalf("queued journal failure = %#v, want needs_attention terminal state", updated)
	}
}
