package repository

import (
	"context"
	"testing"
	"time"

	scheduledapp "github.com/yyhuni/lunafox/server/internal/modules/scheduledscan/application"
)

func horizonTestQuery(now time.Time) scheduledapp.ScheduledScanOverviewQuery {
	return scheduledapp.ScheduledScanOverviewQuery{
		TodayStart: now.Add(-10 * time.Hour), TodayEnd: now.Add(14 * time.Hour),
		Next24HoursStart: now, Next24HoursEnd: now.Add(24 * time.Hour), UpcomingItemsLimit: 5,
		HorizonStart: now.Add(-scheduledapp.ScheduledScanHorizonLookback),
		HorizonEnd:   now.Add(scheduledapp.ScheduledScanHorizonDuration),
	}
}

func TestHorizonExpandsHourlyScheduleAcrossWindow(t *testing.T) {
	repo := newScheduledScanRepositoryForTest(t)
	now := time.Date(2026, 6, 15, 10, 0, 0, 0, time.UTC)
	seedScheduledScanOverviewModel(t, repo, 1, true, timePtr(now), 7)

	overview, err := repo.GetOverviewSummary(context.Background(), horizonTestQuery(now))
	if err != nil {
		t.Fatalf("GetOverviewSummary() error = %v", err)
	}
	buckets := overview.HorizonBuckets
	// Window is [04:00, next-day 10:00); `0 * * * *` first matches 05:00 and
	// last matches next-day 09:00, so 29 non-empty hour buckets exist.
	if len(buckets) != 29 {
		t.Fatalf("horizon buckets = %d, want 29 (first %v last %v)", len(buckets), buckets[0].HourStart, buckets[len(buckets)-1].HourStart)
	}
	if !buckets[0].HourStart.Equal(time.Date(2026, 6, 15, 5, 0, 0, 0, time.UTC)) {
		t.Fatalf("first bucket = %v, want 2026-06-15T05:00Z", buckets[0].HourStart)
	}
	if !buckets[len(buckets)-1].HourStart.Equal(time.Date(2026, 6, 16, 9, 0, 0, 0, time.UTC)) {
		t.Fatalf("last bucket = %v, want 2026-06-16T09:00Z", buckets[len(buckets)-1].HourStart)
	}
	for index, bucket := range buckets {
		if len(bucket.Items) != 1 || bucket.Items[0].ID != 1 {
			t.Fatalf("bucket %d items = %+v, want exactly schedule 1", index, bucket.Items)
		}
		if index > 0 && !bucket.HourStart.After(buckets[index-1].HourStart) {
			t.Fatalf("buckets not ordered ascending at %d", index)
		}
	}
}

func TestHorizonDeduplicatesSameSchedulePerHourAndClustersSchedules(t *testing.T) {
	repo := newScheduledScanRepositoryForTest(t)
	now := time.Date(2026, 6, 15, 10, 0, 0, 0, time.UTC)
	seedScheduledScanOverviewModel(t, repo, 1, true, timePtr(now), 7)
	if err := repo.db.Exec(`UPDATE scheduled_scan SET cron_expression = '0,30 * * * *' WHERE id = 1`).Error; err != nil {
		t.Fatalf("tighten cron: %v", err)
	}
	seedScheduledScanOverviewModel(t, repo, 2, true, timePtr(now), 7)

	overview, err := repo.GetOverviewSummary(context.Background(), horizonTestQuery(now))
	if err != nil {
		t.Fatalf("GetOverviewSummary() error = %v", err)
	}
	// `0,30 * * * *` first matches 04:30, so it also fills the 04:00 bucket
	// that `0 * * * *` misses; together they cover all 30 window buckets.
	if len(overview.HorizonBuckets) != 30 {
		t.Fatalf("horizon buckets = %d, want 30", len(overview.HorizonBuckets))
	}
	for _, bucket := range overview.HorizonBuckets {
		perSchedule := map[int]int{}
		for _, item := range bucket.Items {
			perSchedule[item.ID]++
		}
		// The 04:00 bucket only holds the half-hour schedule (its 04:30 run);
		// the hourly schedule starts at 05:00.
		wantScheduleTwo := 1
		if bucket.HourStart.Equal(time.Date(2026, 6, 15, 4, 0, 0, 0, time.UTC)) {
			wantScheduleTwo = 0
		}
		if perSchedule[1] != 1 || perSchedule[2] != wantScheduleTwo {
			t.Fatalf("bucket %v per-schedule counts = %v, want schedule1=1 schedule2=%d", bucket.HourStart, perSchedule, wantScheduleTwo)
		}
	}
}

