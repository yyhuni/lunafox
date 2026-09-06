package repository

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

const (
	engineDiagnosticsPostgresDSNEnv = "LUNAFOX_POSTGRES_ENGINE_DIAGNOSTICS_DSN"
	engineDiagnosticsPostgresSchema = "scan_engine_diagnostics"
)

func TestScanTaskRepositoryBulkLifecycleDiagnosticsPostgres(t *testing.T) {
	db := openEngineDiagnosticsPostgres(t)
	for _, statement := range []string{
		`INSERT INTO scan_task (id, scan_id, status, resolved_execution_plan)
		 VALUES (1, 7, 'pending', decode('01', 'hex')),
		        (2, 7, 'blocked', ''::bytea),
		        (3, 7, 'running', decode('02', 'hex'))`,
	} {
		if err := db.Exec(statement).Error; err != nil {
			t.Fatalf("prepare Engine diagnostics fixture: %v", err)
		}
	}

	affected, err := NewScanTaskRepository(db).CancelUnstartedTasksByScanID(context.Background(), 7)
	if err != nil {
		t.Fatalf("CancelUnstartedTasksByScanID returned error: %v", err)
	}
	if affected != 2 {
		t.Fatalf("CancelUnstartedTasksByScanID affected %d tasks, want 2", affected)
	}
	assertPersistedUnavailableEngineDiagnostics(t, db, 1)
	assertNoPersistedEngineDiagnostics(t, db, 2)

	var running struct {
		Status string
	}
	if err := db.Table("scan_task").Select("status").Where("id = ?", 3).Take(&running).Error; err != nil {
		t.Fatalf("read running task: %v", err)
	}
	if running.Status != taskStatusRunning {
		t.Fatalf("running task status = %q, want %q", running.Status, taskStatusRunning)
	}
	assertNoPersistedEngineDiagnostics(t, db, 3)
}

func TestDeleteAgentLifecycleDiagnosticsPostgres(t *testing.T) {
	db := openEngineDiagnosticsPostgres(t)
	for _, statement := range []string{
		`INSERT INTO agent (id) VALUES (42)`,
		`INSERT INTO scan (id, agent_id, assignment_mode, status) VALUES (7, 42, 'pinned', 'pending')`,
		`INSERT INTO scan_task (id, scan_id, status, resolved_execution_plan)
		 VALUES (1, 7, 'pending', decode('01', 'hex')),
		        (2, 7, 'blocked', ''::bytea)`,
	} {
		if err := db.Exec(statement).Error; err != nil {
			t.Fatalf("prepare Agent deletion fixture: %v", err)
		}
	}

	if err := (&ScanRepository{db: db}).DeleteAgent(context.Background(), 42); err != nil {
		t.Fatalf("DeleteAgent returned error: %v", err)
	}
	assertPersistedUnavailableEngineDiagnostics(t, db, 1)
	assertNoPersistedEngineDiagnostics(t, db, 2)
}

func openEngineDiagnosticsPostgres(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := strings.TrimSpace(os.Getenv(engineDiagnosticsPostgresDSNEnv))
	if dsn == "" {
		t.Skip("set " + engineDiagnosticsPostgresDSNEnv + " with search_path=" + engineDiagnosticsPostgresSchema + " to run PostgreSQL Engine diagnostics verification")
	}
	if !strings.Contains(dsn, "search_path="+engineDiagnosticsPostgresSchema) {
		t.Fatalf("Engine diagnostics PostgreSQL DSN must pin search_path=%s", engineDiagnosticsPostgresSchema)
	}
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatalf("open Engine diagnostics PostgreSQL database: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("open Engine diagnostics PostgreSQL SQL handle: %v", err)
	}
	sqlDB.SetMaxOpenConns(4)
	t.Cleanup(func() { _ = sqlDB.Close() })
	if err := db.Exec("DROP SCHEMA IF EXISTS " + engineDiagnosticsPostgresSchema + " CASCADE").Error; err != nil {
		t.Fatalf("drop Engine diagnostics schema: %v", err)
	}
	if err := db.Exec("CREATE SCHEMA " + engineDiagnosticsPostgresSchema).Error; err != nil {
		t.Fatalf("create Engine diagnostics schema: %v", err)
	}
	t.Cleanup(func() {
		cleanupCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		_ = db.WithContext(cleanupCtx).Exec("DROP SCHEMA IF EXISTS " + engineDiagnosticsPostgresSchema + " CASCADE").Error
	})
	for _, statement := range []string{
		`CREATE TABLE agent (id INTEGER PRIMARY KEY)`,
		`CREATE TABLE scan (
			id INTEGER PRIMARY KEY,
			agent_id INTEGER,
			assignment_mode TEXT NOT NULL,
			status TEXT NOT NULL,
			deleted_at TIMESTAMPTZ,
			stopped_at TIMESTAMPTZ,
			error_message TEXT,
			failure_kind TEXT
		)`,
		`CREATE TABLE scan_task (
		id INTEGER PRIMARY KEY,
		scan_id INTEGER NOT NULL,
		status VARCHAR(20) NOT NULL,
		resolved_execution_plan BYTEA NOT NULL DEFAULT ''::bytea,
		engine_diagnostics JSONB,
		completed_at TIMESTAMPTZ,
		terminal_reconciliation_pending BOOLEAN NOT NULL DEFAULT FALSE,
		error_message TEXT,
		failure_kind TEXT,
		failure_detail TEXT
	)`,
	} {
		if err := db.Exec(statement).Error; err != nil {
			t.Fatalf("prepare Engine diagnostics schema: %v", err)
		}
	}
	return db
}
