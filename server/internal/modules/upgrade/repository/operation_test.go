package repository

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/yyhuni/lunafox/server/internal/modules/upgrade/domain"
	model "github.com/yyhuni/lunafox/server/internal/modules/upgrade/repository/persistence"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newUpgradeOperationRepositoryForTest(t *testing.T) *UpgradeOperationRepository {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+strings.ReplaceAll(t.Name(), "/", "_")+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&model.Operation{}); err != nil {
		t.Fatalf("migrate upgrade_operation: %v", err)
	}
	t.Cleanup(func() {
		sqlDB, err := db.DB()
		if err == nil {
			_ = sqlDB.Close()
		}
	})
	return NewUpgradeOperationRepository(db)
}

func newTestOperation() *domain.Operation {
	now := time.Date(2026, 9, 13, 12, 0, 0, 123456789, time.UTC)
	return &domain.Operation{
		OperationID:              "11111111-1111-4111-8111-111111111111",
		RequestID:                "22222222-2222-4222-8222-222222222222",
		OperatorID:               7,
		ManifestID:               "lunafox-1.2.3",
		ManifestDigest:           "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		ReleaseVersion:           "1.2.3",
		CompatibilityRange:       ">=1.0.0 <2.0.0",
		MaintenanceWindowMinutes: 15,
		Status:                   domain.StatusQueued,
		MigrationStatus:          domain.MigrationStatusNotStarted,
		MigrationType:            "none",
		ObservedDigests: map[string]string{
			"server":   "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
			"frontend": "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
		},
		StageTimes: map[domain.Status]time.Time{
			domain.StatusQueued: now,
		},
		ProgressEvents: []domain.ProgressEvent{{
			Timestamp: now, Stage: domain.StatusQueued, MessageKey: "preflightStarted",
			Message: "Preflight checks are running", Metadata: map[string]string{"scope": "release"},
		}},
		AgentSummary:        domain.AgentSummary{Expected: 3, Ready: 1, Missing: 1, Unhealthy: 1},
		CancelledScanCount:  2,
		CancelledTaskCount:  5,
		AgentDesiredVersion: "1.2.3",
		AgentTargetDigest:   "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		Diagnostic:          "",
		CreatedAt:           now,
		UpdatedAt:           now,
	}
}

func TestUpgradeOperationRepositoryRoundTripsAuditFields(t *testing.T) {
	repository := newUpgradeOperationRepositoryForTest(t)
	want := newTestOperation()
	got, created, err := repository.CreateOrGet(context.Background(), want)
	if err != nil {
		t.Fatalf("CreateOrGet() error = %v", err)
	}
	if !created {
		t.Fatal("CreateOrGet() created = false, want true")
	}
	if got.OperationID != want.OperationID || got.ManifestDigest != want.ManifestDigest || got.AgentSummary != want.AgentSummary {
		t.Fatalf("created operation = %#v, want %#v", got, want)
	}

	loaded, err := repository.Get(context.Background(), want.OperationID)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if loaded.ManifestID != want.ManifestID || loaded.MigrationType != want.MigrationType || loaded.CancelledTaskCount != want.CancelledTaskCount {
		t.Fatalf("loaded identity fields = %#v, want %#v", loaded, want)
	}
	if loaded.AgentSummary != want.AgentSummary {
		t.Fatalf("loaded agent summary = %#v, want %#v", loaded.AgentSummary, want.AgentSummary)
	}
	if loaded.ObservedDigests["server"] != want.ObservedDigests["server"] || !loaded.StageTimes[domain.StatusQueued].Equal(want.StageTimes[domain.StatusQueued]) {
		t.Fatalf("loaded JSON evidence = %#v/%#v, want %#v/%#v", loaded.ObservedDigests, loaded.StageTimes, want.ObservedDigests, want.StageTimes)
	}
	if len(loaded.ProgressEvents) != 1 || loaded.ProgressEvents[0].MessageKey != "preflightStarted" || loaded.ProgressEvents[0].Metadata["scope"] != "release" {
		t.Fatalf("loaded progress events = %#v", loaded.ProgressEvents)
	}
}

