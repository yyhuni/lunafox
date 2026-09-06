package repository

import (
	"context"
	"fmt"
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestDeleteTargetScopedForCleanupUsesScheduleCascadeAndPreservesOrganizationSchedule(t *testing.T) {
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared&_foreign_keys=on", t.Name())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	for _, statement := range []string{
		"CREATE TABLE target (id INTEGER PRIMARY KEY, name TEXT NOT NULL, deleted_at DATETIME)",
		`CREATE TABLE scheduled_scan (
			id INTEGER PRIMARY KEY, name TEXT NOT NULL, scan_workflow_id TEXT NOT NULL, configuration TEXT NOT NULL,
			organization_id INTEGER, target_id INTEGER, agent_id INTEGER, cron_expression TEXT NOT NULL,
			is_enabled BOOLEAN NOT NULL, run_count INTEGER NOT NULL, last_run_time DATETIME, next_run_time DATETIME,
			created_at DATETIME NOT NULL, updated_at DATETIME NOT NULL
		)`,
		`CREATE TABLE scheduled_scan_occurrence (
			id INTEGER PRIMARY KEY, scheduled_scan_id INTEGER NOT NULL REFERENCES scheduled_scan(id) ON DELETE CASCADE,
			scheduled_for DATETIME NOT NULL, attempted_at DATETIME, dispatched_at DATETIME, failure_kind TEXT, failure_message TEXT,
			created_at DATETIME NOT NULL
		)`,
	} {
		if err := db.Exec(statement).Error; err != nil {
			t.Fatalf("create fixture table: %v", err)
		}
	}
	now := time.Date(2026, 8, 6, 12, 0, 0, 0, time.UTC)
	if err := db.Exec("INSERT INTO target (id, name, deleted_at) VALUES (1, 'deleted.example', ?)", now).Error; err != nil {
		t.Fatalf("seed tombstone: %v", err)
	}
	if err := db.Exec(`INSERT INTO scheduled_scan
		(id, name, scan_workflow_id, configuration, target_id, cron_expression, is_enabled, run_count, next_run_time, created_at, updated_at)
		VALUES (10, 'target-owned', 'default', '{}', 1, '* * * * *', TRUE, 0, ?, ?, ?),
		(11, 'organization-owned', 'default', '{}', NULL, '* * * * *', TRUE, 0, ?, ?, ?)`, now, now, now, now, now, now).Error; err != nil {
		t.Fatalf("seed Schedules: %v", err)
	}
	if err := db.Exec("INSERT INTO scheduled_scan_occurrence (id, scheduled_scan_id, scheduled_for, created_at) VALUES (100, 10, ?, ?)", now, now).Error; err != nil {
		t.Fatalf("seed occurrence: %v", err)
	}

	repo := NewScheduledScanRepository(db)
	deleted, err := repo.DeleteTargetScopedForCleanup(context.Background(), 1)
	if err != nil || deleted != 1 {
		t.Fatalf("DeleteTargetScopedForCleanup() = %d, %v", deleted, err)
	}
	for _, check := range []struct {
		table string
		id    int
		want  int64
	}{
		{table: "scheduled_scan", id: 10, want: 0},
		{table: "scheduled_scan_occurrence", id: 100, want: 0},
		{table: "scheduled_scan", id: 11, want: 1},
	} {
		var count int64
		if err := db.Table(check.table).Where("id = ?", check.id).Count(&count).Error; err != nil {
			t.Fatalf("count %s/%d: %v", check.table, check.id, err)
		}
		if count != check.want {
			t.Fatalf("%s/%d count = %d, want %d", check.table, check.id, count, check.want)
		}
	}
}
