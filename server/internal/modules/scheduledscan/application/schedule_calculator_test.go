package application

import (
	"testing"
	"time"
)

func TestCronScheduleCalculatorValidatesStrictFiveFieldRules(t *testing.T) {
	calculator := NewCronScheduleCalculator()
	if err := calculator.Validate("* * * * *"); err != nil {
		t.Fatalf("Validate() rejected minute-level rule: %v", err)
	}

	for name, cron := range map[string]string{
		"seconds":          "0 * * * * *",
		"calendar":         "@daily",
		"fixed interval":   "@every 5m",
		"embedded TZ":      "TZ=UTC 0 2 * * *",
		"embedded CRON_TZ": "CRON_TZ=UTC 0 2 * * *",
	} {
		t.Run(name, func(t *testing.T) {
			if err := calculator.Validate(cron); err == nil {
				t.Fatal("Validate() accepted unsupported rule")
			}
		})
	}
}

func TestCronScheduleCalculatorUsesStrictFutureAndClosedDueBoundaries(t *testing.T) {
	calculator := NewCronScheduleCalculator()
	boundary := time.Date(2026, 8, 4, 10, 0, 0, 0, time.UTC)

	first, err := calculator.FirstAfter("* * * * *", boundary)
	if err != nil {
		t.Fatalf("FirstAfter() error = %v", err)
	}
	wantFirst := boundary.Add(time.Minute)
	if !first.Equal(wantFirst) {
		t.Fatalf("FirstAfter() = %s, want %s", first, wantFirst)
	}

	latest, err := calculator.LatestAtOrBefore("* * * * *", boundary, boundary.Add(10*time.Minute))
	if err != nil {
		t.Fatalf("LatestAtOrBefore() error = %v", err)
	}
	if want := boundary.Add(10 * time.Minute); !latest.Equal(want) {
		t.Fatalf("LatestAtOrBefore() = %s, want %s", latest, want)
	}
}

func TestCronScheduleCalculatorEvaluatesRulesInUTC(t *testing.T) {
	calculator := NewCronScheduleCalculator()
	instant := time.Date(2026, 8, 4, 1, 59, 0, 0, time.FixedZone("UTC+8", 8*60*60))
	next, err := calculator.FirstAfter("0 2 * * *", instant)
	if err != nil {
		t.Fatalf("FirstAfter() error = %v", err)
	}
	if want := time.Date(2026, 8, 4, 2, 0, 0, 0, time.UTC); !next.Equal(want) || next.Location() != time.UTC {
		t.Fatalf("FirstAfter() = %s (%s), want UTC %s", next, next.Location(), want)
	}
}

func TestCronScheduleCalculatorLatestDueIsBoundedAcrossLongOutage(t *testing.T) {
	calculator := NewCronScheduleCalculator()
	cursor := time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)
	evaluationAt := time.Date(2026, 8, 4, 10, 47, 31, 0, time.UTC)

	latest, err := calculator.LatestAtOrBefore("* * * * *", cursor, evaluationAt)
	if err != nil {
		t.Fatalf("LatestAtOrBefore() error = %v", err)
	}
	if want := time.Date(2026, 8, 4, 10, 47, 0, 0, time.UTC); !latest.Equal(want) {
		t.Fatalf("LatestAtOrBefore() = %s, want %s", latest, want)
	}
}
