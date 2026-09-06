package main

import (
	"context"
	"database/sql"
	"math"
	"os"
	"strings"
	"testing"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	serverdatabase "github.com/yyhuni/lunafox/server/internal/database"
)

const emptyPostgresMigrationTestDSNEnv = "LUNAFOX_EMPTY_POSTGRES_MIGRATION_DSN"

func TestInitialSchemaBootstrapsEmptyPostgres(t *testing.T) {
	dsn := strings.TrimSpace(os.Getenv(emptyPostgresMigrationTestDSNEnv))
	if dsn == "" {
		t.Skip(emptyPostgresMigrationTestDSNEnv + " is not configured")
	}

	db, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatalf("open PostgreSQL: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		t.Fatalf("ping PostgreSQL: %v", err)
	}

	var databaseName string
	if err := db.QueryRowContext(ctx, "SELECT current_database()").Scan(&databaseName); err != nil {
		t.Fatalf("read current database: %v", err)
	}
	if !strings.HasPrefix(databaseName, "lunafox_migration_test_") {
		t.Fatalf("refusing destructive migration test against database %q", databaseName)
	}
	if count := countPublicTables(t, ctx, db, false); count != 0 {
		t.Fatalf("migration test database must start empty, found %d public tables", count)
	}

	previousFS, previousPath := serverdatabase.MigrationsFS, serverdatabase.MigrationsPath
	serverdatabase.MigrationsFS = migrationsFS
	serverdatabase.MigrationsPath = "migrations"
	t.Cleanup(func() {
		serverdatabase.MigrationsFS = previousFS
		serverdatabase.MigrationsPath = previousPath
	})
	if err := serverdatabase.RunMigrations(db); err != nil {
		t.Fatalf("run baseline migration: %v", err)
	}
	version, dirty, err := serverdatabase.GetMigrationVersion(db)
	if err != nil {
		t.Fatalf("read migration version: %v", err)
	}
	if version != 1 || dirty {
		t.Fatalf("migration version = %d dirty=%t, want version 1 clean", version, dirty)
	}

	for _, table := range []string{"registration_token", "agent", "agent_runtime_status", "agent_location", "server_location_snapshot"} {
		if !postgresTableExists(t, ctx, db, table) {
			t.Fatalf("baseline migration did not create %s", table)
		}
	}
	for _, constraint := range []string{
		"agent_registration_token_id_fkey",
		"agent_runtime_status_observed_ip_generation_nonnegative",
		"agent_location_latitude_range",
		"server_location_snapshot_singleton",
	} {
		if !postgresConstraintExists(t, ctx, db, constraint) {
			t.Fatalf("baseline migration did not create constraint %s", constraint)
		}
	}
	verifyNotificationFreshPostgresPipeline(t, ctx, db, dsn)
	verifyDirectoryPostgresContract(t, ctx, db)
	verifyObservedAssetURLPostgresContract(t, ctx, db)
	verifyHostPortPostgresContract(t, ctx, db)
	verifyBlacklistPostgresContract(t, ctx, db)
	verifyScanTriggerProvenancePostgresContract(t, ctx, db)
	verifyScanInputSourcePostgresContract(t, ctx, db)
	verifyAuthUserTokenVersionPostgresContract(t, ctx, db)
	verifyTargetCleanupPostgresContract(t, ctx, db)

	if err := serverdatabase.MigrateDown(db); err != nil {
		t.Fatalf("run baseline down migration: %v", err)
	}
	if count := countPublicTables(t, ctx, db, true); count != 0 {
		t.Fatalf("down migration left %d application tables", count)
	}
	if err := serverdatabase.RunMigrations(db); err != nil {
		t.Fatalf("rerun baseline migration after destructive down: %v", err)
	}
	verifyBlacklistPostgresContract(t, ctx, db)
	verifyScanInputSourcePostgresContract(t, ctx, db)
	verifyAuthUserTokenVersionPostgresContract(t, ctx, db)
	verifyTargetCleanupPostgresContract(t, ctx, db)
	if err := serverdatabase.MigrateDown(db); err != nil {
		t.Fatalf("run second baseline down migration: %v", err)
	}
	if count := countPublicTables(t, ctx, db, true); count != 0 {
		t.Fatalf("second down migration left %d application tables", count)
	}
}

