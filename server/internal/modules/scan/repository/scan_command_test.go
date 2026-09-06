package repository

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	scandomain "github.com/yyhuni/lunafox/server/internal/modules/scan/domain"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestScanRepositoryUpdateScanStatus_RejectsEmptyFailureKind(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:scan_update_status_failure_kind?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite failed: %v", err)
	}
	if err := db.Exec(`
		CREATE TABLE scan (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			status TEXT NOT NULL,
			error_message TEXT NOT NULL DEFAULT '',
			failure_kind TEXT NOT NULL DEFAULT '',
			stopped_at DATETIME
		);
		INSERT INTO scan (id, status, error_message, failure_kind) VALUES (1, 'running', '', '');
	`).Error; err != nil {
		t.Fatalf("setup scan table failed: %v", err)
	}

	repo := &ScanRepository{db: db}
	failure := &scandomain.FailureDetail{Kind: "", Message: "task timed out"}
	if err := repo.UpdateScanStatus(1, scanStatusFailed, failure); err == nil {
		t.Fatal("expected UpdateScanStatus to reject empty failure kind")
	}
}

func TestScanRepositoryUpdateScanStatus_SucceededUpdatesTargetLastScannedAt(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:scan_update_success_target_last_scanned?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite failed: %v", err)
	}
	if err := db.Exec(`
		CREATE TABLE target (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			last_scanned_at DATETIME
		);
		CREATE TABLE scan (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			target_id INTEGER NOT NULL,
			status TEXT NOT NULL,
			error_message TEXT NOT NULL DEFAULT '',
			failure_kind TEXT NOT NULL DEFAULT '',
			stopped_at DATETIME
		);
		INSERT INTO target (id, last_scanned_at) VALUES (7, NULL);
		INSERT INTO scan (id, target_id, status, error_message, failure_kind) VALUES (11, 7, 'running', '', '');
	`).Error; err != nil {
		t.Fatalf("setup tables failed: %v", err)
	}

	repo := &ScanRepository{db: db}
	if err := repo.UpdateScanStatus(11, scanStatusSucceeded, nil); err != nil {
		t.Fatalf("UpdateScanStatus failed: %v", err)
	}

	var row struct {
		ScanStoppedAt       *time.Time
		TargetLastScannedAt *time.Time
	}
	if err := db.Raw(`
		SELECT scan.stopped_at AS scan_stopped_at, target.last_scanned_at AS target_last_scanned_at
		FROM scan
		JOIN target ON target.id = scan.target_id
		WHERE scan.id = 11
	`).Scan(&row).Error; err != nil {
		t.Fatalf("query scan target timestamps failed: %v", err)
	}
	if row.ScanStoppedAt == nil {
		t.Fatal("expected scan stopped_at to be set")
	}
	if row.TargetLastScannedAt == nil {
		t.Fatal("expected target last_scanned_at to be set")
	}
	firstStoppedAt := *row.ScanStoppedAt
	firstLastScannedAt := *row.TargetLastScannedAt
	if err := repo.UpdateScanStatus(11, scanStatusSucceeded, nil); err != nil {
		t.Fatalf("duplicate UpdateScanStatus failed: %v", err)
	}
	if err := db.Raw(`
		SELECT scan.stopped_at AS scan_stopped_at, target.last_scanned_at AS target_last_scanned_at
		FROM scan
		JOIN target ON target.id = scan.target_id
		WHERE scan.id = 11
	`).Scan(&row).Error; err != nil {
		t.Fatalf("query duplicate scan target timestamps failed: %v", err)
	}
	if row.ScanStoppedAt == nil || !row.ScanStoppedAt.Equal(firstStoppedAt) || row.TargetLastScannedAt == nil || !row.TargetLastScannedAt.Equal(firstLastScannedAt) {
		t.Fatalf("duplicate terminal reconciliation changed timestamps: stopped=%v last_scanned=%v", row.ScanStoppedAt, row.TargetLastScannedAt)
	}
}

