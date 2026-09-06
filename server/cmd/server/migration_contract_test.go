package main

import (
	"os"
	"regexp"
	"slices"
	"strings"
	"testing"

	notificationdomain "github.com/yyhuni/lunafox/server/internal/modules/notification/domain"
)

func readInitialSchema(t *testing.T) string {
	t.Helper()

	sql, err := os.ReadFile("migrations/000001_init_schema.up.sql")
	if err != nil {
		t.Fatalf("read initial schema migration: %v", err)
	}

	return string(sql)
}

func readInitialSchemaDown(t *testing.T) string {
	t.Helper()

	sql, err := os.ReadFile("migrations/000001_init_schema.down.sql")
	if err != nil {
		t.Fatalf("read initial schema down migration: %v", err)
	}

	return string(sql)
}

func extractTableNames(pattern, text string) []string {
	re := regexp.MustCompile(pattern)
	matches := re.FindAllStringSubmatch(text, -1)
	names := make([]string, 0, len(matches))
	seen := make(map[string]struct{}, len(matches))
	for _, match := range matches {
		if len(match) < 2 {
			continue
		}
		name := match[1]
		if _, ok := seen[name]; ok {
			continue
		}
		seen[name] = struct{}{}
		names = append(names, name)
	}
	slices.Sort(names)
	return names
}

func TestInitialSchemaDefinesSubfinderProviderSettingsOnceAsSingleton(t *testing.T) {
	sql := readInitialSchema(t)
	re := regexp.MustCompile(`(?s)CREATE TABLE IF NOT EXISTS subfinder_provider_settings \((.*?)\);`)
	matches := re.FindAllStringSubmatch(sql, -1)
	if len(matches) != 1 {
		t.Fatalf("expected exactly one subfinder_provider_settings table definition, got %d", len(matches))
	}

	if !strings.Contains(matches[0][1], "CHECK (id = 1)") {
		t.Fatal("subfinder_provider_settings baseline must enforce singleton id = 1 shape")
	}
}

func TestInitialSchemaDefinesBlacklistPolicyAndScanSnapshot(t *testing.T) {
	up := readInitialSchema(t)
	policy := regexp.MustCompile(`(?s)CREATE TABLE IF NOT EXISTS blacklist_policy \((.*?)\);`).FindStringSubmatch(up)
	if len(policy) != 2 {
		t.Fatal("expected exactly one blacklist_policy table definition")
	}
	for _, required := range []string{
		"id SERIAL PRIMARY KEY",
		"scope VARCHAR(20) NOT NULL",
		"target_id INTEGER REFERENCES target(id) ON DELETE CASCADE",
		"patterns JSONB NOT NULL DEFAULT '[]'::jsonb",
		"updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP",
		"CONSTRAINT blacklist_policy_scope CHECK",
		"CONSTRAINT blacklist_policy_scope_target_shape CHECK",
		"CONSTRAINT blacklist_policy_patterns_string_array CHECK",
	} {
		if !strings.Contains(policy[1], required) {
			t.Fatalf("blacklist_policy baseline must contain %q", required)
		}
	}
	for _, forbidden := range []string{"rule_type", "etag", "version", "created_at", "description"} {
		if strings.Contains(policy[1], forbidden) {
			t.Fatalf("blacklist_policy baseline must not contain %q", forbidden)
		}
	}
	for _, required := range []string{
		"CREATE UNIQUE INDEX IF NOT EXISTS idx_blacklist_policy_global_singleton",
		"CREATE UNIQUE INDEX IF NOT EXISTS idx_blacklist_policy_target_singleton",
		"INSERT INTO blacklist_policy (scope, target_id, patterns)",
	} {
		if !strings.Contains(up, required) {
			t.Fatalf("blacklist policy baseline missing %q", required)
		}
	}

	snapshot := regexp.MustCompile(`(?s)CREATE TABLE IF NOT EXISTS scan_blacklist_snapshot \((.*?)\);`).FindStringSubmatch(up)
	if len(snapshot) != 2 {
		t.Fatal("expected exactly one scan_blacklist_snapshot table definition")
	}
	for _, required := range []string{
		"scan_id INTEGER PRIMARY KEY REFERENCES scan(id) ON DELETE CASCADE",
		"patterns JSONB NOT NULL",
		"CONSTRAINT scan_blacklist_snapshot_patterns_string_array CHECK",
	} {
		if !strings.Contains(snapshot[1], required) {
			t.Fatalf("scan_blacklist_snapshot baseline must contain %q", required)
		}
	}
	for _, forbidden := range []string{"id SERIAL", "etag", "updated_at", "created_at", "target_id"} {
		if strings.Contains(snapshot[1], forbidden) {
			t.Fatalf("scan_blacklist_snapshot baseline must not contain %q", forbidden)
		}
	}
	if strings.Contains(up, "blacklist_rule") {
		t.Fatal("initial schema must not retain blacklist_rule")
	}
	for _, required := range []string{
		"DROP TABLE IF EXISTS scan_blacklist_snapshot CASCADE;",
		"DROP TABLE IF EXISTS blacklist_policy CASCADE;",
	} {
		if !strings.Contains(readInitialSchemaDown(t), required) {
			t.Fatalf("initial schema down migration must contain %q", required)
		}
	}
}

func TestInitialSchemaDefinesScanBackedMCPOperationAndBoundedReplay(t *testing.T) {
	up := readInitialSchema(t)
	operation := regexp.MustCompile(`(?s)CREATE TABLE IF NOT EXISTS scan_operation \((.*?)\);`).FindStringSubmatch(up)
	if len(operation) != 2 {
		t.Fatal("expected exactly one scan_operation table definition")
	}
	for _, required := range []string{
		"id UUID PRIMARY KEY",
		"scan_id INTEGER NOT NULL UNIQUE REFERENCES scan(id) ON DELETE CASCADE",
		"target_id INTEGER NOT NULL",
		"request_id VARCHAR(128)",
		"request_fingerprint VARCHAR(64) NOT NULL",
		"CONSTRAINT scan_operation_request_fingerprint_format CHECK",
	} {
		if !strings.Contains(operation[1], required) {
			t.Fatalf("scan_operation baseline must contain %q", required)
		}
	}
	if strings.Contains(operation[1], "status ") || strings.Contains(operation[1], "expires_at") {
		t.Fatal("scan_operation must project Scan lifecycle and must not own a TTL")
	}
	for _, required := range []string{
		"CREATE INDEX IF NOT EXISTS idx_scan_operation_target_id",
		"CREATE INDEX IF NOT EXISTS idx_scan_operation_request_id",
	} {
		if !strings.Contains(up, required) {
			t.Fatalf("scan_operation baseline missing %q", required)
		}
	}

	replay := regexp.MustCompile(`(?s)CREATE TABLE IF NOT EXISTS mcp_request_replay \((.*?)\);`).FindStringSubmatch(up)
	if len(replay) != 2 {
		t.Fatal("expected exactly one mcp_request_replay table definition")
	}
	for _, required := range []string{
		"request_id VARCHAR(128) PRIMARY KEY",
		"action VARCHAR(80) NOT NULL",
		"request_fingerprint VARCHAR(64) NOT NULL",
		"response JSONB NOT NULL",
		"expires_at TIMESTAMPTZ NOT NULL",
		"CONSTRAINT mcp_request_replay_fingerprint_format CHECK",
		"CREATE INDEX IF NOT EXISTS idx_mcp_request_replay_expires_at",
	} {
		if !strings.Contains(up, required) {
			t.Fatalf("mcp_request_replay baseline must contain %q", required)
		}
	}
	for _, required := range []string{
		"DROP TABLE IF EXISTS mcp_request_replay CASCADE;",
		"DROP TABLE IF EXISTS scan_operation CASCADE;",
	} {
		if !strings.Contains(readInitialSchemaDown(t), required) {
			t.Fatalf("initial schema down migration must contain %q", required)
		}
	}
}

func TestInitialSchemaDefinesAuthUserTokenVersion(t *testing.T) {
	up := readInitialSchema(t)
	user := regexp.MustCompile(`(?s)CREATE TABLE IF NOT EXISTS auth_user \((.*?)\);`).FindStringSubmatch(up)
	if len(user) != 2 {
		t.Fatal("expected exactly one auth_user table definition")
	}
	if !strings.Contains(user[1], "token_version INTEGER NOT NULL DEFAULT 0") {
		t.Fatal("auth_user baseline must define a non-null token_version defaulting to 0")
	}
	seed := "INSERT INTO auth_user (username, password, is_superuser, is_staff, is_active, date_joined)"
	if !strings.Contains(up, seed) {
		t.Fatal("initial administrator seed must retain its existing column list")
	}
	if strings.Contains(up, "INSERT INTO auth_user (username, password, token_version") {
		t.Fatal("initial administrator seed must rely on the token_version default")
	}
}

