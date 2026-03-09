package repository

import (
	"context"
	"strings"
	"testing"

	scandomain "github.com/yyhuni/lunafox/server/internal/modules/scan/domain"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestScanTaskRepositoryUpdateStatus_PersistsFailureDetail(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:scan_task_update_status_failure_kind?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite failed: %v", err)
	}
	if err := db.Exec(`
		CREATE TABLE scan_task (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			scan_id INTEGER NOT NULL,
			stage INTEGER NOT NULL,
			workflow_id TEXT NOT NULL,
			status TEXT NOT NULL,
			agent_id INTEGER,
			workflow_config JSON,
			error_message TEXT,
			failure_kind TEXT NOT NULL DEFAULT '',
			created_at DATETIME,
			started_at DATETIME,
			completed_at DATETIME
		);
		INSERT INTO scan_task (id, scan_id, stage, workflow_id, status, failure_kind) VALUES (1, 10, 1, 'subdomain_discovery', 'running', '');
	`).Error; err != nil {
		t.Fatalf("setup scan_task table failed: %v", err)
	}

	repo := &scanTaskRepository{db: db}
	failure := &scandomain.FailureDetail{Kind: "runtime_error", Message: "boom"}
	if err := repo.UpdateStatus(context.Background(), 1, taskStatusFailed, failure); err != nil {
		t.Fatalf("UpdateStatus failed: %v", err)
	}

	var row struct {
		Status       string
		ErrorMessage string
		FailureKind  string
	}
	if err := db.Table("scan_task").Select("status, error_message, failure_kind").Where("id = 1").Scan(&row).Error; err != nil {
		t.Fatalf("query scan_task failed: %v", err)
	}
	if row.Status != taskStatusFailed || row.ErrorMessage != "boom" || row.FailureKind != "runtime_error" {
		t.Fatalf("unexpected scan_task row: %+v", row)
	}
}

func TestScanTaskRepositoryFailPulledTask_PersistsFailureDetail(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:scan_task_fail_claim_failure_kind?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite failed: %v", err)
	}
	if err := db.Exec(`
		CREATE TABLE scan_task (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			scan_id INTEGER NOT NULL,
			stage INTEGER NOT NULL,
			workflow_id TEXT NOT NULL,
			status TEXT NOT NULL,
			agent_id INTEGER,
			workflow_config JSON,
			error_message TEXT,
			failure_kind TEXT NOT NULL DEFAULT '',
			created_at DATETIME,
			started_at DATETIME,
			completed_at DATETIME
		);
		INSERT INTO scan_task (id, scan_id, stage, workflow_id, status, failure_kind) VALUES (1, 10, 1, 'subdomain_discovery', 'running', '');
	`).Error; err != nil {
		t.Fatalf("setup scan_task table failed: %v", err)
	}

	repo := &scanTaskRepository{db: db}
	failure := &scandomain.FailureDetail{Kind: "schema_invalid", Message: "bad workflow"}
	if err := repo.FailPulledTask(context.Background(), 1, failure); err != nil {
		t.Fatalf("FailPulledTask failed: %v", err)
	}

	var row struct {
		Status       string
		ErrorMessage string
		FailureKind  string
	}
	if err := db.Table("scan_task").Select("status, error_message, failure_kind").Where("id = 1").Scan(&row).Error; err != nil {
		t.Fatalf("query scan_task failed: %v", err)
	}
	if row.Status != taskStatusFailed || row.ErrorMessage != "bad workflow" || row.FailureKind != "schema_invalid" {
		t.Fatalf("unexpected scan_task row: %+v", row)
	}
}

func TestFailTasksSQL_ContainsAgentDisconnectedFailureKind(t *testing.T) {
	if !strings.Contains(failTasksSQL, "failure_kind") {
		t.Fatalf("expected failTasksSQL to update failure_kind")
	}
	if !strings.Contains(failTasksSQL, "agent_disconnected") {
		t.Fatalf("expected failTasksSQL to mark agent_disconnected")
	}
}

func TestCanonicalTaskFailureKind_DoesNotRewriteLegacyCamelCase(t *testing.T) {
	if got := canonicalTaskFailureKind("runtimeError"); got != "runtimeError" {
		t.Fatalf("expected legacy value preserved without compatibility rewrite, got %q", got)
	}
}