func verifyScanTriggerProvenancePostgresContract(t *testing.T, ctx context.Context, db *sql.DB) {
	t.Helper()

	var targetID int
	if err := db.QueryRowContext(ctx, `INSERT INTO target (name, type) VALUES ('scan-trigger-provenance-contract.example', 'domain') RETURNING id`).Scan(&targetID); err != nil {
		t.Fatalf("seed Scan trigger provenance Target: %v", err)
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO scan (target_id, scan_workflow_id, input_source, status) VALUES ($1, 'scan-trigger-provenance', 'scan_snapshot', 'pending')`, targetID); err == nil {
		t.Fatal("scan baseline accepted a missing trigger_type")
	}
	for _, triggerType := range []string{"legacy", "unknown"} {
		if _, err := db.ExecContext(ctx, `INSERT INTO scan (target_id, scan_workflow_id, input_source, trigger_type, status) VALUES ($1, 'scan-trigger-provenance', 'scan_snapshot', $2, 'pending')`, targetID, triggerType); err == nil {
			t.Fatalf("scan baseline accepted invalid trigger_type %q", triggerType)
		}
	}

	var manualID, scheduledID int
	if err := db.QueryRowContext(ctx, `
		INSERT INTO scan (target_id, scan_workflow_id, input_source, trigger_type, status)
		VALUES ($1, 'scan-trigger-provenance-manual', 'scan_snapshot', 'manual', 'pending')
		RETURNING id
	`, targetID).Scan(&manualID); err != nil {
		t.Fatalf("create manual Scan from fresh baseline: %v", err)
	}
	if err := db.QueryRowContext(ctx, `
		INSERT INTO scan (target_id, scan_workflow_id, input_source, trigger_type, status)
		VALUES ($1, 'scan-trigger-provenance-scheduled', 'scan_snapshot', 'scheduled', 'pending')
		RETURNING id
	`, targetID).Scan(&scheduledID); err != nil {
		t.Fatalf("create scheduled Scan from fresh baseline: %v", err)
	}

	for scanID, want := range map[int]string{manualID: "manual", scheduledID: "scheduled"} {
		var got string
		if err := db.QueryRowContext(ctx, `SELECT trigger_type FROM scan WHERE id = $1`, scanID).Scan(&got); err != nil {
			t.Fatalf("read Scan %d trigger type: %v", scanID, err)
		}
		if got != want {
			t.Fatalf("Scan %d trigger_type = %q, want %q", scanID, got, want)
		}
	}

	var scheduleID int
	if err := db.QueryRowContext(ctx, `
		INSERT INTO scheduled_scan (name, scan_workflow_id, input_source, target_id, is_enabled)
		VALUES ('scan-trigger-provenance-schedule', 'scan-trigger-provenance-scheduled', 'scan_snapshot', $1, FALSE)
		RETURNING id
	`, targetID).Scan(&scheduleID); err != nil {
		t.Fatalf("seed Schedule for provenance deletion rehearsal: %v", err)
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO scheduled_scan_occurrence (scheduled_scan_id, scheduled_for) VALUES ($1, CURRENT_TIMESTAMP)`, scheduleID); err != nil {
		t.Fatalf("seed Schedule occurrence for provenance deletion rehearsal: %v", err)
	}
	if _, err := db.ExecContext(ctx, `DELETE FROM scheduled_scan WHERE id = $1`, scheduleID); err != nil {
		t.Fatalf("delete Schedule after scheduled Scan creation: %v", err)
	}
	var occurrenceCount int
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM scheduled_scan_occurrence WHERE scheduled_scan_id = $1`, scheduleID).Scan(&occurrenceCount); err != nil {
		t.Fatalf("count deleted Schedule occurrences: %v", err)
	}
	if occurrenceCount != 0 {
		t.Fatalf("Schedule deletion left %d occurrences", occurrenceCount)
	}
	var scheduledTriggerType string
	if err := db.QueryRowContext(ctx, `SELECT trigger_type FROM scan WHERE id = $1`, scheduledID).Scan(&scheduledTriggerType); err != nil {
		t.Fatalf("read scheduled Scan after Schedule deletion: %v", err)
	}
	if scheduledTriggerType != "scheduled" {
		t.Fatalf("scheduled Scan trigger_type after Schedule deletion = %q, want scheduled", scheduledTriggerType)
	}
}

func verifyScanInputSourcePostgresContract(t *testing.T, ctx context.Context, db *sql.DB) {
	t.Helper()
	for _, table := range []string{"scan", "scheduled_scan"} {
		var nullable string
		var defaultValue sql.NullString
		if err := db.QueryRowContext(ctx, `
			SELECT is_nullable, column_default
			FROM information_schema.columns
			WHERE table_schema = 'public' AND table_name = $1 AND column_name = 'input_source'
		`, table).Scan(&nullable, &defaultValue); err != nil {
			t.Fatalf("read %s.input_source metadata: %v", table, err)
		}
		if nullable != "NO" || defaultValue.Valid {
			t.Fatalf("%s.input_source metadata = nullable=%q default=%q, want NOT NULL without default", table, nullable, defaultValue.String)
		}
	}

	var targetID int
	if err := db.QueryRowContext(ctx, `INSERT INTO target (name, type) VALUES ('scan-input-source-contract.example', 'domain') RETURNING id`).Scan(&targetID); err != nil {
		t.Fatalf("seed input source contract Target: %v", err)
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO scan (target_id, scan_workflow_id, trigger_type, status) VALUES ($1, 'input-source-contract', 'manual', 'pending')`, targetID); err == nil {
		t.Fatal("scan baseline accepted a missing input_source")
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO scheduled_scan (name, scan_workflow_id, target_id, is_enabled) VALUES ('input-source-contract', 'input-source-contract', $1, FALSE)`, targetID); err == nil {
		t.Fatal("scheduled_scan baseline accepted a missing input_source")
	}
	for _, source := range []string{"", "unknown"} {
		if _, err := db.ExecContext(ctx, `INSERT INTO scan (target_id, scan_workflow_id, input_source, trigger_type, status) VALUES ($1, 'input-source-contract', $2, 'manual', 'pending')`, targetID, source); err == nil {
			t.Fatalf("scan baseline accepted invalid input_source %q", source)
		}
		if _, err := db.ExecContext(ctx, `INSERT INTO scheduled_scan (name, scan_workflow_id, input_source, target_id, is_enabled) VALUES ('input-source-contract-' || $2, 'input-source-contract', $2, $1, FALSE)`, targetID, source); err == nil {
			t.Fatalf("scheduled_scan baseline accepted invalid input_source %q", source)
		}
	}
	for _, source := range []string{"scan_snapshot", "target_inventory"} {
		if _, err := db.ExecContext(ctx, `INSERT INTO scan (target_id, scan_workflow_id, input_source, trigger_type, status) VALUES ($1, 'input-source-contract-' || $2, $2, 'manual', 'pending')`, targetID, source); err != nil {
			t.Fatalf("scan baseline rejected valid input_source %q: %v", source, err)
		}
		if _, err := db.ExecContext(ctx, `INSERT INTO scheduled_scan (name, scan_workflow_id, input_source, target_id, is_enabled) VALUES ('input-source-contract-' || $2, 'input-source-contract-' || $2, $2, $1, FALSE)`, targetID, source); err != nil {
			t.Fatalf("scheduled_scan baseline rejected valid input_source %q: %v", source, err)
		}
	}
}

func verifyAuthUserTokenVersionPostgresContract(t *testing.T, ctx context.Context, db *sql.DB) {
	t.Helper()
	var seedVersion int
	if err := db.QueryRowContext(ctx, `SELECT token_version FROM auth_user WHERE username = 'admin'`).Scan(&seedVersion); err != nil {
		t.Fatalf("read initial admin token_version: %v", err)
	}
	if seedVersion != 0 {
		t.Fatalf("initial admin token_version = %d, want 0", seedVersion)
	}

	var createdVersion int
	if err := db.QueryRowContext(ctx, `INSERT INTO auth_user (username, password) VALUES ('token-version-contract-user', 'hash') RETURNING token_version`).Scan(&createdVersion); err != nil {
		t.Fatalf("create token version contract user: %v", err)
	}
	if createdVersion != 0 {
		t.Fatalf("new auth_user token_version = %d, want 0", createdVersion)
	}
}

func verifyTargetCleanupPostgresContract(t *testing.T, ctx context.Context, db *sql.DB) {
	t.Helper()
	if !postgresTableExists(t, ctx, db, "target_cleanup_job") {
		t.Fatal("baseline migration did not create target_cleanup_job")
	}
	assertPostgresExactColumns(t, ctx, db, "target_cleanup_job", []string{
		"id", "target_id", "status", "retry_count", "next_retry_at", "last_error", "completed_at", "created_at", "updated_at",
	})

	var oldTargetID int
	if err := db.QueryRowContext(ctx, `INSERT INTO target (name, type) VALUES ('target-cleanup-contract.example', 'domain') RETURNING id`).Scan(&oldTargetID); err != nil {
		t.Fatalf("seed target cleanup target: %v", err)
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO target_cleanup_job (target_id) VALUES ($1)`, oldTargetID); err != nil {
		t.Fatalf("seed target cleanup job: %v", err)
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO target_cleanup_job (target_id) VALUES ($1)`, oldTargetID); err == nil {
		t.Fatal("target_cleanup_job accepted a second row for one target")
	}
	if _, err := db.ExecContext(ctx, `UPDATE target SET deleted_at = CURRENT_TIMESTAMP WHERE id = $1`, oldTargetID); err != nil {
		t.Fatalf("tombstone target cleanup target: %v", err)
	}
	var replacementID int
	if err := db.QueryRowContext(ctx, `INSERT INTO target (name, type) VALUES ('target-cleanup-contract.example', 'domain') RETURNING id`).Scan(&replacementID); err != nil {
		t.Fatalf("active target identity did not become reusable after tombstone: %v", err)
	}
	if replacementID == oldTargetID {
		t.Fatal("replacement target must receive a distinct identity")
	}
}

func verifyBlacklistPostgresContract(t *testing.T, ctx context.Context, db *sql.DB) {
	t.Helper()
	for _, table := range []string{"blacklist_policy", "scan_blacklist_snapshot"} {
		if !postgresTableExists(t, ctx, db, table) {
			t.Fatalf("baseline migration did not create %s", table)
		}
	}
	assertPostgresExactColumns(t, ctx, db, "blacklist_policy", []string{"id", "scope", "target_id", "patterns", "updated_at"})
	assertPostgresExactColumns(t, ctx, db, "scan_blacklist_snapshot", []string{"scan_id", "patterns"})

	var globalCount int
	var globalPatterns string
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*), COALESCE(MAX(patterns::text), '') FROM blacklist_policy WHERE scope = 'global' AND target_id IS NULL`).Scan(&globalCount, &globalPatterns); err != nil {
		t.Fatalf("read global blacklist policy: %v", err)
	}
	if globalCount != 1 || globalPatterns != "[]" {
		t.Fatalf("global blacklist policy = count %d patterns %q, want one empty policy", globalCount, globalPatterns)
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO blacklist_policy (scope, target_id, patterns) VALUES ('global', NULL, '[]'::jsonb)`); err == nil {
		t.Fatal("blacklist_policy accepted a second global row")
	}
	for _, patterns := range []string{`{}`, `[1]`, `[null]`} {
		if _, err := db.ExecContext(ctx, `INSERT INTO blacklist_policy (scope, target_id, patterns) VALUES ('target', NULL, $1::jsonb)`, patterns); err == nil {
			t.Fatalf("blacklist_policy accepted malformed patterns %s", patterns)
		}
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO blacklist_policy (scope, target_id, patterns) VALUES ('target', NULL, '[]'::jsonb)`); err == nil {
		t.Fatal("blacklist_policy accepted target scope without target_id")
	}

	var targetID int
	if err := db.QueryRowContext(ctx, `INSERT INTO target (name, type) VALUES ('blacklist-contract.example', 'domain') RETURNING id`).Scan(&targetID); err != nil {
		t.Fatalf("seed blacklist target: %v", err)
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO blacklist_policy (scope, target_id, patterns) VALUES ('global', $1, '[]'::jsonb)`, targetID); err == nil {
		t.Fatal("blacklist_policy accepted global scope with target_id")
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO blacklist_policy (scope, target_id, patterns) VALUES ('target', $1, '[]'::jsonb)`, targetID); err != nil {
		t.Fatalf("insert target blacklist policy: %v", err)
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO blacklist_policy (scope, target_id, patterns) VALUES ('target', $1, '[]'::jsonb)`, targetID); err == nil {
		t.Fatal("blacklist_policy accepted a duplicate target row")
	}

	var scanID int
	if err := db.QueryRowContext(ctx, `INSERT INTO scan (target_id, scan_workflow_id, input_source, trigger_type, status) VALUES ($1, 'blacklist-contract', 'scan_snapshot', 'manual', 'initiated') RETURNING id`, targetID).Scan(&scanID); err != nil {
		t.Fatalf("seed blacklist scan: %v", err)
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO scan_blacklist_snapshot (scan_id, patterns) VALUES ($1, '[]'::jsonb)`, scanID); err != nil {
		t.Fatalf("insert blacklist snapshot: %v", err)
	}
	if _, err := db.ExecContext(ctx, `DELETE FROM scan WHERE id = $1`, scanID); err != nil {
		t.Fatalf("delete blacklist scan: %v", err)
	}
	var snapshots int
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM scan_blacklist_snapshot WHERE scan_id = $1`, scanID).Scan(&snapshots); err != nil {
		t.Fatalf("count cascading snapshot: %v", err)
	}
	if snapshots != 0 {
		t.Fatalf("scan deletion left %d blacklist snapshots", snapshots)
	}
}

func assertPostgresExactColumns(t *testing.T, ctx context.Context, db *sql.DB, table string, want []string) {
	t.Helper()
	rows, err := db.QueryContext(ctx, `
		SELECT column_name
		FROM information_schema.columns
		WHERE table_schema = 'public' AND table_name = $1
		ORDER BY ordinal_position
	`, table)
	if err != nil {
		t.Fatalf("list %s columns: %v", table, err)
	}
	defer rows.Close()
	var got []string
	for rows.Next() {
		var column string
		if err := rows.Scan(&column); err != nil {
			t.Fatalf("scan %s column: %v", table, err)
		}
		got = append(got, column)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("read %s columns: %v", table, err)
	}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("%s columns = %v, want %v", table, got, want)
	}
}

func verifyHostPortPostgresContract(t *testing.T, ctx context.Context, db *sql.DB) {
	t.Helper()
	for _, constraint := range []string{
		"host_port_mapping_host_nonblank",
		"host_port_mapping_ip_ipv4_host",
		"host_port_mapping_port_range",
		"host_port_mapping_snapshot_host_nonblank",
		"host_port_mapping_snapshot_ip_ipv4_host",
		"host_port_mapping_snapshot_port_range",
	} {
		if !postgresConstraintExists(t, ctx, db, constraint) {
			t.Fatalf("baseline migration did not create constraint %s", constraint)
		}
	}

	var targetID int
	if err := db.QueryRowContext(ctx, `INSERT INTO target (name, type) VALUES ('host-port.example', 'domain') RETURNING id`).Scan(&targetID); err != nil {
		t.Fatalf("seed HostPort Target: %v", err)
	}
	var scanID int
	if err := db.QueryRowContext(ctx, `
		INSERT INTO scan (target_id, scan_workflow_id, input_source, trigger_type, status)
		VALUES ($1, 'host-port-postgres-contract', 'scan_snapshot', 'manual', 'initiated')
		RETURNING id
	`, targetID).Scan(&scanID); err != nil {
		t.Fatalf("seed HostPort Scan: %v", err)
	}

	for table, scope := range map[string]struct {
		column string
		id     int
	}{
		"host_port_mapping":          {column: "target_id", id: targetID},
		"host_port_mapping_snapshot": {column: "scan_id", id: scanID},
	} {
		if _, err := db.ExecContext(ctx, `INSERT INTO `+table+` (`+scope.column+`, host, ip, port) VALUES ($1, $2, $3, $4)`, scope.id, "api.host-port.example", "192.0.2.10", 443); err != nil {
			t.Fatalf("insert valid HostPort into %s: %v", table, err)
		}
		for _, observation := range []struct {
			host any
			ip   any
			port any
		}{
			{host: nil, ip: "192.0.2.11", port: 80},
			{host: "", ip: "192.0.2.11", port: 80},
			{host: "   ", ip: "192.0.2.11", port: 80},
			{host: "ipv6.host-port.example", ip: "2001:db8::1", port: 80},
			{host: "network.host-port.example", ip: "192.0.2.0/24", port: 80},
			{host: "zero.host-port.example", ip: "192.0.2.12", port: 0},
			{host: "negative.host-port.example", ip: "192.0.2.13", port: -1},
			{host: "large.host-port.example", ip: "192.0.2.14", port: 65536},
		} {
			if _, err := db.ExecContext(ctx, `INSERT INTO `+table+` (`+scope.column+`, host, ip, port) VALUES ($1, $2, $3, $4)`, scope.id, observation.host, observation.ip, observation.port); err == nil {
				t.Fatalf("%s accepted invalid HostPort observation %#v", table, observation)
			}
		}
	}
}

func verifyDirectoryPostgresContract(t *testing.T, ctx context.Context, db *sql.DB) {
	t.Helper()
	for _, table := range []string{"directory", "directory_snapshot"} {
		assertPostgresColumnShape(t, ctx, db, table, "content_length", "int8", 0)
		assertPostgresColumnShape(t, ctx, db, table, "duration", "int8", 0)
		assertPostgresColumnShape(t, ctx, db, table, "content_type", "varchar", 1024)
	}

	var targetID int
	if err := db.QueryRowContext(ctx, `INSERT INTO target (name, type) VALUES ('example.com', 'domain') RETURNING id`).Scan(&targetID); err != nil {
		t.Fatalf("seed Directory Target: %v", err)
	}
	var scanID int
	if err := db.QueryRowContext(ctx, `
		INSERT INTO scan (target_id, scan_workflow_id, input_source, trigger_type, status)
		VALUES ($1, 'directory-postgres-contract', 'scan_snapshot', 'manual', 'initiated')
		RETURNING id
	`, targetID).Scan(&scanID); err != nil {
		t.Fatalf("seed Directory Scan: %v", err)
	}

	url := "https://Example.com/%00?x=1#frag"
	contentType := strings.Repeat("x", 1024)
	if _, err := db.ExecContext(ctx, `
		INSERT INTO directory (target_id, url, status, content_length, content_type, duration)
		VALUES ($1, $2, 999, $3, $4, $3)
	`, targetID, url, int64(math.MaxInt64), contentType); err != nil {
		t.Fatalf("insert MaxInt64 Directory Asset: %v", err)
	}
	if _, err := db.ExecContext(ctx, `
		INSERT INTO directory_snapshot (scan_id, url, status, content_length, content_type, duration)
		VALUES ($1, $2, 999, $3, $4, $3)
	`, scanID, url, int64(math.MaxInt64), contentType); err != nil {
		t.Fatalf("insert MaxInt64 Directory Snapshot: %v", err)
	}
	assertPostgresDirectoryObservation(t, ctx, db, "directory", "target_id", targetID, url, 999, math.MaxInt64, contentType, math.MaxInt64)
	assertPostgresDirectoryObservation(t, ctx, db, "directory_snapshot", "scan_id", scanID, url, 999, math.MaxInt64, contentType, math.MaxInt64)

	if _, err := db.ExecContext(ctx, `
		INSERT INTO directory (target_id, url, status, content_length, content_type, duration)
		VALUES ($1, $2, 0, 0, '', 0)
		ON CONFLICT (target_id, url) DO UPDATE SET
			status = EXCLUDED.status,
			content_length = EXCLUDED.content_length,
			content_type = EXCLUDED.content_type,
			duration = EXCLUDED.duration
	`, targetID, url); err != nil {
		t.Fatalf("replace Directory Asset with zero observation: %v", err)
	}
	if _, err := db.ExecContext(ctx, `
		INSERT INTO directory_snapshot (scan_id, url, status, content_length, content_type, duration)
		VALUES ($1, $2, 0, 0, '', 0)
		ON CONFLICT (scan_id, url) DO UPDATE SET
			status = EXCLUDED.status,
			content_length = EXCLUDED.content_length,
			content_type = EXCLUDED.content_type,
			duration = EXCLUDED.duration
	`, scanID, url); err != nil {
		t.Fatalf("replace Directory Snapshot with zero observation: %v", err)
	}
	assertPostgresDirectoryObservation(t, ctx, db, "directory", "target_id", targetID, url, 0, 0, "", 0)
	assertPostgresDirectoryObservation(t, ctx, db, "directory_snapshot", "scan_id", scanID, url, 0, 0, "", 0)

	if _, err := db.ExecContext(ctx, `UPDATE scan SET status = 'failed', stopped_at = CURRENT_TIMESTAMP WHERE id = $1`, scanID); err != nil {
		t.Fatalf("mark Directory Scan failed: %v", err)
	}
	for table, scope := range map[string]struct {
		column string
		id     int
	}{
		"directory":          {column: "target_id", id: targetID},
		"directory_snapshot": {column: "scan_id", id: scanID},
	} {
		var count int
		query := `SELECT COUNT(*) FROM ` + table + ` WHERE ` + scope.column + ` = $1 AND url = $2`
		if err := db.QueryRowContext(ctx, query, scope.id, url).Scan(&count); err != nil {
			t.Fatalf("query %s after failed Scan: %v", table, err)
		}
		if count != 1 {
			t.Fatalf("%s rows after failed Scan = %d, want 1", table, count)
		}
	}
}

// verifyObservedAssetURLPostgresContract proves the full-value natural-key
// boundary against PostgreSQL itself. Application validation rejects oversized
// values first, while this test prevents a future schema/index change from
// silently making an accepted 2000-byte URL unpersistable.
func verifyObservedAssetURLPostgresContract(t *testing.T, ctx context.Context, db *sql.DB) {
	t.Helper()
	for _, table := range []string{
		"website", "endpoint", "directory", "screenshot", "vulnerability",
		"website_snapshot", "endpoint_snapshot", "directory_snapshot", "screenshot_snapshot", "vulnerability_snapshot",
	} {
		assertPostgresColumnShape(t, ctx, db, table, "url", "varchar", 2000)
	}

	var targetID int
	if err := db.QueryRowContext(ctx, `INSERT INTO target (name, type) VALUES ('raw-observed-url-contract.example', 'domain') RETURNING id`).Scan(&targetID); err != nil {
		t.Fatalf("seed observed URL contract Target: %v", err)
	}
	var scanID int
	if err := db.QueryRowContext(ctx, `
		INSERT INTO scan (target_id, scan_workflow_id, input_source, trigger_type, status)
		VALUES ($1, 'raw-observed-url-postgres-contract', 'scan_snapshot', 'manual', 'initiated')
		RETURNING id
	`, targetID).Scan(&scanID); err != nil {
		t.Fatalf("seed observed URL contract Scan: %v", err)
	}

	const prefix = "https://raw-observed-url-contract.example/"
	rawURL := prefix + strings.Repeat("a", 2000-len(prefix))
	if len(rawURL) != 2000 {
		t.Fatalf("contract URL length = %d, want 2000", len(rawURL))
	}

	assertObservedURLPostgresRoundTrip(t, ctx, db, "Website Asset", rawURL,
		`INSERT INTO website (target_id, url, host, title) VALUES ($1, $2, 'raw-observed-url-contract.example', 'first')`, []any{targetID, rawURL},
		`INSERT INTO website (target_id, url, host, title) VALUES ($1, $2, 'raw-observed-url-contract.example', 'second') ON CONFLICT (url, target_id) DO UPDATE SET title = EXCLUDED.title`, []any{targetID, rawURL},
		`SELECT url FROM website WHERE target_id = $1 AND url = $2`, []any{targetID, rawURL})
	assertObservedURLPostgresRoundTrip(t, ctx, db, "Endpoint Asset", rawURL,
		`INSERT INTO endpoint (target_id, url, host, title) VALUES ($1, $2, 'raw-observed-url-contract.example', 'first')`, []any{targetID, rawURL},
		`INSERT INTO endpoint (target_id, url, host, title) VALUES ($1, $2, 'raw-observed-url-contract.example', 'second') ON CONFLICT (url, target_id) DO UPDATE SET title = EXCLUDED.title`, []any{targetID, rawURL},
		`SELECT url FROM endpoint WHERE target_id = $1 AND url = $2`, []any{targetID, rawURL})
	assertObservedURLPostgresRoundTrip(t, ctx, db, "Directory Asset", rawURL,
		`INSERT INTO directory (target_id, url, status) VALUES ($1, $2, 200)`, []any{targetID, rawURL},
		`INSERT INTO directory (target_id, url, status) VALUES ($1, $2, 201) ON CONFLICT (target_id, url) DO UPDATE SET status = EXCLUDED.status`, []any{targetID, rawURL},
		`SELECT url FROM directory WHERE target_id = $1 AND url = $2`, []any{targetID, rawURL})
	assertObservedURLPostgresRoundTrip(t, ctx, db, "Screenshot Asset", rawURL,
		`INSERT INTO screenshot (target_id, url, image) VALUES ($1, $2, '\x01')`, []any{targetID, rawURL},
		`INSERT INTO screenshot (target_id, url, image) VALUES ($1, $2, '\x02') ON CONFLICT (target_id, url) DO UPDATE SET image = EXCLUDED.image`, []any{targetID, rawURL},
		`SELECT url FROM screenshot WHERE target_id = $1 AND url = $2`, []any{targetID, rawURL})
	assertObservedURLPostgresRoundTrip(t, ctx, db, "Vulnerability Asset", rawURL,
		`INSERT INTO vulnerability (target_id, url, vuln_type, severity, description, reviewed) VALUES ($1, $2, 'raw-url-contract', 'high', 'first', TRUE)`, []any{targetID, rawURL},
		`INSERT INTO vulnerability (target_id, url, vuln_type, severity, description) VALUES ($1, $2, 'raw-url-contract', 'high', 'second') ON CONFLICT (target_id, url, vuln_type, severity) DO UPDATE SET description = EXCLUDED.description`, []any{targetID, rawURL},
		`SELECT url FROM vulnerability WHERE target_id = $1 AND url = $2 AND vuln_type = 'raw-url-contract' AND severity = 'high'`, []any{targetID, rawURL})
	var reviewed bool
	if err := db.QueryRowContext(ctx, `SELECT reviewed FROM vulnerability WHERE target_id = $1 AND url = $2 AND vuln_type = 'raw-url-contract' AND severity = 'high'`, targetID, rawURL).Scan(&reviewed); err != nil || !reviewed {
		t.Fatalf("Vulnerability Asset conflict must preserve reviewed state: reviewed=%t err=%v", reviewed, err)
	}

	assertObservedURLPostgresRoundTrip(t, ctx, db, "Website Snapshot", rawURL,
		`INSERT INTO website_snapshot (scan_id, url, host, title) VALUES ($1, $2, 'raw-observed-url-contract.example', 'first')`, []any{scanID, rawURL},
		`INSERT INTO website_snapshot (scan_id, url, host, title) VALUES ($1, $2, 'raw-observed-url-contract.example', 'second') ON CONFLICT (scan_id, url) DO UPDATE SET title = EXCLUDED.title`, []any{scanID, rawURL},
		`SELECT url FROM website_snapshot WHERE scan_id = $1 AND url = $2`, []any{scanID, rawURL})
	assertObservedURLPostgresRoundTrip(t, ctx, db, "Endpoint Snapshot", rawURL,
		`INSERT INTO endpoint_snapshot (scan_id, url, host, title) VALUES ($1, $2, 'raw-observed-url-contract.example', 'first')`, []any{scanID, rawURL},
		`INSERT INTO endpoint_snapshot (scan_id, url, host, title) VALUES ($1, $2, 'raw-observed-url-contract.example', 'second') ON CONFLICT (scan_id, url) DO UPDATE SET title = EXCLUDED.title`, []any{scanID, rawURL},
		`SELECT url FROM endpoint_snapshot WHERE scan_id = $1 AND url = $2`, []any{scanID, rawURL})
	assertObservedURLPostgresRoundTrip(t, ctx, db, "Directory Snapshot", rawURL,
		`INSERT INTO directory_snapshot (scan_id, url, status) VALUES ($1, $2, 200)`, []any{scanID, rawURL},
		`INSERT INTO directory_snapshot (scan_id, url, status) VALUES ($1, $2, 201) ON CONFLICT (scan_id, url) DO UPDATE SET status = EXCLUDED.status`, []any{scanID, rawURL},
		`SELECT url FROM directory_snapshot WHERE scan_id = $1 AND url = $2`, []any{scanID, rawURL})
	assertObservedURLPostgresRoundTrip(t, ctx, db, "Screenshot Snapshot", rawURL,
		`INSERT INTO screenshot_snapshot (scan_id, url, image) VALUES ($1, $2, '\x01')`, []any{scanID, rawURL},
		`INSERT INTO screenshot_snapshot (scan_id, url, image) VALUES ($1, $2, '\x02') ON CONFLICT (scan_id, url) DO UPDATE SET image = EXCLUDED.image`, []any{scanID, rawURL},
		`SELECT url FROM screenshot_snapshot WHERE scan_id = $1 AND url = $2`, []any{scanID, rawURL})
	assertObservedURLPostgresRoundTrip(t, ctx, db, "Vulnerability Snapshot", rawURL,
		`INSERT INTO vulnerability_snapshot (scan_id, url, vuln_type, severity, description) VALUES ($1, $2, 'raw-url-contract', 'high', 'first')`, []any{scanID, rawURL},
		`INSERT INTO vulnerability_snapshot (scan_id, url, vuln_type, severity, description) VALUES ($1, $2, 'raw-url-contract', 'high', 'second') ON CONFLICT (scan_id, url, vuln_type, severity) DO UPDATE SET description = EXCLUDED.description`, []any{scanID, rawURL},
		`SELECT url FROM vulnerability_snapshot WHERE scan_id = $1 AND url = $2 AND vuln_type = 'raw-url-contract' AND severity = 'high'`, []any{scanID, rawURL})

	overLimitURL := rawURL + "x"
	for _, path := range []struct {
		name, table, scopeColumn string
		scopeID                  int
	}{
		{"Website Asset", "website", "target_id", targetID},
		{"Endpoint Asset", "endpoint", "target_id", targetID},
		{"Directory Asset", "directory", "target_id", targetID},
		{"Screenshot Asset", "screenshot", "target_id", targetID},
		{"Vulnerability Asset", "vulnerability", "target_id", targetID},
		{"Website Snapshot", "website_snapshot", "scan_id", scanID},
		{"Endpoint Snapshot", "endpoint_snapshot", "scan_id", scanID},
		{"Directory Snapshot", "directory_snapshot", "scan_id", scanID},
		{"Screenshot Snapshot", "screenshot_snapshot", "scan_id", scanID},
		{"Vulnerability Snapshot", "vulnerability_snapshot", "scan_id", scanID},
	} {
		statement := `INSERT INTO ` + path.table + ` (` + path.scopeColumn + `, url) VALUES ($1, $2)`
		if _, err := db.ExecContext(ctx, statement, path.scopeID, overLimitURL); err == nil {
			t.Fatalf("%s accepted a 2001-byte URL", path.name)
		}
	}
}

func assertObservedURLPostgresRoundTrip(t *testing.T, ctx context.Context, db *sql.DB, name, wantURL, insertSQL string, insertArgs []any, conflictSQL string, conflictArgs []any, readSQL string, readArgs []any) {
	t.Helper()
	if _, err := db.ExecContext(ctx, insertSQL, insertArgs...); err != nil {
		t.Fatalf("insert %s: %v", name, err)
	}
	assertObservedURLPostgresRead(t, ctx, db, name, wantURL, readSQL, readArgs)
	if _, err := db.ExecContext(ctx, conflictSQL, conflictArgs...); err != nil {
		t.Fatalf("conflict %s: %v", name, err)
	}
	assertObservedURLPostgresRead(t, ctx, db, name, wantURL, readSQL, readArgs)
}

func assertObservedURLPostgresRead(t *testing.T, ctx context.Context, db *sql.DB, name, wantURL, readSQL string, args []any) {
	t.Helper()
	var gotURL string
	if err := db.QueryRowContext(ctx, readSQL, args...).Scan(&gotURL); err != nil {
		t.Fatalf("read %s: %v", name, err)
	}
	if gotURL != wantURL {
		t.Fatalf("%s URL round trip changed bytes: got length %d want length %d", name, len(gotURL), len(wantURL))
	}
}

func assertPostgresColumnShape(t *testing.T, ctx context.Context, db *sql.DB, table, column, wantType string, wantMaxLength int64) {
	t.Helper()
	var columnKind string
	var maxLength sql.NullInt64
	if err := db.QueryRowContext(ctx, `
		SELECT udt_name, character_maximum_length
		FROM information_schema.columns
		WHERE table_schema = 'public' AND table_name = $1 AND column_name = $2
	`, table, column).Scan(&columnKind, &maxLength); err != nil {
		t.Fatalf("read %s.%s column shape: %v", table, column, err)
	}
	if columnKind != wantType {
		t.Fatalf("%s.%s column kind = %q, want %q", table, column, columnKind, wantType)
	}
	if wantMaxLength == 0 {
		if maxLength.Valid {
			t.Fatalf("%s.%s maximum length = %d, want null", table, column, maxLength.Int64)
		}
		return
	}
	if !maxLength.Valid || maxLength.Int64 != wantMaxLength {
		t.Fatalf("%s.%s maximum length = %v, want %d", table, column, maxLength, wantMaxLength)
	}
}

func assertPostgresDirectoryObservation(t *testing.T, ctx context.Context, db *sql.DB, table, scopeColumn string, scopeID int, url string, wantStatus int, wantLength int64, wantType string, wantDuration int64) {
	t.Helper()
	query := `SELECT status, content_length, content_type, duration FROM ` + table + ` WHERE ` + scopeColumn + ` = $1 AND url = $2`
	var status int
	var contentLength, duration int64
	var contentType string
	if err := db.QueryRowContext(ctx, query, scopeID, url).Scan(&status, &contentLength, &contentType, &duration); err != nil {
		t.Fatalf("read %s Directory observation: %v", table, err)
	}
	if status != wantStatus || contentLength != wantLength || contentType != wantType || duration != wantDuration {
		t.Fatalf("%s observation = status %d length %d type %q duration %d; want %d %d %q %d", table, status, contentLength, contentType, duration, wantStatus, wantLength, wantType, wantDuration)
	}
}

func countPublicTables(t *testing.T, ctx context.Context, db *sql.DB, excludeMigrationMetadata bool) int {
	t.Helper()
	query := `SELECT COUNT(*) FROM pg_tables WHERE schemaname = 'public'`
	if excludeMigrationMetadata {
		query += ` AND tablename <> 'schema_migrations'`
	}
	var count int
	if err := db.QueryRowContext(ctx, query).Scan(&count); err != nil {
		t.Fatalf("count public tables: %v", err)
	}
	return count
}

func postgresTableExists(t *testing.T, ctx context.Context, db *sql.DB, table string) bool {
	t.Helper()
	var exists bool
	if err := db.QueryRowContext(ctx, `SELECT to_regclass('public.' || $1) IS NOT NULL`, table).Scan(&exists); err != nil {
		t.Fatalf("look up table %s: %v", table, err)
	}
	return exists
}

func postgresConstraintExists(t *testing.T, ctx context.Context, db *sql.DB, constraint string) bool {
	t.Helper()
	var exists bool
	if err := db.QueryRowContext(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM pg_constraint
			WHERE conname = $1
		)
	`, constraint).Scan(&exists); err != nil {
		t.Fatalf("look up constraint %s: %v", constraint, err)
	}
	return exists
}