func TestInitialSchemaDefinesTargetCleanupBaseline(t *testing.T) {
	up := readInitialSchema(t)
	job := regexp.MustCompile(`(?s)CREATE TABLE IF NOT EXISTS target_cleanup_job \((.*?)\);`).FindStringSubmatch(up)
	if len(job) != 2 {
		t.Fatal("expected exactly one target_cleanup_job table definition")
	}
	for _, required := range []string{
		"id SERIAL PRIMARY KEY",
		"target_id INTEGER NOT NULL UNIQUE REFERENCES target(id) ON DELETE CASCADE",
		"status VARCHAR(20) NOT NULL DEFAULT 'pending'",
		"retry_count INTEGER NOT NULL DEFAULT 0",
		"next_retry_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP",
		"last_error VARCHAR(2000) NOT NULL DEFAULT ''",
		"completed_at TIMESTAMPTZ",
		"created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP",
		"updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP",
	} {
		if !strings.Contains(job[1], required) {
			t.Fatalf("target_cleanup_job baseline must contain %q", required)
		}
	}
	for _, forbidden := range []string{"phase", "cursor", "lease_owner", "lease_expires_at", "claimed_at", "worker"} {
		if strings.Contains(job[1], forbidden) {
			t.Fatalf("target_cleanup_job baseline must not contain %q", forbidden)
		}
	}
	for _, required := range []string{
		"CREATE UNIQUE INDEX IF NOT EXISTS idx_target_active_identity_unique",
		"ON target(type, name) WHERE deleted_at IS NULL",
		"CREATE INDEX IF NOT EXISTS idx_target_cleanup_job_due",
		"ON target_cleanup_job(status, next_retry_at, id) WHERE status = 'pending'",
		"CREATE INDEX IF NOT EXISTS idx_subdomain_target_id ON subdomain(target_id, id)",
		"CREATE INDEX IF NOT EXISTS idx_hpm_target_id ON host_port_mapping(target_id, id)",
		"CREATE INDEX IF NOT EXISTS idx_website_target_id ON website(target_id, id)",
		"CREATE INDEX IF NOT EXISTS idx_endpoint_target_id ON endpoint(target_id, id)",
		"CREATE INDEX IF NOT EXISTS idx_directory_target_id ON directory(target_id, id)",
		"CREATE INDEX IF NOT EXISTS idx_screenshot_target_id ON screenshot(target_id, id)",
		"CREATE INDEX IF NOT EXISTS idx_vuln_target_id ON vulnerability(target_id, id)",
	} {
		if !strings.Contains(up, required) {
			t.Fatalf("initial schema missing target cleanup contract %q", required)
		}
	}
	down := readInitialSchemaDown(t)
	jobDrop := strings.Index(down, "DROP TABLE IF EXISTS target_cleanup_job CASCADE;")
	targetDrop := strings.Index(down, "DROP TABLE IF EXISTS target CASCADE;")
	if jobDrop < 0 {
		t.Fatal("initial schema down migration must drop target_cleanup_job before target")
	}
	if targetDrop < 0 || jobDrop > targetDrop {
		t.Fatal("initial schema down migration must drop target_cleanup_job before its target foreign-key parent")
	}
}

func TestInitialSchemaUsesExactDirectoryObservationWidths(t *testing.T) {
	sql := readInitialSchema(t)
	tablePatterns := map[string]string{
		"directory":          `(?s)CREATE TABLE IF NOT EXISTS directory \((.*?)\);`,
		"directory_snapshot": `(?s)CREATE TABLE IF NOT EXISTS directory_snapshot \((.*?)\) PARTITION BY RANGE \(scan_id\);`,
	}
	for table, pattern := range tablePatterns {
		matches := regexp.MustCompile(pattern).FindStringSubmatch(sql)
		if len(matches) != 2 {
			t.Fatalf("expected exactly one %s table definition", table)
		}
		body := matches[1]
		for _, required := range []string{
			"content_length BIGINT",
			"content_type VARCHAR(1024) NOT NULL DEFAULT ''",
			"duration BIGINT",
		} {
			if !strings.Contains(body, required) {
				t.Fatalf("%s baseline must contain %q", table, required)
			}
		}
		for _, forbidden := range []string{"content_length INTEGER", "content_type VARCHAR(200)", "duration INTEGER"} {
			if strings.Contains(body, forbidden) {
				t.Fatalf("%s baseline retains lossy field %q", table, forbidden)
			}
		}
	}
}

func TestInitialSchemaDefinesFingerprintArtifactStateAndMetadata(t *testing.T) {
	sql := readInitialSchema(t)
	for _, required := range []string{
		"CREATE TABLE IF NOT EXISTS fingerprint_library_state",
		"INSERT INTO fingerprint_library_state (library, source_generation)",
		"('fingerprinthub', 0)",
		"CREATE TABLE IF NOT EXISTS fingerprint_library_artifact",
		"CONSTRAINT fingerprint_library_artifact_generation_key UNIQUE (library, source_generation)",
		"CONSTRAINT fingerprint_library_artifact_digest_format CHECK (sha256_digest ~ '^sha256:[0-9a-f]{64}$')",
		"CREATE INDEX IF NOT EXISTS idx_fingerprint_library_artifact_digest",
	} {
		if !strings.Contains(sql, required) {
			t.Fatalf("fingerprint artifact baseline missing %q", required)
		}
	}
	for _, forbidden := range []string{"fingerprint_library_artifact_descriptor", "task_artifact_binding", "execution_attempt"} {
		if strings.Contains(sql, forbidden) {
			t.Fatalf("fingerprint artifact baseline must not create %q", forbidden)
		}
	}
}

func TestInitialSchemaDefinesCurrentInstalledEngineInventory(t *testing.T) {
	sql := readInitialSchema(t)
	re := regexp.MustCompile(`(?s)CREATE TABLE IF NOT EXISTS engine \((.*?)\);`)
	matches := re.FindAllStringSubmatch(sql, -1)
	if len(matches) != 1 {
		t.Fatalf("expected exactly one engine table definition, got %d", len(matches))
	}
	for _, required := range []string{
		"engine_id VARCHAR(255) NOT NULL UNIQUE",
		"publisher VARCHAR(255) NOT NULL",
		"package_version VARCHAR(255) NOT NULL",
		"artifact_ref VARCHAR(1000) NOT NULL UNIQUE",
		"package_digest VARCHAR(71) NOT NULL UNIQUE",
		"manifest JSONB NOT NULL",
	} {
		if !strings.Contains(matches[0][1], required) {
			t.Fatalf("engine table must contain %q", required)
		}
	}
	for _, forbidden := range []string{
		"source_kind",
		"pending",
		"failed",
		"status",
		"current_version",
		"cache_path",
		"artifact_manifest_digest",
		"runtime_image_digest",
		"expanded_digest",
		"result_type",
		"result_types",
		"result_schema",
		"result_schema_ref",
		"result_schemas",
		"output_allow_set",
		"output_authorization",
		"outputs",
	} {
		if strings.Contains(matches[0][1], forbidden) {
			t.Fatalf("engine table must not contain unsupported state field %q", forbidden)
		}
	}
}

func TestInitialSchemaDefinesScanWorkflowAggregate(t *testing.T) {
	sql := readInitialSchema(t)
	re := regexp.MustCompile(`(?s)CREATE TABLE IF NOT EXISTS scan_workflow \((.*?)\);`)
	matches := re.FindAllStringSubmatch(sql, -1)
	if len(matches) != 1 {
		t.Fatalf("expected exactly one scan_workflow table definition, got %d", len(matches))
	}
	for _, required := range []string{
		"scan_workflow_id VARCHAR(100) PRIMARY KEY",
		"display_name VARCHAR(300) NOT NULL",
		"stages JSONB NOT NULL",
		"is_builtin BOOLEAN NOT NULL",
		"definition_digest VARCHAR(64)",
		"request_id UUID",
		"version BIGINT NOT NULL DEFAULT 1",
		"create_time TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP",
		"update_time TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP",
		"CONSTRAINT scan_workflow_namespace CHECK",
		"CONSTRAINT scan_workflow_builtin_digest_shape CHECK",
		"CONSTRAINT scan_workflow_request_ownership CHECK",
	} {
		if !strings.Contains(matches[0][1], required) {
			t.Fatalf("scan_workflow table must contain %q", required)
		}
	}
	for _, required := range []string{
		"CREATE UNIQUE INDEX IF NOT EXISTS idx_scan_workflow_request_id",
		"CREATE INDEX IF NOT EXISTS idx_scan_workflow_list_order",
	} {
		if !strings.Contains(sql, required) {
			t.Fatalf("initial schema must contain %q", required)
		}
	}
	if !strings.Contains(readInitialSchemaDown(t), "DROP TABLE IF EXISTS scan_workflow CASCADE;") {
		t.Fatal("initial schema down migration must drop scan_workflow")
	}
}