func TestHorizonExcludesDisabledAndDeletedTargetSchedules(t *testing.T) {
	repo := newScheduledScanRepositoryForTest(t)
	now := time.Date(2026, 6, 15, 10, 0, 0, 0, time.UTC)
	seedScheduledScanOverviewModel(t, repo, 1, false, timePtr(now), 7)
	if err := repo.db.Exec(`INSERT INTO target (id, name, deleted_at) VALUES (8, 'deleted-target.example', ?)`, now).Error; err != nil {
		t.Fatalf("seed deleted target: %v", err)
	}
	seedScheduledScanOverviewModel(t, repo, 2, true, timePtr(now), 8)

	overview, err := repo.GetOverviewSummary(context.Background(), horizonTestQuery(now))
	if err != nil {
		t.Fatalf("GetOverviewSummary() error = %v", err)
	}
	if overview.HorizonBuckets == nil || len(overview.HorizonBuckets) != 0 {
		t.Fatalf("horizon must be empty but non-nil: %+v", overview.HorizonBuckets)
	}
}

func TestHorizonMinuteCronStaysBoundedAndFillsEveryHour(t *testing.T) {
	repo := newScheduledScanRepositoryForTest(t)
	now := time.Date(2026, 6, 15, 10, 0, 0, 0, time.UTC)
	seedScheduledScanOverviewModel(t, repo, 1, true, timePtr(now), 7)
	if err := repo.db.Exec(`UPDATE scheduled_scan SET cron_expression = '* * * * *' WHERE id = 1`).Error; err != nil {
		t.Fatalf("tighten cron: %v", err)
	}

	overview, err := repo.GetOverviewSummary(context.Background(), horizonTestQuery(now))
	if err != nil {
		t.Fatalf("GetOverviewSummary() error = %v", err)
	}
	if len(overview.HorizonBuckets) != 30 {
		t.Fatalf("minute cron must cover all 30 hour buckets once, got %d", len(overview.HorizonBuckets))
	}
	for _, bucket := range overview.HorizonBuckets {
		if len(bucket.Items) != 1 {
			t.Fatalf("bucket %v items = %d, want 1 (per-hour dedupe)", bucket.HourStart, len(bucket.Items))
		}
	}
}

func TestHorizonOmittedWhenWindowMissing(t *testing.T) {
	repo := newScheduledScanRepositoryForTest(t)
	now := time.Date(2026, 6, 15, 10, 0, 0, 0, time.UTC)
	seedScheduledScanOverviewModel(t, repo, 1, true, timePtr(now), 7)

	overview, err := repo.GetOverviewSummary(context.Background(), scheduledapp.ScheduledScanOverviewQuery{
		TodayStart: now.Add(-time.Hour), TodayEnd: now.Add(24 * time.Hour),
		Next24HoursStart: now, Next24HoursEnd: now.Add(24 * time.Hour), UpcomingItemsLimit: 5,
	})
	if err != nil {
		t.Fatalf("GetOverviewSummary() error = %v", err)
	}
	if overview.HorizonBuckets != nil {
		t.Fatalf("horizon must stay nil when the window is omitted: %+v", overview.HorizonBuckets)
	}
}

func TestHorizonBucketsStayAlignedToWholeHoursForNonIntegralAsOfTime(t *testing.T) {
	repo := newScheduledScanRepositoryForTest(t)
	// Non-integral evaluation instant: the window start carries a minute and
	// second remainder while every bucket hourStart must stay hour-aligned.
	now := time.Date(2026, 6, 15, 10, 23, 45, 0, time.UTC)
	seedScheduledScanOverviewModel(t, repo, 1, true, timePtr(now), 7)

	overview, err := repo.GetOverviewSummary(context.Background(), horizonTestQuery(now))
	if err != nil {
		t.Fatalf("GetOverviewSummary() error = %v", err)
	}
	if len(overview.HorizonBuckets) == 0 {
		t.Fatalf("horizon buckets must not be empty for an hourly schedule")
	}
	for _, bucket := range overview.HorizonBuckets {
		if bucket.HourStart.Minute() != 0 || bucket.HourStart.Second() != 0 || bucket.HourStart.Nanosecond() != 0 {
			t.Fatalf("bucket %v is not aligned to a whole UTC hour", bucket.HourStart)
		}
	}
}
