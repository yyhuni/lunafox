package main

import (
	"os"
	"strings"
	"testing"
)

func TestTargetBlacklistPolicyBackfillMigrationIsIdempotentAndPreservesRows(t *testing.T) {
	up, err := os.ReadFile("migrations/000010_backfill_target_blacklist_policy.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	sql := string(up)
	for _, required := range []string{
		"INSERT INTO blacklist_policy (scope, target_id, patterns)",
		"SELECT 'target', target.id, '[]'::jsonb",
		"FROM target",
		"WHERE NOT EXISTS",
		"blacklist_policy.scope = 'target'",
		"blacklist_policy.target_id = target.id",
	} {
		if !strings.Contains(sql, required) {
			t.Fatalf("backfill migration missing %q", required)
		}
	}
	if strings.Contains(sql, "UPDATE blacklist_policy") || strings.Contains(sql, "DELETE FROM blacklist_policy") {
		t.Fatal("backfill migration must not overwrite or delete existing policy rows")
	}
}