func TestInitialSchemaDefinesGlobalFingerprintLibraries(t *testing.T) {
	sql := readInitialSchema(t)
	re := regexp.MustCompile(`(?s)CREATE TABLE IF NOT EXISTS fingerprint_fingerprinthub \((.*?)\);`)
	matches := re.FindAllStringSubmatch(sql, -1)
	if len(matches) != 1 {
		t.Fatalf("expected exactly one FingerprintHub table definition, got %d", len(matches))
	}
	for _, required := range []string{
		"resource_id UUID NOT NULL PRIMARY KEY", "identity_key TEXT NOT NULL", "content_hash BYTEA NOT NULL", "payload JSONB NOT NULL",
		"fingerprint_id TEXT NOT NULL", "name TEXT NOT NULL", "severity TEXT",
		"CONSTRAINT fingerprint_fingerprinthub_identity_key_key UNIQUE (identity_key)",
	} {
		if !strings.Contains(matches[0][1], required) {
			t.Fatalf("FingerprintHub table must contain %q", required)
		}
	}

	for _, forbidden := range []string{
		"uuid-ossp",
		"pgcrypto",
		"gen_random_uuid",
		"uuid_generate",
		"fingerprint_identity_registry",
		"fingerprint_ehole", "fingerprint_goby", "fingerprint_wappalyzer", "fingerprint_fingers", "fingerprint_arl",
	} {
		if strings.Contains(sql, forbidden) {
			t.Fatalf("initial fingerprint schema must not contain %q", forbidden)
		}
	}
}

func TestInitialSchemaIndexesGlobalFingerprintLibraryQueries(t *testing.T) {
	sql := readInitialSchema(t)

	for _, required := range []string{
		"CREATE EXTENSION IF NOT EXISTS pg_trgm",
		"CREATE INDEX IF NOT EXISTS idx_fingerprint_fingerprinthub_name_trgm ON fingerprint_fingerprinthub USING GIN (name gin_trgm_ops)",
		"CREATE INDEX IF NOT EXISTS idx_fingerprint_fingerprinthub_severity ON fingerprint_fingerprinthub(severity)",
		"CREATE INDEX IF NOT EXISTS idx_fingerprint_fingerprinthub_created_at_resource_id ON fingerprint_fingerprinthub(created_at DESC, resource_id DESC)",
		"CREATE INDEX IF NOT EXISTS idx_fingerprint_fingerprinthub_name_resource_id ON fingerprint_fingerprinthub(name, resource_id)",
	} {
		if !strings.Contains(sql, required) {
			t.Fatalf("initial schema must contain approved fingerprint query index %q", required)
		}
	}

	if regexp.MustCompile(`(?m)^CREATE INDEX IF NOT EXISTS .* ON fingerprint_.* USING GIN \(payload`).MatchString(sql) {
		t.Fatal("initial fingerprint schema must not add speculative JSONB payload GIN indexes")
	}
}

func TestInitialSchemaExcludesHistoricalWorkflowCutoverArtifacts(t *testing.T) {
	sql := readInitialSchema(t)

	for _, artifact := range []string{
		"scan_workflow_migration_quarantine",
		"scheduled_scan_workflow_migration_quarantine",
		"Scan workflow singleton cutover",
		"column_name = 'workflow_ids'",
	} {
		if strings.Contains(sql, artifact) {
			t.Fatalf("initial schema baseline must not include historical workflow cutover artifact %q", artifact)
		}
	}
}

func TestInitialSchemaUsesCanonicalScanWorkflowColumns(t *testing.T) {
	sql := readInitialSchema(t)

	for _, required := range []string{
		"scan_workflow_id VARCHAR(100) NOT NULL",
		"stage_order INTEGER NOT NULL DEFAULT 0",
		"stage_id VARCHAR(100) NOT NULL",
		"step_id VARCHAR(100) NOT NULL",
		"engine_id VARCHAR(255) NOT NULL",
		"task_execution_config JSONB NOT NULL DEFAULT '{}'",
		"assigned_session_epoch BIGINT",
		"skip_reason VARCHAR(1000) NOT NULL DEFAULT ''",
	} {
		if !strings.Contains(sql, required) {
			t.Fatalf("initial schema must contain %q", required)
		}
	}

	for _, forbidden := range []string{
		strings.Join([]string{"workflow", "_id"}, "") + " VARCHAR(100) NOT NULL DEFAULT ''",
		strings.Join([]string{"workflow", "_config"}, "") + " JSONB NOT NULL DEFAULT '{}'",
		"stage           INT NOT NULL DEFAULT 0",
		"scan_workflow_stage_order",
	} {
		if strings.Contains(sql, forbidden) {
			t.Fatalf("initial schema must not retain legacy task/workflow column %q", forbidden)
		}
	}
}

func TestInitialSchemaDefinesScheduledScanExecutionLedger(t *testing.T) {
	sql := readInitialSchema(t)
	scheduleTable := regexp.MustCompile(`(?s)CREATE TABLE IF NOT EXISTS scheduled_scan \((.*?)\);`).FindStringSubmatch(sql)
	if len(scheduleTable) != 2 {
		t.Fatal("expected exactly one scheduled_scan table definition")
	}
	for _, required := range []string{
		"run_count INTEGER NOT NULL DEFAULT 0 CHECK (run_count >= 0)",
		"successful_handoff_count INTEGER NOT NULL DEFAULT 0 CHECK (successful_handoff_count >= 0)",
		"failed_handoff_count INTEGER NOT NULL DEFAULT 0 CHECK (failed_handoff_count >= 0)",
		"CONSTRAINT scheduled_scan_enabled_cursor_consistency CHECK",
	} {
		if !strings.Contains(scheduleTable[1], required) {
			t.Fatalf("scheduled_scan table must contain %q", required)
		}
	}

	occurrenceTable := regexp.MustCompile(`(?s)CREATE TABLE IF NOT EXISTS scheduled_scan_occurrence \((.*?)\);`).FindStringSubmatch(sql)
	if len(occurrenceTable) != 2 {
		t.Fatal("expected exactly one scheduled_scan_occurrence table definition")
	}
	for _, required := range []string{
		"scheduled_scan_id INTEGER NOT NULL REFERENCES scheduled_scan(id) ON DELETE CASCADE",
		"scheduled_for TIMESTAMPTZ NOT NULL",
		"attempted_at TIMESTAMPTZ",
		"dispatched_at TIMESTAMPTZ",
		"failure_kind VARCHAR(100)",
		"failure_message VARCHAR(2000)",
		"CONSTRAINT scheduled_scan_occurrence_identity UNIQUE (scheduled_scan_id, scheduled_for)",
		"CONSTRAINT scheduled_scan_occurrence_dispatch_requires_attempt CHECK",
		"CONSTRAINT scheduled_scan_occurrence_outcome_consistency CHECK",
	} {
		if !strings.Contains(occurrenceTable[1], required) {
			t.Fatalf("scheduled_scan_occurrence table must contain %q", required)
		}
	}
	for _, required := range []string{
		"CREATE INDEX IF NOT EXISTS idx_scheduled_scan_due ON scheduled_scan(next_run_time, id)",
		"CREATE INDEX IF NOT EXISTS idx_scheduled_scan_occurrence_candidate",
		"CREATE INDEX IF NOT EXISTS idx_scheduled_scan_occurrence_retention",
	} {
		if !strings.Contains(sql, required) {
			t.Fatalf("initial schema must contain %q", required)
		}
	}
	for _, forbidden := range []string{
		"schedule_id INTEGER REFERENCES scan",
		"occurrence_id INTEGER REFERENCES scan",
		"scheduled_scan_id INTEGER REFERENCES scan",
	} {
		if strings.Contains(sql, forbidden) {
			t.Fatalf("normal Scan schema must not gain scheduling provenance %q", forbidden)
		}
	}

	down := readInitialSchemaDown(t)
	if !strings.Contains(down, "DROP TABLE IF EXISTS scheduled_scan_occurrence CASCADE;") {
		t.Fatal("initial schema down migration must drop scheduled_scan_occurrence")
	}
	if strings.Index(down, "DROP TABLE IF EXISTS scheduled_scan_occurrence CASCADE;") > strings.Index(down, "DROP TABLE IF EXISTS scheduled_scan CASCADE;") {
		t.Fatal("occurrence teardown must precede scheduled_scan teardown")
	}
}

