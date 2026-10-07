package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	scandomain "github.com/yyhuni/lunafox/server/internal/modules/scan/domain"
	scheduledapp "github.com/yyhuni/lunafox/server/internal/modules/scheduledscan/application"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestScheduledScanRepositoryLifecycleMaintainsCursorAndOccurrenceInvariants(t *testing.T) {
	now := time.Date(2026, 8, 4, 10, 0, 30, 0, time.UTC)
	repo := newScheduledScanRepositoryForTest(t).WithClock(func() time.Time { return now })
	targetID := 7

	enabled, err := repo.Create(context.Background(), &scheduledapp.ScheduledScanCreate{
		Name: "enabled", ScanWorkflowID: "default", Configuration: map[string]any{"version": 1}, InputSource: scandomain.InputSourceScanSnapshot,
		TargetID: &targetID, TimeZone: "UTC", CronExpression: "* * * * *", IsEnabled: true,
	})
	if err != nil {
		t.Fatalf("Create(enabled) error = %v", err)
	}
	wantNext := time.Date(2026, 8, 4, 10, 1, 0, 0, time.UTC)
	if enabled.NextRunTime == nil || !enabled.NextRunTime.Equal(wantNext) {
		t.Fatalf("enabled nextRunTime = %v, want %s", enabled.NextRunTime, wantNext)
	}

	disabled, err := repo.Create(context.Background(), &scheduledapp.ScheduledScanCreate{
		Name: "disabled", ScanWorkflowID: "default", Configuration: map[string]any{"version": 1}, InputSource: scandomain.InputSourceScanSnapshot,
		TargetID: &targetID, TimeZone: "UTC", CronExpression: "* * * * *", IsEnabled: false,
	})
	if err != nil || disabled.NextRunTime != nil {
		t.Fatalf("Create(disabled) = %+v, %v; want null cursor", disabled, err)
	}

	name := "renamed"
	unchanged, err := repo.Update(context.Background(), enabled.ID, &scheduledapp.ScheduledScanUpdate{Name: &name})
	if err != nil || unchanged.NextRunTime == nil || !unchanged.NextRunTime.Equal(wantNext) {
		t.Fatalf("non-time Update changed cursor: %+v error=%v", unchanged, err)
	}

	if err := repo.db.Create(&scheduledScanOccurrenceModel{ScheduledScanID: enabled.ID, ScheduledFor: wantNext}).Error; err != nil {
		t.Fatalf("seed unattempted occurrence: %v", err)
	}
	falseValue := false
	disabledResult, err := repo.Update(context.Background(), enabled.ID, &scheduledapp.ScheduledScanUpdate{IsEnabled: &falseValue})
	if err != nil || disabledResult.IsEnabled || disabledResult.NextRunTime != nil {
		t.Fatalf("disable Update = %+v, %v", disabledResult, err)
	}
	var unattempted int64
	repo.db.Model(&scheduledScanOccurrenceModel{}).Where("scheduled_scan_id = ? AND attempted_at IS NULL", enabled.ID).Count(&unattempted)
	if unattempted != 0 {
		t.Fatalf("disable retained %d unattempted occurrences", unattempted)
	}

	now = now.Add(2 * time.Hour)
	trueValue := true
	reenabled, err := repo.Update(context.Background(), enabled.ID, &scheduledapp.ScheduledScanUpdate{IsEnabled: &trueValue})
	if err != nil || reenabled.NextRunTime == nil || !reenabled.NextRunTime.After(now) {
		t.Fatalf("re-enable did not create a fresh future cursor: %+v error=%v", reenabled, err)
	}
}

func TestScheduledScanRepositoryBatchStatusUpdatePreservesLifecycleForMixedStates(t *testing.T) {
	now := time.Date(2026, 8, 4, 10, 0, 30, 0, time.UTC)
	repo := newScheduledScanRepositoryForTest(t).WithClock(func() time.Time { return now })
	targetID := 7
	enabled, err := repo.Create(context.Background(), &scheduledapp.ScheduledScanCreate{
		Name: "enabled", ScanWorkflowID: "default", Configuration: map[string]any{"version": 1}, InputSource: scandomain.InputSourceScanSnapshot,
		TargetID: &targetID, TimeZone: "UTC", CronExpression: "* * * * *", IsEnabled: true,
	})
	if err != nil {
		t.Fatalf("Create(enabled) error = %v", err)
	}
	disabled, err := repo.Create(context.Background(), &scheduledapp.ScheduledScanCreate{
		Name: "disabled", ScanWorkflowID: "default", Configuration: map[string]any{"version": 1}, InputSource: scandomain.InputSourceScanSnapshot,
		TargetID: &targetID, TimeZone: "UTC", CronExpression: "* * * * *", IsEnabled: false,
	})
	if err != nil {
		t.Fatalf("Create(disabled) error = %v", err)
	}
	attemptedAt := now.Add(-time.Minute)
	settledKind := "scan_create_failed"
	settledMessage := "settled failure"
	if err := repo.db.Create(&scheduledScanOccurrenceModel{ScheduledScanID: enabled.ID, ScheduledFor: now.Add(-3 * time.Minute), AttemptedAt: &attemptedAt, FailureKind: &settledKind, FailureMessage: &settledMessage}).Error; err != nil {
		t.Fatalf("seed settled occurrence: %v", err)
	}
	retryAt := now.Add(-30 * time.Second)
	if err := repo.db.Create(&scheduledScanOccurrenceModel{ScheduledScanID: enabled.ID, ScheduledFor: now.Add(-2 * time.Minute), AttemptedAt: &attemptedAt, NextRetryAt: &retryAt}).Error; err != nil {
		t.Fatalf("seed retry-pending occurrence: %v", err)
	}
	if err := repo.db.Create(&scheduledScanOccurrenceModel{ScheduledScanID: enabled.ID, ScheduledFor: now.Add(-time.Minute)}).Error; err != nil {
		t.Fatalf("seed unattempted occurrence: %v", err)
	}

	updatedCount, err := repo.BatchUpdateStatus(context.Background(), []scheduledapp.ScheduledScanStatusUpdate{
		{ID: disabled.ID, IsEnabled: true},
		{ID: enabled.ID, IsEnabled: false},
	})
	if err != nil || updatedCount != 2 {
		t.Fatalf("BatchUpdateStatus() = %d, %v", updatedCount, err)
	}

	disabledAfter, err := repo.GetByID(context.Background(), enabled.ID)
	if err != nil || disabledAfter.IsEnabled || disabledAfter.NextRunTime != nil {
		t.Fatalf("enabled schedule was not disabled: %+v, %v", disabledAfter, err)
	}
	enabledAfter, err := repo.GetByID(context.Background(), disabled.ID)
	wantNext := time.Date(2026, 8, 4, 10, 1, 0, 0, time.UTC)
	if err != nil || !enabledAfter.IsEnabled || enabledAfter.NextRunTime == nil || !enabledAfter.NextRunTime.Equal(wantNext) {
		t.Fatalf("disabled schedule was not enabled from shared reference time: %+v, %v", enabledAfter, err)
	}
	var unattemptedCount int64
	if err := repo.db.Model(&scheduledScanOccurrenceModel{}).
		Where("scheduled_scan_id = ? AND attempted_at IS NULL", enabled.ID).
		Count(&unattemptedCount).Error; err != nil {
		t.Fatalf("count unattempted occurrences: %v", err)
	}
	if unattemptedCount != 0 {
		t.Fatalf("disable retained %d unattempted occurrences", unattemptedCount)
	}
	var settledCount int64
	if err := repo.db.Model(&scheduledScanOccurrenceModel{}).
		Where("scheduled_scan_id = ? AND failure_kind IS NOT NULL", enabled.ID).
		Count(&settledCount).Error; err != nil {
		t.Fatalf("count settled occurrences: %v", err)
	}
	if settledCount != 1 {
		t.Fatalf("disable removed %d settled occurrences", settledCount)
	}
	var pendingRetryCount int64
	if err := repo.db.Model(&scheduledScanOccurrenceModel{}).
		Where("scheduled_scan_id = ? AND next_retry_at IS NOT NULL", enabled.ID).
		Count(&pendingRetryCount).Error; err != nil {
		t.Fatalf("count retry-pending occurrences: %v", err)
	}
	if pendingRetryCount != 0 {
		t.Fatalf("disable retained %d retry-pending occurrences", pendingRetryCount)
	}
}

func TestScheduledScanRepositoryBatchStatusUpdateIsNaturallyIdempotent(t *testing.T) {
	now := time.Date(2026, 8, 4, 10, 0, 30, 0, time.UTC)
	repo := newScheduledScanRepositoryForTest(t).WithClock(func() time.Time { return now })
	targetID := 7
	enabled, err := repo.Create(context.Background(), &scheduledapp.ScheduledScanCreate{
		Name: "enabled", ScanWorkflowID: "default", Configuration: map[string]any{"version": 1}, InputSource: scandomain.InputSourceScanSnapshot,
		TargetID: &targetID, TimeZone: "UTC", CronExpression: "* * * * *", IsEnabled: true,
	})
	if err != nil {
		t.Fatalf("Create(enabled) error = %v", err)
	}
	disabled, err := repo.Create(context.Background(), &scheduledapp.ScheduledScanCreate{
		Name: "disabled", ScanWorkflowID: "default", Configuration: map[string]any{"version": 1}, InputSource: scandomain.InputSourceScanSnapshot,
		TargetID: &targetID, TimeZone: "UTC", CronExpression: "* * * * *", IsEnabled: false,
	})
	if err != nil {
		t.Fatalf("Create(disabled) error = %v", err)
	}
	originalCursor := *enabled.NextRunTime

	updatedCount, err := repo.BatchUpdateStatus(context.Background(), []scheduledapp.ScheduledScanStatusUpdate{
		{ID: disabled.ID, IsEnabled: false},
		{ID: enabled.ID, IsEnabled: true},
	})
	if err != nil || updatedCount != 2 {
		t.Fatalf("first BatchUpdateStatus() = %d, %v", updatedCount, err)
	}
	updatedCount, err = repo.BatchUpdateStatus(context.Background(), []scheduledapp.ScheduledScanStatusUpdate{
		{ID: enabled.ID, IsEnabled: true},
		{ID: disabled.ID, IsEnabled: false},
	})
	if err != nil || updatedCount != 2 {
		t.Fatalf("replayed BatchUpdateStatus() = %d, %v", updatedCount, err)
	}

	enabledAfter, err := repo.GetByID(context.Background(), enabled.ID)
	if err != nil || !enabledAfter.IsEnabled || enabledAfter.NextRunTime == nil || !enabledAfter.NextRunTime.Equal(originalCursor) {
		t.Fatalf("idempotent enable changed cursor: %+v, %v", enabledAfter, err)
	}
	disabledAfter, err := repo.GetByID(context.Background(), disabled.ID)
	if err != nil || disabledAfter.IsEnabled || disabledAfter.NextRunTime != nil {
		t.Fatalf("idempotent disable changed status: %+v, %v", disabledAfter, err)
	}
}

