package application

import (
	"context"
	"fmt"
	"time"
)

const scheduledScanOverviewUpcomingLimit = 5

// Horizon projection constants: the management axis renders a server-defined
// rolling window (past lookback + future horizon) aggregated into UTC hour
// buckets. The expansion cap is a fuse against calculator pathologies; a
// minute-level cron inside the 30-hour window peaks at 1800 occurrences, so
// the cap safely exceeds it and only guards against runaway loops (AD-04).
const (
	ScheduledScanHorizonLookback    = 6 * time.Hour
	ScheduledScanHorizonDuration    = 24 * time.Hour
	MaxHorizonExpansionsPerSchedule = 2000
)

// ScheduledScanOverviewInput is intentionally empty. Overview windows are
// derived server-side from one UTC clock reading.
type ScheduledScanOverviewInput struct{}

type ScheduledScanOverviewQuery struct {
	TodayStart         time.Time
	TodayEnd           time.Time
	Next24HoursStart   time.Time
	Next24HoursEnd     time.Time
	UpcomingItemsLimit int
	HorizonStart       time.Time
	HorizonEnd         time.Time
}

type ScheduledScanOverviewProjection struct {
	EnabledScheduledScanCount     int64
	PausedScheduledScanCount      int64
	TodayScheduledScanCount       int64
	Next24HoursScheduledScanCount int64
	UpcomingScheduledScans        []ScheduledScanOverviewUpcoming
	HorizonBuckets                []ScheduledScanHorizonBucket
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

// ScheduledScanHorizonItem mirrors the upcoming item scope projection so both
// lists share one response shape.
type ScheduledScanHorizonItem struct {
	ID                      int
	DisplayName             string
	OrganizationID          *int
	OrganizationDisplayName *string
	TargetID                *int
	TargetDisplayName       *string
}

type ScheduledScanHorizonWindow struct {
	Start time.Time
	End   time.Time
}

type ScheduledScanHorizonBucket struct {
	HourStart time.Time
	Items     []ScheduledScanHorizonItem
}

type ScheduledScanOverview struct {
	AsOfTime                      time.Time
	EnabledScheduledScanCount     int64
	PausedScheduledScanCount      int64
	TodayScheduledScanCount       int64
	Next24HoursScheduledScanCount int64
	UpcomingScheduledScans        []ScheduledScanOverviewUpcoming
	HorizonWindow                 ScheduledScanHorizonWindow
	HorizonBuckets                []ScheduledScanHorizonBucket
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
		HorizonStart:       asOfTime.Add(-ScheduledScanHorizonLookback),
		HorizonEnd:         asOfTime.Add(ScheduledScanHorizonDuration),
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
		HorizonWindow: ScheduledScanHorizonWindow{
			Start: asOfTime.Add(-ScheduledScanHorizonLookback),
			End:   asOfTime.Add(ScheduledScanHorizonDuration),
		},
		HorizonBuckets: append([]ScheduledScanHorizonBucket(nil), projection.HorizonBuckets...),
	}, nil
}