func TestInitialSchemaUsesScanTaskAsMVPWorkflowStepRecord(t *testing.T) {
	sql := readInitialSchema(t)

	for _, table := range []string{"workflow_execution", "workflow_step_execution", "engine_task_execution"} {
		if strings.Contains(sql, "CREATE TABLE IF NOT EXISTS "+table) {
			t.Fatalf("initial schema must not create %s in the MVP execution model", table)
		}
	}

	re := regexp.MustCompile(`(?s)CREATE TABLE IF NOT EXISTS scan_task \((.*?)\);`)
	matches := re.FindAllStringSubmatch(sql, -1)
	if len(matches) != 1 {
		t.Fatalf("expected one scan_task table definition, got %d", len(matches))
	}
	if !regexp.MustCompile(`(?m)^\s*status\s+`).MatchString(matches[0][1]) {
		t.Fatal("scan_task must carry the executable task status")
	}
	for _, required := range []string{
		"stage_order",
		"stage_id",
		"step_order",
		"step_id",
		"engine_id",
		"engine_config",
		"resolved_execution_plan",
		"terminal_reconciliation_pending",
	} {
		if !regexp.MustCompile(`(?m)^\s*` + required + `\s+`).MatchString(matches[0][1]) {
			t.Fatalf("scan_task must directly carry %s", required)
		}
	}
	if !regexp.MustCompile(`(?m)^\s*resolved_execution_plan\s+BYTEA\s+NOT NULL`).MatchString(matches[0][1]) {
		t.Fatal("scan_task resolved_execution_plan must persist immutable protobuf bytes")
	}
	if !regexp.MustCompile(`(?m)^\s*terminal_reconciliation_pending\s+BOOLEAN\s+NOT NULL\s+DEFAULT\s+FALSE`).MatchString(matches[0][1]) {
		t.Fatal("scan_task terminal reconciliation marker must be durable and default false")
	}
	for _, required := range []string{
		"CONSTRAINT scan_task_claim_tuple_shape CHECK",
		"assigned_agent_id IS NULL AND assigned_session_id IS NULL AND assigned_request_id IS NULL",
		"assigned_agent_id IS NOT NULL AND assigned_session_id IS NOT NULL AND assigned_session_epoch IS NOT NULL AND assigned_request_id IS NOT NULL",
		"CONSTRAINT scan_task_running_plan_claim_shape CHECK",
		"status <> 'running'",
		"OCTET_LENGTH(resolved_execution_plan) > 0",
	} {
		if !strings.Contains(matches[0][1], required) {
			t.Fatalf("scan_task claim shape must contain %q", required)
		}
	}
	if !strings.Contains(sql, "CREATE INDEX IF NOT EXISTS idx_scan_task_terminal_reconciliation_pending ON scan_task(id) WHERE terminal_reconciliation_pending = TRUE") {
		t.Fatal("scan_task terminal reconciliation marker must have a partial replay index")
	}
	if regexp.MustCompile(`(?m)^\s*scan_workflow_id\s+`).MatchString(matches[0][1]) {
		t.Fatal("scan_task must not duplicate scan.scan_workflow_id")
	}
	if regexp.MustCompile(`(?m)^\s*agent_id\s+`).MatchString(matches[0][1]) {
		t.Fatal("scan_task must not duplicate scan.agent_id")
	}
	if strings.Contains(matches[0][1], "OCTET_LENGTH(resolved_execution_plan) = 0") {
		t.Fatal("a running task must never be accepted without its persisted execution plan")
	}
}

func TestInitialSchemaDefinesScanPartitionedProgressLogs(t *testing.T) {
	sql := readInitialSchema(t)
	re := regexp.MustCompile(`(?s)CREATE TABLE IF NOT EXISTS task_progress_log \((.*?)\) PARTITION BY RANGE \(scan_id\);`)
	matches := re.FindAllStringSubmatch(sql, -1)
	if len(matches) != 1 {
		t.Fatalf("expected one task_progress_log table definition, got %d", len(matches))
	}

	for _, required := range []string{
		"CREATE INDEX IF NOT EXISTS idx_scan_task_scan ON scan_task(scan_id)",
		"CREATE UNIQUE INDEX IF NOT EXISTS unique_scan_task_scan_id_id ON scan_task(scan_id, id)",
		"scan_id INTEGER NOT NULL",
		"PRIMARY KEY (scan_id, id)",
		"FOREIGN KEY (scan_id, task_id) REFERENCES scan_task(scan_id, id) ON DELETE CASCADE",
		"CREATE INDEX IF NOT EXISTS idx_task_progress_log_task ON task_progress_log(scan_id, task_id, id)",
		"CREATE UNIQUE INDEX IF NOT EXISTS uniq_task_progress_log_request ON task_progress_log(scan_id, task_id, request_id, sequence)",
		"CREATE TABLE IF NOT EXISTS task_progress_log_p00000000 PARTITION OF task_progress_log",
		"CREATE TABLE IF NOT EXISTS task_progress_log_p00010000 PARTITION OF task_progress_log",
	} {
		if !strings.Contains(sql, required) {
			t.Fatalf("initial schema must define partitioned task progress log contract %q", required)
		}
	}
}

func TestInitialSchemaPartitionsAllScanHistoryParentsAtTenThousandIDs(t *testing.T) {
	sql := readInitialSchema(t)
	for _, table := range []string{
		"subdomain_snapshot",
		"website_snapshot",
		"endpoint_snapshot",
		"directory_snapshot",
		"host_port_mapping_snapshot",
		"screenshot_snapshot",
		"vulnerability_snapshot",
		"task_progress_log",
	} {
		parent := regexp.MustCompile(`(?s)CREATE TABLE IF NOT EXISTS ` + table + ` \((.*?)\) PARTITION BY RANGE \(scan_id\);`)
		matches := parent.FindAllStringSubmatch(sql, -1)
		if len(matches) != 1 {
			t.Fatalf("%s must be one RANGE(scan_id) parent, got %d definitions", table, len(matches))
		}
		if !strings.Contains(matches[0][1], "PRIMARY KEY (scan_id, id)") {
			t.Fatalf("%s must use scan_id in its primary key", table)
		}
		for _, start := range []string{"00000000", "00010000"} {
			required := "CREATE TABLE IF NOT EXISTS " + table + "_p" + start + " PARTITION OF " + table
			if !strings.Contains(sql, required) {
				t.Fatalf("%s must provision aligned initial partition %q", table, required)
			}
		}
	}
	if strings.Contains(sql, "DEFAULT PARTITION") {
		t.Fatal("scan history parents must not have a default partition")
	}
}

func TestInitialSchemaOmitsRetiredWorkerNode(t *testing.T) {
	up := readInitialSchema(t)
	down := readInitialSchemaDown(t)
	for _, statement := range []string{"CREATE TABLE IF NOT EXISTS worker_node", "DROP TABLE IF EXISTS worker_node"} {
		if strings.Contains(up, statement) || strings.Contains(down, statement) {
			t.Fatalf("fresh-install migration retained retired Worker node schema %q", statement)
		}
	}
}

func TestInitialSchemaIndexesWordlistTagsAsJSONB(t *testing.T) {
	sql := readInitialSchema(t)

	if !strings.Contains(sql, "tags JSONB NOT NULL DEFAULT '[]'::jsonb") {
		t.Fatal("wordlist.tags must remain a JSONB string-array metadata column")
	}
	if !strings.Contains(sql, "CREATE INDEX IF NOT EXISTS idx_wordlist_tags_gin ON wordlist USING GIN (tags)") {
		t.Fatal("wordlist.tags must have a GIN index for tag filtering and summaries")
	}
}

func TestInitialSchemaIndexesWordlistSortableFields(t *testing.T) {
	sql := readInitialSchema(t)

	for _, required := range []string{
		"CREATE UNIQUE INDEX IF NOT EXISTS unique_wordlist_file_name ON wordlist(file_name)",
		"CREATE INDEX IF NOT EXISTS idx_wordlist_updated_at_id ON wordlist(updated_at, id)",
		"CREATE INDEX IF NOT EXISTS idx_wordlist_line_count_id ON wordlist(line_count, id)",
		"CREATE INDEX IF NOT EXISTS idx_wordlist_file_size_id ON wordlist(file_size, id)",
	} {
		if !strings.Contains(sql, required) {
			t.Fatalf("initial schema must index wordlist sortable field with stable id tie-breaker: %q", required)
		}
	}
}

func TestInitialSchemaIndexesTargetListQueryFields(t *testing.T) {
	sql := readInitialSchema(t)

	for _, required := range []string{
		"CREATE EXTENSION IF NOT EXISTS pg_trgm",
		"CREATE INDEX IF NOT EXISTS idx_target_name_trgm_active ON target USING GIN (name gin_trgm_ops) WHERE deleted_at IS NULL",
		"CREATE INDEX IF NOT EXISTS idx_target_name_id_active ON target(name, id) WHERE deleted_at IS NULL",
		"CREATE INDEX IF NOT EXISTS idx_target_created_at_id_active ON target(created_at, id) WHERE deleted_at IS NULL",
		"CREATE INDEX IF NOT EXISTS idx_target_last_scanned_at_desc_id_active ON target(last_scanned_at DESC NULLS LAST, id DESC) WHERE deleted_at IS NULL",
		"CREATE INDEX IF NOT EXISTS idx_target_last_scanned_at_asc_id_active ON target(last_scanned_at ASC NULLS LAST, id ASC) WHERE deleted_at IS NULL",
		"CREATE INDEX IF NOT EXISTS idx_target_type_created_at_id_active ON target(type, created_at, id) WHERE deleted_at IS NULL",
	} {
		if !strings.Contains(sql, required) {
			t.Fatalf("initial schema must index target list query field: %q", required)
		}
	}
}

func TestInitialSchemaIndexesOrganizationListQueryFields(t *testing.T) {
	sql := readInitialSchema(t)

	for _, required := range []string{
		"CREATE EXTENSION IF NOT EXISTS pg_trgm",
		"CREATE INDEX IF NOT EXISTS idx_org_name_trgm_active ON organization USING GIN (name gin_trgm_ops) WHERE deleted_at IS NULL",
		"CREATE INDEX IF NOT EXISTS idx_org_name_id_active ON organization(name, id) WHERE deleted_at IS NULL",
		"CREATE INDEX IF NOT EXISTS idx_org_created_at_id_active ON organization(created_at, id) WHERE deleted_at IS NULL",
		"CREATE UNIQUE INDEX IF NOT EXISTS idx_org_active_name_unique ON organization(name) WHERE deleted_at IS NULL",
	} {
		if !strings.Contains(sql, required) {
			t.Fatalf("initial schema must index organization list query field: %q", required)
		}
	}
}