func TestScanRepositoryUpdateScanStatus_FailedDoesNotUpdateTargetLastScannedAt(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:scan_update_failed_target_last_scanned?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite failed: %v", err)
	}
	if err := db.Exec(`
		CREATE TABLE target (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			last_scanned_at DATETIME
		);
		CREATE TABLE scan (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			target_id INTEGER NOT NULL,
			status TEXT NOT NULL,
			error_message TEXT NOT NULL DEFAULT '',
			failure_kind TEXT NOT NULL DEFAULT '',
			stopped_at DATETIME
		);
		INSERT INTO target (id, last_scanned_at) VALUES (7, NULL);
		INSERT INTO scan (id, target_id, status, error_message, failure_kind) VALUES (11, 7, 'running', '', '');
	`).Error; err != nil {
		t.Fatalf("setup tables failed: %v", err)
	}

	repo := &ScanRepository{db: db}
	failure := &scandomain.FailureDetail{Kind: "runtime_error", Message: "engine crashed"}
	if err := repo.UpdateScanStatus(11, scanStatusFailed, failure); err != nil {
		t.Fatalf("UpdateScanStatus failed: %v", err)
	}

	var row struct {
		LastScannedAt *time.Time
	}
	if err := db.Table("target").Select("last_scanned_at").Where("id = ?", 7).Scan(&row).Error; err != nil {
		t.Fatalf("query target failed: %v", err)
	}
	if row.LastScannedAt != nil {
		t.Fatalf("failed scan must not update target last_scanned_at, got %v", row.LastScannedAt)
	}
}

func TestScanRepositoryUpdateScanStatus_DoesNotReplaceTerminalOutcomeOrEmitAnotherNotification(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:scan_update_terminal_immutable?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite failed: %v", err)
	}
	if err := db.Exec(`
		CREATE TABLE scan (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			status TEXT NOT NULL,
			error_message TEXT NOT NULL DEFAULT '',
			failure_kind TEXT NOT NULL DEFAULT '',
			stopped_at DATETIME
		);
		INSERT INTO scan (id, status, error_message, failure_kind) VALUES (11, 'succeeded', '', '');
	`).Error; err != nil {
		t.Fatalf("setup scan table failed: %v", err)
	}

	sink := &scanTerminalNotificationSinkFake{}
	repo := NewScanRepository(db, sink)
	failure := &scandomain.FailureDetail{Kind: "runtime_error", Message: "late task report"}
	if err := repo.UpdateScanStatus(11, scanStatusFailed, failure); err != nil {
		t.Fatalf("late terminal update failed: %v", err)
	}

	var row struct {
		Status       string
		ErrorMessage string
		FailureKind  string
	}
	if err := db.Table("scan").Select("status, error_message, failure_kind").Where("id = ?", 11).Scan(&row).Error; err != nil {
		t.Fatalf("read scan after late terminal update: %v", err)
	}
	if row.Status != scanStatusSucceeded || row.ErrorMessage != "" || row.FailureKind != "" {
		t.Fatalf("terminal scan changed after late report: %#v", row)
	}
	if sink.calls != 0 {
		t.Fatalf("late terminal update emitted %d notification candidates, want none", sink.calls)
	}
}

type scanTerminalNotificationSinkFake struct {
	calls int
}

func (sink *scanTerminalNotificationSinkFake) WriteScanTerminal(*gorm.DB, int, string, string, string, time.Time) error {
	sink.calls++
	return nil
}

