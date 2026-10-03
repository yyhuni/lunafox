package main

import (
	"os"
	"strings"
	"testing"
)

func TestScheduledScanDisabledCursorCleanupMigrationClearsOnlyDisabledStaleCursors(t *testing.T) {
	up, err := os.ReadFile("migrations/000006_scheduled_scan_disabled_cursor_cleanup.up.sql")
	if err != nil {
		t.Fatalf("read scheduled scan disabled-cursor cleanup migration: %v", err)
	}
	down, err := os.ReadFile("migrations/000006_scheduled_scan_disabled_cursor_cleanup.down.sql")
	if err != nil {
		t.Fatalf("read scheduled scan disabled-cursor cleanup migration down: %v", err)
	}
	upSQL := string(up)
	for _, required := range []string{
		"UPDATE scheduled_scan",
		"SET next_run_time = NULL",
		"WHERE is_enabled = FALSE AND next_run_time IS NOT NULL",
	} {
		if !strings.Contains(upSQL, required) {
			t.Fatalf("scheduled scan disabled-cursor cleanup migration missing %q", required)
		}
	}
	for _, forbidden := range []string{
		"SET cron_expression", "SET time_zone", "SET is_enabled", "DELETE", "ALTER TABLE", "DROP",
	} {
		if strings.Contains(upSQL, forbidden) {
			t.Fatalf("scheduled scan disabled-cursor cleanup migration must not contain %q", forbidden)
		}
	}
	if !strings.HasPrefix(string(down), "-- DESTRUCTIVE TEST TEARDOWN ONLY.") {
		t.Fatalf("scheduled scan disabled-cursor cleanup down migration is not an explicit destructive teardown: %s", down)
	}
	if strings.Contains(strings.ToUpper(string(down)), "UPDATE") {
		t.Fatalf("scheduled scan disabled-cursor cleanup down migration must stay a no-op placeholder: %s", down)
	}
}