func TestInitialSchemaIndexesTargetSubdomainListQueryFields(t *testing.T) {
	sql := readInitialSchema(t)

	for _, required := range []string{
		"CREATE EXTENSION IF NOT EXISTS pg_trgm",
		"CREATE INDEX IF NOT EXISTS idx_subdomain_dns_name_trgm ON subdomain USING GIN (dns_name gin_trgm_ops)",
		"CREATE INDEX IF NOT EXISTS idx_subdomain_target_created_at_id ON subdomain(target_id, created_at, id)",
		"CREATE INDEX IF NOT EXISTS idx_subdomain_target_dns_name_id ON subdomain(target_id, dns_name, id)",
	} {
		if !strings.Contains(sql, required) {
			t.Fatalf("initial schema must index target subdomain list query field: %q", required)
		}
	}
}

func TestInitialSchemaIndexesHostPortIPAggregateListQueryFields(t *testing.T) {
	sql := readInitialSchema(t)

	for _, required := range []string{
		"CREATE INDEX IF NOT EXISTS idx_hpm_target_port_ip ON host_port_mapping(target_id, port, ip)",
		"CREATE INDEX IF NOT EXISTS idx_hpm_target_ip ON host_port_mapping(target_id, ip)",
		"CREATE INDEX IF NOT EXISTS idx_hpm_target_host_ip ON host_port_mapping(target_id, host, ip)",
		"CREATE INDEX IF NOT EXISTS idx_hpm_host_trgm ON host_port_mapping USING GIN (host gin_trgm_ops)",
		"CREATE INDEX IF NOT EXISTS idx_hpm_target_created_at_ip ON host_port_mapping(target_id, created_at, ip)",
		"CREATE INDEX IF NOT EXISTS idx_hpm_snap_scan_port_ip ON host_port_mapping_snapshot(scan_id, port, ip)",
		"CREATE INDEX IF NOT EXISTS idx_hpm_snap_scan_ip ON host_port_mapping_snapshot(scan_id, ip)",
		"CREATE INDEX IF NOT EXISTS idx_hpm_snap_scan_host_ip ON host_port_mapping_snapshot(scan_id, host, ip)",
		"CREATE INDEX IF NOT EXISTS idx_hpm_snap_host_trgm ON host_port_mapping_snapshot USING GIN (host gin_trgm_ops)",
		"CREATE INDEX IF NOT EXISTS idx_hpm_snap_scan_created_at_ip ON host_port_mapping_snapshot(scan_id, created_at, ip)",
		"CREATE INDEX IF NOT EXISTS idx_hpm_snap_scan_effective_host_port ON host_port_mapping_snapshot(scan_id, (COALESCE(NULLIF(LOWER(BTRIM(host)), ''), HOST(ip))), port)",
		"CREATE UNIQUE INDEX IF NOT EXISTS unique_scan_host_ip_port_snapshot ON host_port_mapping_snapshot(scan_id, host, ip, port)",
	} {
		if !strings.Contains(sql, required) {
			t.Fatalf("initial schema must index host-port IP aggregate list query field: %q", required)
		}
	}
}

func TestInitialSchemaEnforcesCompleteHostPortFacts(t *testing.T) {
	sql := readInitialSchema(t)
	tablePatterns := map[string]string{
		"host_port_mapping":          `(?s)CREATE TABLE IF NOT EXISTS host_port_mapping \((.*?)\);`,
		"host_port_mapping_snapshot": `(?s)CREATE TABLE IF NOT EXISTS host_port_mapping_snapshot \((.*?)\) PARTITION BY RANGE \(scan_id\);`,
	}
	for table, pattern := range tablePatterns {
		matches := regexp.MustCompile(pattern).FindStringSubmatch(sql)
		if len(matches) != 2 {
			t.Fatalf("expected exactly one %s table definition", table)
		}
		for _, required := range []string{
			"host VARCHAR(1000) NOT NULL",
			"ip INET NOT NULL",
			"port INTEGER NOT NULL",
			"CONSTRAINT " + table + "_host_nonblank CHECK (BTRIM(host) <> '')",
			"CONSTRAINT " + table + "_ip_ipv4_host CHECK (family(ip) = 4 AND masklen(ip) = 32)",
			"CONSTRAINT " + table + "_port_range CHECK (port BETWEEN 1 AND 65535)",
		} {
			if !strings.Contains(matches[1], required) {
				t.Fatalf("%s baseline missing %q", table, required)
			}
		}
	}
}

func TestInitialSchemaIndexesScanSubdomainListQueryFields(t *testing.T) {
	sql := readInitialSchema(t)

	for _, required := range []string{
		"CREATE EXTENSION IF NOT EXISTS pg_trgm",
		"CREATE INDEX IF NOT EXISTS idx_subdomain_snap_dns_name_trgm ON subdomain_snapshot USING GIN (dns_name gin_trgm_ops)",
		"CREATE INDEX IF NOT EXISTS idx_subdomain_snap_scan_created_at_id ON subdomain_snapshot(scan_id, created_at, id)",
		"CREATE INDEX IF NOT EXISTS idx_subdomain_snap_scan_dns_name_id ON subdomain_snapshot(scan_id, dns_name, id)",
	} {
		if !strings.Contains(sql, required) {
			t.Fatalf("initial schema must index scan subdomain list query field: %q", required)
		}
	}
}

func TestInitialSchemaIndexesWebsiteListQueryFields(t *testing.T) {
	sql := readInitialSchema(t)

	for _, required := range []string{
		"CREATE EXTENSION IF NOT EXISTS pg_trgm",
		"CREATE INDEX IF NOT EXISTS idx_website_target_created_at_id ON website(target_id, created_at, id)",
		"CREATE INDEX IF NOT EXISTS idx_website_url_trgm ON website USING GIN (url gin_trgm_ops)",
		"CREATE INDEX IF NOT EXISTS idx_website_host_trgm ON website USING GIN (host gin_trgm_ops)",
		"CREATE INDEX IF NOT EXISTS idx_website_title_trgm ON website USING GIN (title gin_trgm_ops)",
		"CREATE INDEX IF NOT EXISTS idx_website_target_status_code_id ON website(target_id, status_code, id)",
		"CREATE INDEX IF NOT EXISTS idx_website_target_content_length_id ON website(target_id, content_length, id)",
		"CREATE INDEX IF NOT EXISTS idx_website_target_webserver_id ON website(target_id, webserver, id)",
		"CREATE INDEX IF NOT EXISTS idx_website_target_content_type_id ON website(target_id, content_type, id)",
		"CREATE INDEX IF NOT EXISTS idx_website_target_vhost_id ON website(target_id, vhost, id)",
		"CREATE INDEX IF NOT EXISTS idx_website_tech_gin ON website USING GIN (tech)",
		"CREATE INDEX IF NOT EXISTS idx_website_snap_scan_created_at_id ON website_snapshot(scan_id, created_at, id)",
		"CREATE INDEX IF NOT EXISTS idx_website_snap_url_trgm ON website_snapshot USING GIN (url gin_trgm_ops)",
		"CREATE INDEX IF NOT EXISTS idx_website_snap_scan_status_code_id ON website_snapshot(scan_id, status_code, id)",
		"CREATE INDEX IF NOT EXISTS idx_website_snap_scan_content_length_id ON website_snapshot(scan_id, content_length, id)",
		"CREATE INDEX IF NOT EXISTS idx_website_snap_scan_webserver_id ON website_snapshot(scan_id, webserver, id)",
		"CREATE INDEX IF NOT EXISTS idx_website_snap_scan_content_type_id ON website_snapshot(scan_id, content_type, id)",
		"CREATE INDEX IF NOT EXISTS idx_website_snap_scan_vhost_id ON website_snapshot(scan_id, vhost, id)",
		"CREATE INDEX IF NOT EXISTS idx_website_snap_tech_gin ON website_snapshot USING GIN (tech)",
	} {
		if !strings.Contains(sql, required) {
			t.Fatalf("initial schema must index website list query field: %q", required)
		}
	}
}

func TestInitialSchemaKeepsWebsiteTechnologyStorageAtVarchar100Arrays(t *testing.T) {
	sql := readInitialSchema(t)
	for _, table := range []string{"website", "website_snapshot"} {
		re := regexp.MustCompile(`(?s)CREATE TABLE IF NOT EXISTS ` + table + ` \((.*?)\);`)
		matches := re.FindAllStringSubmatch(sql, -1)
		if len(matches) != 1 {
			t.Fatalf("expected one %s table definition, got %d", table, len(matches))
		}
		if !regexp.MustCompile(`(?m)^\s*tech\s+VARCHAR\(100\)\[\]\s+NOT NULL\s+DEFAULT '\{\}'`).MatchString(matches[0][1]) {
			t.Fatalf("%s.tech must remain varchar(100)[] with an explicit empty-array default", table)
		}
	}
}

