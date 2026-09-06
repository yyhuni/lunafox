package main

import (
	"os"
	"strings"
	"testing"
)

func TestInitialSchemaCreatesScanTaskBeforeTaskProgressLog(t *testing.T) {
	sql, err := os.ReadFile("migrations/000001_init_schema.up.sql")
	if err != nil {
		t.Fatalf("read initial schema migration: %v", err)
	}

	text := string(sql)
	scanTaskPos := strings.Index(text, "CREATE TABLE IF NOT EXISTS scan_task")
	if scanTaskPos < 0 {
		t.Fatal("initial schema migration must create scan_task")
	}
	if strings.Contains(text, "CREATE TABLE IF NOT EXISTS engine_task_execution") {
		t.Fatal("initial schema migration must not create engine_task_execution")
	}

	taskProgressLogPos := strings.Index(text, "CREATE TABLE IF NOT EXISTS task_progress_log")
	if taskProgressLogPos < 0 {
		t.Fatal("initial schema migration must create task_progress_log")
	}

	if taskProgressLogPos < scanTaskPos {
		t.Fatal("task_progress_log references scan_task, so scan_task must be created first")
	}
	for _, required := range []string{
		"task_id INTEGER NOT NULL",
		"FOREIGN KEY (scan_id, task_id) REFERENCES scan_task(scan_id, id) ON DELETE CASCADE",
	} {
		if !strings.Contains(text, required) {
			t.Fatalf("task_progress_log must retain scan-scoped task reference %q", required)
		}
	}
}
