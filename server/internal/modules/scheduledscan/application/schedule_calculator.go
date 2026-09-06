package application

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/robfig/cron/v3"
)

var ErrInvalidScheduleRule = errors.New("invalid scheduled scan time rule")

// ScheduleCalculator is the only boundary that interprets persisted Cron.
// Every schedule is evaluated in UTC and callers keep cursors and occurrences
// in the same absolute time basis.
type ScheduleCalculator interface {
	Validate(cronExpression string) error
	FirstAfter(cronExpression string, instant time.Time) (time.Time, error)
	LatestAtOrBefore(cronExpression string, persistedCursor, instant time.Time) (time.Time, error)
	AdvanceAfter(cronExpression string, instant time.Time) (time.Time, error)
}

type CronScheduleCalculator struct {
	parser cron.Parser
}

func NewCronScheduleCalculator() *CronScheduleCalculator {
	return &CronScheduleCalculator{parser: cron.NewParser(
		cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow,
	)}
}

func (calculator *CronScheduleCalculator) Validate(cronExpression string) error {
	_, err := calculator.parse(cronExpression)
	return err
}

func (calculator *CronScheduleCalculator) FirstAfter(cronExpression string, instant time.Time) (time.Time, error) {
	schedule, err := calculator.parse(cronExpression)
	if err != nil {
		return time.Time{}, err
	}
	return schedule.Next(instant.UTC()).UTC(), nil
}

func (calculator *CronScheduleCalculator) AdvanceAfter(cronExpression string, instant time.Time) (time.Time, error) {
	return calculator.FirstAfter(cronExpression, instant)
}

func (calculator *CronScheduleCalculator) LatestAtOrBefore(
	cronExpression string,
	persistedCursor time.Time,
	instant time.Time,
) (time.Time, error) {
	schedule, err := calculator.parse(cronExpression)
	if err != nil {
		return time.Time{}, err
	}

	cursor := persistedCursor.UTC()
	evaluationAt := instant.UTC()
	if cursor.IsZero() || cursor.After(evaluationAt) {
		return time.Time{}, fmt.Errorf("%w: persisted cursor must be due", ErrInvalidScheduleRule)
	}

	// Schedule.Next is monotonic in its input. Searching the UTC-second range
	// finds the last matching instant in logarithmic calls, even after a long
	// outage, while preserving distinct instants in a repeated DST hour.
	latest := cursor
	low, high := cursor.Unix(), evaluationAt.Unix()
	for low <= high {
		mid := low + (high-low)/2
		next := schedule.Next(time.Unix(mid, 0).UTC()).UTC()
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

func (calculator *CronScheduleCalculator) parse(cronExpression string) (cron.Schedule, error) {
	if calculator == nil {
		return nil, fmt.Errorf("%w: calculator is required", ErrInvalidScheduleRule)
	}
	expression := strings.TrimSpace(cronExpression)
	if expression == "" {
		return nil, fmt.Errorf("%w: cronExpression is required", ErrInvalidScheduleRule)
	}
	if strings.HasPrefix(expression, "@") {
		return nil, fmt.Errorf("%w: Cron descriptors are not supported", ErrInvalidScheduleRule)
	}
	fields := strings.Fields(expression)
	if len(fields) != 5 {
		return nil, fmt.Errorf("%w: cronExpression must contain exactly five fields", ErrInvalidScheduleRule)
	}
	for _, field := range fields {
		upper := strings.ToUpper(field)
		if strings.HasPrefix(upper, "TZ=") || strings.HasPrefix(upper, "CRON_TZ=") {
			return nil, fmt.Errorf("%w: embedded time zones are not supported", ErrInvalidScheduleRule)
		}
	}
	schedule, err := calculator.parser.Parse(strings.Join(fields, " "))
	if err != nil {
		return nil, fmt.Errorf("%w: invalid cronExpression: %v", ErrInvalidScheduleRule, err)
	}
	return schedule, nil
}