func TestScanRepositoryRefreshScanResultSummary(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:scan_refresh_scan_result_summary?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite failed: %v", err)
	}
	if err := db.Exec(`
		CREATE TABLE scan (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			target_id INTEGER NOT NULL,
			deleted_at DATETIME,
			cached_subdomains_count INTEGER DEFAULT 0,
			cached_websites_count INTEGER DEFAULT 0,
			cached_endpoints_count INTEGER DEFAULT 0,
			cached_ips_count INTEGER DEFAULT 0,
			cached_directories_count INTEGER DEFAULT 0,
			cached_screenshots_count INTEGER DEFAULT 0,
			cached_vulns_total INTEGER DEFAULT 0,
			cached_vulns_critical INTEGER DEFAULT 0,
			cached_vulns_high INTEGER DEFAULT 0,
			cached_vulns_medium INTEGER DEFAULT 0,
			cached_vulns_low INTEGER DEFAULT 0,
			stats_updated_at DATETIME
		);
		CREATE TABLE subdomain_snapshot (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			scan_id INTEGER NOT NULL,
			dns_name TEXT NOT NULL
		);
		CREATE TABLE host_port_mapping_snapshot (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			scan_id INTEGER NOT NULL,
			host TEXT NOT NULL,
			ip TEXT NOT NULL,
			port INTEGER NOT NULL
		);
		CREATE TABLE website_snapshot (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			scan_id INTEGER NOT NULL,
			url TEXT NOT NULL
		);
		CREATE TABLE endpoint_snapshot (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			scan_id INTEGER NOT NULL,
			url TEXT NOT NULL
		);
		CREATE TABLE directory_snapshot (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			scan_id INTEGER NOT NULL,
			url TEXT NOT NULL
		);
		CREATE TABLE screenshot_snapshot (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			scan_id INTEGER NOT NULL,
			url TEXT NOT NULL
		);
		CREATE TABLE vulnerability_snapshot (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			scan_id INTEGER NOT NULL,
			severity TEXT NOT NULL
		);
		INSERT INTO scan (id, target_id) VALUES (11, 7);
		INSERT INTO subdomain_snapshot (scan_id, dns_name) VALUES
			(11, 'api.example.com'),
			(11, 'admin.example.com'),
			(12, 'other.example.com');
		INSERT INTO host_port_mapping_snapshot (scan_id, host, ip, port) VALUES
			(11, 'api.example.com', '192.0.2.10', 443),
			(11, 'admin.example.com', '192.0.2.11', 443),
			(12, 'other.example.com', '192.0.2.12', 443);
		INSERT INTO website_snapshot (scan_id, url) VALUES
			(11, 'https://example.com'),
			(12, 'https://other.example.com');
		INSERT INTO endpoint_snapshot (scan_id, url) VALUES
			(11, 'https://example.com/api'),
			(11, 'https://example.com/login'),
			(12, 'https://other.example.com/api');
		INSERT INTO directory_snapshot (scan_id, url) VALUES
			(11, 'https://example.com/admin');
		INSERT INTO screenshot_snapshot (scan_id, url) VALUES
			(11, 'https://example.com'),
			(11, 'https://example.com/login');
		INSERT INTO vulnerability_snapshot (scan_id, severity) VALUES
			(11, 'critical'),
			(11, 'high'),
			(11, 'medium'),
			(11, 'low'),
			(11, 'info'),
			(12, 'critical');
	`).Error; err != nil {
		t.Fatalf("setup tables failed: %v", err)
	}

	repo := &ScanRepository{db: db}
	if err := repo.RefreshScanResultSummary(context.Background(), 11, 7); err != nil {
		t.Fatalf("RefreshScanResultSummary failed: %v", err)
	}

	var row struct {
		CachedSubdomainsCount  int
		CachedWebsitesCount    int
		CachedEndpointsCount   int
		CachedIPsCount         int
		CachedDirectoriesCount int
		CachedScreenshotsCount int
		CachedVulnsTotal       int
		CachedVulnsCritical    int
		CachedVulnsHigh        int
		CachedVulnsMedium      int
		CachedVulnsLow         int
		StatsUpdatedAt         *time.Time
	}
	if err := db.Table("scan").Where("id = ?", 11).Scan(&row).Error; err != nil {
		t.Fatalf("query scan summary failed: %v", err)
	}
	if row.CachedSubdomainsCount != 2 {
		t.Fatalf("expected cached_subdomains_count=2, got %d", row.CachedSubdomainsCount)
	}
	if row.CachedIPsCount != 2 || row.CachedWebsitesCount != 1 || row.CachedEndpointsCount != 2 || row.CachedDirectoriesCount != 1 || row.CachedScreenshotsCount != 2 {
		t.Fatalf("unexpected asset summary: %+v", row)
	}
	if row.CachedVulnsTotal != 5 || row.CachedVulnsCritical != 1 || row.CachedVulnsHigh != 1 || row.CachedVulnsMedium != 1 || row.CachedVulnsLow != 1 {
		t.Fatalf("unexpected vulnerability summary: %+v", row)
	}
	if row.StatsUpdatedAt == nil {
		t.Fatal("expected stats_updated_at to be set")
	}
}

