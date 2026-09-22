package main

import (
	"os"
	"strings"
	"testing"
)

func TestScheduledScanTimeZoneMigrationBackfillsUTCWithoutRewritingCursor(t *testing.T) {
	up, err := os.ReadFile("migrations/000003_scheduled_scan_time_zone.up.sql")
	if err != nil {
		t.Fatalf("read scheduled scan time-zone migration: %v", err)
	}
	down, err := os.ReadFile("migrations/000003_scheduled_scan_time_zone.down.sql")
	if err != nil {
		t.Fatalf("read scheduled scan time-zone migration down: %v", err)
	}
	upSQL := string(up)
	for _, required := range []string{
		"ADD COLUMN IF NOT EXISTS time_zone VARCHAR(100)",
		"UPDATE scheduled_scan\nSET time_zone = 'UTC'\nWHERE time_zone IS NULL",
		"ALTER COLUMN time_zone SET NOT NULL",
	} {
		if !strings.Contains(upSQL, required) {
			t.Fatalf("scheduled scan time-zone migration missing %q", required)
		}
	}
	for _, forbidden := range []string{"SET cron_expression", "SET next_run_time"} {
		if strings.Contains(upSQL, forbidden) {
			t.Fatalf("scheduled scan time-zone migration must not rewrite %q", forbidden)
		}
	}
	if !strings.HasPrefix(string(down), "-- DESTRUCTIVE TEST TEARDOWN ONLY.") || !strings.Contains(string(down), "DROP COLUMN IF EXISTS time_zone") {
		t.Fatalf("scheduled scan time-zone down migration is not an explicit destructive teardown: %s", down)
	}
}
