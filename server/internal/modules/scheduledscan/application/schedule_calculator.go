package application

import (
	"errors"
	"fmt"
	"strings"
	"time"
	_ "time/tzdata"

	"github.com/robfig/cron/v3"
)

var ErrInvalidScheduleRule = errors.New("invalid scheduled scan time rule")

// ScheduleCalculator is the only boundary that interprets persisted Cron and
// time-zone rules. Callers keep every persisted cursor and occurrence in UTC.
type ScheduleCalculator interface {
	Validate(cronExpression, timeZone string) error
	FirstAfter(cronExpression, timeZone string, instant time.Time) (time.Time, error)
	LatestAtOrBefore(cronExpression, timeZone string, persistedCursor, instant time.Time) (time.Time, error)
	AdvanceAfter(cronExpression, timeZone string, instant time.Time) (time.Time, error)
}

type CronScheduleCalculator struct {
	parser cron.Parser
}

func NewCronScheduleCalculator() *CronScheduleCalculator {
	return &CronScheduleCalculator{parser: cron.NewParser(
		cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow,
	)}
}

func (calculator *CronScheduleCalculator) Validate(cronExpression, timeZone string) error {
	_, _, err := calculator.parse(cronExpression, timeZone)
	return err
}

func (calculator *CronScheduleCalculator) FirstAfter(cronExpression, timeZone string, instant time.Time) (time.Time, error) {
	schedule, location, err := calculator.parse(cronExpression, timeZone)
	if err != nil {
		return time.Time{}, err
	}
	return firstAfter(schedule, location, instant)
}

func (calculator *CronScheduleCalculator) AdvanceAfter(cronExpression, timeZone string, instant time.Time) (time.Time, error) {
	return calculator.FirstAfter(cronExpression, timeZone, instant)
}

func (calculator *CronScheduleCalculator) LatestAtOrBefore(
	cronExpression string,
	timeZone string,
	persistedCursor time.Time,
	instant time.Time,
) (time.Time, error) {
	schedule, location, err := calculator.parse(cronExpression, timeZone)
	if err != nil {
		return time.Time{}, err
	}

	cursor := persistedCursor.UTC()
	evaluationAt := instant.UTC()
	if cursor.IsZero() || cursor.After(evaluationAt) {
		return time.Time{}, fmt.Errorf("%w: persisted cursor must be due", ErrInvalidScheduleRule)
	}

	// Search the UTC-second range in logarithmic calls, even after a long
	// outage. firstAfter keeps only the earlier instant of a repeated local hour.
	latest := cursor
	low, high := cursor.Unix(), evaluationAt.Unix()
	for low <= high {
		mid := low + (high-low)/2
		next, err := firstAfter(schedule, location, time.Unix(mid, 0).UTC())
		if err != nil {
			return time.Time{}, err
		}
		if next.After(evaluationAt) {
			high = mid - 1
			continue
		}
		if next.After(latest) {
			latest = next
		}
		low = mid + 1
	}
	return latest, nil
}

func (calculator *CronScheduleCalculator) parse(cronExpression, timeZone string) (cron.Schedule, *time.Location, error) {
	if calculator == nil {
		return nil, nil, fmt.Errorf("%w: calculator is required", ErrInvalidScheduleRule)
	}
	expression := strings.TrimSpace(cronExpression)
	if expression == "" {
		return nil, nil, fmt.Errorf("%w: cronExpression is required", ErrInvalidScheduleRule)
	}
	if strings.HasPrefix(expression, "@") {
		return nil, nil, fmt.Errorf("%w: Cron descriptors are not supported", ErrInvalidScheduleRule)
	}
	fields := strings.Fields(expression)
	if len(fields) != 5 {
		return nil, nil, fmt.Errorf("%w: cronExpression must contain exactly five fields", ErrInvalidScheduleRule)
	}
	for _, field := range fields {
		upper := strings.ToUpper(field)
		if strings.HasPrefix(upper, "TZ=") || strings.HasPrefix(upper, "CRON_TZ=") {
			return nil, nil, fmt.Errorf("%w: embedded time zones are not supported", ErrInvalidScheduleRule)
		}
	}
	location, err := parseIANAZone(timeZone)
	if err != nil {
		return nil, nil, err
	}
	schedule, err := calculator.parser.Parse(strings.Join(fields, " "))
	if err != nil {
		return nil, nil, fmt.Errorf("%w: invalid cronExpression: %v", ErrInvalidScheduleRule, err)
	}
	return schedule, location, nil
}

func parseIANAZone(value string) (*time.Location, error) {
	zone := strings.TrimSpace(value)
	if zone == "" {
		return nil, fmt.Errorf("%w: timeZone is required", ErrInvalidScheduleRule)
	}
	if zone == "Local" || looksLikeFixedOffsetZone(zone) {
		return nil, fmt.Errorf("%w: timeZone must be an explicit IANA location", ErrInvalidScheduleRule)
	}
	location, err := time.LoadLocation(zone)
	if err != nil {
		return nil, fmt.Errorf("%w: invalid timeZone", ErrInvalidScheduleRule)
	}
	return location, nil
}

func looksLikeFixedOffsetZone(zone string) bool {
	upper := strings.ToUpper(zone)
	return strings.HasPrefix(zone, "+") || strings.HasPrefix(zone, "-") ||
		strings.HasPrefix(upper, "UTC+") || strings.HasPrefix(upper, "UTC-") ||
		strings.HasPrefix(upper, "GMT+") || strings.HasPrefix(upper, "GMT-")
}

func firstAfter(schedule cron.Schedule, location *time.Location, instant time.Time) (time.Time, error) {
	if schedule == nil || location == nil {
		return time.Time{}, fmt.Errorf("%w: schedule and timeZone are required", ErrInvalidScheduleRule)
	}
	cursor := instant.In(location)
	for {
		next := schedule.Next(cursor)
		if next.IsZero() {
			return time.Time{}, fmt.Errorf("%w: cronExpression has no future occurrence", ErrInvalidScheduleRule)
		}
		nextUTC := next.UTC()
		if earliestInstantForLocalMinute(nextUTC, location).Equal(nextUTC) {
			return nextUTC, nil
		}
		// robfig/cron can return the second representation of a repeated local
		// minute. A schedule must fire once, at that minute's earlier instant.
		cursor = next
	}
}

func earliestInstantForLocalMinute(instant time.Time, location *time.Location) time.Time {
	local := instant.In(location)
	nominal := time.Date(local.Year(), local.Month(), local.Day(), local.Hour(), local.Minute(), 0, 0, time.UTC)
	earliest := instant.UTC()
	offsets := make(map[int]struct{})
	for hour := -36; hour <= 36; hour++ {
		_, offset := instant.Add(time.Duration(hour) * time.Hour).In(location).Zone()
		offsets[offset] = struct{}{}
	}
	for offset := range offsets {
		candidate := nominal.Add(-time.Duration(offset) * time.Second)
		projected := candidate.In(location)
		if projected.Year() == local.Year() &&
			projected.Month() == local.Month() &&
			projected.Day() == local.Day() &&
			projected.Hour() == local.Hour() &&
			projected.Minute() == local.Minute() &&
			candidate.Before(earliest) {
			earliest = candidate.UTC()
		}
	}
	return earliest
}
