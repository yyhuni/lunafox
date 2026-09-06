package repository

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"github.com/yyhuni/lunafox/contracts/resourcenames"
	scandomain "github.com/yyhuni/lunafox/server/internal/modules/scan/domain"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestSavedExecutionPlanReaderRejectsMissingAndCorruptPlans(t *testing.T) {
	db := openSavedExecutionPlanTestDB(t)
	if err := db.Exec(`CREATE TABLE scan (id INTEGER PRIMARY KEY, input_source TEXT NOT NULL CHECK (input_source IN ('scan_snapshot', 'target_inventory')), agent_id INTEGER, deleted_at DATETIME)`).Error; err != nil {
		t.Fatalf("create scan table: %v", err)
	}
	if err := db.Exec(`CREATE TABLE scan_task (id INTEGER PRIMARY KEY, scan_id INTEGER NOT NULL, status TEXT NOT NULL, assigned_agent_id INTEGER, assigned_session_id TEXT, assigned_session_epoch INTEGER, assigned_request_id TEXT, terminal_reconciliation_pending BOOLEAN NOT NULL DEFAULT FALSE, resolved_execution_plan BLOB NOT NULL DEFAULT '')`).Error; err != nil {
		t.Fatalf("create scan_task table: %v", err)
	}
	if err := db.Exec(`INSERT INTO scan (id, input_source, agent_id) VALUES (7, 'scan_snapshot', 42)`).Error; err != nil {
		t.Fatalf("insert scan: %v", err)
	}
	if err := db.Exec(`INSERT INTO scan_task (id, scan_id, status) VALUES (1, 7, 'pending'), (2, 7, 'skipped'), (3, 7, 'pending')`).Error; err != nil {
		t.Fatalf("insert rows: %v", err)
	}
	repo := &scanTaskRepository{db: db}
	if _, err := repo.GetSavedExecutionPlanLease(nil, 1); err == nil {
		t.Fatal("expected nil context to fail fast")
	}
	if _, err := repo.GetSavedExecutionPlan(context.Background(), 1); err == nil {
		t.Fatal("expected missing plan to fail closed")
	}
	plan, err := repo.GetSavedExecutionPlan(context.Background(), 2)
	if err != nil || plan != nil {
		t.Fatalf("expected skipped task to return no plan, got %#v, %v", plan, err)
	}
	if err := db.Exec(`UPDATE scan_task SET resolved_execution_plan = ? WHERE id = 3`, []byte("not protobuf")).Error; err != nil {
		t.Fatalf("update corrupt row: %v", err)
	}
	if _, err := repo.GetSavedExecutionPlan(context.Background(), 3); err == nil {
		t.Fatal("expected corrupt plan to fail closed")
	}
}

func TestSavedExecutionPlanLeaseReadsTaskAgentAndEpochInOneSnapshot(t *testing.T) {
	db := openSavedExecutionPlanTestDB(t)
	if err := db.Exec(`CREATE TABLE scan (id INTEGER PRIMARY KEY, input_source TEXT NOT NULL CHECK (input_source IN ('scan_snapshot', 'target_inventory')), agent_id INTEGER, deleted_at DATETIME)`).Error; err != nil {
		t.Fatalf("create scan table: %v", err)
	}
	if err := db.Exec(`CREATE TABLE scan_task (id INTEGER PRIMARY KEY, scan_id INTEGER NOT NULL, status TEXT NOT NULL, assigned_agent_id INTEGER, assigned_session_id TEXT, assigned_session_epoch INTEGER, assigned_request_id TEXT, terminal_reconciliation_pending BOOLEAN NOT NULL DEFAULT FALSE, resolved_execution_plan BLOB NOT NULL DEFAULT '')`).Error; err != nil {
		t.Fatalf("create scan_task table: %v", err)
	}
	if err := db.Exec(`INSERT INTO scan (id, input_source, agent_id) VALUES (7, 'scan_snapshot', 42)`).Error; err != nil {
		t.Fatalf("insert scan: %v", err)
	}
	savedPlan := mustSavedPlanForTask(t, resourcenames.Task(7, 9), resourcenames.Scan(7), "engine.lunafox.port_scan", "ports", "run")
	if err := db.Exec(`INSERT INTO scan_task (id, scan_id, status, assigned_agent_id, assigned_session_id, assigned_session_epoch, resolved_execution_plan) VALUES (9, 7, 'running', 42, 'session-a', 11, ?)`, savedPlan).Error; err != nil {
		t.Fatalf("insert task: %v", err)
	}

	repo := &scanTaskRepository{db: db}
	lease, err := repo.GetSavedExecutionPlanLease(context.Background(), 9)
	if err != nil {
		t.Fatalf("GetSavedExecutionPlanLease failed: %v", err)
	}
	if lease.TaskID != 9 || lease.ScanID != 7 || lease.Status != "running" || lease.AgentID == nil || *lease.AgentID != 42 || lease.AssignedSessionID == nil || *lease.AssignedSessionID != "session-a" || lease.AssignedSessionEpoch == nil || *lease.AssignedSessionEpoch != 11 || string(lease.ResolvedExecutionPlan) != string(savedPlan) {
		t.Fatalf("unexpected lease snapshot: %#v", lease)
	}

	if _, err := repo.GetSavedExecutionPlanLease(context.Background(), 404); !errors.Is(err, scandomain.ErrSavedExecutionPlanLeaseNotFound) {
		t.Fatalf("missing lease error = %v, want ErrSavedExecutionPlanLeaseNotFound", err)
	}
}