// --- CreateWithScanTasks tests ---

const createWithScanTasksScanDDL = `
CREATE TABLE scan (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	target_id INTEGER NOT NULL,
	scan_workflow_id TEXT NOT NULL,
	configuration TEXT,
	input_source TEXT NOT NULL CHECK (input_source IN ('scan_snapshot', 'target_inventory')),
	trigger_type TEXT NOT NULL CHECK (trigger_type IN ('manual', 'scheduled', 'ai')),
	status TEXT DEFAULT 'pending',
	results_dir TEXT DEFAULT '',
	container_ids TEXT DEFAULT '[]',
	agent_id INTEGER,
	assignment_mode TEXT NOT NULL DEFAULT 'automatic',
	error_message TEXT DEFAULT '',
	failure_kind TEXT DEFAULT '',
	progress INTEGER DEFAULT 0,
	current_stage TEXT DEFAULT '',
	stage_progress TEXT DEFAULT '{}',
	created_at DATETIME,
	stopped_at DATETIME,
	deleted_at DATETIME,
	cached_subdomains_count INTEGER DEFAULT 0,
	cached_websites_count INTEGER DEFAULT 0,
	cached_endpoints_count INTEGER DEFAULT 0,
	cached_ips_count INTEGER DEFAULT 0,
	cached_directories_count INTEGER DEFAULT 0,
	cached_screenshots_count INTEGER DEFAULT 0,
	cached_vulns_total INTEGER DEFAULT 0,
	cached_vulns_critical INTEGER DEFAULT 0,
	cached_vulns_high INTEGER DEFAULT 0,
	cached_vulns_medium INTEGER DEFAULT 0,
	cached_vulns_low INTEGER DEFAULT 0,
	stats_updated_at DATETIME
);
CREATE TABLE scan_blacklist_snapshot (
	scan_id INTEGER PRIMARY KEY,
	patterns TEXT NOT NULL
);`

const createWithScanTasksTargetDDL = `
CREATE TABLE target (
	id INTEGER PRIMARY KEY,
	deleted_at DATETIME
);
INSERT INTO target (id, deleted_at) VALUES
	(1, NULL), (2, NULL), (3, NULL), (4, NULL), (5, NULL), (6, NULL),
	(7, NULL), (8, NULL), (9, NULL), (10, NULL), (11, NULL);`

func setupActiveTargetsForScanCreate(t *testing.T, db *gorm.DB) {
	t.Helper()
	if err := db.Exec(createWithScanTasksTargetDDL).Error; err != nil {
		t.Fatalf("setup active Targets: %v", err)
	}
}

func resolveEmptyEffectiveBlacklistPatterns(context.Context, int) ([]string, error) {
	return []string{}, nil
}