func TestScheduledScanRepositoryBatchStatusUpdateRollsBackEveryScheduleOnCursorFailure(t *testing.T) {
	now := time.Date(2026, 8, 4, 10, 0, 30, 0, time.UTC)
	repo := newScheduledScanRepositoryForTest(t).WithClock(func() time.Time { return now })
	targetID := 7
	enabled, err := repo.Create(context.Background(), &scheduledapp.ScheduledScanCreate{
		Name: "enabled", ScanWorkflowID: "default", Configuration: map[string]any{"version": 1}, InputSource: scandomain.InputSourceScanSnapshot,
		TargetID: &targetID, TimeZone: "UTC", CronExpression: "* * * * *", IsEnabled: true,
	})
	if err != nil {
		t.Fatalf("Create(enabled) error = %v", err)
	}
	disabled, err := repo.Create(context.Background(), &scheduledapp.ScheduledScanCreate{
		Name: "disabled", ScanWorkflowID: "default", Configuration: map[string]any{"version": 1}, InputSource: scandomain.InputSourceScanSnapshot,
		TargetID: &targetID, TimeZone: "UTC", CronExpression: "* * * * *", IsEnabled: false,
	})
	if err != nil {
		t.Fatalf("Create(disabled) error = %v", err)
	}
	originalCursor := *enabled.NextRunTime
	occurrence := &scheduledScanOccurrenceModel{ScheduledScanID: enabled.ID, ScheduledFor: now}
	if err := repo.db.Create(occurrence).Error; err != nil {
		t.Fatalf("seed occurrence: %v", err)
	}
	wantErr := errors.New("batch cursor calculation failed")
	repo.calculator = firstAfterFailingScheduleCalculator{ScheduleCalculator: scheduledapp.NewCronScheduleCalculator(), err: wantErr}

	_, err = repo.BatchUpdateStatus(context.Background(), []scheduledapp.ScheduledScanStatusUpdate{
		{ID: enabled.ID, IsEnabled: false},
		{ID: disabled.ID, IsEnabled: true},
	})
	if !errors.Is(err, wantErr) {
		t.Fatalf("BatchUpdateStatus() error = %v, want %v", err, wantErr)
	}
	enabledAfter, err := repo.GetByID(context.Background(), enabled.ID)
	if err != nil || !enabledAfter.IsEnabled || enabledAfter.NextRunTime == nil || !enabledAfter.NextRunTime.Equal(originalCursor) {
		t.Fatalf("cursor failure changed first schedule: %+v, %v", enabledAfter, err)
	}
	disabledAfter, err := repo.GetByID(context.Background(), disabled.ID)
	if err != nil || disabledAfter.IsEnabled || disabledAfter.NextRunTime != nil {
		t.Fatalf("cursor failure changed second schedule: %+v, %v", disabledAfter, err)
	}
	var occurrenceCount int64
	if err := repo.db.Model(&scheduledScanOccurrenceModel{}).Where("id = ?", occurrence.ID).Count(&occurrenceCount).Error; err != nil {
		t.Fatalf("count occurrence: %v", err)
	}
	if occurrenceCount != 1 {
		t.Fatalf("cursor failure committed occurrence cleanup: count=%d", occurrenceCount)
	}
}

func TestScheduledScanRepositoryBatchStatusUpdateRollsBackEarlierWrites(t *testing.T) {
	now := time.Date(2026, 8, 4, 10, 0, 30, 0, time.UTC)
	repo := newScheduledScanRepositoryForTest(t).WithClock(func() time.Time { return now })
	targetID := 7
	first, err := repo.Create(context.Background(), &scheduledapp.ScheduledScanCreate{
		Name: "first", ScanWorkflowID: "default", Configuration: map[string]any{"version": 1}, InputSource: scandomain.InputSourceScanSnapshot,
		TargetID: &targetID, TimeZone: "UTC", CronExpression: "* * * * *", IsEnabled: true,
	})
	if err != nil {
		t.Fatalf("Create(first) error = %v", err)
	}
	second, err := repo.Create(context.Background(), &scheduledapp.ScheduledScanCreate{
		Name: "second", ScanWorkflowID: "default", Configuration: map[string]any{"version": 1}, InputSource: scandomain.InputSourceScanSnapshot,
		TargetID: &targetID, TimeZone: "UTC", CronExpression: "* * * * *", IsEnabled: true,
	})
	if err != nil {
		t.Fatalf("Create(second) error = %v", err)
	}
	trigger := fmt.Sprintf(`CREATE TRIGGER reject_second_batch_schedule_write BEFORE UPDATE ON scheduled_scan WHEN NEW.id = %d BEGIN SELECT RAISE(ABORT, 'injected batch schedule write failure'); END`, second.ID)
	if err := repo.db.Exec(trigger).Error; err != nil {
		t.Fatalf("create failure trigger: %v", err)
	}

	if _, err := repo.BatchUpdateStatus(context.Background(), []scheduledapp.ScheduledScanStatusUpdate{
		{ID: first.ID, IsEnabled: false},
		{ID: second.ID, IsEnabled: false},
	}); err == nil {
		t.Fatal("BatchUpdateStatus unexpectedly succeeded")
	}
	firstAfter, err := repo.GetByID(context.Background(), first.ID)
	if err != nil || !firstAfter.IsEnabled || firstAfter.NextRunTime == nil {
		t.Fatalf("failed batch changed first schedule: %+v, %v", firstAfter, err)
	}
	secondAfter, err := repo.GetByID(context.Background(), second.ID)
	if err != nil || !secondAfter.IsEnabled || secondAfter.NextRunTime == nil {
		t.Fatalf("failed batch changed second schedule: %+v, %v", secondAfter, err)
	}
}

func TestScheduledScanRepositoryBatchStatusUpdateRejectsMissingSchedule(t *testing.T) {
	repo := newScheduledScanRepositoryForTest(t)
	_, err := repo.BatchUpdateStatus(context.Background(), []scheduledapp.ScheduledScanStatusUpdate{{ID: 999, IsEnabled: false}})
	if !errors.Is(err, scheduledapp.ErrScheduledScanNotFound) {
		t.Fatalf("BatchUpdateStatus() error = %v, want not found", err)
	}
}