func TestSavedExecutionPlanLeaseRejectsPlanFromDifferentPersistedScope(t *testing.T) {
	db := openSavedExecutionPlanTestDB(t)
	if err := db.Exec(`CREATE TABLE scan (id INTEGER PRIMARY KEY, input_source TEXT NOT NULL CHECK (input_source IN ('scan_snapshot', 'target_inventory')), agent_id INTEGER, deleted_at DATETIME)`).Error; err != nil {
		t.Fatalf("create scan table: %v", err)
	}
	if err := db.Exec(`CREATE TABLE scan_task (id INTEGER PRIMARY KEY, scan_id INTEGER NOT NULL, status TEXT NOT NULL, assigned_agent_id INTEGER, assigned_session_id TEXT, assigned_session_epoch INTEGER, assigned_request_id TEXT, terminal_reconciliation_pending BOOLEAN NOT NULL DEFAULT FALSE, resolved_execution_plan BLOB NOT NULL DEFAULT '')`).Error; err != nil {
		t.Fatalf("create scan_task table: %v", err)
	}
	if err := db.Exec(`INSERT INTO scan (id, input_source, agent_id) VALUES (7, 'scan_snapshot', 42), (8, 'scan_snapshot', 42)`).Error; err != nil {
		t.Fatalf("insert scans: %v", err)
	}
	if err := db.Exec(`INSERT INTO scan_task (id, scan_id, status, assigned_agent_id, assigned_session_id, assigned_session_epoch) VALUES (9, 7, 'running', 42, 'session-a', 11)`).Error; err != nil {
		t.Fatalf("insert task: %v", err)
	}

	repo := &scanTaskRepository{db: db}
	for _, test := range []struct {
		name        string
		plan        []byte
		wantErrPart string
	}{
		{
			name:        "different task in same scan",
			plan:        mustSavedPlanForTask(t, resourcenames.Task(7, 10), resourcenames.Scan(7), "engine.lunafox.port_scan", "ports", "run"),
			wantErrPart: "task scope does not match persisted task",
		},
		{
			name:        "same task id in different scan",
			plan:        mustSavedPlanForTask(t, resourcenames.Task(8, 9), resourcenames.Scan(8), "engine.lunafox.port_scan", "ports", "run"),
			wantErrPart: "scan scope does not match persisted task",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			if err := db.Exec(`UPDATE scan_task SET resolved_execution_plan = ? WHERE id = 9`, test.plan).Error; err != nil {
				t.Fatalf("update saved plan: %v", err)
			}
			if _, err := repo.GetSavedExecutionPlanLease(context.Background(), 9); err == nil || !strings.Contains(err.Error(), test.wantErrPart) {
				t.Fatalf("GetSavedExecutionPlanLease() error = %v, want containing %q", err, test.wantErrPart)
			}
		})
	}
}

func openSavedExecutionPlanTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "repository.db")), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("open sqlite connection: %v", err)
	}
	t.Cleanup(func() {
		if err := sqlDB.Close(); err != nil {
			t.Errorf("close sqlite: %v", err)
		}
	})
	return db
}