const createWithScanTasksScanTaskDDL = `
CREATE TABLE scan_task (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	scan_id INTEGER NOT NULL,
	stage_order INTEGER NOT NULL DEFAULT 0,
	stage_id TEXT NOT NULL,
	step_order INTEGER NOT NULL DEFAULT 0,
	step_id TEXT NOT NULL,
	engine_id TEXT NOT NULL,
	engine_config TEXT NOT NULL DEFAULT '{}',
	task_execution_config TEXT NOT NULL DEFAULT '{}',
	status TEXT NOT NULL DEFAULT 'pending',
	assigned_agent_id INTEGER,
	assigned_session_id TEXT,
	assigned_session_epoch INTEGER,
	assigned_request_id TEXT,
	resolved_execution_plan BLOB NOT NULL DEFAULT '',
	terminal_reconciliation_pending BOOLEAN NOT NULL DEFAULT FALSE,
	error_message TEXT DEFAULT '',
	failure_kind TEXT DEFAULT '',
	failure_detail TEXT DEFAULT '',
	engine_diagnostics TEXT,
	skip_reason TEXT DEFAULT '',
	created_at DATETIME,
	started_at DATETIME,
	completed_at DATETIME
);`

func TestScanRepositoryCreateWithScanTasksAndPlans_PersistsScanAndScanTasks(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:create_with_scan_task_happy?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite failed: %v", err)
	}
	if err := db.Exec(createWithScanTasksScanDDL).Error; err != nil {
		t.Fatalf("setup scan table failed: %v", err)
	}
	setupActiveTargetsForScanCreate(t, db)
	if err := db.Exec(createWithScanTasksScanTaskDDL).Error; err != nil {
		t.Fatalf("setup scan_task table failed: %v", err)
	}

	repo := NewScanRepository(db)
	scan := &ScanCreateRecord{
		TargetID:       1,
		ScanWorkflowID: "subdomain_discovery",
		InputSource:    scandomain.InputSourceScanSnapshot,
		TriggerType:    scandomain.ScanTriggerTypeManual,
		Status:         "pending",
		ScanTasks: []CreateScanTaskRecord{{
			StageOrder:          1,
			StageID:             "discovery",
			StepOrder:           1,
			StepID:              "subdomain_discovery",
			EngineID:            "engine.lunafox.subdomain_discovery",
			EngineConfig:        map[string]any{"enabled": true},
			TaskExecutionConfig: map[string]any{"workflowStepExecution": map[string]any{"schemaVersion": 1, "scanWorkflowId": "subdomain_discovery", "step": map[string]any{"stageId": "discovery", "stepId": "subdomain_discovery", "engineId": "engine.lunafox.subdomain_discovery", "engineConfig": map[string]any{"enabled": true}}}},
			Status:              "skipped",
			SkipReason:          "target_not_applicable",
		}},
	}

	if err := repo.CreateWithScanTasksAndPlans(context.Background(), scan, resolveEmptyEffectiveBlacklistPatterns, func(_ int, _ int, _ *scandomain.CreateScanTask) error { return nil }); err != nil {
		t.Fatalf("CreateWithScanTasksAndPlans failed: %v", err)
	}
	if scan.ID == 0 {
		t.Fatal("expected scan ID to be populated after persistence")
	}

	// Verify scan row.
	var scanRow struct {
		ID             int
		TargetID       int
		ScanWorkflowID string
		Status         string
	}
	if err := db.Table("scan").Select("id, target_id, scan_workflow_id, status").Where("id = ?", scan.ID).Scan(&scanRow).Error; err != nil {
		t.Fatalf("query scan failed: %v", err)
	}
	if scanRow.TargetID != 1 || scanRow.ScanWorkflowID != "subdomain_discovery" || scanRow.Status != "pending" {
		t.Fatalf("unexpected scan row: %+v", scanRow)
	}

	var scanTaskRow struct {
		ScanID              int
		StageOrder          int
		StageID             string
		StepOrder           int
		StepID              string
		EngineID            string
		EngineConfig        string
		TaskExecutionConfig string
		Status              string
		SkipReason          string
	}
	if err := db.Table("scan_task").Select("scan_id, stage_order, stage_id, step_order, step_id, engine_id, engine_config, task_execution_config, status, skip_reason").Where("scan_id = ?", scan.ID).Scan(&scanTaskRow).Error; err != nil {
		t.Fatalf("query scan_task failed: %v", err)
	}
	if scanTaskRow.ScanID != scan.ID || scanTaskRow.StageOrder != 1 || scanTaskRow.StageID != "discovery" || scanTaskRow.StepOrder != 1 || scanTaskRow.StepID != "subdomain_discovery" || scanTaskRow.EngineID != "engine.lunafox.subdomain_discovery" || scanTaskRow.Status != "skipped" {
		t.Fatalf("unexpected scan_task row: %+v", scanTaskRow)
	}
	if scanTaskRow.SkipReason != "target_not_applicable" {
		t.Fatalf("expected skip reason persisted, got %+v", scanTaskRow)
	}
	if scanTaskRow.EngineConfig == "" || scanTaskRow.EngineConfig == "{}" || scanTaskRow.EngineConfig == "null" {
		t.Fatalf("expected scan task engine_config persisted, got %q", scanTaskRow.EngineConfig)
	}
	if scanTaskRow.TaskExecutionConfig == "" || scanTaskRow.TaskExecutionConfig == "{}" || scanTaskRow.TaskExecutionConfig == "null" {
		t.Fatalf("expected scan task task_execution_config persisted, got %q", scanTaskRow.TaskExecutionConfig)
	}
}

