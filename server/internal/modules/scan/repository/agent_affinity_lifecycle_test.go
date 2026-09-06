package repository

import (
	"context"
	"testing"
)

func TestDeleteAgentTerminalizesNotStartedPinnedTasks(t *testing.T) {
	db := openSavedExecutionPlanTestDB(t)
	for _, statement := range []string{
		`CREATE TABLE agent (id INTEGER PRIMARY KEY)`,
		`CREATE TABLE scan (id INTEGER PRIMARY KEY, agent_id INTEGER, assignment_mode TEXT NOT NULL, status TEXT NOT NULL, deleted_at DATETIME, stopped_at DATETIME, error_message TEXT, failure_kind TEXT)`,
		`CREATE TABLE scan_task (id INTEGER PRIMARY KEY, scan_id INTEGER NOT NULL, status TEXT NOT NULL, completed_at DATETIME, terminal_reconciliation_pending BOOLEAN NOT NULL DEFAULT FALSE, error_message TEXT, failure_kind TEXT, failure_detail TEXT, resolved_execution_plan BLOB NOT NULL DEFAULT '', engine_diagnostics TEXT)`,
		`INSERT INTO agent (id) VALUES (42)`,
		`INSERT INTO scan (id, agent_id, assignment_mode, status) VALUES (7, 42, 'pinned', 'pending')`,
		`INSERT INTO scan_task (id, scan_id, status, resolved_execution_plan) VALUES (1, 7, 'pending', X'01'), (2, 7, 'blocked', X'')`,
	} {
		if err := db.Exec(statement).Error; err != nil {
			t.Fatalf("fixture setup failed: %v", err)
		}
	}

	if err := (&ScanRepository{db: db}).DeleteAgent(context.Background(), 42); err != nil {
		t.Fatalf("DeleteAgent returned error: %v", err)
	}
	assertPersistedUnavailableEngineDiagnostics(t, db, 1)
	assertNoPersistedEngineDiagnostics(t, db, 2)

	var taskCount int64
	if err := db.Table("scan_task").Where("status = ? AND failure_kind = ?", taskStatusFailed, "agent_deleted").Count(&taskCount).Error; err != nil || taskCount != 2 {
		t.Fatalf("expected two terminalized Agent-deleted tasks, count=%d err=%v", taskCount, err)
	}
	var scan struct {
		Status      string
		FailureKind string `gorm:"column:failure_kind"`
	}
	if err := db.Table("scan").Where("id = 7").Take(&scan).Error; err != nil || scan.Status != scanStatusFailed || scan.FailureKind != "agent_deleted" {
		t.Fatalf("expected failed Scan convergence, scan=%+v err=%v", scan, err)
	}
	var agents int64
	if err := db.Table("agent").Where("id = 42").Count(&agents).Error; err != nil || agents != 0 {
		t.Fatalf("expected Agent deletion, count=%d err=%v", agents, err)
	}
}