func TestScheduledScanRepositoryDisabledRuleEditsKeepNullCursorUntilReenabled(t *testing.T) {
	now := time.Date(2026, 8, 4, 10, 0, 30, 0, time.UTC)
	repo := newScheduledScanRepositoryForTest(t).WithClock(func() time.Time { return now })
	targetID := 7
	schedule, err := repo.Create(context.Background(), &scheduledapp.ScheduledScanCreate{
		Name: "disabled", ScanWorkflowID: "default", Configuration: map[string]any{"version": 1}, InputSource: scandomain.InputSourceScanSnapshot,
		TargetID: &targetID, TimeZone: "UTC", CronExpression: "0 * * * *", IsEnabled: false,
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	cron := "30 9 * * *"
	edited, err := repo.Update(context.Background(), schedule.ID, &scheduledapp.ScheduledScanUpdate{
		CronExpression: &cron,
	})
	if err != nil || edited.NextRunTime != nil || edited.CronExpression != cron {
		t.Fatalf("disabled time edit = %+v, %v; want saved rule and null cursor", edited, err)
	}

	now = now.Add(12 * time.Hour)
	enabled := true
	reenabled, err := repo.Update(context.Background(), schedule.ID, &scheduledapp.ScheduledScanUpdate{IsEnabled: &enabled})
	if err != nil || reenabled.NextRunTime == nil || !reenabled.NextRunTime.After(now) {
		t.Fatalf("re-enable = %+v, %v; want fresh future cursor", reenabled, err)
	}
}

func TestScheduledScanRepositoryTimeRuleUpdateReplacesCursorFromOneReferenceTime(t *testing.T) {
	now := time.Date(2026, 8, 4, 10, 0, 30, 0, time.UTC)
	repo := newScheduledScanRepositoryForTest(t).WithClock(func() time.Time { return now })
	targetID := 7
	schedule, err := repo.Create(context.Background(), &scheduledapp.ScheduledScanCreate{
		Name: "time-rule", ScanWorkflowID: "default", Configuration: map[string]any{"version": 1}, InputSource: scandomain.InputSourceScanSnapshot,
		TargetID: &targetID, TimeZone: "UTC", CronExpression: "* * * * *", IsEnabled: true,
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	cron := "15 19 * * *"
	updated, err := repo.Update(context.Background(), schedule.ID, &scheduledapp.ScheduledScanUpdate{
		CronExpression: &cron,
	})
	want := time.Date(2026, 8, 4, 19, 15, 0, 0, time.UTC)
	if err != nil || updated.NextRunTime == nil || !updated.NextRunTime.Equal(want) {
		t.Fatalf("time-rule Update cursor = %+v, %v; want %s", updated, err, want)
	}
}

func TestScheduledScanRepositoryDoesNotRecalculateCursorWhenTimeRuleValuesAreUnchanged(t *testing.T) {
	now := time.Date(2026, 8, 4, 10, 0, 30, 0, time.UTC)
	repo := newScheduledScanRepositoryForTest(t).WithClock(func() time.Time { return now })
	targetID := 7
	schedule, err := repo.Create(context.Background(), &scheduledapp.ScheduledScanCreate{
		Name: "unchanged-time-rule", ScanWorkflowID: "default", Configuration: map[string]any{"version": 1}, InputSource: scandomain.InputSourceScanSnapshot,
		TargetID: &targetID, TimeZone: "UTC", CronExpression: "0 19 * * *", IsEnabled: true,
	})
	if err != nil || schedule.NextRunTime == nil {
		t.Fatalf("Create() = %+v, %v", schedule, err)
	}
	originalCursor := *schedule.NextRunTime

	// Advance beyond the stored cursor so an unnecessary recalculation would be observable.
	now = time.Date(2026, 8, 4, 20, 0, 30, 0, time.UTC)
	cron := "0 19 * * *"
	timeZone := "UTC"
	updated, err := repo.Update(context.Background(), schedule.ID, &scheduledapp.ScheduledScanUpdate{
		CronExpression: &cron,
		TimeZone:       &timeZone,
	})
	if err != nil || updated.NextRunTime == nil || !updated.NextRunTime.Equal(originalCursor) {
		t.Fatalf("unchanged time-rule Update changed cursor: %+v error=%v, want %s", updated, err, originalCursor)
	}
}

func TestScheduledScanRepositoryTimeZoneUpdateReplacesCursorWithoutReadSideEffects(t *testing.T) {
	now := time.Date(2026, 8, 4, 10, 0, 30, 0, time.UTC)
	repo := newScheduledScanRepositoryForTest(t).WithClock(func() time.Time { return now })
	targetID := 7
	schedule, err := repo.Create(context.Background(), &scheduledapp.ScheduledScanCreate{
		Name: "shanghai-wall-clock", ScanWorkflowID: "default", Configuration: map[string]any{"version": 1}, InputSource: scandomain.InputSourceScanSnapshot,
		TargetID: &targetID, TimeZone: "UTC", CronExpression: "0 19 * * *", IsEnabled: true,
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if schedule.TimeZone != "UTC" || schedule.CronExpression != "0 19 * * *" || schedule.NextRunTime == nil {
		t.Fatalf("Create() did not persist the original UTC rule: %+v", schedule)
	}
	if want := time.Date(2026, 8, 4, 19, 0, 0, 0, time.UTC); !schedule.NextRunTime.Equal(want) {
		t.Fatalf("initial cursor = %s, want %s", schedule.NextRunTime, want)
	}

	timeZone := "Asia/Shanghai"
	updated, err := repo.Update(context.Background(), schedule.ID, &scheduledapp.ScheduledScanUpdate{TimeZone: &timeZone})
	if err != nil {
		t.Fatalf("Update(timeZone) error = %v", err)
	}
	if updated.TimeZone != timeZone || updated.CronExpression != "0 19 * * *" || updated.NextRunTime == nil {
		t.Fatalf("timeZone update changed the stored rule unexpectedly: %+v", updated)
	}
	wantCursor := time.Date(2026, 8, 4, 11, 0, 0, 0, time.UTC)
	if !updated.NextRunTime.Equal(wantCursor) {
		t.Fatalf("timeZone update cursor = %s, want Shanghai wall-clock cursor %s", updated.NextRunTime, wantCursor)
	}

	read, err := repo.GetByID(context.Background(), schedule.ID)
	if err != nil || read.NextRunTime == nil || !read.NextRunTime.Equal(wantCursor) || read.TimeZone != timeZone {
		t.Fatalf("GetByID() rewrote the persisted time rule or cursor: %+v, %v", read, err)
	}
	listed, _, err := repo.List(context.Background(), scheduledapp.ScheduledScanListQuery{Page: 1, PageSize: 20})
	if err != nil || len(listed) != 1 || listed[0].NextRunTime == nil || !listed[0].NextRunTime.Equal(wantCursor) || listed[0].TimeZone != timeZone {
		t.Fatalf("List() rewrote the persisted time rule or cursor: %+v, %v", listed, err)
	}
}

func TestScheduledScanRepositoryMaterializesLatestDueAndStartsOneFrozenAttempt(t *testing.T) {
	evaluationAt := time.Date(2026, 8, 4, 10, 10, 0, 0, time.UTC)
	repo := newScheduledScanRepositoryForTest(t).WithClock(func() time.Time { return evaluationAt })
	targetID := 7
	schedule, err := repo.Create(context.Background(), &scheduledapp.ScheduledScanCreate{
		Name: "due", ScanWorkflowID: "default", Configuration: map[string]any{"version": 1}, InputSource: scandomain.InputSourceScanSnapshot,
		TargetID: &targetID, TimeZone: "UTC", CronExpression: "* * * * *", IsEnabled: true,
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	persistedCursor := evaluationAt.Add(-10 * time.Minute)
	if err := repo.db.Model(&scheduledScanModel{}).Where("id = ?", schedule.ID).Update("next_run_time", persistedCursor).Error; err != nil {
		t.Fatalf("make schedule due: %v", err)
	}

	due, err := repo.ListDueSchedules(context.Background(), evaluationAt)
	if err != nil || len(due) != 1 || due[0].ID != schedule.ID {
		t.Fatalf("ListDueSchedules() = %+v, %v", due, err)
	}
	committed, err := repo.MaterializeDue(context.Background(), schedule.ID, evaluationAt)
	if err != nil || !committed {
		t.Fatalf("MaterializeDue() = %v, %v", committed, err)
	}
	var occurrences []scheduledScanOccurrenceModel
	if err := repo.db.Where("scheduled_scan_id = ?", schedule.ID).Find(&occurrences).Error; err != nil {
		t.Fatalf("read occurrences: %v", err)
	}
	if len(occurrences) != 1 || !occurrences[0].ScheduledFor.Equal(evaluationAt) {
		t.Fatalf("latest-only occurrence = %+v, want %s", occurrences, evaluationAt)
	}
	stored, err := repo.GetByID(context.Background(), schedule.ID)
	if err != nil || stored.NextRunTime == nil || !stored.NextRunTime.Equal(evaluationAt.Add(time.Minute)) {
		t.Fatalf("advanced cursor = %+v, %v", stored, err)
	}

	candidate, err := repo.SelectAttemptCandidate(context.Background())
	if err != nil || candidate == nil || candidate.ID != occurrences[0].ID {
		t.Fatalf("SelectAttemptCandidate() = %+v, %v", candidate, err)
	}
	attemptedAt := evaluationAt.Add(5 * time.Second)
	frozen, err := repo.StartAttempt(context.Background(), *candidate, attemptedAt)
	if err != nil || frozen == nil {
		t.Fatalf("StartAttempt() = %+v, %v", frozen, err)
	}
	if frozen.Configuration["version"] != float64(1) || !frozen.TargetScoped || len(frozen.TargetIDs) != 1 || frozen.TargetIDs[0] != targetID {
		t.Fatalf("frozen input mismatch: %+v", frozen)
	}
	stored, _ = repo.GetByID(context.Background(), schedule.ID)
	if stored.RunCount != 1 || stored.SuccessfulHandoffCount != 0 || stored.FailedHandoffCount != 0 || stored.LastRunTime == nil || !stored.LastRunTime.Equal(attemptedAt) {
		t.Fatalf("attempt aggregates mismatch: %+v", stored)
	}
	if replay, err := repo.StartAttempt(context.Background(), *candidate, attemptedAt.Add(time.Second)); err != nil || replay != nil {
		t.Fatalf("attempt replay = %+v, %v; want no work", replay, err)
	}
}

func TestScheduledScanRepositoryOrganizationAttemptFreezesCurrentActiveTargetsOnly(t *testing.T) {
	repo := newScheduledScanRepositoryForTest(t)
	organizationID := 41
	if err := repo.db.Exec(`CREATE TABLE IF NOT EXISTS organization_target (
		organization_id INTEGER NOT NULL,
		target_id INTEGER NOT NULL,
		PRIMARY KEY (organization_id, target_id)
	)`).Error; err != nil {
		t.Fatalf("create organization_target table: %v", err)
	}
	if err := repo.db.Exec("INSERT OR IGNORE INTO organization (id, name) VALUES (?, ?)", organizationID, "Acme").Error; err != nil {
		t.Fatalf("seed organization: %v", err)
	}
	if err := repo.db.Exec(`
		INSERT OR IGNORE INTO target (id, name, deleted_at) VALUES
			(8, 'active-eight.example', NULL),
			(9, 'active-nine.example', NULL),
			(10, 'deleted-ten.example', CURRENT_TIMESTAMP),
			(11, 'active-eleven.example', NULL)`).Error; err != nil {
		t.Fatalf("seed Targets: %v", err)
	}
	if err := repo.db.Exec(`INSERT INTO organization_target (organization_id, target_id) VALUES (?, ?), (?, ?), (?, ?)`, organizationID, 8, organizationID, 9, organizationID, 10).Error; err != nil {
		t.Fatalf("seed organization membership: %v", err)
	}

	schedule, err := repo.Create(context.Background(), &scheduledapp.ScheduledScanCreate{
		Name: "organization-attempt", ScanWorkflowID: "default", Configuration: map[string]any{"version": 1}, InputSource: scandomain.InputSourceScanSnapshot,
		OrganizationID: &organizationID, TimeZone: "UTC", CronExpression: "* * * * *", IsEnabled: true,
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	firstOccurrence := &scheduledScanOccurrenceModel{ScheduledScanID: schedule.ID, ScheduledFor: time.Date(2026, 8, 4, 10, 0, 0, 0, time.UTC)}
	if err := repo.db.Create(firstOccurrence).Error; err != nil {
		t.Fatalf("seed first occurrence: %v", err)
	}
	first, err := repo.StartAttempt(context.Background(), scheduledapp.OccurrenceCandidate{
		ID: firstOccurrence.ID, ScheduledScanID: schedule.ID, ScheduledFor: firstOccurrence.ScheduledFor,
	}, firstOccurrence.ScheduledFor.Add(time.Second))
	if err != nil || first == nil {
		t.Fatalf("StartAttempt(first) = %+v, %v", first, err)
	}
	if first.TargetScoped || len(first.TargetIDs) != 2 || first.TargetIDs[0] != 8 || first.TargetIDs[1] != 9 {
		t.Fatalf("first frozen Target IDs = %+v, want active [8 9]", first)
	}

	if err := repo.db.Exec("DELETE FROM organization_target WHERE organization_id = ? AND target_id = ?", organizationID, 8).Error; err != nil {
		t.Fatalf("remove member: %v", err)
	}
	if err := repo.db.Exec("INSERT INTO organization_target (organization_id, target_id) VALUES (?, ?)", organizationID, 11).Error; err != nil {
		t.Fatalf("add member: %v", err)
	}
	secondOccurrence := &scheduledScanOccurrenceModel{ScheduledScanID: schedule.ID, ScheduledFor: firstOccurrence.ScheduledFor.Add(time.Minute)}
	if err := repo.db.Create(secondOccurrence).Error; err != nil {
		t.Fatalf("seed second occurrence: %v", err)
	}
	second, err := repo.StartAttempt(context.Background(), scheduledapp.OccurrenceCandidate{
		ID: secondOccurrence.ID, ScheduledScanID: schedule.ID, ScheduledFor: secondOccurrence.ScheduledFor,
	}, secondOccurrence.ScheduledFor.Add(time.Second))
	if err != nil || second == nil {
		t.Fatalf("StartAttempt(second) = %+v, %v", second, err)
	}
	if second.TargetScoped || len(second.TargetIDs) != 2 || second.TargetIDs[0] != 9 || second.TargetIDs[1] != 11 {
		t.Fatalf("second frozen Target IDs = %+v, want current active [9 11]", second)
	}

	var scheduleRow struct {
		TargetID *int `gorm:"column:target_id"`
	}
	if err := repo.db.Table("scheduled_scan").Select("target_id").Where("id = ?", schedule.ID).Take(&scheduleRow).Error; err != nil {
		t.Fatalf("read organization Schedule: %v", err)
	}
	if scheduleRow.TargetID != nil {
		t.Fatalf("organization Schedule retained a Target association: %+v", scheduleRow)
	}
}

func TestScheduledScanRepositoryRestartKeepsExistingOccurrenceIndependent(t *testing.T) {
	evaluationAt := time.Date(2026, 8, 4, 10, 10, 0, 0, time.UTC)
	repo := newScheduledScanRepositoryForTest(t)
	targetID := 7
	schedule, err := repo.Create(context.Background(), &scheduledapp.ScheduledScanCreate{
		Name: "restart", ScanWorkflowID: "default", Configuration: map[string]any{"version": 1}, InputSource: scandomain.InputSourceScanSnapshot,
		TargetID: &targetID, TimeZone: "UTC", CronExpression: "* * * * *", IsEnabled: true,
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	oldOccurrence := &scheduledScanOccurrenceModel{ScheduledScanID: schedule.ID, ScheduledFor: evaluationAt.Add(-20 * time.Minute)}
	if err := repo.db.Create(oldOccurrence).Error; err != nil {
		t.Fatalf("seed old occurrence: %v", err)
	}
	persistedCursor := evaluationAt.Add(-10 * time.Minute)
	if err := repo.db.Model(&scheduledScanModel{}).Where("id = ?", schedule.ID).Update("next_run_time", persistedCursor).Error; err != nil {
		t.Fatalf("set persisted cursor: %v", err)
	}
	if committed, err := repo.MaterializeDue(context.Background(), schedule.ID, evaluationAt); err != nil || !committed {
		t.Fatalf("MaterializeDue() = %v, %v", committed, err)
	}
	var count int64
	repo.db.Model(&scheduledScanOccurrenceModel{}).Where("scheduled_scan_id = ?", schedule.ID).Count(&count)
	if count != 2 {
		t.Fatalf("restart folded existing occurrence into missed-run recovery: count=%d", count)
	}
	candidate, err := repo.SelectAttemptCandidate(context.Background())
	if err != nil || candidate == nil || candidate.ID != oldOccurrence.ID {
		t.Fatalf("restart candidate = %+v, %v; want pre-existing oldest row", candidate, err)
	}
}

func TestScheduledScanRepositoryMaterializationRollsBackCursorWhenInsertFails(t *testing.T) {
	evaluationAt := time.Date(2026, 8, 4, 10, 10, 0, 0, time.UTC)
	repo := newScheduledScanRepositoryForTest(t)
	targetID := 7
	schedule, err := repo.Create(context.Background(), &scheduledapp.ScheduledScanCreate{
		Name: "rollback", ScanWorkflowID: "default", Configuration: map[string]any{"version": 1}, InputSource: scandomain.InputSourceScanSnapshot,
		TargetID: &targetID, TimeZone: "UTC", CronExpression: "* * * * *", IsEnabled: true,
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	persistedCursor := evaluationAt.Add(-time.Minute)
	if err := repo.db.Model(&scheduledScanModel{}).Where("id = ?", schedule.ID).Update("next_run_time", persistedCursor).Error; err != nil {
		t.Fatalf("set persisted cursor: %v", err)
	}
	if err := repo.db.Exec(`CREATE TRIGGER reject_occurrence_insert BEFORE INSERT ON scheduled_scan_occurrence BEGIN SELECT RAISE(ABORT, 'injected insert failure'); END`).Error; err != nil {
		t.Fatalf("create failure trigger: %v", err)
	}
	committed, err := repo.MaterializeDue(context.Background(), schedule.ID, evaluationAt)
	if err == nil || committed {
		t.Fatalf("MaterializeDue() = %v, %v; want rollback", committed, err)
	}
	stored, readErr := repo.GetByID(context.Background(), schedule.ID)
	if readErr != nil || stored.NextRunTime == nil || !stored.NextRunTime.Equal(persistedCursor) {
		t.Fatalf("failed materialization consumed cursor: %+v, %v", stored, readErr)
	}
	var count int64
	repo.db.Model(&scheduledScanOccurrenceModel{}).Count(&count)
	if count != 0 {
		t.Fatalf("failed materialization committed %d occurrences", count)
	}
}

func TestScheduledScanRepositoryMaterializationRollsBackOccurrenceWhenCursorWriteFails(t *testing.T) {
	evaluationAt := time.Date(2026, 8, 4, 10, 10, 0, 0, time.UTC)
	repo := newScheduledScanRepositoryForTest(t)
	targetID := 7
	schedule, err := repo.Create(context.Background(), &scheduledapp.ScheduledScanCreate{
		Name: "rollback-cursor", ScanWorkflowID: "default", Configuration: map[string]any{"version": 1}, InputSource: scandomain.InputSourceScanSnapshot,
		TargetID: &targetID, TimeZone: "UTC", CronExpression: "* * * * *", IsEnabled: true,
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	persistedCursor := evaluationAt.Add(-time.Minute)
	if err := repo.db.Model(&scheduledScanModel{}).Where("id = ?", schedule.ID).Update("next_run_time", persistedCursor).Error; err != nil {
		t.Fatalf("set persisted cursor: %v", err)
	}
	if err := repo.db.Exec(`CREATE TRIGGER reject_cursor_update BEFORE UPDATE OF next_run_time ON scheduled_scan BEGIN SELECT RAISE(ABORT, 'injected cursor failure'); END`).Error; err != nil {
		t.Fatalf("create failure trigger: %v", err)
	}

	committed, err := repo.MaterializeDue(context.Background(), schedule.ID, evaluationAt)
	if err == nil || committed {
		t.Fatalf("MaterializeDue() = %v, %v; want rollback", committed, err)
	}
	var count int64
	repo.db.Model(&scheduledScanOccurrenceModel{}).Where("scheduled_scan_id = ?", schedule.ID).Count(&count)
	if count != 0 {
		t.Fatalf("failed cursor update committed %d occurrences", count)
	}
	var stored scheduledScanModel
	if err := repo.db.Where("id = ?", schedule.ID).Take(&stored).Error; err != nil || stored.NextRunTime == nil || !stored.NextRunTime.Equal(persistedCursor) {
		t.Fatalf("failed cursor update changed Schedule: %+v, %v", stored, err)
	}
}

func TestScheduledScanRepositoryFairCandidateRotatesByLastTrigger(t *testing.T) {
	now := time.Date(2026, 8, 4, 10, 0, 0, 0, time.UTC)
	repo := newScheduledScanRepositoryForTest(t).WithClock(func() time.Time { return now })
	targetID := 7
	create := func(name string) *scheduledapp.ScheduledScan {
		item, err := repo.Create(context.Background(), &scheduledapp.ScheduledScanCreate{
			Name: name, ScanWorkflowID: "default", Configuration: map[string]any{"version": name}, InputSource: scandomain.InputSourceScanSnapshot,
			TargetID: &targetID, TimeZone: "UTC", CronExpression: "* * * * *", IsEnabled: true,
		})
		if err != nil {
			t.Fatalf("Create(%s): %v", name, err)
		}
		return item
	}
	first, second := create("first"), create("second")
	for index, scheduleID := range []int{first.ID, first.ID, second.ID} {
		if err := repo.db.Create(&scheduledScanOccurrenceModel{ScheduledScanID: scheduleID, ScheduledFor: now.Add(time.Duration(index+1) * time.Minute)}).Error; err != nil {
			t.Fatalf("seed occurrence: %v", err)
		}
	}

	candidate, err := repo.SelectAttemptCandidate(context.Background())
	if err != nil || candidate == nil || candidate.ScheduledScanID != first.ID {
		t.Fatalf("first candidate = %+v, %v", candidate, err)
	}
	if _, err := repo.StartAttempt(context.Background(), *candidate, now); err != nil {
		t.Fatalf("StartAttempt(first): %v", err)
	}
	candidate, err = repo.SelectAttemptCandidate(context.Background())
	if err != nil || candidate == nil || candidate.ScheduledScanID != second.ID {
		t.Fatalf("rotated candidate = %+v, %v", candidate, err)
	}
}

func TestScheduledScanRepositoryRestartNeverReplaysAttemptedOccurrence(t *testing.T) {
	now := time.Date(2026, 8, 4, 10, 0, 0, 0, time.UTC)
	repo := newScheduledScanRepositoryForTest(t).WithClock(func() time.Time { return now })
	targetID := 7
	schedule, err := repo.Create(context.Background(), &scheduledapp.ScheduledScanCreate{
		Name: "attempted", ScanWorkflowID: "default", Configuration: map[string]any{"version": 1}, InputSource: scandomain.InputSourceScanSnapshot,
		TargetID: &targetID, TimeZone: "UTC", CronExpression: "* * * * *", IsEnabled: true,
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	attemptedAt := now.Add(-time.Minute)
	attempted := &scheduledScanOccurrenceModel{
		ScheduledScanID: schedule.ID, ScheduledFor: now.Add(-2 * time.Minute), AttemptedAt: &attemptedAt,
	}
	unattempted := &scheduledScanOccurrenceModel{ScheduledScanID: schedule.ID, ScheduledFor: now.Add(-time.Minute)}
	if err := repo.db.Create(attempted).Error; err != nil {
		t.Fatalf("seed attempted occurrence: %v", err)
	}
	if err := repo.db.Create(unattempted).Error; err != nil {
		t.Fatalf("seed unattempted occurrence: %v", err)
	}

	candidate, err := repo.SelectAttemptCandidate(context.Background())
	if err != nil || candidate == nil || candidate.ID != unattempted.ID {
		t.Fatalf("restart candidate = %+v, %v; want only unattempted row %d", candidate, err, unattempted.ID)
	}
}

func TestScheduledScanRepositoryDueQueryIsStableAndLimitedToOneHundred(t *testing.T) {
	evaluationAt := time.Date(2026, 8, 4, 10, 0, 0, 0, time.UTC)
	repo := newScheduledScanRepositoryForTest(t)
	models := make([]scheduledScanModel, 0, 101)
	for index := 0; index < 101; index++ {
		next := evaluationAt.Add(-time.Duration(index%3) * time.Minute)
		models = append(models, scheduledScanModel{
			Name: fmt.Sprintf("schedule-%03d", index), ScanWorkflowID: "default", Configuration: []byte(`{}`),
			TimeZone: "UTC", CronExpression: "* * * * *", IsEnabled: true, NextRunTime: &next,
		})
	}
	if err := repo.db.CreateInBatches(models, 50).Error; err != nil {
		t.Fatalf("seed due schedules: %v", err)
	}

	due, err := repo.ListDueSchedules(context.Background(), evaluationAt)
	if err != nil || len(due) != scheduledapp.DueScheduleBatchSize {
		t.Fatalf("ListDueSchedules() count = %d error=%v", len(due), err)
	}
	for index := 1; index < len(due); index++ {
		previous, current := due[index-1], due[index]
		if previous.NextRunTime.After(current.NextRunTime) || (previous.NextRunTime.Equal(current.NextRunTime) && previous.ID >= current.ID) {
			t.Fatalf("due ordering is unstable at %d: %+v then %+v", index, previous, current)
		}
	}
	var stillDue int64
	repo.db.Model(&scheduledScanModel{}).Where("next_run_time <= ?", evaluationAt).Count(&stillDue)
	if stillDue != 101 {
		t.Fatalf("due query mutated rows outside materialization: %d", stillDue)
	}
}

func TestScheduledScanRepositoryRetentionUsesAttemptedAtBoundaryAndFixedBatch(t *testing.T) {
	repo := newScheduledScanRepositoryForTest(t)
	cutoff := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)
	rows := make([]scheduledScanOccurrenceModel, 0, 1003)
	for index := 0; index < 1001; index++ {
		attemptedAt := cutoff
		rows = append(rows, scheduledScanOccurrenceModel{ScheduledScanID: index + 1, ScheduledFor: cutoff.Add(-time.Hour), AttemptedAt: &attemptedAt})
	}
	younger := cutoff.Add(time.Nanosecond)
	rows = append(rows, scheduledScanOccurrenceModel{ScheduledScanID: 2001, ScheduledFor: cutoff.Add(-24 * time.Hour), AttemptedAt: &younger})
	rows = append(rows, scheduledScanOccurrenceModel{ScheduledScanID: 2002, ScheduledFor: cutoff.Add(-30 * 24 * time.Hour)})
	if err := repo.db.CreateInBatches(rows, 200).Error; err != nil {
		t.Fatalf("seed retention rows: %v", err)
	}

	deleted, err := repo.DeleteOccurrenceBatch(context.Background(), cutoff)
	if err != nil || deleted != scheduledapp.OccurrenceDeleteBatchSize {
		t.Fatalf("first DeleteOccurrenceBatch() = %d, %v", deleted, err)
	}
	var oldestRemaining scheduledScanOccurrenceModel
	if err := repo.db.Where("attempted_at <= ?", cutoff).Order("attempted_at ASC, id ASC").Take(&oldestRemaining).Error; err != nil {
		t.Fatalf("read stable retention remainder: %v", err)
	}
	if oldestRemaining.ID != 1001 {
		t.Fatalf("stable retention order left occurrence %d, want highest tie-break ID 1001", oldestRemaining.ID)
	}
	deleted, err = repo.DeleteOccurrenceBatch(context.Background(), cutoff)
	if err != nil || deleted != 1 {
		t.Fatalf("second DeleteOccurrenceBatch() = %d, %v", deleted, err)
	}
	var remaining int64
	repo.db.Model(&scheduledScanOccurrenceModel{}).Count(&remaining)
	if remaining != 2 {
		t.Fatalf("retention removed younger or unattempted rows: remaining=%d", remaining)
	}
}

func TestScheduledScanRepositoryRecordsHandoffOutcomeAggregatesExactlyOnce(t *testing.T) {
	repo := newScheduledScanRepositoryForTest(t)
	attemptedAt := time.Date(2026, 8, 4, 10, 0, 0, 0, time.UTC)
	targetID := 7
	schedule, err := repo.Create(context.Background(), &scheduledapp.ScheduledScanCreate{
		Name: "outcomes", ScanWorkflowID: "default", Configuration: map[string]any{"version": 1}, InputSource: scandomain.InputSourceScanSnapshot,
		TargetID: &targetID, TimeZone: "UTC", CronExpression: "* * * * *", IsEnabled: true,
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	tests := []struct {
		name    string
		outcome scheduledapp.HandoffOutcome
		success int
		failure int
	}{
		{name: "completed", outcome: scheduledapp.HandoffOutcome{Kind: scheduledapp.HandoffCompleted}, success: 1},
		{name: "partial", outcome: scheduledapp.HandoffOutcome{Kind: scheduledapp.HandoffPartial, Message: "partial"}, success: 1, failure: 1},
		{name: "deadline", outcome: scheduledapp.HandoffOutcome{Kind: scheduledapp.HandoffDeadlineExceeded, Message: "deadline"}, success: 1, failure: 2},
		{name: "canceled", outcome: scheduledapp.HandoffOutcome{Kind: scheduledapp.HandoffCanceled, Message: "canceled"}, success: 1, failure: 3},
	}
	for index, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			occurrence := &scheduledScanOccurrenceModel{
				ScheduledScanID: schedule.ID,
				ScheduledFor:    attemptedAt.Add(time.Duration(index) * time.Minute),
				AttemptedAt:     &attemptedAt,
			}
			if err := repo.db.Create(occurrence).Error; err != nil {
				t.Fatalf("seed occurrence: %v", err)
			}
			recorded, err := repo.RecordOutcome(context.Background(), occurrence.ID, test.outcome, attemptedAt.Add(time.Second))
			if err != nil || !recorded {
				t.Fatalf("RecordOutcome() = %v, %v", recorded, err)
			}
			stored, err := repo.GetByID(context.Background(), schedule.ID)
			if err != nil || stored.SuccessfulHandoffCount != test.success || stored.FailedHandoffCount != test.failure {
				t.Fatalf("outcome aggregates = %+v, %v; want success=%d failure=%d", stored, err, test.success, test.failure)
			}

			if index == 0 {
				recorded, err = repo.RecordOutcome(context.Background(), occurrence.ID, scheduledapp.HandoffOutcome{Kind: scheduledapp.HandoffPartial, Message: "retry"}, attemptedAt.Add(2*time.Second))
				if err != nil || recorded {
					t.Fatalf("repeated RecordOutcome() = %v, %v; want no-op", recorded, err)
				}
				stored, err = repo.GetByID(context.Background(), schedule.ID)
				if err != nil || stored.SuccessfulHandoffCount != test.success || stored.FailedHandoffCount != test.failure {
					t.Fatalf("repeat changed outcome aggregates = %+v, %v", stored, err)
				}
			}
		})
	}

	interrupted := &scheduledScanOccurrenceModel{ScheduledScanID: schedule.ID, ScheduledFor: attemptedAt.Add(6 * time.Minute)}
	if err := repo.db.Create(interrupted).Error; err != nil {
		t.Fatalf("seed interrupted occurrence: %v", err)
	}
	frozen, err := repo.StartAttempt(context.Background(), scheduledapp.OccurrenceCandidate{
		ID: interrupted.ID, ScheduledScanID: schedule.ID, ScheduledFor: interrupted.ScheduledFor,
	}, attemptedAt.Add(6*time.Minute))
	if err != nil || frozen == nil {
		t.Fatalf("StartAttempt(interrupted) = %+v, %v", frozen, err)
	}
	stored, err := repo.GetByID(context.Background(), schedule.ID)
	if err != nil || stored.RunCount != 1 || stored.SuccessfulHandoffCount != 1 || stored.FailedHandoffCount != 3 {
		t.Fatalf("interrupted attempt aggregates = %+v, %v", stored, err)
	}

	deletedSchedule, err := repo.Create(context.Background(), &scheduledapp.ScheduledScanCreate{
		Name: "deleted-owner", ScanWorkflowID: "default", Configuration: map[string]any{"version": 1}, InputSource: scandomain.InputSourceScanSnapshot,
		TargetID: &targetID, TimeZone: "UTC", CronExpression: "* * * * *", IsEnabled: true,
	})
	if err != nil {
		t.Fatalf("Create(deleted owner) error = %v", err)
	}
	deletedOccurrence := &scheduledScanOccurrenceModel{ScheduledScanID: deletedSchedule.ID, ScheduledFor: attemptedAt, AttemptedAt: &attemptedAt}
	if err := repo.db.Create(deletedOccurrence).Error; err != nil {
		t.Fatalf("seed deleted owner occurrence: %v", err)
	}
	if err := repo.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Delete(&scheduledScanOccurrenceModel{}, deletedOccurrence.ID).Error; err != nil {
			return err
		}
		return tx.Delete(&scheduledScanModel{}, deletedSchedule.ID).Error
	}); err != nil {
		t.Fatalf("delete Schedule-owned occurrence: %v", err)
	}
	if recorded, err := repo.RecordOutcome(context.Background(), deletedOccurrence.ID, scheduledapp.HandoffOutcome{Kind: scheduledapp.HandoffCompleted}, attemptedAt); err != nil || recorded {
		t.Fatalf("RecordOutcome() after owner deletion = %v, %v; want expected no-op", recorded, err)
	}
}

func TestScheduledScanRepositoryRollsBackOutcomeWhenAggregateWriteFails(t *testing.T) {
	repo := newScheduledScanRepositoryForTest(t)
	attemptedAt := time.Date(2026, 8, 4, 10, 0, 0, 0, time.UTC)
	targetID := 7
	schedule, err := repo.Create(context.Background(), &scheduledapp.ScheduledScanCreate{
		Name: "outcome-rollback", ScanWorkflowID: "default", Configuration: map[string]any{"version": 1}, InputSource: scandomain.InputSourceScanSnapshot,
		TargetID: &targetID, TimeZone: "UTC", CronExpression: "* * * * *", IsEnabled: true,
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	occurrence := &scheduledScanOccurrenceModel{ScheduledScanID: schedule.ID, ScheduledFor: attemptedAt, AttemptedAt: &attemptedAt}
	if err := repo.db.Create(occurrence).Error; err != nil {
		t.Fatalf("seed occurrence: %v", err)
	}
	if err := repo.db.Exec(`CREATE TRIGGER reject_handoff_aggregate_write BEFORE UPDATE OF successful_handoff_count ON scheduled_scan BEGIN SELECT RAISE(ABORT, 'injected handoff aggregate failure'); END`).Error; err != nil {
		t.Fatalf("create failure trigger: %v", err)
	}

	recorded, err := repo.RecordOutcome(context.Background(), occurrence.ID, scheduledapp.HandoffOutcome{Kind: scheduledapp.HandoffCompleted}, attemptedAt.Add(time.Second))
	if err == nil || recorded {
		t.Fatalf("RecordOutcome() = %v, %v; want aggregate write failure", recorded, err)
	}
	var storedOccurrence scheduledScanOccurrenceModel
	if err := repo.db.First(&storedOccurrence, occurrence.ID).Error; err != nil {
		t.Fatalf("read occurrence: %v", err)
	}
	if storedOccurrence.DispatchedAt != nil || storedOccurrence.FailureKind != nil || storedOccurrence.FailureMessage != nil {
		t.Fatalf("failed aggregate write settled occurrence: %+v", storedOccurrence)
	}
	stored, err := repo.GetByID(context.Background(), schedule.ID)
	if err != nil || stored.SuccessfulHandoffCount != 0 || stored.FailedHandoffCount != 0 {
		t.Fatalf("failed aggregate write changed Schedule: %+v, %v", stored, err)
	}
}

type failingScheduleCalculator struct{ err error }

func (calculator failingScheduleCalculator) Validate(string, string) error { return calculator.err }
func (calculator failingScheduleCalculator) FirstAfter(string, string, time.Time) (time.Time, error) {
	return time.Time{}, calculator.err
}
func (calculator failingScheduleCalculator) LatestAtOrBefore(string, string, time.Time, time.Time) (time.Time, error) {
	return time.Time{}, calculator.err
}
func (calculator failingScheduleCalculator) AdvanceAfter(string, string, time.Time) (time.Time, error) {
	return time.Time{}, calculator.err
}

type firstAfterFailingScheduleCalculator struct {
	ScheduleCalculator scheduledapp.ScheduleCalculator
	err                error
}

func (calculator firstAfterFailingScheduleCalculator) Validate(expression, timeZone string) error {
	return calculator.ScheduleCalculator.Validate(expression, timeZone)
}

func (calculator firstAfterFailingScheduleCalculator) FirstAfter(string, string, time.Time) (time.Time, error) {
	return time.Time{}, calculator.err
}

func (calculator firstAfterFailingScheduleCalculator) LatestAtOrBefore(expression, timeZone string, cursor, instant time.Time) (time.Time, error) {
	return calculator.ScheduleCalculator.LatestAtOrBefore(expression, timeZone, cursor, instant)
}

func (calculator firstAfterFailingScheduleCalculator) AdvanceAfter(expression, timeZone string, instant time.Time) (time.Time, error) {
	return calculator.ScheduleCalculator.AdvanceAfter(expression, timeZone, instant)
}

func TestScheduledScanRepositoryCreateRollsBackCalculationFailure(t *testing.T) {
	wantErr := errors.New("calculator failed")
	db := openScheduledScanTestDB(t)
	repo := NewScheduledScanRepository(db, failingScheduleCalculator{err: wantErr})
	targetID := 7
	_, err := repo.Create(context.Background(), &scheduledapp.ScheduledScanCreate{
		Name: "invalid", ScanWorkflowID: "default", InputSource: scandomain.InputSourceScanSnapshot, TargetID: &targetID,
		TimeZone: "UTC", CronExpression: "* * * * *", IsEnabled: true,
	})
	if !errors.Is(err, wantErr) {
		t.Fatalf("Create() error = %v, want %v", err, wantErr)
	}
	var count int64
	db.Model(&scheduledScanModel{}).Count(&count)
	if count != 0 {
		t.Fatalf("calculation failure committed %d schedules", count)
	}
}

func TestScheduledScanRepositoryUpdateRollsBackCalculationFailure(t *testing.T) {
	now := time.Date(2026, 8, 4, 10, 0, 30, 0, time.UTC)
	repo := newScheduledScanRepositoryForTest(t).WithClock(func() time.Time { return now })
	targetID := 7
	schedule, err := repo.Create(context.Background(), &scheduledapp.ScheduledScanCreate{
		Name: "calculation", ScanWorkflowID: "default", Configuration: map[string]any{"version": 1}, InputSource: scandomain.InputSourceScanSnapshot,
		TargetID: &targetID, TimeZone: "UTC", CronExpression: "* * * * *", IsEnabled: true,
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	originalCursor := *schedule.NextRunTime
	wantErr := errors.New("first future calculation failed")
	repo.calculator = firstAfterFailingScheduleCalculator{ScheduleCalculator: scheduledapp.NewCronScheduleCalculator(), err: wantErr}
	cron := "15 * * * *"

	if _, err := repo.Update(context.Background(), schedule.ID, &scheduledapp.ScheduledScanUpdate{CronExpression: &cron}); !errors.Is(err, wantErr) {
		t.Fatalf("Update() error = %v, want %v", err, wantErr)
	}
	stored, err := repo.GetByID(context.Background(), schedule.ID)
	if err != nil || stored.CronExpression != "* * * * *" || stored.NextRunTime == nil || !stored.NextRunTime.Equal(originalCursor) {
		t.Fatalf("calculation failure changed Schedule: %+v, %v", stored, err)
	}
}

func TestScheduledScanRepositoryDisableRollsBackOccurrenceDeletionWhenScheduleWriteFails(t *testing.T) {
	now := time.Date(2026, 8, 4, 10, 0, 30, 0, time.UTC)
	repo := newScheduledScanRepositoryForTest(t).WithClock(func() time.Time { return now })
	targetID := 7
	schedule, err := repo.Create(context.Background(), &scheduledapp.ScheduledScanCreate{
		Name: "write-rollback", ScanWorkflowID: "default", Configuration: map[string]any{"version": 1}, InputSource: scandomain.InputSourceScanSnapshot,
		TargetID: &targetID, TimeZone: "UTC", CronExpression: "* * * * *", IsEnabled: true,
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	occurrence := &scheduledScanOccurrenceModel{ScheduledScanID: schedule.ID, ScheduledFor: now}
	if err := repo.db.Create(occurrence).Error; err != nil {
		t.Fatalf("seed occurrence: %v", err)
	}
	if err := repo.db.Exec(`CREATE TRIGGER reject_schedule_write BEFORE UPDATE ON scheduled_scan BEGIN SELECT RAISE(ABORT, 'injected schedule write failure'); END`).Error; err != nil {
		t.Fatalf("create failure trigger: %v", err)
	}
	disabled := false

	if _, err := repo.Update(context.Background(), schedule.ID, &scheduledapp.ScheduledScanUpdate{IsEnabled: &disabled}); err == nil {
		t.Fatal("disable Update unexpectedly succeeded")
	}
	stored, err := repo.GetByID(context.Background(), schedule.ID)
	if err != nil || !stored.IsEnabled || stored.NextRunTime == nil {
		t.Fatalf("failed disable changed Schedule: %+v, %v", stored, err)
	}
	var count int64
	repo.db.Model(&scheduledScanOccurrenceModel{}).Where("id = ?", occurrence.ID).Count(&count)
	if count != 1 {
		t.Fatalf("failed disable committed occurrence deletion: count=%d", count)
	}
}

func TestScheduledScanRepositoryRejectsUnencodableConfigurationWithoutWrites(t *testing.T) {
	repo := newScheduledScanRepositoryForTest(t)
	targetID := 7
	invalid := map[string]any{"unsupported": make(chan int)}
	if _, err := repo.Create(context.Background(), &scheduledapp.ScheduledScanCreate{
		Name: "invalid-create", ScanWorkflowID: "default", Configuration: invalid, InputSource: scandomain.InputSourceScanSnapshot,
		TargetID: &targetID, TimeZone: "UTC", CronExpression: "* * * * *", IsEnabled: true,
	}); err == nil {
		t.Fatal("Create() accepted an unencodable configuration")
	}
	var count int64
	repo.db.Model(&scheduledScanModel{}).Count(&count)
	if count != 0 {
		t.Fatalf("failed Create committed %d schedules", count)
	}

	valid, err := repo.Create(context.Background(), &scheduledapp.ScheduledScanCreate{
		Name: "valid", ScanWorkflowID: "default", Configuration: map[string]any{"version": 1}, InputSource: scandomain.InputSourceScanSnapshot,
		TargetID: &targetID, TimeZone: "UTC", CronExpression: "* * * * *", IsEnabled: true,
	})
	if err != nil {
		t.Fatalf("seed valid Schedule: %v", err)
	}
	if _, err := repo.Update(context.Background(), valid.ID, &scheduledapp.ScheduledScanUpdate{Configuration: invalid}); err == nil {
		t.Fatal("Update() accepted an unencodable configuration")
	}
	stored, err := repo.GetByID(context.Background(), valid.ID)
	if err != nil || stored.Configuration["version"] != float64(1) {
		t.Fatalf("failed Update changed persisted configuration: %+v, %v", stored, err)
	}
}

func TestScheduledScanRepositoryTargetScopedWritesRequireAnActiveTarget(t *testing.T) {
	repo := newScheduledScanRepositoryForTest(t)
	activeTargetID := 7
	tombstonedTargetID := 8
	if err := repo.db.Exec(`INSERT INTO target (id, name, deleted_at) VALUES (?, ?, CURRENT_TIMESTAMP)`, tombstonedTargetID, "target-8.example").Error; err != nil {
		t.Fatalf("seed tombstoned target: %v", err)
	}

	schedule, err := repo.Create(context.Background(), &scheduledapp.ScheduledScanCreate{
		Name: "active-target-schedule", ScanWorkflowID: "default", Configuration: map[string]any{"version": 1}, InputSource: scandomain.InputSourceScanSnapshot,
		TargetID: &activeTargetID, TimeZone: "UTC", CronExpression: "* * * * *", IsEnabled: true,
	})
	if err != nil {
		t.Fatalf("create active Target Schedule: %v", err)
	}
	if _, err := repo.Create(context.Background(), &scheduledapp.ScheduledScanCreate{
		Name: "tombstoned-target-schedule", ScanWorkflowID: "default", Configuration: map[string]any{"version": 1}, InputSource: scandomain.InputSourceScanSnapshot,
		TargetID: &tombstonedTargetID, TimeZone: "UTC", CronExpression: "* * * * *", IsEnabled: true,
	}); !errors.Is(err, scheduledapp.ErrScheduledScanInvalidArgument) {
		t.Fatalf("create for tombstoned Target error = %v, want invalid argument", err)
	}

	if _, err := repo.Update(context.Background(), schedule.ID, &scheduledapp.ScheduledScanUpdate{TargetID: &tombstonedTargetID}); !errors.Is(err, scheduledapp.ErrScheduledScanInvalidArgument) {
		t.Fatalf("rebind to tombstoned Target error = %v, want invalid argument", err)
	}
	stored, err := repo.GetByID(context.Background(), schedule.ID)
	if err != nil || stored == nil || stored.TargetID == nil || *stored.TargetID != activeTargetID {
		t.Fatalf("failed rebind changed existing Schedule: %+v, %v", stored, err)
	}

	if err := repo.db.Exec("UPDATE target SET deleted_at = CURRENT_TIMESTAMP WHERE id = ?", activeTargetID).Error; err != nil {
		t.Fatalf("tombstone active Target: %v", err)
	}
	name := "must-not-update"
	if _, err := repo.Update(context.Background(), schedule.ID, &scheduledapp.ScheduledScanUpdate{Name: &name}); !errors.Is(err, scheduledapp.ErrScheduledScanInvalidArgument) {
		t.Fatalf("update after Target deletion error = %v, want invalid argument", err)
	}
	if _, err := repo.GetByID(context.Background(), schedule.ID); !errors.Is(err, scheduledapp.ErrScheduledScanNotFound) {
		t.Fatalf("Target-scoped Schedule must become immediately invisible, got %v", err)
	}
	items, total, err := repo.List(context.Background(), scheduledapp.ScheduledScanListQuery{Page: 1, PageSize: 20})
	if err != nil || total != 0 || len(items) != 0 {
		t.Fatalf("deleted Target Schedule remained listed: items=%+v total=%d err=%v", items, total, err)
	}
}

func TestScheduledScanRepositoryRuntimeSkipsTombstonedTargetSchedule(t *testing.T) {
	repo := newScheduledScanRepositoryForTest(t)
	evaluationAt := time.Date(2026, 8, 6, 10, 0, 0, 0, time.UTC)
	targetID := 7
	schedule, err := repo.Create(context.Background(), &scheduledapp.ScheduledScanCreate{
		Name: "target-cleanup-pending", ScanWorkflowID: "default", Configuration: map[string]any{"version": 1}, InputSource: scandomain.InputSourceScanSnapshot,
		TargetID: &targetID, TimeZone: "UTC", CronExpression: "* * * * *", IsEnabled: true,
	})
	if err != nil {
		t.Fatalf("seed Target-scoped Schedule: %v", err)
	}
	if err := repo.db.Model(&scheduledScanModel{}).Where("id = ?", schedule.ID).Update("next_run_time", evaluationAt).Error; err != nil {
		t.Fatalf("make Schedule due: %v", err)
	}
	occurrence := &scheduledScanOccurrenceModel{ScheduledScanID: schedule.ID, ScheduledFor: evaluationAt.Add(-time.Minute)}
	if err := repo.db.Create(occurrence).Error; err != nil {
		t.Fatalf("seed unattempted occurrence: %v", err)
	}
	if err := repo.db.Exec("UPDATE target SET deleted_at = CURRENT_TIMESTAMP WHERE id = ?", targetID).Error; err != nil {
		t.Fatalf("tombstone Target: %v", err)
	}

	due, err := repo.ListDueSchedules(context.Background(), evaluationAt)
	if err != nil || len(due) != 0 {
		t.Fatalf("ListDueSchedules() = %+v, %v; tombstoned Target must be hidden", due, err)
	}
	committed, err := repo.MaterializeDue(context.Background(), schedule.ID, evaluationAt)
	if err != nil || committed {
		t.Fatalf("MaterializeDue() = %v, %v; tombstoned Target must not materialize", committed, err)
	}
	candidate, err := repo.SelectAttemptCandidate(context.Background())
	if err != nil || candidate != nil {
		t.Fatalf("SelectAttemptCandidate() = %+v, %v; tombstoned Target must not be selected", candidate, err)
	}
	frozen, err := repo.StartAttempt(context.Background(), scheduledapp.OccurrenceCandidate{
		ID: occurrence.ID, ScheduledScanID: schedule.ID, ScheduledFor: occurrence.ScheduledFor,
	}, evaluationAt)
	if err != nil || frozen != nil {
		t.Fatalf("StartAttempt() = %+v, %v; tombstoned Target must not start", frozen, err)
	}
	var stored scheduledScanOccurrenceModel
	if err := repo.db.Where("id = ?", occurrence.ID).First(&stored).Error; err != nil {
		t.Fatalf("read occurrence: %v", err)
	}
	if stored.AttemptedAt != nil {
		t.Fatalf("tombstoned Target occurrence was marked attempted: %+v", stored)
	}
}

func TestScheduledScanRepositoryOperationsHonorCanceledContext(t *testing.T) {
	repo := newScheduledScanRepositoryForTest(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if _, err := repo.ListDueSchedules(ctx, time.Now().UTC()); !errors.Is(err, context.Canceled) {
		t.Fatalf("ListDueSchedules() error = %v, want context canceled", err)
	}
	if _, err := repo.DeleteOccurrenceBatch(ctx, time.Now().UTC()); !errors.Is(err, context.Canceled) {
		t.Fatalf("DeleteOccurrenceBatch() error = %v, want context canceled", err)
	}
}

func newScheduledScanRepositoryForTest(t *testing.T) *ScheduledScanRepository {
	t.Helper()
	return NewScheduledScanRepository(openScheduledScanTestDB(t))
}

func openScheduledScanTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&scheduledScanModel{}, &scheduledScanOccurrenceModel{}); err != nil {
		t.Fatalf("migrate test schema: %v", err)
	}
	if err := db.Exec(`INSERT OR IGNORE INTO target (id, name, deleted_at) VALUES (7, 'target-7.example', NULL)`).Error; err != nil {
		t.Fatalf("seed active target: %v", err)
	}
	return db
}

func TestScheduledScanRepositoryHandoffRetryLifecycleAndSummary(t *testing.T) {
	base := time.Date(2026, 8, 4, 10, 0, 0, 0, time.UTC)
	now := base
	repo := newScheduledScanRepositoryForTest(t).WithClock(func() time.Time { return now })
	targetID := 7
	schedule, err := repo.Create(context.Background(), &scheduledapp.ScheduledScanCreate{
		Name: "retry", ScanWorkflowID: "default", Configuration: map[string]any{"version": 1}, InputSource: scandomain.InputSourceScanSnapshot,
		TargetID: &targetID, TimeZone: "UTC", CronExpression: "* * * * *", IsEnabled: true,
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	seedOccurrence := func(offset time.Duration) *scheduledScanOccurrenceModel {
		occurrence := &scheduledScanOccurrenceModel{ScheduledScanID: schedule.ID, ScheduledFor: base.Add(offset)}
		if err := repo.db.Create(occurrence).Error; err != nil {
			t.Fatalf("seed occurrence: %v", err)
		}
		return occurrence
	}
	loadOccurrence := func(id int64) *scheduledScanOccurrenceModel {
		var occurrence scheduledScanOccurrenceModel
		if err := repo.db.First(&occurrence, "id = ?", id).Error; err != nil {
			t.Fatalf("load occurrence %d: %v", id, err)
		}
		return &occurrence
	}
	loadSchedule := func() *scheduledapp.ScheduledScan {
		stored, err := repo.GetByID(context.Background(), schedule.ID)
		if err != nil {
			t.Fatalf("GetByID() error = %v", err)
		}
		return stored
	}
	failedOutcome := scheduledapp.HandoffOutcome{
		Kind: scheduledapp.HandoffScanCreateFailed, Message: "scan create failed", Cause: scheduledapp.HandoffCauseWorkflowUnavailable,
	}
	recordFailure := func(occurrenceID int64, at time.Time) {
		t.Helper()
		recorded, err := repo.RecordOutcome(context.Background(), occurrenceID, failedOutcome, at)
		if err != nil || !recorded {
			t.Fatalf("RecordOutcome(failed) at %s = %v, %v", at, recorded, err)
		}
	}

	// A fresh schedule and occurrence carry no failure summary and no retry
	// state; both summaries start null and retry_count starts at zero.
	first := seedOccurrence(time.Minute)
	if fresh := loadOccurrence(first.ID); fresh.RetryCount != 0 || fresh.NextRetryAt != nil || fresh.LastFailureCause != nil {
		t.Fatalf("fresh occurrence retry state = %+v; want zero/null defaults", fresh)
	}
	if summary := loadSchedule(); summary.LastHandoffFailureCause != nil || summary.LastHandoffFailureTime != nil {
		t.Fatalf("fresh schedule summary = %+v, %+v; want null", summary.LastHandoffFailureCause, summary.LastHandoffFailureTime)
	}

	// Attempt 1: a zero-created failure records retry state without settling.
	if _, err := repo.StartAttempt(context.Background(), scheduledapp.OccurrenceCandidate{ID: first.ID, ScheduledScanID: schedule.ID, ScheduledFor: first.ScheduledFor}, base.Add(time.Second)); err != nil {
		t.Fatalf("StartAttempt(first) error = %v", err)
	}
	recordFailure(first.ID, base.Add(2*time.Second))
	stored := loadOccurrence(first.ID)
	if stored.RetryCount != 1 || stored.NextRetryAt == nil || !stored.NextRetryAt.Equal(base.Add(32*time.Second)) {
		t.Fatalf("first failure retry state = %+v", stored)
	}
	if stored.FailureKind != nil || stored.LastFailureCause == nil || *stored.LastFailureCause != "WORKFLOW_UNAVAILABLE" {
		t.Fatalf("first failure must not settle terminal diagnostics: %+v", stored)
	}
	summary := loadSchedule()
	if summary.RunCount != 1 || summary.SuccessfulHandoffCount != 0 || summary.FailedHandoffCount != 0 ||
		summary.LastHandoffFailureCause != nil || summary.LastHandoffFailureTime != nil {
		t.Fatalf("intermediate failure touched aggregates or summary: %+v", summary)
	}

	deadline, err := repo.EarliestRetryDeadline(context.Background())
	if err != nil || deadline == nil || !deadline.Equal(base.Add(32*time.Second)) {
		t.Fatalf("EarliestRetryDeadline() = %v, %v", deadline, err)
	}

	// Before the deadline the occurrence is not an eligible candidate.
	now = base.Add(10 * time.Second)
	candidate, err := repo.SelectAttemptCandidate(context.Background())
	if err != nil || candidate != nil {
		t.Fatalf("early SelectAttemptCandidate() = %+v, %v; want nil", candidate, err)
	}

	// After the deadline the retry competes as an ordinary candidate and
	// re-freezes the latest committed inputs (an edit landed meanwhile).
	now = base.Add(40 * time.Second)
	if _, err := repo.Update(context.Background(), schedule.ID, &scheduledapp.ScheduledScanUpdate{Configuration: map[string]any{"version": 2}}); err != nil {
		t.Fatalf("Update() error = %v", err)
	}
	candidate, err = repo.SelectAttemptCandidate(context.Background())
	if err != nil || candidate == nil || candidate.ID != first.ID {
		t.Fatalf("retry-due SelectAttemptCandidate() = %+v, %v", candidate, err)
	}
	frozen, err := repo.StartAttempt(context.Background(), *candidate, now)
	if err != nil || frozen == nil || frozen.Configuration["version"] != float64(2) {
		t.Fatalf("retry StartAttempt() = %+v, %v; want re-frozen latest configuration", frozen, err)
	}

	// Attempts 2 and 3 keep retrying on the 1-minute and 3-minute backoffs.
	recordFailure(first.ID, now)
	stored = loadOccurrence(first.ID)
	if stored.RetryCount != 2 || stored.NextRetryAt == nil || !stored.NextRetryAt.Equal(now.Add(time.Minute)) {
		t.Fatalf("second failure retry state = %+v", stored)
	}
	now = base.Add(160 * time.Second)
	if _, err := repo.StartAttempt(context.Background(), scheduledapp.OccurrenceCandidate{ID: first.ID, ScheduledScanID: schedule.ID, ScheduledFor: first.ScheduledFor}, now); err != nil {
		t.Fatalf("StartAttempt(third) error = %v", err)
	}
	recordFailure(first.ID, now)
	stored = loadOccurrence(first.ID)
	if stored.RetryCount != 3 || stored.NextRetryAt == nil || !stored.NextRetryAt.Equal(now.Add(3*time.Minute)) {
		t.Fatalf("third failure retry state = %+v", stored)
	}
	if summary := loadSchedule(); summary.FailedHandoffCount != 0 {
		t.Fatalf("aggregates moved before exhaustion: %+v", summary)
	}

	// Attempt 4 exhausts the budget and settles terminally with the summary.
	now = base.Add(400 * time.Second)
	if _, err := repo.StartAttempt(context.Background(), scheduledapp.OccurrenceCandidate{ID: first.ID, ScheduledScanID: schedule.ID, ScheduledFor: first.ScheduledFor}, now); err != nil {
		t.Fatalf("StartAttempt(final) error = %v", err)
	}
	recordFailure(first.ID, now)
	stored = loadOccurrence(first.ID)
	if stored.FailureKind == nil || stored.NextRetryAt != nil {
		t.Fatalf("exhausted failure must settle: %+v", stored)
	}
	summary = loadSchedule()
	if summary.RunCount != 4 || summary.SuccessfulHandoffCount != 0 || summary.FailedHandoffCount != 1 ||
		summary.LastHandoffFailureCause == nil || *summary.LastHandoffFailureCause != "WORKFLOW_UNAVAILABLE" ||
		summary.LastHandoffFailureTime == nil || !summary.LastHandoffFailureTime.Equal(now) {
		t.Fatalf("final settlement aggregates/summary = %+v", summary)
	}

	// A later occurrence that succeeds clears the failure summary.
	second := seedOccurrence(10 * time.Minute)
	now = base.Add(500 * time.Second)
	if _, err := repo.StartAttempt(context.Background(), scheduledapp.OccurrenceCandidate{ID: second.ID, ScheduledScanID: schedule.ID, ScheduledFor: second.ScheduledFor}, now); err != nil {
		t.Fatalf("StartAttempt(success) error = %v", err)
	}
	recorded, err := repo.RecordOutcome(context.Background(), second.ID, scheduledapp.HandoffOutcome{Kind: scheduledapp.HandoffCompleted}, now.Add(time.Second))
	if err != nil || !recorded {
		t.Fatalf("RecordOutcome(completed) = %v, %v", recorded, err)
	}
	summary = loadSchedule()
	if summary.RunCount != 5 || summary.SuccessfulHandoffCount != 1 || summary.FailedHandoffCount != 1 ||
		summary.LastHandoffFailureCause != nil || summary.LastHandoffFailureTime != nil {
		t.Fatalf("success settlement must clear the summary: %+v", summary)
	}

	// An attempt interrupted before outcome writeback never retries: its
	// creation state is unknown, so it must stay outside the candidate pool.
	interrupted := seedOccurrence(20 * time.Minute)
	now = base.Add(600 * time.Second)
	if _, err := repo.StartAttempt(context.Background(), scheduledapp.OccurrenceCandidate{ID: interrupted.ID, ScheduledScanID: schedule.ID, ScheduledFor: interrupted.ScheduledFor}, now); err != nil {
		t.Fatalf("StartAttempt(interrupted) error = %v", err)
	}
	now = base.Add(700 * time.Second)
	candidate, err = repo.SelectAttemptCandidate(context.Background())
	if err != nil || candidate != nil {
		t.Fatalf("interrupted occurrence must never replay: %+v, %v", candidate, err)
	}
	if deadline, err = repo.EarliestRetryDeadline(context.Background()); err != nil || deadline != nil {
		t.Fatalf("EarliestRetryDeadline() after interruption = %v, %v", deadline, err)
	}
}

func TestAttemptCandidatePostgresQueryKeepsEligibilityOnTheBaseTable(t *testing.T) {
	query, argCount := attemptCandidateQuery("postgres")
	if argCount != 1 || strings.Count(query, "?") != 1 {
		t.Fatalf("postgres candidate query args = %d, placeholders = %d; want 1", argCount, strings.Count(query, "?"))
	}
	lateralEnd := strings.Index(query, ") AS occurrence")
	if lateralEnd < 0 {
		t.Fatal("postgres candidate query must alias the LATERAL subquery as occurrence")
	}
	inner, outer := query[:lateralEnd], query[lateralEnd:]
	for _, column := range []string{"attempted_at", "next_retry_at", "dispatched_at", "failure_kind"} {
		if !strings.Contains(inner, "oldest."+column) {
			t.Errorf("LATERAL filter missing oldest.%s", column)
		}
		if strings.Contains(outer, "occurrence."+column) {
			t.Errorf("outer query references occurrence.%s, which the LATERAL alias does not project", column)
		}
	}

	sqliteQuery, sqliteArgs := attemptCandidateQuery("sqlite")
	if sqliteArgs != 2 || strings.Count(sqliteQuery, "?") != 2 {
		t.Fatalf("sqlite candidate query args = %d, placeholders = %d; want 2", sqliteArgs, strings.Count(sqliteQuery, "?"))
	}
	if !strings.Contains(sqliteQuery, "JOIN scheduled_scan_occurrence AS occurrence") || !strings.Contains(sqliteQuery, "occurrence.attempted_at") {
		t.Fatal("sqlite candidate query must filter the joined occurrence table")
	}
}
