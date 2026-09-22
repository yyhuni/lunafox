package main

import (
	"os"
	"strings"
	"testing"
)

func TestUpgradeOperationBaselineMigrationContract(t *testing.T) {
	up, err := os.ReadFile("migrations/000001_init_schema.up.sql")
	if err != nil {
		t.Fatalf("read initial schema migration: %v", err)
	}
	down, err := os.ReadFile("migrations/000001_init_schema.down.sql")
	if err != nil {
		t.Fatalf("read initial schema down migration: %v", err)
	}
	upSQL := string(up)
	downSQL := string(down)
	for _, required := range []string{
		"CREATE TABLE IF NOT EXISTS upgrade_operation (",
		"id UUID PRIMARY KEY",
		"request_id UUID NOT NULL UNIQUE",
		"operator_id INTEGER NOT NULL REFERENCES auth_user(id) ON DELETE RESTRICT",
		"manifest_digest VARCHAR(71) NOT NULL CHECK (manifest_digest ~ '^sha256:[a-f0-9]{64}$')",
		"status VARCHAR(32) NOT NULL DEFAULT 'queued' CHECK",
		"migration_status VARCHAR(32) NOT NULL DEFAULT 'not_started' CHECK",
		"migration_checksum VARCHAR(71) NOT NULL DEFAULT '' CHECK",
		"observed_digests JSONB NOT NULL DEFAULT '{}'::jsonb CHECK (jsonb_typeof(observed_digests) = 'object')",
		"stage_times JSONB NOT NULL DEFAULT '{}'::jsonb CHECK (jsonb_typeof(stage_times) = 'object')",
		"progress_events JSONB NOT NULL DEFAULT '[]'::jsonb CHECK (jsonb_typeof(progress_events) = 'array')",
		"CREATE UNIQUE INDEX IF NOT EXISTS idx_upgrade_operation_one_active",
	} {
		if !strings.Contains(upSQL, required) {
			t.Fatalf("initial schema missing Upgrade Operation contract %q", required)
		}
	}
	if !strings.Contains(downSQL, "DROP TABLE IF EXISTS upgrade_operation CASCADE;") {
		t.Fatal("initial schema down migration must drop upgrade_operation")
	}
}

func TestUpgradeOperationScopeMigrationContract(t *testing.T) {
	up, err := os.ReadFile("migrations/000002_upgrade_operation_scope.up.sql")
	if err != nil {
		t.Fatalf("read scope migration: %v", err)
	}
	down, err := os.ReadFile("migrations/000002_upgrade_operation_scope.down.sql")
	if err != nil {
		t.Fatalf("read scope migration down: %v", err)
	}
	upSQL := string(up)
	for _, required := range []string{
		"ADD COLUMN IF NOT EXISTS execution_mode VARCHAR(32)",
		"ADD COLUMN IF NOT EXISTS work_disposition VARCHAR(32)",
		"ADD COLUMN IF NOT EXISTS plan_summary JSONB",
		"ADD COLUMN IF NOT EXISTS plan_digest VARCHAR(71)",
		"ADD COLUMN IF NOT EXISTS baseline_deployment_digest VARCHAR(71)",
		"ADD COLUMN IF NOT EXISTS confirmed_deployment_version VARCHAR(64)",
		"upgrade_operation_not_required_scope",
		"upgrade_operation_frontend_only_scope",
		"upgrade_operation_full_scope_evidence",
		"'{\"touchedServices\":[\"frontend\"]}'::jsonb",
	} {
		if !strings.Contains(upSQL, required) {
			t.Fatalf("scope migration missing Upgrade Operation contract %q", required)
		}
	}
	if !strings.HasPrefix(string(down), "-- DESTRUCTIVE TEST TEARDOWN ONLY.") {
		t.Fatal("scope migration down must be limited to destructive test teardown")
	}
	for _, required := range []string{
		"DROP CONSTRAINT IF EXISTS upgrade_operation_full_scope_evidence",
		"DROP COLUMN IF EXISTS execution_mode",
	} {
		if !strings.Contains(string(down), required) {
			t.Fatalf("scope migration down missing %q", required)
		}
	}
}
