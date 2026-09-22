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

func TestReconcileJournalStaleFrontendOnlyPlanClosesQueuedOperation(t *testing.T) {
	service, repository := newUpgradeServiceForTest(t, &upgradeDispatcherStub{})
	now := time.Date(2026, 9, 21, 9, 0, 0, 0, time.UTC)
	planDigest := "sha256:" + strings.Repeat("a", 64)
	baselineDigest := "sha256:" + strings.Repeat("b", 64)
	operation := &domain.Operation{
		OperationID: "56565656-5656-4565-8565-565656565656", RequestID: "67676767-6767-4676-8676-676767676767", OperatorID: 7,
		ManifestID: "release-1.1.0", ManifestDigest: "sha256:" + strings.Repeat("c", 64), ReleaseVersion: "1.1.0",
		CompatibilityRange: "*", Status: domain.StatusQueued, MigrationStatus: domain.MigrationStatusNotStarted, MigrationType: "none",
		ExecutionMode: domain.ExecutionModeFrontendOnly, WorkDisposition: domain.WorkDispositionNotRequired,
		PlanSummary: domain.PlanSummary{TouchedServices: []string{"frontend"}}, PlanDigest: planDigest,
		BaselineDeploymentDigest: baselineDigest, ConfirmedDeploymentVersion: "1.0.0",
		CreatedAt: now, UpdatedAt: now, StageTimes: map[domain.Status]time.Time{domain.StatusQueued: now}, ObservedDigests: map[string]string{},
	}
	repository.byID[operation.OperationID] = operation
	repository.byRequest[operation.RequestID] = operation
	repository.active = operation

	updated, err := service.ReconcileHostEvent(context.Background(), HostUpgradeEvent{
		OperationID: operation.OperationID, ManifestDigest: operation.ManifestDigest, Stage: string(domain.StatusFailed),
		ExecutionMode: operation.ExecutionMode, PlanDigest: planDigest, BaselineStateDigest: baselineDigest,
		TouchedServices: []string{"frontend"}, ConfirmedDeploymentVersion: "1.0.0",
		UpdatedAt: now.Add(time.Minute), StageUpdatedAt: now.Add(time.Minute), FromJournal: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if updated.Status != domain.StatusFailed || updated.CompletedAt == nil {
		t.Fatalf("stale frontend-only recovery result = %#v", updated)
	}
	if _, err := repository.FindActive(context.Background()); err == nil {
		t.Fatal("stale frontend-only result remained active")
	}
}

func TestReconcileJournalRejectsFrontendOnlyMigrationStage(t *testing.T) {
	service, repository := newUpgradeServiceForTest(t, &upgradeDispatcherStub{})
	now := time.Date(2026, 9, 21, 10, 0, 0, 0, time.UTC)
	planDigest := "sha256:" + strings.Repeat("a", 64)
	baselineDigest := "sha256:" + strings.Repeat("b", 64)
	operation := &domain.Operation{
		OperationID: "78787878-7878-4787-8787-787878787878", RequestID: "79797979-7979-4797-8797-797979797979", OperatorID: 7,
		ManifestID: "release-1.1.0", ManifestDigest: "sha256:" + strings.Repeat("c", 64), ReleaseVersion: "1.1.0",
		CompatibilityRange: "*", Status: domain.StatusUpdating, MigrationStatus: domain.MigrationStatusNotStarted, MigrationType: "none",
		ExecutionMode: domain.ExecutionModeFrontendOnly, WorkDisposition: domain.WorkDispositionNotRequired,
		PlanSummary: domain.PlanSummary{TouchedServices: []string{"frontend"}}, PlanDigest: planDigest,
		BaselineDeploymentDigest: baselineDigest, ConfirmedDeploymentVersion: "1.0.0",
		CreatedAt: now, UpdatedAt: now, StageTimes: map[domain.Status]time.Time{domain.StatusUpdating: now}, ObservedDigests: map[string]string{},
	}
	repository.byID[operation.OperationID] = operation
	repository.byRequest[operation.RequestID] = operation
	repository.active = operation

	_, err := service.ReconcileHostEvent(context.Background(), HostUpgradeEvent{
		OperationID: operation.OperationID, ManifestDigest: operation.ManifestDigest, Stage: string(domain.StatusMigrating),
		ExecutionMode: operation.ExecutionMode, PlanDigest: planDigest, BaselineStateDigest: baselineDigest,
		TouchedServices: []string{"frontend"}, ConfirmedDeploymentVersion: "1.0.0", FromJournal: true,
		UpdatedAt: now.Add(time.Minute), StageUpdatedAt: now.Add(time.Minute),
	})
	if err == nil || !strings.Contains(err.Error(), "frontend-only host journal stage") {
		t.Fatalf("frontend-only migrating journal stage error = %v", err)
	}
	if operation.Status != domain.StatusUpdating || repository.updates != 0 {
		t.Fatalf("frontend-only operation changed after rejected migration stage: status=%s updates=%d", operation.Status, repository.updates)
	}
}

func TestReconcileJournalRejectsFrontendOnlyMigrationEvidence(t *testing.T) {
	service, repository := newUpgradeServiceForTest(t, &upgradeDispatcherStub{})
	now := time.Date(2026, 9, 21, 10, 0, 0, 0, time.UTC)
	planDigest := "sha256:" + strings.Repeat("a", 64)
	baselineDigest := "sha256:" + strings.Repeat("b", 64)
	operation := &domain.Operation{
		OperationID: "82828282-8282-4828-8828-828282828282", RequestID: "83838383-8383-4838-8838-838383838383", OperatorID: 7,
		ManifestID: "release-1.1.0", ManifestDigest: "sha256:" + strings.Repeat("c", 64), ReleaseVersion: "1.1.0",
		CompatibilityRange: "*", Status: domain.StatusUpdating, MigrationStatus: domain.MigrationStatusNotStarted, MigrationType: "none",
		ExecutionMode: domain.ExecutionModeFrontendOnly, WorkDisposition: domain.WorkDispositionNotRequired,
		PlanSummary: domain.PlanSummary{TouchedServices: []string{"frontend"}}, PlanDigest: planDigest,
		BaselineDeploymentDigest: baselineDigest, ConfirmedDeploymentVersion: "1.0.0",
		CreatedAt: now, UpdatedAt: now, StageTimes: map[domain.Status]time.Time{domain.StatusUpdating: now}, ObservedDigests: map[string]string{},
	}
	repository.byID[operation.OperationID] = operation
	repository.byRequest[operation.RequestID] = operation
	repository.active = operation

	_, err := service.ReconcileHostEvent(context.Background(), HostUpgradeEvent{
		OperationID: operation.OperationID, ManifestDigest: operation.ManifestDigest, Stage: string(domain.StatusFailed),
		Migration: string(domain.MigrationStatusFailed), ExecutionMode: operation.ExecutionMode,
		PlanDigest: planDigest, BaselineStateDigest: baselineDigest, TouchedServices: []string{"frontend"},
		ConfirmedDeploymentVersion: "1.0.0", FromJournal: true, UpdatedAt: now.Add(time.Minute), StageUpdatedAt: now.Add(time.Minute),
	})
	if err == nil || !strings.Contains(err.Error(), "frontend-only host journal migration status") {
		t.Fatalf("frontend-only migration evidence error = %v", err)
	}
	if operation.Status != domain.StatusUpdating || operation.MigrationStatus != domain.MigrationStatusNotStarted || repository.updates != 0 {
		t.Fatalf("frontend-only operation changed after rejected migration evidence: status=%s migration=%s updates=%d", operation.Status, operation.MigrationStatus, repository.updates)
	}
}

func TestReconcileJournalRejectsFrontendOnlyProgressAtMigrationStage(t *testing.T) {
	service, repository := newUpgradeServiceForTest(t, &upgradeDispatcherStub{})
	now := time.Date(2026, 9, 21, 10, 0, 0, 0, time.UTC)
	planDigest := "sha256:" + strings.Repeat("a", 64)
	baselineDigest := "sha256:" + strings.Repeat("b", 64)
	operation := &domain.Operation{
		OperationID: "84848484-8484-4848-8848-848484848484", RequestID: "85858585-8585-4858-8858-858585858585", OperatorID: 7,
		ManifestID: "release-1.1.0", ManifestDigest: "sha256:" + strings.Repeat("c", 64), ReleaseVersion: "1.1.0",
		CompatibilityRange: "*", Status: domain.StatusUpdating, MigrationStatus: domain.MigrationStatusNotStarted, MigrationType: "none",
		ExecutionMode: domain.ExecutionModeFrontendOnly, WorkDisposition: domain.WorkDispositionNotRequired,
		PlanSummary: domain.PlanSummary{TouchedServices: []string{"frontend"}}, PlanDigest: planDigest,
		BaselineDeploymentDigest: baselineDigest, ConfirmedDeploymentVersion: "1.0.0",
		CreatedAt: now, UpdatedAt: now, StageTimes: map[domain.Status]time.Time{domain.StatusUpdating: now}, ObservedDigests: map[string]string{},
	}
	repository.byID[operation.OperationID] = operation
	repository.byRequest[operation.RequestID] = operation
	repository.active = operation

	_, err := service.ReconcileHostEvent(context.Background(), HostUpgradeEvent{
		OperationID: operation.OperationID, ManifestDigest: operation.ManifestDigest, Stage: string(domain.StatusUpdating),
		ExecutionMode: operation.ExecutionMode, PlanDigest: planDigest, BaselineStateDigest: baselineDigest,
		TouchedServices: []string{"frontend"}, ConfirmedDeploymentVersion: "1.0.0", FromJournal: true,
		UpdatedAt: now.Add(time.Minute), StageUpdatedAt: now,
		ProgressEvents: []domain.ProgressEvent{{
			Timestamp: now.Add(time.Minute), Stage: domain.StatusMigrating, MessageKey: "migrationStarted",
			Message: "Running database migration", Metadata: map[string]string{},
		}},
	})
	if err == nil || !strings.Contains(err.Error(), "frontend-only host journal progress stage") {
		t.Fatalf("frontend-only migration progress error = %v", err)
	}
	if operation.Status != domain.StatusUpdating || len(operation.ProgressEvents) != 0 || repository.updates != 0 {
		t.Fatalf("frontend-only operation changed after rejected migration progress: status=%s progress=%#v updates=%d", operation.Status, operation.ProgressEvents, repository.updates)
	}
}

func TestReconcileJournalRejectsFrontendOnlyAgentVerificationStage(t *testing.T) {
	service, repository := newUpgradeServiceForTest(t, &upgradeDispatcherStub{})
	now := time.Date(2026, 9, 21, 10, 0, 0, 0, time.UTC)
	planDigest := "sha256:" + strings.Repeat("d", 64)
	baselineDigest := "sha256:" + strings.Repeat("e", 64)
	operation := &domain.Operation{
		OperationID: "80808080-8080-4808-8808-808080808080", RequestID: "81818181-8181-4818-8818-818181818181", OperatorID: 7,
		ManifestID: "release-1.1.0", ManifestDigest: "sha256:" + strings.Repeat("f", 64), ReleaseVersion: "1.1.0",
		CompatibilityRange: "*", Status: domain.StatusRestarting, MigrationStatus: domain.MigrationStatusNotStarted, MigrationType: "none",
		ExecutionMode: domain.ExecutionModeFrontendOnly, WorkDisposition: domain.WorkDispositionNotRequired,
		PlanSummary: domain.PlanSummary{TouchedServices: []string{"frontend"}}, PlanDigest: planDigest,
		BaselineDeploymentDigest: baselineDigest, ConfirmedDeploymentVersion: "1.0.0",
		CreatedAt: now, UpdatedAt: now, StageTimes: map[domain.Status]time.Time{domain.StatusRestarting: now}, ObservedDigests: map[string]string{},
	}
	repository.byID[operation.OperationID] = operation
	repository.byRequest[operation.RequestID] = operation
	repository.active = operation

	_, err := service.ReconcileHostEvent(context.Background(), HostUpgradeEvent{
		OperationID: operation.OperationID, ManifestDigest: operation.ManifestDigest, Stage: string(domain.StatusAgentVerifying),
		ExecutionMode: operation.ExecutionMode, PlanDigest: planDigest, BaselineStateDigest: baselineDigest,
		TouchedServices: []string{"frontend"}, ConfirmedDeploymentVersion: "1.0.0", FromJournal: true,
		UpdatedAt: now.Add(time.Minute), StageUpdatedAt: now.Add(time.Minute),
	})
	if err == nil || !strings.Contains(err.Error(), "frontend-only host journal stage") {
		t.Fatalf("frontend-only agent verification journal stage error = %v", err)
	}
	if operation.Status != domain.StatusRestarting || repository.updates != 0 {
		t.Fatalf("frontend-only operation changed after rejected Agent verification stage: status=%s updates=%d", operation.Status, repository.updates)
	}
}

func TestReconcileJournalRejectsPersistedFrontendOnlyForbiddenStages(t *testing.T) {
	tests := []struct {
		name       string
		persisted  domain.Status
		checkpoint domain.Status
	}{
		{name: "stopping", persisted: domain.StatusStopping, checkpoint: domain.StatusPreflight},
		{name: "migrating", persisted: domain.StatusMigrating, checkpoint: domain.StatusRestarting},
		{name: "agent verifying", persisted: domain.StatusAgentVerifying, checkpoint: domain.StatusVerifying},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			service, repository := newUpgradeServiceForTest(t, &upgradeDispatcherStub{})
			now := time.Date(2026, 9, 21, 11, 0, 0, 0, time.UTC)
			planDigest := "sha256:" + strings.Repeat("a", 64)
			baselineDigest := "sha256:" + strings.Repeat("b", 64)
			operation := &domain.Operation{
				OperationID: "90909090-9090-4090-8090-909090909090", RequestID: "91919191-9191-4191-8191-919191919191", OperatorID: 7,
				ManifestID: "release-1.1.0", ManifestDigest: "sha256:" + strings.Repeat("c", 64), ReleaseVersion: "1.1.0",
				CompatibilityRange: "*", Status: test.persisted, MigrationStatus: domain.MigrationStatusNotStarted, MigrationType: "none",
				ExecutionMode: domain.ExecutionModeFrontendOnly, WorkDisposition: domain.WorkDispositionNotRequired,
				PlanSummary: domain.PlanSummary{TouchedServices: []string{"frontend"}}, PlanDigest: planDigest,
				BaselineDeploymentDigest: baselineDigest, ConfirmedDeploymentVersion: "1.0.0",
				CreatedAt: now, UpdatedAt: now, StageTimes: map[domain.Status]time.Time{test.persisted: now}, ObservedDigests: map[string]string{},
			}
			repository.byID[operation.OperationID] = operation
			repository.byRequest[operation.RequestID] = operation
			repository.active = operation

			_, err := service.ReconcileHostEvent(context.Background(), HostUpgradeEvent{
				OperationID: operation.OperationID, ManifestDigest: operation.ManifestDigest, Stage: string(test.checkpoint),
				ExecutionMode: operation.ExecutionMode, PlanDigest: planDigest, BaselineStateDigest: baselineDigest,
				TouchedServices: []string{"frontend"}, ConfirmedDeploymentVersion: "1.0.0", FromJournal: true,
				UpdatedAt: now.Add(time.Minute), StageUpdatedAt: now.Add(time.Minute),
			})
			if err == nil || !strings.Contains(err.Error(), "frontend-only persisted operation stage") {
				t.Fatalf("persisted frontend-only stage %q error = %v", test.persisted, err)
			}
			if operation.Status != test.persisted || repository.updates != 0 {
				t.Fatalf("persisted frontend-only stage changed after rejection: status=%s updates=%d", operation.Status, repository.updates)
			}
		})
	}
}