func TestInitialSchemaIndexesAgentListQueryFields(t *testing.T) {
	sql := readInitialSchema(t)

	for _, required := range []string{
		"CREATE EXTENSION IF NOT EXISTS pg_trgm",
		"CREATE INDEX IF NOT EXISTS idx_agent_display_name_trgm ON agent USING GIN (display_name gin_trgm_ops)",
		"CREATE INDEX IF NOT EXISTS idx_agent_created_at_id ON agent(created_at, id)",
		"CREATE INDEX IF NOT EXISTS idx_agent_status ON agent(status)",
		"CREATE INDEX IF NOT EXISTS idx_agent_runtime_status_observed_hostname_trgm ON agent_runtime_status USING GIN (observed_hostname gin_trgm_ops)",
		"CREATE INDEX IF NOT EXISTS idx_agent_runtime_status_observed_source_ip_trgm ON agent_runtime_status USING GIN (observed_source_ip gin_trgm_ops)",
		"CREATE INDEX IF NOT EXISTS idx_agent_runtime_status_health_state_agent_id ON agent_runtime_status(health_state, agent_id)",
	} {
		if !strings.Contains(sql, required) {
			t.Fatalf("initial schema must index agent list query field: %q", required)
		}
	}
}

func TestInitialSchemaDefinesAgentOperationalDataPersistence(t *testing.T) {
	sql := readInitialSchema(t)
	down := readInitialSchemaDown(t)

	registrationToken := regexp.MustCompile(`(?s)CREATE TABLE IF NOT EXISTS registration_token \((.*?)\);`).FindStringSubmatch(sql)
	agent := regexp.MustCompile(`(?s)CREATE TABLE IF NOT EXISTS agent \((.*?)\);`).FindStringSubmatch(sql)
	runtimeStatus := regexp.MustCompile(`(?s)CREATE TABLE IF NOT EXISTS agent_runtime_status \((.*?)\);`).FindStringSubmatch(sql)
	agentLocation := regexp.MustCompile(`(?s)CREATE TABLE IF NOT EXISTS agent_location \((.*?)\);`).FindStringSubmatch(sql)
	serverLocation := regexp.MustCompile(`(?s)CREATE TABLE IF NOT EXISTS server_location_snapshot \((.*?)\);`).FindStringSubmatch(sql)
	for name, match := range map[string][]string{
		"registration_token":       registrationToken,
		"agent":                    agent,
		"agent_runtime_status":     runtimeStatus,
		"agent_location":           agentLocation,
		"server_location_snapshot": serverLocation,
	} {
		if len(match) != 2 {
			t.Fatalf("expected exactly one %s table definition", name)
		}
	}

	for _, required := range []string{"ever_attributed_at  TIMESTAMPTZ", "expires_at          TIMESTAMPTZ NOT NULL"} {
		if !strings.Contains(registrationToken[1], required) {
			t.Fatalf("registration_token must contain %q", required)
		}
	}
	if !strings.Contains(agent[1], "registration_token_id  INT NOT NULL REFERENCES registration_token(id) ON DELETE RESTRICT") {
		t.Fatal("agent must require the non-secret registration token foreign key")
	}
	if strings.Contains(agent[1], "registration_token  VARCHAR") {
		t.Fatal("agent must not copy the registration bearer token")
	}
	for _, required := range []string{
		"observed_source_ip",
		"observed_ip_generation BIGINT NOT NULL DEFAULT 0",
		"CHECK (observed_ip_generation >= 0)",
	} {
		if !strings.Contains(runtimeStatus[1], required) {
			t.Fatalf("agent_runtime_status must contain %q", required)
		}
	}
	if regexp.MustCompile(`(?m)^\s*ip_address\s`).MatchString(runtimeStatus[1]) {
		t.Fatal("agent_runtime_status must not retain the ambiguous ip_address column")
	}
	for _, table := range []struct {
		name string
		body string
	}{
		{name: "agent_location", body: agentLocation[1]},
		{name: "server_location_snapshot", body: serverLocation[1]},
	} {
		for _, required := range []string{"latitude", "longitude", "accuracy_radius_km", "provider_key", "resolved_at", "forced_expired"} {
			if !strings.Contains(table.body, required) {
				t.Fatalf("%s must contain %q", table.name, required)
			}
		}
	}
	if !strings.Contains(serverLocation[1], "CHECK (singleton_id = 1)") {
		t.Fatal("server_location_snapshot must enforce the single-Server singleton")
	}

	agentDrop := strings.Index(down, "DROP TABLE IF EXISTS agent CASCADE;")
	tokenDrop := strings.Index(down, "DROP TABLE IF EXISTS registration_token CASCADE;")
	if agentDrop < 0 || tokenDrop < 0 || agentDrop > tokenDrop {
		t.Fatal("down migration must drop agent before its registration_token parent")
	}
}

func TestInitialSchemaIndexesEndpointListQueryFields(t *testing.T) {
	sql := readInitialSchema(t)

	for _, required := range []string{
		"CREATE EXTENSION IF NOT EXISTS pg_trgm",
		"CREATE INDEX IF NOT EXISTS idx_endpoint_target_created_at_id ON endpoint(target_id, created_at, id)",
		"CREATE INDEX IF NOT EXISTS idx_endpoint_url_trgm ON endpoint USING GIN (url gin_trgm_ops)",
		"CREATE INDEX IF NOT EXISTS idx_endpoint_host_trgm ON endpoint USING GIN (host gin_trgm_ops)",
		"CREATE INDEX IF NOT EXISTS idx_endpoint_title_trgm ON endpoint USING GIN (title gin_trgm_ops)",
		"CREATE INDEX IF NOT EXISTS idx_endpoint_target_status_code_id ON endpoint(target_id, status_code, id)",
		"CREATE INDEX IF NOT EXISTS idx_endpoint_target_content_length_id ON endpoint(target_id, content_length, id)",
		"CREATE INDEX IF NOT EXISTS idx_endpoint_target_webserver_id ON endpoint(target_id, webserver, id)",
		"CREATE INDEX IF NOT EXISTS idx_endpoint_target_content_type_id ON endpoint(target_id, content_type, id)",
		"CREATE INDEX IF NOT EXISTS idx_endpoint_target_vhost_id ON endpoint(target_id, vhost, id)",
		"CREATE INDEX IF NOT EXISTS idx_endpoint_tech_gin ON endpoint USING GIN (tech)",
		"CREATE INDEX IF NOT EXISTS idx_endpoint_snap_scan_created_at_id ON endpoint_snapshot(scan_id, created_at, id)",
		"CREATE INDEX IF NOT EXISTS idx_endpoint_snap_url_trgm ON endpoint_snapshot USING GIN (url gin_trgm_ops)",
		"CREATE INDEX IF NOT EXISTS idx_endpoint_snap_scan_status_code_id ON endpoint_snapshot(scan_id, status_code, id)",
		"CREATE INDEX IF NOT EXISTS idx_endpoint_snap_scan_content_length_id ON endpoint_snapshot(scan_id, content_length, id)",
		"CREATE INDEX IF NOT EXISTS idx_endpoint_snap_scan_webserver_id ON endpoint_snapshot(scan_id, webserver, id)",
		"CREATE INDEX IF NOT EXISTS idx_endpoint_snap_scan_content_type_id ON endpoint_snapshot(scan_id, content_type, id)",
		"CREATE INDEX IF NOT EXISTS idx_endpoint_snap_scan_vhost_id ON endpoint_snapshot(scan_id, vhost, id)",
		"CREATE INDEX IF NOT EXISTS idx_endpoint_snap_tech_gin ON endpoint_snapshot USING GIN (tech)",
	} {
		if !strings.Contains(sql, required) {
			t.Fatalf("initial schema must index endpoint list query field: %q", required)
		}
	}
}

