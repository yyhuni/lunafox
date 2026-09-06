package repository

import (
	"context"

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
