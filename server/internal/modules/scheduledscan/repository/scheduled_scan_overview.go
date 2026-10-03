package repository

import (
	"context"
	"sort"
	"time"

	scheduledapp "github.com/yyhuni/lunafox/server/internal/modules/scheduledscan/application"
	"github.com/yyhuni/lunafox/server/internal/pkg/timeutil"
	"gorm.io/gorm"
)

func (repo *ScheduledScanRepository) GetOverviewSummary(
	ctx context.Context,
	query scheduledapp.ScheduledScanOverviewQuery,
) (*scheduledapp.ScheduledScanOverviewProjection, error) {
	if repo == nil || repo.db == nil || query.UpcomingItemsLimit <= 0 {
		return nil, scheduledapp.ErrScheduledScanInvalidArgument
	}

	projection := &scheduledapp.ScheduledScanOverviewProjection{
		UpcomingScheduledScans: make([]scheduledapp.ScheduledScanOverviewUpcoming, 0),
	}
	err := repo.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if tx.Dialector.Name() == "postgres" {
			if err := tx.Exec("SET TRANSACTION ISOLATION LEVEL REPEATABLE READ READ ONLY").Error; err != nil {
				return err
			}
		}

		var counts struct {
			EnabledScheduledScanCount     int64 `gorm:"column:enabled_scheduled_scan_count"`
			PausedScheduledScanCount      int64 `gorm:"column:paused_scheduled_scan_count"`
			TodayScheduledScanCount       int64 `gorm:"column:today_scheduled_scan_count"`
			Next24HoursScheduledScanCount int64 `gorm:"column:next_24_hours_scheduled_scan_count"`
		}
		if err := activeScheduledScanQuery(tx.Model(&scheduledScanModel{})).
			Select(`
				COALESCE(SUM(CASE WHEN scheduled_scan.is_enabled THEN 1 ELSE 0 END), 0) AS enabled_scheduled_scan_count,
				COALESCE(SUM(CASE WHEN NOT scheduled_scan.is_enabled THEN 1 ELSE 0 END), 0) AS paused_scheduled_scan_count,
				COALESCE(SUM(CASE WHEN scheduled_scan.is_enabled AND scheduled_scan.next_run_time >= ? AND scheduled_scan.next_run_time < ? THEN 1 ELSE 0 END), 0) AS today_scheduled_scan_count,
				COALESCE(SUM(CASE WHEN scheduled_scan.is_enabled AND scheduled_scan.next_run_time >= ? AND scheduled_scan.next_run_time < ? THEN 1 ELSE 0 END), 0) AS next_24_hours_scheduled_scan_count`,
				query.TodayStart, query.TodayEnd, query.Next24HoursStart, query.Next24HoursEnd).
			Scan(&counts).Error; err != nil {
			return err
		}

		var upcoming []scheduledScanModel
		if err := activeScheduledScanQuery(tx.Model(&scheduledScanModel{})).
			Select("scheduled_scan.id, scheduled_scan.name, scheduled_scan.organization_id, scheduled_scan.target_id, scheduled_scan.next_run_time").
			Preload("Organization").
			Preload("Target").
			Where("scheduled_scan.is_enabled = ? AND scheduled_scan.next_run_time IS NOT NULL", true).
			Order("scheduled_scan.next_run_time ASC").
			Order("scheduled_scan.id ASC").
			Limit(query.UpcomingItemsLimit).
			Find(&upcoming).Error; err != nil {
			return err
		}

		// The horizon axis is a Cron expansion over every enabled Schedule: it
		// deliberately does not reuse the upcoming LIMIT and shares one
		// consistent snapshot with the counts above.
		var horizonRows []scheduledScanModel
		if !query.HorizonStart.IsZero() && !query.HorizonEnd.IsZero() && query.HorizonEnd.After(query.HorizonStart) {
			if err := activeScheduledScanQuery(tx.Model(&scheduledScanModel{})).
				Select("scheduled_scan.id, scheduled_scan.name, scheduled_scan.organization_id, scheduled_scan.target_id, scheduled_scan.cron_expression, scheduled_scan.time_zone").
				Preload("Organization").
				Preload("Target").
				Where("scheduled_scan.is_enabled = ?", true).
				Order("scheduled_scan.id ASC").
				Find(&horizonRows).Error; err != nil {
				return err
			}
			projection.HorizonBuckets = expandHorizonBuckets(repo.calculator, horizonRows, query.HorizonStart, query.HorizonEnd)
		}

		projection.EnabledScheduledScanCount = counts.EnabledScheduledScanCount
		projection.PausedScheduledScanCount = counts.PausedScheduledScanCount
		projection.TodayScheduledScanCount = counts.TodayScheduledScanCount
		projection.Next24HoursScheduledScanCount = counts.Next24HoursScheduledScanCount
		projection.UpcomingScheduledScans = make([]scheduledapp.ScheduledScanOverviewUpcoming, 0, len(upcoming))
		for index := range upcoming {
			item := upcoming[index]
			if item.NextRunTime == nil {
				continue
			}
			projection.UpcomingScheduledScans = append(projection.UpcomingScheduledScans, scheduledapp.ScheduledScanOverviewUpcoming{
				ID:                      item.ID,
				DisplayName:             item.Name,
				OrganizationID:          cloneIntPtr(item.OrganizationID),
				OrganizationDisplayName: scheduledScanOrganizationDisplayName(item.Organization),
				TargetID:                cloneIntPtr(item.TargetID),
				TargetDisplayName:       scheduledScanTargetDisplayName(item.Target),
				NextRunTime:             timeutil.ToUTC(*item.NextRunTime),
			})
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return projection, nil
}

func scheduledScanOrganizationDisplayName(item *organizationRefModel) *string {
	if item == nil {
		return nil
	}
	name := item.Name
	return &name
}

func scheduledScanTargetDisplayName(item *targetRefModel) *string {
	if item == nil {
		return nil
	}
	name := item.Name
	return &name
}

type horizonBucketKey struct {
	scheduleID int
	hourStart  time.Time
}

// expandHorizonBuckets projects every enabled Schedule's Cron rule onto UTC
// hour buckets inside [start, end). One Schedule contributes at most one item
// per bucket. The per-Schedule expansion cap is a fuse: exceeding it stops
// that Schedule silently instead of failing the overview (AD-04).
func expandHorizonBuckets(
	calculator scheduledapp.ScheduleCalculator,
	rows []scheduledScanModel,
	start, end time.Time,
) []scheduledapp.ScheduledScanHorizonBucket {
	if calculator == nil || len(rows) == 0 {
		return []scheduledapp.ScheduledScanHorizonBucket{}
	}
	start = start.UTC()
	end = end.UTC()
	buckets := make(map[time.Time][]scheduledapp.ScheduledScanHorizonItem)
	seen := make(map[horizonBucketKey]struct{})
	for _, row := range rows {
		cursor := start
		for range scheduledapp.MaxHorizonExpansionsPerSchedule {
			next, err := calculator.FirstAfter(row.CronExpression, row.TimeZone, cursor)
			if err != nil || !next.Before(end) {
				break
			}
			hourStart := next.UTC().Truncate(time.Hour)
			key := horizonBucketKey{scheduleID: row.ID, hourStart: hourStart}
			if _, duplicate := seen[key]; !duplicate {
				seen[key] = struct{}{}
				buckets[hourStart] = append(buckets[hourStart], scheduledapp.ScheduledScanHorizonItem{
					ID:                      row.ID,
					DisplayName:             row.Name,
					OrganizationID:          cloneIntPtr(row.OrganizationID),
					OrganizationDisplayName: scheduledScanOrganizationDisplayName(row.Organization),
					TargetID:                cloneIntPtr(row.TargetID),
					TargetDisplayName:       scheduledScanTargetDisplayName(row.Target),
				})
			}
			cursor = next
		}
	}
	hours := make([]time.Time, 0, len(buckets))
	for hourStart := range buckets {
		hours = append(hours, hourStart)
	}
	sort.Slice(hours, func(left, right int) bool { return hours[left].Before(hours[right]) })
	out := make([]scheduledapp.ScheduledScanHorizonBucket, 0, len(hours))
	for _, hourStart := range hours {
		out = append(out, scheduledapp.ScheduledScanHorizonBucket{
			HourStart: hourStart,
			Items:     buckets[hourStart],
		})
	}
	return out
}