func TestUpgradeOperationRepositoryBindsRequestAndActiveOperation(t *testing.T) {
	repository := newUpgradeOperationRepositoryForTest(t)
	original := newTestOperation()
	if _, created, err := repository.CreateOrGet(context.Background(), original); err != nil || !created {
		t.Fatalf("create original: created=%t err=%v", created, err)
	}

	replayed := *original
	replayed.UpdatedAt = replayed.UpdatedAt.Add(time.Minute)
	got, created, err := repository.CreateOrGet(context.Background(), &replayed)
	if err != nil || created || got.OperationID != original.OperationID {
		t.Fatalf("same request replay = operation=%#v created=%t err=%v", got, created, err)
	}

	conflict := *original
	conflict.ManifestDigest = "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
	if _, _, err := repository.CreateOrGet(context.Background(), &conflict); !errors.Is(err, domain.ErrUpgradeRequestConflict) {
		t.Fatalf("expected request target conflict, got %v", err)
	}

	activeConflict := *original
	activeConflict.RequestID = "33333333-3333-4333-8333-333333333333"
	activeConflict.OperationID = "44444444-4444-4444-8444-444444444444"
	if _, _, err := repository.CreateOrGet(context.Background(), &activeConflict); !errors.Is(err, domain.ErrUpgradeAlreadyRunning) {
		t.Fatalf("expected active operation conflict, got %v", err)
	}

	completedAt := original.UpdatedAt.Add(2 * time.Minute)
	original.Status = domain.StatusSucceeded
	original.MigrationStatus = domain.MigrationStatusSucceeded
	original.CompletedAt = &completedAt
	original.UpdatedAt = completedAt
	if err := repository.Update(context.Background(), original); err != nil {
		t.Fatalf("Update() terminal operation: %v", err)
	}
	if _, err := repository.FindActive(context.Background()); !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("FindActive() error = %v, want gorm.ErrRecordNotFound", err)
	}
	if _, created, err := repository.CreateOrGet(context.Background(), &activeConflict); err != nil || !created {
		t.Fatalf("create operation after terminal state: created=%t err=%v", created, err)
	}
}

func TestUpgradeOperationRepositoryRejectsMissingIdentity(t *testing.T) {
	repository := newUpgradeOperationRepositoryForTest(t)
	operation := newTestOperation()
	operation.ManifestDigest = ""
	if _, _, err := repository.CreateOrGet(context.Background(), operation); err == nil {
		t.Fatal("CreateOrGet() accepted missing manifest digest")
	}
	if _, err := repository.Get(context.Background(), " "); !errors.Is(err, domain.ErrUpgradeNotFound) {
		t.Fatalf("Get() blank operation id error = %v", err)
	}
}

func TestUpgradeOperationRepositoryGetByRequestAndTransitionFence(t *testing.T) {
	repository := newUpgradeOperationRepositoryForTest(t)
	want := newTestOperation()
	if _, created, err := repository.CreateOrGet(context.Background(), want); err != nil || !created {
		t.Fatalf("create operation: created=%t err=%v", created, err)
	}
	loaded, err := repository.GetByRequest(context.Background(), want.RequestID)
	if err != nil || loaded.OperationID != want.OperationID {
		t.Fatalf("GetByRequest=%#v err=%v", loaded, err)
	}

	advanced := *want
	advanced.Status = domain.StatusStopping
	advanced.UpdatedAt = want.UpdatedAt.Add(time.Minute)
	advanced.StageTimes = map[domain.Status]time.Time{domain.StatusQueued: want.UpdatedAt, domain.StatusStopping: advanced.UpdatedAt}
	if err := repository.UpdateTransition(context.Background(), &advanced, domain.StatusQueued); err != nil {
		t.Fatalf("first transition: %v", err)
	}

	// A newer writer advances the row first. The delayed event still carries
	// the old expected status and must lose the compare-and-set fence.
	newer := advanced
	newer.Status = domain.StatusPreflight
	newer.UpdatedAt = advanced.UpdatedAt.Add(30 * time.Second)
	newer.StageTimes = map[domain.Status]time.Time{domain.StatusQueued: want.UpdatedAt, domain.StatusStopping: advanced.UpdatedAt, domain.StatusPreflight: newer.UpdatedAt}
	if err := repository.UpdateTransition(context.Background(), &newer, domain.StatusStopping); err != nil {
		t.Fatalf("newer transition: %v", err)
	}
	stale := newer
	stale.UpdatedAt = newer.UpdatedAt.Add(time.Minute)
	stale.StageTimes = map[domain.Status]time.Time{domain.StatusQueued: want.UpdatedAt, domain.StatusStopping: advanced.UpdatedAt, domain.StatusPreflight: stale.UpdatedAt}
	if err := repository.UpdateTransition(context.Background(), &stale, domain.StatusStopping); !errors.Is(err, domain.ErrUpgradeTransitionConflict) {
		t.Fatalf("stale transition error=%v", err)
	}
	current, err := repository.Get(context.Background(), want.OperationID)
	if err != nil {
		t.Fatal(err)
	}
	if current.Status != domain.StatusPreflight {
		t.Fatalf("stale transition changed status=%q", current.Status)
	}
}