func TestRecoveryJobRejectsCustomJournalObservedDigestOutsideFrontendScope(t *testing.T) {
	service, repository := newUpgradeServiceForTest(t, &upgradeDispatcherStub{})
	now := time.Date(2026, 9, 21, 11, 0, 0, 0, time.UTC)
	planDigest := "sha256:" + strings.Repeat("a", 64)
	baselineDigest := "sha256:" + strings.Repeat("b", 64)
	operation := &domain.Operation{
		OperationID: "92929292-9292-4292-8292-929292929292", RequestID: "93939393-9393-4393-8393-939393939393", OperatorID: 7,
		ManifestID: "release-1.1.0", ManifestDigest: "sha256:" + strings.Repeat("c", 64), ReleaseVersion: "1.1.0",
		CompatibilityRange: "*", Status: domain.StatusUpdating, MigrationStatus: domain.MigrationStatusNotStarted, MigrationType: "none",
		ExecutionMode: domain.ExecutionModeFrontendOnly, WorkDisposition: domain.WorkDispositionNotRequired,
		PlanSummary: domain.PlanSummary{TouchedServices: []string{"frontend"}}, PlanDigest: planDigest,
		BaselineDeploymentDigest: baselineDigest, ConfirmedDeploymentVersion: "1.0.0",
		CreatedAt: now, UpdatedAt: now, StageTimes: map[domain.Status]time.Time{domain.StatusUpdating: now}, ObservedDigests: map[string]string{},
	}
	repository.byID[operation.OperationID] = operation
	repository.byRequest[operation.RequestID] = operation
	repository.active = operation

	job, err := NewRecoveryJob(service, recoveryJournalReaderStub{event: HostUpgradeEvent{
		OperationID: operation.OperationID, ManifestDigest: operation.ManifestDigest, Stage: string(domain.StatusUpdating),
		ExecutionMode: operation.ExecutionMode, PlanDigest: planDigest, BaselineStateDigest: baselineDigest,
		TouchedServices: []string{"frontend"}, ConfirmedDeploymentVersion: "1.0.0",
		ObservedDigests: map[string]string{"server": "sha256:" + strings.Repeat("d", 64)},
		UpdatedAt:       now.Add(time.Minute), StageUpdatedAt: now.Add(time.Minute),
	}}, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := job.RunOnce(context.Background()); err == nil || !strings.Contains(err.Error(), "host journal observed digest service") {
		t.Fatalf("custom journal observed digest error = %v", err)
	}
	if operation.Status != domain.StatusUpdating || len(operation.ObservedDigests) != 0 || repository.updates != 0 {
		t.Fatalf("custom journal changed frontend-only operation: status=%s digests=%#v updates=%d", operation.Status, operation.ObservedDigests, repository.updates)
	}
}

func TestReconcileJournalRejectsCorruptPersistedFrontendOnlyOperationBeforeNoop(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*domain.Operation)
	}{
		{
			name: "migration evidence",
			mutate: func(operation *domain.Operation) {
				operation.MigrationStatus = domain.MigrationStatusRunning
			},
		},
		{
			name: "Agent evidence",
			mutate: func(operation *domain.Operation) {
				operation.AgentExpectations = []domain.AgentExpectation{{AgentID: 42}}
			},
		},
	}
	for index, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			service, repository := newUpgradeServiceForTest(t, &upgradeDispatcherStub{})
			now := time.Date(2026, 9, 21, 12, index, 0, 0, time.UTC)
			planDigest := "sha256:" + strings.Repeat("a", 64)
			baselineDigest := "sha256:" + strings.Repeat("b", 64)
			operation := &domain.Operation{
				OperationID: "94949494-9494-4494-8494-949494949494", RequestID: "95959595-9595-4595-8595-959595959595", OperatorID: 7,
				ManifestID: "release-1.1.0", ManifestDigest: "sha256:" + strings.Repeat("c", 64), ReleaseVersion: "1.1.0",
				CompatibilityRange: "*", Status: domain.StatusUpdating, MigrationStatus: domain.MigrationStatusNotStarted, MigrationType: "none",
				ExecutionMode: domain.ExecutionModeFrontendOnly, WorkDisposition: domain.WorkDispositionNotRequired,
				PlanSummary: domain.PlanSummary{TouchedServices: []string{"frontend"}}, PlanDigest: planDigest,
				BaselineDeploymentDigest: baselineDigest, ConfirmedDeploymentVersion: "1.0.0",
				CreatedAt: now, UpdatedAt: now, StageTimes: map[domain.Status]time.Time{domain.StatusUpdating: now}, ObservedDigests: map[string]string{},
			}
			test.mutate(operation)
			repository.byID[operation.OperationID] = operation
			repository.byRequest[operation.RequestID] = operation
			repository.active = operation

			_, err := service.ReconcileHostEvent(context.Background(), HostUpgradeEvent{
				OperationID: operation.OperationID, ManifestDigest: operation.ManifestDigest, Stage: string(domain.StatusUpdating),
				ExecutionMode: operation.ExecutionMode, PlanDigest: planDigest, BaselineStateDigest: baselineDigest,
				TouchedServices: []string{"frontend"}, ConfirmedDeploymentVersion: "1.0.0",
				UpdatedAt: now, StageUpdatedAt: now, FromJournal: true,
			})
			if err == nil || !strings.Contains(err.Error(), "persisted frontend-only operation is invalid") {
				t.Fatalf("corrupt persisted operation error = %v", err)
			}
			if repository.updates != 0 || operation.Status != domain.StatusUpdating {
				t.Fatalf("corrupt persisted operation changed during no-op replay: status=%s updates=%d", operation.Status, repository.updates)
			}
		})
	}
}