func TestInitialSchemaGlobalAssetSearchKeepsSearchProjectionOutOfBaseline(t *testing.T) {
	sql := readInitialSchema(t)

	// The search feature is intentionally read-only over the existing current-state
	// tables. Keep this column inventory fixed so a later search-only field or
	// projection table cannot silently become part of the baseline schema.
	assertInitialSchemaTableColumns(t, sql, "website", []string{
		"id", "target_id", "url", "host", "location", "created_at", "title", "webserver",
		"response_body", "response_body_truncated", "content_type", "tech", "status_code",
		"content_length", "vhost", "response_headers",
	})
	assertInitialSchemaTableColumns(t, sql, "endpoint", []string{
		"id", "target_id", "url", "host", "location", "created_at", "title", "webserver",
		"response_body", "response_body_truncated", "content_type", "tech", "status_code",
		"content_length", "vhost", "response_headers", "response_headers_truncated",
	})

	if regexp.MustCompile(`(?im)^\s*CREATE\s+TABLE\b[^;]*\bglobal[_ ]asset[_ ]search\b`).MatchString(sql) {
		t.Fatal("global asset search must not add a projection table")
	}
	if regexp.MustCompile(`(?im)^\s*ALTER\s+TABLE\s+(website|endpoint)\s+ADD\s+COLUMN\b`).MatchString(sql) {
		t.Fatal("global asset search must not add Website or Endpoint fields")
	}

	expected := []string{
		"CREATE INDEX IF NOT EXISTS idx_website_host_trgm ON website USING GIN (host gin_trgm_ops)",
		"CREATE INDEX IF NOT EXISTS idx_website_title_trgm ON website USING GIN (title gin_trgm_ops)",
		"CREATE INDEX IF NOT EXISTS idx_endpoint_host_trgm ON endpoint USING GIN (host gin_trgm_ops)",
		"CREATE INDEX IF NOT EXISTS idx_endpoint_title_trgm ON endpoint USING GIN (title gin_trgm_ops)",
	}
	for _, statement := range expected {
		if strings.Count(sql, statement) != 1 {
			t.Fatalf("global asset search index must appear exactly once: %q", statement)
		}
	}

	globalSearchIndexes := regexp.MustCompile(`(?m)^CREATE INDEX IF NOT EXISTS idx_(website|endpoint)_(host|title)_trgm ON (website|endpoint) USING GIN \((host|title) gin_trgm_ops\);$`).FindAllString(sql, -1)
	if len(globalSearchIndexes) != 4 {
		t.Fatalf("global asset search must add exactly four host/title trigram indexes, got %d: %v", len(globalSearchIndexes), globalSearchIndexes)
	}

	for _, statement := range []string{
		"CREATE INDEX IF NOT EXISTS idx_website_host ON website(host)",
		"CREATE INDEX IF NOT EXISTS idx_website_title ON website(title)",
		"CREATE INDEX IF NOT EXISTS idx_endpoint_host ON endpoint(host)",
		"CREATE INDEX IF NOT EXISTS idx_endpoint_title ON endpoint(title)",
		"CREATE INDEX IF NOT EXISTS idx_website_url_trgm ON website USING GIN (url gin_trgm_ops)",
		"CREATE INDEX IF NOT EXISTS idx_endpoint_url_trgm ON endpoint USING GIN (url gin_trgm_ops)",
		"CREATE INDEX IF NOT EXISTS idx_website_tech_gin ON website USING GIN (tech)",
		"CREATE INDEX IF NOT EXISTS idx_endpoint_tech_gin ON endpoint USING GIN (tech)",
	} {
		if !strings.Contains(sql, statement) {
			t.Fatalf("global asset search must preserve existing index %q", statement)
		}
	}

	assetGINIndexes := regexp.MustCompile(`(?m)^CREATE INDEX IF NOT EXISTS \S+ ON (website|endpoint) USING GIN \(([^)]*)\);$`).FindAllStringSubmatch(sql, -1)
	for _, index := range assetGINIndexes {
		if strings.Contains(index[2], "response_body") || strings.Contains(index[2], "response_headers") {
			t.Fatalf("global asset search must not index response evidence: %q", index[0])
		}
	}
}

func TestInitialSchemaDefinesNotificationHardCut(t *testing.T) {
	up := readInitialSchema(t)
	down := readInitialSchemaDown(t)

	inboxKindConstraint := "kind VARCHAR(64) NOT NULL CHECK (kind IN (" + notificationKindSQLList(notificationdomain.InboxSupportedKinds()) + "))"
	for _, table := range []string{"notification_outbox", "notification_fact", "notification_inbox"} {
		definition := regexp.MustCompile(`(?s)CREATE TABLE IF NOT EXISTS ` + regexp.QuoteMeta(table) + ` \((.*?)\);`).FindStringSubmatch(up)
		if len(definition) != 2 || !strings.Contains(definition[1], inboxKindConstraint) {
			t.Fatalf("%s must accept the complete inbox kind vocabulary", table)
		}
	}
	destinationKindConstraint := "kind VARCHAR(64) NOT NULL CHECK (kind IN (" + notificationKindSQLList(notificationdomain.ExternallyDeliverableKinds()) + "))"
	destination := regexp.MustCompile(`(?s)CREATE TABLE IF NOT EXISTS notification_destination_subscription \((.*?)\);`).FindStringSubmatch(up)
	if len(destination) != 2 || !strings.Contains(destination[1], destinationKindConstraint) {
		t.Fatal("notification destination subscriptions must keep Nuclei terminal kinds inbox-only")
	}

	for _, table := range []string{
		"notification_outbox",
		"notification_fact",
		"notification_recipient",
		"notification_inbox",
		"notification_destination",
		"notification_destination_subscription",
		"notification_delivery",
		"notification_delivery_attempt",
	} {
		definitions := regexp.MustCompile(`(?m)^CREATE TABLE IF NOT EXISTS `+regexp.QuoteMeta(table)+` \(`).FindAllString(up, -1)
		if len(definitions) != 1 {
			t.Fatalf("notification baseline must define %s exactly once, got %d definitions", table, len(definitions))
		}
		if !strings.Contains(down, "DROP TABLE IF EXISTS "+table+" CASCADE;") {
			t.Fatalf("notification baseline down migration must drop %s", table)
		}
	}

	for _, forbidden := range []string{
		`(?m)^CREATE TABLE IF NOT EXISTS notification \(`,
		`(?m)^CREATE TABLE IF NOT EXISTS notification_settings \(`,
		`(?m)^CREATE TABLE IF NOT EXISTS notification_preference \(`,
		`(?m)^ALTER TABLE notification\b`,
		`(?m)^ALTER TABLE notification_settings\b`,
		`\binbox_enabled\b`,
		`(?s)INSERT INTO notification_destination.*?SELECT`,
	} {
		if regexp.MustCompile(forbidden).MatchString(up) {
			t.Fatalf("notification hard cut must not retain or import legacy state: %q", forbidden)
		}
	}

	for _, required := range []string{
		"locale VARCHAR(2) NOT NULL DEFAULT 'en' CHECK (locale IN ('zh', 'en'))",
		"locale_initialized_at TIMESTAMPTZ",
		"event_id VARCHAR(255) NOT NULL UNIQUE",
		"lease_owner VARCHAR(128)",
		"lease_expires_at TIMESTAMPTZ",
		"UNIQUE (fact_id, user_id)",
		"UNIQUE (event_id, destination_id)",
		"UNIQUE (delivery_id, attempt_number)",
		"CREATE INDEX IF NOT EXISTS idx_notification_outbox_claim",
		"CREATE INDEX IF NOT EXISTS idx_notification_outbox_published_retention",
		"CREATE INDEX IF NOT EXISTS idx_notification_inbox_user_visibility",
		"CREATE INDEX IF NOT EXISTS idx_notification_inbox_user_unread",
		"CREATE INDEX IF NOT EXISTS idx_notification_delivery_claim",
		"CREATE INDEX IF NOT EXISTS idx_notification_delivery_terminal_retention",
		"provider VARCHAR(16) NOT NULL UNIQUE CHECK (provider IN ('discord', 'wecom', 'feishu'))",
		"provider VARCHAR(16) NOT NULL CHECK (provider IN ('discord', 'wecom', 'feishu'))",
		"VALUES ('discord', '', FALSE), ('wecom', '', FALSE), ('feishu', '', FALSE)",
	} {
		if !strings.Contains(up, required) {
			t.Fatalf("notification hard-cut baseline must contain %q", required)
		}
	}

	assertInitialSchemaTableColumns(t, up, "notification_recipient", []string{
		"id", "fact_id", "user_id", "locale", "category", "created_at",
	})
	if strings.Contains(down, "notification_preference") {
		t.Fatal("notification hard cut must not retain a preference teardown path")
	}

	if strings.Index(down, "DROP TABLE IF EXISTS notification_delivery_attempt CASCADE;") > strings.Index(down, "DROP TABLE IF EXISTS notification_delivery CASCADE;") {
		t.Fatal("notification down migration must drop delivery attempts before deliveries")
	}
	if strings.Index(down, "DROP TABLE IF EXISTS notification_delivery CASCADE;") > strings.Index(down, "DROP TABLE IF EXISTS notification_destination CASCADE;") {
		t.Fatal("notification down migration must drop deliveries before destinations")
	}
	if strings.Index(down, "DROP TABLE IF EXISTS notification_inbox CASCADE;") > strings.Index(down, "DROP TABLE IF EXISTS notification_fact CASCADE;") {
		t.Fatal("notification down migration must drop inbox projections before facts")
	}
}

func TestInitialSchemaDefaultsNucleiPOCEnablementToDisabled(t *testing.T) {
	up := readInitialSchema(t)
	definition := regexp.MustCompile(`(?s)CREATE TABLE IF NOT EXISTS nuclei_poc \((.*?)\);`).FindStringSubmatch(up)
	if len(definition) != 2 {
		t.Fatal("initial schema must define nuclei_poc exactly once")
	}
	if !strings.Contains(definition[1], "is_enabled BOOLEAN NOT NULL DEFAULT FALSE") {
		t.Fatal("nuclei_poc.is_enabled must default to FALSE in the disposable development baseline")
	}
	if strings.Contains(definition[1], "is_enabled BOOLEAN NOT NULL DEFAULT TRUE") {
		t.Fatal("nuclei_poc.is_enabled must not default to TRUE")
	}
	if !strings.Contains(up, "existing development rows are not backfilled") {
		t.Fatal("initial schema must document rebuild-only enablement default rollout")
	}
}

