package repository

import (
	"context"
	"testing"
	"time"

	scheduledapp "github.com/yyhuni/lunafox/server/internal/modules/scheduledscan/application"
)

func TestScheduledScanOverviewUsesCompleteVisibleSetAndHalfOpenWindows(t *testing.T) {
	repo := newScheduledScanRepositoryForTest(t)
	asOfTime := time.Date(2026, 6, 15, 10, 0, 0, 0, time.UTC)
	todayStart := time.Date(2026, 6, 15, 0, 0, 0, 0, time.UTC)
	todayEnd := todayStart.AddDate(0, 0, 1)
	query := scheduledapp.ScheduledScanOverviewQuery{
		TodayStart: todayStart, TodayEnd: todayEnd,
		Next24HoursStart: asOfTime, Next24HoursEnd: asOfTime.Add(24 * time.Hour), UpcomingItemsLimit: 5,
	}

	seedScheduledScanOverviewModel(t, repo, 1, true, timePtr(asOfTime), 7)
	seedScheduledScanOverviewModel(t, repo, 2, true, timePtr(todayEnd.Add(-time.Nanosecond)), 7)
	seedScheduledScanOverviewModel(t, repo, 3, true, timePtr(asOfTime.Add(23*time.Hour)), 7)
	seedScheduledScanOverviewModel(t, repo, 4, true, timePtr(asOfTime.Add(24*time.Hour)), 7)
	seedScheduledScanOverviewModel(t, repo, 5, false, timePtr(asOfTime.Add(time.Hour)), 7)
	seedScheduledScanOverviewModel(t, repo, 6, true, nil, 7)
	if err := repo.db.Exec(`INSERT INTO target (id, name, deleted_at) VALUES (8, 'deleted-target.example', ?)`, asOfTime).Error; err != nil {
		t.Fatalf("seed deleted target: %v", err)
	}
	seedScheduledScanOverviewModel(t, repo, 7, true, timePtr(asOfTime), 8)

	overview, err := repo.GetOverviewSummary(context.Background(), query)
	if err != nil {
		t.Fatalf("GetOverviewSummary() error = %v", err)
	}
	if overview.EnabledScheduledScanCount != 5 || overview.PausedScheduledScanCount != 1 {
		t.Fatalf("enabled/paused counts = %d/%d, want 5/1", overview.EnabledScheduledScanCount, overview.PausedScheduledScanCount)
	}
	if overview.TodayScheduledScanCount != 2 || overview.Next24HoursScheduledScanCount != 3 {
		t.Fatalf("window counts = %d/%d, want 2/3", overview.TodayScheduledScanCount, overview.Next24HoursScheduledScanCount)
	}
	for _, item := range overview.UpcomingScheduledScans {
		if item.ID == 7 {
			t.Fatalf("deleted target schedule leaked into overview: %+v", overview)
		}
	}
}

func TestScheduledScanOverviewOrdersAndTruncatesUpcomingItems(t *testing.T) {
	repo := newScheduledScanRepositoryForTest(t)
	now := time.Date(2026, 6, 15, 10, 0, 0, 0, time.UTC)
	for _, item := range []struct {
		id int
		at time.Time
	}{
		{3, now}, {1, now}, {2, now.Add(time.Minute)}, {4, now.Add(2 * time.Minute)},
		{5, now.Add(3 * time.Minute)}, {6, now.Add(4 * time.Minute)},
	} {
		seedScheduledScanOverviewModel(t, repo, item.id, true, timePtr(item.at), 7)
	}
	overview, err := repo.GetOverviewSummary(context.Background(), scheduledapp.ScheduledScanOverviewQuery{
		TodayStart: now.Add(-time.Hour), TodayEnd: now.Add(24 * time.Hour),
		Next24HoursStart: now, Next24HoursEnd: now.Add(24 * time.Hour), UpcomingItemsLimit: 5,
	})
	if err != nil {
		t.Fatalf("GetOverviewSummary() error = %v", err)
	}
	if len(overview.UpcomingScheduledScans) != 5 {
		t.Fatalf("upcoming length = %d, want 5", len(overview.UpcomingScheduledScans))
	}
	for index, want := range []int{1, 3, 2, 4, 5} {
		if overview.UpcomingScheduledScans[index].ID != want {
			t.Fatalf("upcoming[%d] id = %d, want %d: %+v", index, overview.UpcomingScheduledScans[index].ID, want, overview.UpcomingScheduledScans)
		}
	}
}

func TestScheduledScanOverviewReturnsZeroCountsAndEmptyUpcomingForEmptyVisibleSet(t *testing.T) {
	repo := newScheduledScanRepositoryForTest(t)
	now := time.Date(2026, 6, 15, 10, 0, 0, 0, time.UTC)
	overview, err := repo.GetOverviewSummary(context.Background(), scheduledapp.ScheduledScanOverviewQuery{
		TodayStart: now.Add(-10 * time.Hour), TodayEnd: now.Add(14 * time.Hour),
		Next24HoursStart: now, Next24HoursEnd: now.Add(24 * time.Hour), UpcomingItemsLimit: 5,
	})
	if err != nil {
		t.Fatalf("GetOverviewSummary() error = %v", err)
	}
	if overview.EnabledScheduledScanCount != 0 || overview.PausedScheduledScanCount != 0 || overview.TodayScheduledScanCount != 0 || overview.Next24HoursScheduledScanCount != 0 {
		t.Fatalf("empty overview has non-zero counts: %+v", overview)
	}
	if overview.UpcomingScheduledScans == nil || len(overview.UpcomingScheduledScans) != 0 {
		t.Fatalf("empty overview must return a non-nil empty upcoming list: %+v", overview)
	}
}

func seedScheduledScanOverviewModel(t *testing.T, repo *ScheduledScanRepository, id int, enabled bool, nextRunTime *time.Time, targetID int) {
	t.Helper()
	if err := repo.db.Create(&scheduledScanModel{
		ID: id, Name: "schedule", ScanWorkflowID: "default", Configuration: []byte(`{}`), TargetID: &targetID,
		CronExpression: "0 * * * *", IsEnabled: enabled, NextRunTime: nextRunTime,
	}).Error; err != nil {
		t.Fatalf("seed schedule %d: %v", id, err)
	}
}

func timePtr(value time.Time) *time.Time { return &value }