func TestReconcileHostEventRejectsNonJournalFrontendOnlyDigestOutsideScope(t *testing.T) {
	service, repository := newUpgradeServiceForTest(t, &upgradeDispatcherStub{})
	now := time.Date(2026, 9, 21, 13, 0, 0, 0, time.UTC)
	planDigest := "sha256:" + strings.Repeat("a", 64)
	baselineDigest := "sha256:" + strings.Repeat("b", 64)
	operation := &domain.Operation{
		OperationID: "96969696-9696-4696-8696-969696969696", RequestID: "97979797-9797-4797-8797-979797979797", OperatorID: 7,
		ManifestID: "release-1.1.0", ManifestDigest: "sha256:" + strings.Repeat("c", 64), ReleaseVersion: "1.1.0",
		CompatibilityRange: "*", Status: domain.StatusUpdating, MigrationStatus: domain.MigrationStatusNotStarted, MigrationType: "none",
		ExecutionMode: domain.ExecutionModeFrontendOnly, WorkDisposition: domain.WorkDispositionNotRequired,
		PlanSummary: domain.PlanSummary{TouchedServices: []string{"frontend"}}, PlanDigest: planDigest,
		BaselineDeploymentDigest: baselineDigest, ConfirmedDeploymentVersion: "1.0.0",
		CreatedAt: now, UpdatedAt: now, StageTimes: map[domain.Status]time.Time{domain.StatusUpdating: now}, ObservedDigests: map[string]string{},
	}
	repository.byID[operation.OperationID] = operation
	repository.byRequest[operation.RequestID] = operation
	repository.active = operation

	_, err := service.ReconcileHostEvent(context.Background(), HostUpgradeEvent{
		OperationID: operation.OperationID, ManifestDigest: operation.ManifestDigest, Stage: string(domain.StatusUpdating),
		ObservedDigests: map[string]string{"server": "sha256:" + strings.Repeat("d", 64)},
	})
	if err == nil || !strings.Contains(err.Error(), "host observed digest service") {
		t.Fatalf("non-journal out-of-scope digest error = %v", err)
	}
	if repository.updates != 0 || len(operation.ObservedDigests) != 0 {
		t.Fatalf("non-journal out-of-scope digest changed operation: digests=%#v updates=%d", operation.ObservedDigests, repository.updates)
	}
}

func TestValidateJournalObservedDigestsRejectsMalformedDigest(t *testing.T) {
	operation := &domain.Operation{ExecutionMode: domain.ExecutionModeFrontendOnly}
	if err := validateJournalObservedDigests(operation, map[string]string{"frontend": " sha256:" + strings.Repeat("a", 64)}); err == nil || !strings.Contains(err.Error(), "host journal observed digest") {
		t.Fatalf("malformed journal observed digest error = %v", err)
	}
}

type recoveryJournalReaderStub struct {
	event HostUpgradeEvent
	err   error
}

func (stub recoveryJournalReaderStub) ReadCurrent(context.Context) (HostUpgradeEvent, error) {
	return stub.event, stub.err
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