func TestScanRepositoryCreateWithScanTasksAndPlans_WithoutScanTasks(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:create_with_scan_task_no_tasks?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite failed: %v", err)
	}
	if err := db.Exec(createWithScanTasksScanDDL).Error; err != nil {
		t.Fatalf("setup scan table failed: %v", err)
	}
	setupActiveTargetsForScanCreate(t, db)

	repo := NewScanRepository(db)
	scan := &ScanCreateRecord{
		TargetID:       2,
		ScanWorkflowID: "full_scan",
		InputSource:    scandomain.InputSourceScanSnapshot,
		TriggerType:    scandomain.ScanTriggerTypeManual,
		Status:         "pending",
	}

	if err := repo.CreateWithScanTasksAndPlans(context.Background(), scan, resolveEmptyEffectiveBlacklistPatterns, func(_ int, _ int, _ *scandomain.CreateScanTask) error { return nil }); err != nil {
		t.Fatalf("CreateWithScanTasksAndPlans without workflow execution failed: %v", err)
	}
	if scan.ID == 0 {
		t.Fatal("expected scan ID to be populated")
	}

}

func TestScanRepositoryCreateWithScanTasksAndPlansRejectsUnavailableTargetBeforeDependentWrites(t *testing.T) {
	tests := []struct {
		name      string
		targetID  int
		tombstone bool
	}{
		{name: "never existed", targetID: 99},
		{name: "tombstoned", targetID: 1, tombstone: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			db, err := gorm.Open(sqlite.Open("file:"+strings.ReplaceAll(t.Name(), "/", "_")+"?mode=memory&cache=shared"), &gorm.Config{})
			if err != nil {
				t.Fatalf("open sqlite failed: %v", err)
			}
			if err := db.Exec(createWithScanTasksScanDDL).Error; err != nil {
				t.Fatalf("setup tables failed: %v", err)
			}
			setupActiveTargetsForScanCreate(t, db)
			if test.tombstone {
				if err := db.Exec("UPDATE target SET deleted_at = CURRENT_TIMESTAMP WHERE id = ?", test.targetID).Error; err != nil {
					t.Fatalf("tombstone target: %v", err)
				}
			}

			resolverCalled := false
			scan := &ScanCreateRecord{TargetID: test.targetID, ScanWorkflowID: "default", InputSource: scandomain.InputSourceScanSnapshot, TriggerType: scandomain.ScanTriggerTypeManual, Status: "pending"}
			err = NewScanRepository(db).CreateWithScanTasksAndPlans(context.Background(), scan, func(context.Context, int) ([]string, error) {
				resolverCalled = true
				return []string{}, nil
			}, func(int, int, *scandomain.CreateScanTask) error { return nil })
			if !errors.Is(err, gorm.ErrRecordNotFound) {
				t.Fatalf("CreateWithScanTasksAndPlans() error = %v, want record not found", err)
			}
			if resolverCalled {
				t.Fatal("effective policy resolver ran after the Target fence failed")
			}
			assertNoScanOrBlacklistSnapshot(t, db)
		})
	}
}

