package application

import (
	"context"
	"fmt"
	"time"
)

const scheduledScanOverviewUpcomingLimit = 5

// ScheduledScanOverviewInput is intentionally empty. Overview windows are
// derived server-side from one UTC clock reading.
type ScheduledScanOverviewInput struct{}

type ScheduledScanOverviewQuery struct {
	TodayStart         time.Time
	TodayEnd           time.Time
	Next24HoursStart   time.Time
	Next24HoursEnd     time.Time
	UpcomingItemsLimit int
}

type ScheduledScanOverviewProjection struct {
	EnabledScheduledScanCount     int64
	PausedScheduledScanCount      int64
	TodayScheduledScanCount       int64
	Next24HoursScheduledScanCount int64
	UpcomingScheduledScans        []ScheduledScanOverviewUpcoming
}

type ScheduledScanOverviewUpcoming struct {
	ID                      int
	DisplayName             string
	OrganizationID          *int
	OrganizationDisplayName *string
	TargetID                *int
	TargetDisplayName       *string
	NextRunTime             time.Time
}

type ScheduledScanOverview struct {
	AsOfTime                      time.Time
	EnabledScheduledScanCount     int64
	PausedScheduledScanCount      int64
	TodayScheduledScanCount       int64
	Next24HoursScheduledScanCount int64
	UpcomingScheduledScans        []ScheduledScanOverviewUpcoming
}

func (service *ScheduledScanService) GetOverviewSummary(ctx context.Context, input *ScheduledScanOverviewInput) (*ScheduledScanOverview, error) {
	if service == nil || service.store == nil || input == nil {
		return nil, ErrScheduledScanInvalidArgument
	}

	// Read the clock once: every returned window and asOfTime refers to this instant.
	asOfTime := service.now().UTC()
	todayStart := time.Date(asOfTime.Year(), asOfTime.Month(), asOfTime.Day(), 0, 0, 0, 0, time.UTC)
	todayEnd := todayStart.Add(24 * time.Hour)

	projection, err := service.store.GetOverviewSummary(ctx, ScheduledScanOverviewQuery{
		TodayStart:         todayStart,
		TodayEnd:           todayEnd,
		Next24HoursStart:   asOfTime,
		Next24HoursEnd:     asOfTime.Add(24 * time.Hour),
		UpcomingItemsLimit: scheduledScanOverviewUpcomingLimit,
	})
	if err != nil {
		return nil, err
	}
	if projection == nil {
		return nil, fmt.Errorf("scheduled scan overview projection is required")
	}

	return &ScheduledScanOverview{
		AsOfTime:                      asOfTime,
		EnabledScheduledScanCount:     projection.EnabledScheduledScanCount,
		PausedScheduledScanCount:      projection.PausedScheduledScanCount,
		TodayScheduledScanCount:       projection.TodayScheduledScanCount,
		Next24HoursScheduledScanCount: projection.Next24HoursScheduledScanCount,
		UpcomingScheduledScans:        append([]ScheduledScanOverviewUpcoming(nil), projection.UpcomingScheduledScans...),
	}, nil
}
