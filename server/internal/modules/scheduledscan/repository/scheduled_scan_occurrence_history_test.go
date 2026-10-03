package repository

import (
	"context"
	"testing"
	"time"

	scandomain "github.com/yyhuni/lunafox/server/internal/modules/scan/domain"
	scheduledapp "github.com/yyhuni/lunafox/server/internal/modules/scheduledscan/application"
)

// The occurrence history contract at the repository boundary: newest-first
// pagination plus status aggregation that stays in lockstep with the Go-side
// DeriveOccurrenceStatus precedence.
func TestScheduledScanRepositoryOccurrenceHistoryOrderingPaginationAndCounts(t *testing.T) {
	base := time.Date(2026, 9, 28, 2, 0, 0, 0, time.UTC)
	repo := newScheduledScanRepositoryForTest(t)
	targetID := 7
	schedule, err := repo.Create(context.Background(), &scheduledapp.ScheduledScanCreate{
		Name: "history", ScanWorkflowID: "default", Configuration: map[string]any{"version": 1},
		InputSource: scandomain.InputSourceScanSnapshot, TargetID: &targetID,
		TimeZone: "UTC", CronExpression: "0 2 * * *", IsEnabled: true,
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	attempted := base.Add(2*time.Hour + time.Second)
	dispatched := base.Add(2*time.Hour + 2*time.Second)
	retryAt := base.Add(3 * time.Hour)
	failureKind := "scan_create_failed"
	failureMessage := "Scan creation did not complete."
	failureCause := "ENGINE_UNAVAILABLE"
	seed := func(offset time.Duration, mutate func(*scheduledScanOccurrenceModel)) *scheduledScanOccurrenceModel {
		occurrence := &scheduledScanOccurrenceModel{ScheduledScanID: schedule.ID, ScheduledFor: base.Add(offset)}
		if mutate != nil {
			mutate(occurrence)
		}
		if err := repo.db.Create(occurrence).Error; err != nil {
			t.Fatalf("seed occurrence: %v", err)
		}
		return occurrence
	}
	succeeded := seed(3*24*time.Hour, func(row *scheduledScanOccurrenceModel) {
		row.AttemptedAt = &attempted
		row.DispatchedAt = &dispatched
	})
	failed := seed(2*24*time.Hour, func(row *scheduledScanOccurrenceModel) {
		attempt := base.Add(2*24*time.Hour + time.Second)
		row.AttemptedAt = &attempt
		row.FailureKind = &failureKind
		row.FailureMessage = &failureMessage
		row.LastFailureCause = &failureCause
		row.RetryCount = 3
	})
	retrying := seed(24*time.Hour, func(row *scheduledScanOccurrenceModel) {
		row.AttemptedAt = &attempted
		row.NextRetryAt = &retryAt
		row.RetryCount = 2
	})
	dispatching := seed(0, func(row *scheduledScanOccurrenceModel) {
		row.AttemptedAt = &attempted
	})
	pending := seed(-24*time.Hour, nil)

	snapshots, total, err := repo.ListOccurrences(context.Background(), scheduledapp.OccurrenceHistoryQuery{ScheduledScanID: schedule.ID, Page: 1, PageSize: 10})
	if err != nil {
		t.Fatalf("ListOccurrences() error = %v", err)
	}
	if total != 5 || len(snapshots) != 5 {
		t.Fatalf("expected all 5 rows, got total=%d len=%d", total, len(snapshots))
	}
	wantOrder := []int64{succeeded.ID, failed.ID, retrying.ID, dispatching.ID, pending.ID}
	for index, want := range wantOrder {
		if snapshots[index].ID != want {
			t.Fatalf("ordering mismatch at %d: got %d want %d (%+v)", index, snapshots[index].ID, want, snapshots)
		}
	}
	wantStatuses := []scheduledapp.OccurrenceStatus{
		scheduledapp.OccurrenceStatusSucceeded,
		scheduledapp.OccurrenceStatusFailed,
		scheduledapp.OccurrenceStatusRetrying,
		scheduledapp.OccurrenceStatusDispatching,
		scheduledapp.OccurrenceStatusPending,
	}
	for index, snapshot := range snapshots {
		if status := scheduledapp.DeriveOccurrenceStatus(snapshot); status != wantStatuses[index] {
			t.Fatalf("derived status mismatch for %d: got %s want %s", snapshot.ID, status, wantStatuses[index])
		}
	}

	pageOne, totalOne, err := repo.ListOccurrences(context.Background(), scheduledapp.OccurrenceHistoryQuery{ScheduledScanID: schedule.ID, Page: 1, PageSize: 2})
	if err != nil || totalOne != 5 || len(pageOne) != 2 || pageOne[0].ID != succeeded.ID || pageOne[1].ID != failed.ID {
		t.Fatalf("first page wrong: rows=%+v total=%d err=%v", pageOne, totalOne, err)
	}
	pageTwo, _, err := repo.ListOccurrences(context.Background(), scheduledapp.OccurrenceHistoryQuery{ScheduledScanID: schedule.ID, Page: 2, PageSize: 2})
	if err != nil || len(pageTwo) != 2 || pageTwo[0].ID != retrying.ID {
		t.Fatalf("second page wrong: rows=%+v err=%v", pageTwo, err)
	}

	counts, err := repo.CountOccurrencesByStatus(context.Background(), schedule.ID)
	if err != nil {
		t.Fatalf("CountOccurrencesByStatus() error = %v", err)
	}
	wantCounts := scheduledapp.OccurrenceStatusCounts{Pending: 1, Dispatching: 1, Retrying: 1, Succeeded: 1, Failed: 1}
	if counts != wantCounts {
		t.Fatalf("counts = %+v, want %+v", counts, wantCounts)
	}
	if counts.Total() != 5 {
		t.Fatalf("counts total = %d, want 5", counts.Total())
	}

	if _, _, err := repo.ListOccurrences(context.Background(), scheduledapp.OccurrenceHistoryQuery{ScheduledScanID: 0}); err == nil {
		t.Fatalf("expected invalid argument for zero schedule id")
	}
	if _, err := repo.CountOccurrencesByStatus(context.Background(), 0); err == nil {
		t.Fatalf("expected invalid argument for zero schedule id (counts)")
	}
}

func TestScheduledScanRepositoryOccurrenceHistoryEmptySchedule(t *testing.T) {
	repo := newScheduledScanRepositoryForTest(t)
	targetID := 7
	schedule, err := repo.Create(context.Background(), &scheduledapp.ScheduledScanCreate{
		Name: "empty", ScanWorkflowID: "default", Configuration: map[string]any{"version": 1},
		InputSource: scandomain.InputSourceScanSnapshot, TargetID: &targetID,
		TimeZone: "UTC", CronExpression: "0 2 * * *", IsEnabled: true,
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	snapshots, total, err := repo.ListOccurrences(context.Background(), scheduledapp.OccurrenceHistoryQuery{ScheduledScanID: schedule.ID, Page: 1, PageSize: 10})
	if err != nil || total != 0 || len(snapshots) != 0 {
		t.Fatalf("empty history wrong: rows=%+v total=%d err=%v", snapshots, total, err)
	}
	counts, err := repo.CountOccurrencesByStatus(context.Background(), schedule.ID)
	if err != nil || counts != (scheduledapp.OccurrenceStatusCounts{}) {
		t.Fatalf("empty counts wrong: %+v err=%v", counts, err)
	}
}