func TestUpgradeOperationRepositoryMissingRequestIsNotFound(t *testing.T) {
	repository := newUpgradeOperationRepositoryForTest(t)
	if _, err := repository.GetByRequest(context.Background(), "22222222-2222-4222-8222-222222222222"); !errors.Is(err, domain.ErrUpgradeNotFound) {
		t.Fatalf("missing request error=%v", err)
	}
}

func TestUpgradeOperationRepositoryResetForRetryBindsTerminalTarget(t *testing.T) {
	repository := newUpgradeOperationRepositoryForTest(t)
	operation := newTestOperation()
	if _, created, err := repository.CreateOrGet(context.Background(), operation); err != nil || !created {
		t.Fatalf("create operation: created=%v err=%v", created, err)
	}

	failedAt := operation.UpdatedAt.Add(time.Minute)
	failed := *operation
	failed.Status = domain.StatusFailed
	failed.CompletedAt = &failedAt
	failed.UpdatedAt = failedAt
	failed.StageTimes = map[domain.Status]time.Time{domain.StatusQueued: operation.UpdatedAt, domain.StatusFailed: failedAt}
	if err := repository.Update(context.Background(), &failed); err != nil {
		t.Fatalf("persist failed operation: %v", err)
	}

	retryAt := failedAt.Add(time.Minute)
	retry := failed
	retry.Status = domain.StatusQueued
	retry.MigrationStatus = domain.MigrationStatusNotStarted
	retry.CompletedAt = nil
	retry.UpdatedAt = retryAt
	retry.StageTimes = map[domain.Status]time.Time{domain.StatusQueued: retryAt}
	if err := repository.ResetForRetry(context.Background(), &retry, domain.StatusFailed); err != nil {
		t.Fatalf("ResetForRetry() error=%v", err)
	}
	loaded, err := repository.Get(context.Background(), operation.OperationID)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Status != domain.StatusQueued || loaded.CompletedAt != nil || loaded.ManifestDigest != operation.ManifestDigest {
		t.Fatalf("retry reset changed unexpected fields: %#v", loaded)
	}

	failedAgain := retry
	failedAgain.Status = domain.StatusFailed
	failedAgain.CompletedAt = &retryAt
	failedAgain.UpdatedAt = retryAt.Add(time.Minute)
	failedAgain.StageTimes = map[domain.Status]time.Time{domain.StatusFailed: failedAgain.UpdatedAt}
	if err := repository.Update(context.Background(), &failedAgain); err != nil {
		t.Fatalf("persist second failed operation: %v", err)
	}
	badTarget := retry
	badTarget.ManifestDigest = "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
	badTarget.UpdatedAt = failedAgain.UpdatedAt.Add(time.Minute)
	if err := repository.ResetForRetry(context.Background(), &badTarget, domain.StatusFailed); !errors.Is(err, domain.ErrUpgradeTransitionConflict) {
		t.Fatalf("target drift reset error=%v", err)
	}
	if err := repository.ResetForRetry(context.Background(), &retry, domain.StatusSucceeded); err == nil {
		t.Fatal("accepted retry reset with non-retryable expected status")
	}
}