func notificationKindSQLList(kinds []notificationdomain.Kind) string {
	quoted := make([]string, 0, len(kinds))
	for _, kind := range kinds {
		quoted = append(quoted, "'"+strings.ReplaceAll(string(kind), "'", "''")+"'")
	}
	return strings.Join(quoted, ", ")
}

func TestInitialSchemaDefinesStrictScanTriggerType(t *testing.T) {
	definition := regexp.MustCompile(`(?s)CREATE TABLE IF NOT EXISTS scan \((.*?)\);`).FindStringSubmatch(readInitialSchema(t))
	if len(definition) != 2 {
		t.Fatal("expected exactly one scan table definition")
	}
	if regexp.MustCompile(`(?m)^\s*scan_mode\s+`).MatchString(definition[1]) {
		t.Fatal("scan baseline must not retain the legacy scan_mode column")
	}
	triggerLine := regexp.MustCompile(`(?m)^\s*trigger_type\s+.*$`).FindString(definition[1])
	if !regexp.MustCompile(`(?m)^\s*trigger_type\s+VARCHAR\(16\)\s+NOT NULL\s+CHECK \(trigger_type IN \('manual', 'scheduled', 'ai'\)\),?$`).MatchString(triggerLine) {
		t.Fatalf("scan trigger type baseline is not the closed non-null vocabulary: %q", triggerLine)
	}
	for _, forbidden := range []string{"DEFAULT", "legacy", "unknown"} {
		if strings.Contains(triggerLine, forbidden) {
			t.Fatalf("scan trigger type baseline must not include %q: %q", forbidden, triggerLine)
		}
	}
}

func assertInitialSchemaTableColumns(t *testing.T, sql, table string, want []string) {
	t.Helper()

	definition := regexp.MustCompile(`(?s)CREATE TABLE IF NOT EXISTS ` + regexp.QuoteMeta(table) + ` \((.*?)\);`).FindStringSubmatch(sql)
	if len(definition) != 2 {
		t.Fatalf("expected exactly one %s table definition", table)
	}

	columns := make([]string, 0, len(want))
	for _, line := range strings.Split(definition[1], "\n") {
		field := regexp.MustCompile(`^\s*([a-z][a-z0-9_]*)\s+`).FindStringSubmatch(line)
		if len(field) == 2 {
			columns = append(columns, field[1])
		}
	}
	if !slices.Equal(columns, want) {
		t.Fatalf("%s columns changed: got %v want %v", table, columns, want)
	}
}

func TestInitialSchemaIndexesDirectoryListQueryFields(t *testing.T) {
	sql := readInitialSchema(t)

	for _, required := range []string{
		"CREATE EXTENSION IF NOT EXISTS pg_trgm",
		"CREATE INDEX IF NOT EXISTS idx_directory_target_created_at_id ON directory(target_id, created_at, id)",
		"CREATE INDEX IF NOT EXISTS idx_directory_url_trgm ON directory USING GIN (url gin_trgm_ops)",
		"CREATE INDEX IF NOT EXISTS idx_directory_target_status_id ON directory(target_id, status, id)",
		"CREATE INDEX IF NOT EXISTS idx_directory_target_content_length_id ON directory(target_id, content_length, id)",
		"CREATE INDEX IF NOT EXISTS idx_directory_target_content_type_id ON directory(target_id, content_type, id)",
		"CREATE INDEX IF NOT EXISTS idx_directory_snap_scan_created_at_id ON directory_snapshot(scan_id, created_at, id)",
		"CREATE INDEX IF NOT EXISTS idx_directory_snap_url_trgm ON directory_snapshot USING GIN (url gin_trgm_ops)",
		"CREATE INDEX IF NOT EXISTS idx_directory_snap_scan_status_id ON directory_snapshot(scan_id, status, id)",
		"CREATE INDEX IF NOT EXISTS idx_directory_snap_scan_content_length_id ON directory_snapshot(scan_id, content_length, id)",
		"CREATE INDEX IF NOT EXISTS idx_directory_snap_scan_content_type_id ON directory_snapshot(scan_id, content_type, id)",
	} {
		if !strings.Contains(sql, required) {
			t.Fatalf("initial schema must index directory list query field: %q", required)
		}
	}
}

func TestInitialSchemaIndexesScreenshotListQueryFields(t *testing.T) {
	sql := readInitialSchema(t)

	for _, required := range []string{
		"CREATE EXTENSION IF NOT EXISTS pg_trgm",
		"CREATE INDEX IF NOT EXISTS idx_screenshot_target_created_at_id ON screenshot(target_id, created_at, id)",
		"CREATE INDEX IF NOT EXISTS idx_screenshot_url_trgm ON screenshot USING GIN (url gin_trgm_ops)",
		"CREATE INDEX IF NOT EXISTS idx_screenshot_target_status_code_id ON screenshot(target_id, status_code, id)",
		"CREATE INDEX IF NOT EXISTS idx_screenshot_snap_scan_created_at_id ON screenshot_snapshot(scan_id, created_at, id)",
		"CREATE INDEX IF NOT EXISTS idx_screenshot_snap_url_trgm ON screenshot_snapshot USING GIN (url gin_trgm_ops)",
		"CREATE INDEX IF NOT EXISTS idx_screenshot_snap_scan_status_code_id ON screenshot_snapshot(scan_id, status_code, id)",
	} {
		if !strings.Contains(sql, required) {
			t.Fatalf("initial schema must index screenshot list query field: %q", required)
		}
	}
}

func TestInitialSchemaIndexesVulnerabilityListQueryFields(t *testing.T) {
	sql := readInitialSchema(t)

	for _, required := range []string{
		"CREATE EXTENSION IF NOT EXISTS pg_trgm",
		"CREATE INDEX IF NOT EXISTS idx_vuln_url_trgm ON vulnerability USING GIN (url gin_trgm_ops)",
		"CREATE INDEX IF NOT EXISTS idx_vuln_created_at_id ON vulnerability(created_at, id)",
		"CREATE INDEX IF NOT EXISTS idx_vuln_target_created_at_id ON vulnerability(target_id, created_at, id)",
		"CREATE INDEX IF NOT EXISTS idx_vuln_target_severity_id ON vulnerability(target_id, severity, id)",
		"CREATE INDEX IF NOT EXISTS idx_vuln_target_source_id ON vulnerability(target_id, source, id)",
		"CREATE INDEX IF NOT EXISTS idx_vuln_target_type_id ON vulnerability(target_id, vuln_type, id)",
		"CREATE INDEX IF NOT EXISTS idx_vuln_reviewed_created_at_id ON vulnerability(reviewed, created_at, id)",
		"CREATE INDEX IF NOT EXISTS idx_vuln_target_reviewed_created_at_id ON vulnerability(target_id, reviewed, created_at, id)",
		"CREATE INDEX IF NOT EXISTS idx_vuln_snap_url_trgm ON vulnerability_snapshot USING GIN (url gin_trgm_ops)",
		"CREATE INDEX IF NOT EXISTS idx_vuln_snap_scan_created_at_id ON vulnerability_snapshot(scan_id, created_at, id)",
		"CREATE INDEX IF NOT EXISTS idx_vuln_snap_scan_severity_id ON vulnerability_snapshot(scan_id, severity, id)",
		"CREATE INDEX IF NOT EXISTS idx_vuln_snap_scan_source_id ON vulnerability_snapshot(scan_id, source, id)",
		"CREATE INDEX IF NOT EXISTS idx_vuln_snap_scan_type_id ON vulnerability_snapshot(scan_id, vuln_type, id)",
	} {
		if !strings.Contains(sql, required) {
			t.Fatalf("initial schema must index vulnerability list query field: %q", required)
		}
	}
}

func TestDevelopmentSchemaKeepsSingleSquashedMigrationPair(t *testing.T) {
	entries, err := os.ReadDir("migrations")
	if err != nil {
		t.Fatalf("read migrations directory: %v", err)
	}

	var migrationFiles []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if strings.HasSuffix(name, ".up.sql") || strings.HasSuffix(name, ".down.sql") {
			migrationFiles = append(migrationFiles, name)
		}
	}
	slices.Sort(migrationFiles)

	expected := []string{
		"000001_init_schema.down.sql",
		"000001_init_schema.up.sql",
	}
	if !slices.Equal(migrationFiles, expected) {
		t.Fatalf("development schema migrations should be squashed into 000001 pair; got %v", migrationFiles)
	}
}

func TestInitialSchemaDownDropsEveryCreatedTable(t *testing.T) {
	upTables := extractTableNames(`(?m)^CREATE TABLE IF NOT EXISTS ([a-zA-Z0-9_]+)`, readInitialSchema(t))
	downTables := extractTableNames(`(?m)^DROP TABLE IF EXISTS ([a-zA-Z0-9_]+)`, readInitialSchemaDown(t))

	var missing []string
	for _, table := range upTables {
		if !slices.Contains(downTables, table) {
			missing = append(missing, table)
		}
	}

	if len(missing) > 0 {
		t.Fatalf("down migration must drop every table created by up; missing %v", missing)
	}
}
