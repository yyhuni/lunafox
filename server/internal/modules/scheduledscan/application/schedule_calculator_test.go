package application

import (
	"testing"
	"time"
)

func TestCronScheduleCalculatorValidatesStrictFiveFieldRulesAndIANAZone(t *testing.T) {
	calculator := NewCronScheduleCalculator()
	if err := calculator.Validate("* * * * *", "Asia/Shanghai"); err != nil {
		t.Fatalf("Validate() rejected minute-level IANA rule: %v", err)
	}
	if err := calculator.Validate("0 2 * * *", "UTC"); err != nil {
		t.Fatalf("Validate() rejected UTC: %v", err)
	}

	for name, rule := range map[string]struct {
		cron string
		zone string
	}{
		"seconds":          {cron: "0 * * * * *", zone: "UTC"},
		"calendar":         {cron: "@daily", zone: "UTC"},
		"fixed interval":   {cron: "@every 5m", zone: "UTC"},
		"embedded TZ":      {cron: "TZ=UTC 0 2 * * *", zone: "UTC"},
		"embedded CRON_TZ": {cron: "CRON_TZ=UTC 0 2 * * *", zone: "UTC"},
		"missing zone":     {cron: "0 2 * * *"},
		"blank zone":       {cron: "0 2 * * *", zone: "   "},
		"local zone":       {cron: "0 2 * * *", zone: "Local"},
		"utc offset":       {cron: "0 2 * * *", zone: "UTC+8"},
		"numeric offset":   {cron: "0 2 * * *", zone: "+08:00"},
		"invalid zone":     {cron: "0 2 * * *", zone: "Mars/Olympus"},
	} {
		t.Run(name, func(t *testing.T) {
			if err := calculator.Validate(rule.cron, rule.zone); err == nil {
				t.Fatal("Validate() accepted unsupported rule")
			}
		})
	}
}

func TestCronScheduleCalculatorUsesStrictFutureAndClosedDueBoundaries(t *testing.T) {
	calculator := NewCronScheduleCalculator()
	boundary := time.Date(2026, 8, 4, 10, 0, 0, 0, time.UTC)

	first, err := calculator.FirstAfter("* * * * *", "UTC", boundary)
	if err != nil {
		t.Fatalf("FirstAfter() error = %v", err)
	}
	wantFirst := boundary.Add(time.Minute)
	if !first.Equal(wantFirst) {
		t.Fatalf("FirstAfter() = %s, want %s", first, wantFirst)
	}

	latest, err := calculator.LatestAtOrBefore("* * * * *", "UTC", boundary, boundary.Add(10*time.Minute))
	if err != nil {
		t.Fatalf("LatestAtOrBefore() error = %v", err)
	}
	if want := boundary.Add(10 * time.Minute); !latest.Equal(want) {
		t.Fatalf("LatestAtOrBefore() = %s, want %s", latest, want)
	}
}

func TestCronScheduleCalculatorInterpretsCronAsShanghaiWallClockTime(t *testing.T) {
	calculator := NewCronScheduleCalculator()
	instant := time.Date(2026, 8, 4, 10, 0, 0, 0, time.UTC)

	next, err := calculator.FirstAfter("0 19 * * *", "Asia/Shanghai", instant)
	if err != nil {
		t.Fatalf("FirstAfter() error = %v", err)
	}
	if want := time.Date(2026, 8, 4, 11, 0, 0, 0, time.UTC); !next.Equal(want) {
		t.Fatalf("FirstAfter() = %s, want Shanghai 19:00 at %s", next, want)
	}
}

func TestCronScheduleCalculatorSkipsDSTGapAndKeepsEarlierRepeatedInstant(t *testing.T) {
	calculator := NewCronScheduleCalculator()

	springStart := time.Date(2026, 3, 7, 7, 31, 0, 0, time.UTC)
	spring, err := calculator.FirstAfter("30 2 * * *", "America/New_York", springStart)
	if err != nil {
		t.Fatalf("spring FirstAfter() error = %v", err)
	}
	if want := time.Date(2026, 3, 9, 6, 30, 0, 0, time.UTC); !spring.Equal(want) {
		t.Fatalf("spring FirstAfter() = %s, want gap skipped to %s", spring, want)
	}

	beforeFall := time.Date(2026, 11, 1, 4, 0, 0, 0, time.UTC)
	firstFall, err := calculator.FirstAfter("30 1 * * *", "America/New_York", beforeFall)
	if err != nil {
		t.Fatalf("fall FirstAfter() error = %v", err)
	}
	if want := time.Date(2026, 11, 1, 5, 30, 0, 0, time.UTC); !firstFall.Equal(want) {
		t.Fatalf("fall FirstAfter() = %s, want earlier repeated instant %s", firstFall, want)
	}

	afterFirstFall, err := calculator.AdvanceAfter("30 1 * * *", "America/New_York", firstFall)
	if err != nil {
		t.Fatalf("fall AdvanceAfter() error = %v", err)
	}
	if want := time.Date(2026, 11, 2, 6, 30, 0, 0, time.UTC); !afterFirstFall.Equal(want) {
		t.Fatalf("fall AdvanceAfter() = %s, want next local day %s", afterFirstFall, want)
	}

	latest, err := calculator.LatestAtOrBefore("30 1 * * *", "America/New_York", firstFall, time.Date(2026, 11, 1, 6, 45, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("fall LatestAtOrBefore() error = %v", err)
	}
	if !latest.Equal(firstFall) {
		t.Fatalf("fall LatestAtOrBefore() = %s, want only earlier repeated instant %s", latest, firstFall)
	}
}

func TestCronScheduleCalculatorLatestDueIsBoundedAcrossLongOutage(t *testing.T) {
	calculator := NewCronScheduleCalculator()
	cursor := time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)
	evaluationAt := time.Date(2026, 8, 4, 10, 47, 31, 0, time.UTC)

	latest, err := calculator.LatestAtOrBefore("* * * * *", "UTC", cursor, evaluationAt)
	if err != nil {
		t.Fatalf("LatestAtOrBefore() error = %v", err)
	}
	if want := time.Date(2026, 8, 4, 10, 47, 0, 0, time.UTC); !latest.Equal(want) {
		t.Fatalf("LatestAtOrBefore() = %s, want %s", latest, want)
	}
}