func TestScanRepositoryCreateWithScanTasksAndPlans_RollsBackOnScanTaskInsertFailure(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:create_with_scan_task_rollback?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite failed: %v", err)
	}
	if err := db.Exec(createWithScanTasksScanDDL).Error; err != nil {
		t.Fatalf("setup scan table failed: %v", err)
	}
	setupActiveTargetsForScanCreate(t, db)
	// Do NOT create scan_task table → scan task INSERT fails and rolls back.

	repo := NewScanRepository(db)
	scan := &ScanCreateRecord{
		TargetID:       1,
		ScanWorkflowID: "subdomain_discovery",
		InputSource:    scandomain.InputSourceScanSnapshot,
		TriggerType:    scandomain.ScanTriggerTypeManual,
		Status:         "pending",
		ScanTasks: []CreateScanTaskRecord{{
			StageOrder:          1,
			StageID:             "discovery",
			StepOrder:           1,
			StepID:              "subdomain_discovery",
			EngineID:            "engine.lunafox.subdomain_discovery",
			TaskExecutionConfig: map[string]any{"workflowStepExecution": map[string]any{"schemaVersion": 1, "scanWorkflowId": "subdomain_discovery", "step": map[string]any{"stageId": "discovery", "stepId": "subdomain_discovery", "engineId": "engine.lunafox.subdomain_discovery", "engineConfig": map[string]any{}}}},
			Status:              "pending",
		}},
	}

	if err := repo.CreateWithScanTasksAndPlans(context.Background(), scan, resolveEmptyEffectiveBlacklistPatterns, func(_ int, _ int, _ *scandomain.CreateScanTask) error { return nil }); err == nil {
		t.Fatal("expected CreateWithScanTasksAndPlans to fail when scan_task table is missing")
	}

	// Verify scan was rolled back (no rows in scan table).
	var count int64
	if err := db.Table("scan").Count(&count).Error; err != nil {
		t.Fatalf("count scan failed: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected 0 scans after rollback, got %d", count)
	}
}

func TestScanRepositoryCreateWithScanTasksAndPlansHonorsCanceledContext(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:create_with_scan_task_canceled?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite failed: %v", err)
	}
	if err := db.Exec(createWithScanTasksScanDDL).Error; err != nil {
		t.Fatalf("setup scan table failed: %v", err)
	}
	setupActiveTargetsForScanCreate(t, db)
	if err := db.Exec(createWithScanTasksScanTaskDDL).Error; err != nil {
		t.Fatalf("setup scan_task table failed: %v", err)
	}
	repo := NewScanRepository(db)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	scan := &ScanCreateRecord{TargetID: 1, ScanWorkflowID: "default", InputSource: scandomain.InputSourceScanSnapshot, TriggerType: scandomain.ScanTriggerTypeManual, Status: "pending"}

	err = repo.CreateWithScanTasksAndPlans(ctx, scan, resolveEmptyEffectiveBlacklistPatterns, func(int, int, *scandomain.CreateScanTask) error { return nil })
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("CreateWithScanTasksAndPlans() error = %v, want context canceled", err)
	}
	var count int64
	if err := db.Table("scan").Count(&count).Error; err != nil {
		t.Fatalf("count scans: %v", err)
	}
	if count != 0 {
		t.Fatalf("canceled create committed %d scans", count)
	}
}
