package repository

import (
	"context"
	"errors"
	"testing"

	scandomain "github.com/yyhuni/lunafox/server/internal/modules/scan/domain"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

const scanTaskParentScanDDL = `
CREATE TABLE scan (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	target_id INTEGER NOT NULL,
	scan_workflow_id TEXT NOT NULL,
	configuration TEXT,
	trigger_type TEXT NOT NULL DEFAULT 'manual' CHECK (trigger_type IN ('manual', 'scheduled', 'ai')),
	status TEXT DEFAULT 'pending',
	results_dir TEXT DEFAULT '',
	container_ids TEXT DEFAULT '[]',
	agent_id INTEGER,
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
);`

const scanTaskDDL = `
CREATE TABLE scan_task (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	scan_id INTEGER NOT NULL,
	stage_order INTEGER NOT NULL DEFAULT 0,
	stage_id TEXT NOT NULL,
	step_order INTEGER NOT NULL DEFAULT 0,
	step_id TEXT NOT NULL,
	engine_id TEXT NOT NULL,
	engine_config TEXT,
	task_execution_config TEXT,
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

func setupScanTaskDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite failed: %v", err)
	}
	for _, ddl := range []string{scanTaskParentScanDDL, scanTaskDDL} {
		if err := db.Exec(ddl).Error; err != nil {
			t.Fatalf("setup ddl failed: %v", err)
		}
	}
	return db
}

func TestScanTaskRepositoryRuntimeProjectionExposesSavedPlanPresence(t *testing.T) {
	db := setupScanTaskDB(t)
	if err := db.Exec(`
		INSERT INTO scan (id, target_id, scan_workflow_id, status, configuration, created_at)
		VALUES (10, 20, 'default', 'pending', '{}', CURRENT_TIMESTAMP);
		INSERT INTO scan_task (id, scan_id, stage_id, step_id, engine_id, status, resolved_execution_plan, created_at)
		VALUES
			(10000, 10, 'ports', 'planned', 'engine.lunafox.port_scan', 'pending', X'01', CURRENT_TIMESTAMP),
			(10001, 10, 'ports', 'legacy', 'engine.lunafox.port_scan', 'pending', X'', CURRENT_TIMESTAMP);
	`).Error; err != nil {
		t.Fatalf("insert runtime projection fixtures failed: %v", err)
	}

	repo := NewScanTaskRepository(db)
	planned, err := repo.GetByID(context.Background(), 10000)
	if err != nil {
		t.Fatalf("GetByID planned task failed: %v", err)
	}
	legacy, err := repo.GetByID(context.Background(), 10001)
	if err != nil {
		t.Fatalf("GetByID legacy task failed: %v", err)
	}
	if planned == nil || !planned.HasResolvedExecutionPlan {
		t.Fatalf("planned runtime projection = %+v, want plan presence", planned)
	}
	if legacy == nil || legacy.HasResolvedExecutionPlan {
		t.Fatalf("legacy runtime projection = %+v, want no plan presence", legacy)
	}
}

func TestScanTaskRepositoryCountsSkippedTasksSeparately(t *testing.T) {
	db := setupScanTaskDB(t)
	if err := db.Exec(`
		INSERT INTO scan (id, target_id, scan_workflow_id, status, configuration, created_at)
		VALUES (10, 20, 'subdomain_discovery', 'pending', '{}', CURRENT_TIMESTAMP);
		INSERT INTO scan_task (id, scan_id, stage_order, stage_id, step_order, step_id, engine_id, engine_config, task_execution_config, status, created_at)
		VALUES
			(10000, 10, 1, 'workflow', 1, 'workflow', 'engine.lunafox.subdomain_discovery', '{}', '{}', 'skipped', CURRENT_TIMESTAMP),
			(10001, 10, 1, 'workflow', 1, 'workflow', 'engine.lunafox.subdomain_discovery', '{}', '{}', 'succeeded', CURRENT_TIMESTAMP),
			(10002, 10, 2, 'ports', 1, 'port_scan', 'engine.lunafox.port_scan', '{}', '{}', 'blocked', CURRENT_TIMESTAMP);
	`).Error; err != nil {
		t.Fatalf("insert scan task count fixture failed: %v", err)
	}
	repo := NewScanTaskRepository(db)

	pending, running, completed, failed, cancelled, skipped, err := repo.CountByStatusForScanID(context.Background(), 10)
	if err != nil {
		t.Fatalf("CountByStatusForScanID failed: %v", err)
	}
	if pending != 0 || running != 0 || completed != 1 || failed != 0 || cancelled != 0 || skipped != 1 {
		t.Fatalf("unexpected status counts pending=%d running=%d completed=%d failed=%d cancelled=%d skipped=%d", pending, running, completed, failed, cancelled, skipped)
	}
}

func TestScanTaskRepositorySkipUnstartedTasksByScanIDSkipsOnlyPendingAndBlocked(t *testing.T) {
	db := setupScanTaskDB(t)
	if err := db.Exec(`
		INSERT INTO scan (id, target_id, scan_workflow_id, status, configuration, created_at)
		VALUES (10, 20, 'staged_port_scan', 'running', '{}', CURRENT_TIMESTAMP);
		INSERT INTO scan_task (id, scan_id, stage_order, stage_id, step_order, step_id, engine_id, engine_config, task_execution_config, status, created_at)
		VALUES
			(10000, 10, 1, 'discovery', 1, 'subdomain_discovery', 'engine.lunafox.subdomain_discovery', '{}', '{}', 'failed', CURRENT_TIMESTAMP),
			(10001, 10, 2, 'ports', 1, 'port_scan', 'engine.lunafox.port_scan', '{}', '{}', 'blocked', CURRENT_TIMESTAMP),
			(10002, 10, 3, 'http', 1, 'http_probe', 'engine.lunafox.http_probe', '{}', '{}', 'pending', CURRENT_TIMESTAMP),
			(10003, 10, 4, 'later', 1, 'later', 'engine.lunafox.later', '{}', '{}', 'running', CURRENT_TIMESTAMP);
	`).Error; err != nil {
		t.Fatalf("insert skip unstarted fixture failed: %v", err)
	}
	repo := NewScanTaskRepository(db)

	affected, err := repo.SkipUnstartedTasksByScanID(context.Background(), 10, "upstream task failed")
	if err != nil {
		t.Fatalf("SkipUnstartedTasksByScanID failed: %v", err)
	}
	if affected != 2 {
		t.Fatalf("expected two unstarted tasks skipped, got %d", affected)
	}

	var rows []struct {
		ID          int
		Status      string
		SkipReason  string
		CompletedAt *string
	}
	if err := db.Table("scan_task").Select("id, status, skip_reason, completed_at").Where("scan_id = ?", 10).Order("id ASC").Scan(&rows).Error; err != nil {
		t.Fatalf("query skip unstarted rows failed: %v", err)
	}
	got := map[int]struct {
		status      string
		skipReason  string
		completedAt *string
	}{}
	for _, row := range rows {
		got[row.ID] = struct {
			status      string
			skipReason  string
			completedAt *string
		}{status: row.Status, skipReason: row.SkipReason, completedAt: row.CompletedAt}
	}
	if got[10001].status != taskStatusSkipped || got[10002].status != taskStatusSkipped {
		t.Fatalf("expected pending and blocked tasks skipped, got %+v", got)
	}
	if got[10001].skipReason != "upstream task failed" || got[10002].skipReason != "upstream task failed" {
		t.Fatalf("expected skip reason recorded, got %+v", got)
	}
	if got[10001].completedAt == nil || got[10002].completedAt == nil {
		t.Fatalf("expected skipped tasks completed_at set, got %+v", got)
	}
	if got[10003].status != taskStatusRunning || got[10003].completedAt != nil {
		t.Fatalf("expected running task untouched, got %+v", got[10003])
	}
}

func TestScanTaskRepositoryCancelUnstartedTasksByScanIDCancelsOnlyPendingAndBlocked(t *testing.T) {
	db := setupScanTaskDB(t)
	if err := db.Exec(`
		INSERT INTO scan (id, target_id, scan_workflow_id, status, configuration, created_at)
		VALUES (10, 20, 'staged_port_scan', 'running', '{}', CURRENT_TIMESTAMP);
		INSERT INTO scan_task (id, scan_id, stage_order, stage_id, step_order, step_id, engine_id, engine_config, task_execution_config, status, created_at)
		VALUES
			(10001, 10, 2, 'ports', 1, 'port_scan', 'engine.lunafox.port_scan', '{}', '{}', 'blocked', CURRENT_TIMESTAMP),
			(10002, 10, 3, 'http', 1, 'http_probe', 'engine.lunafox.http_probe', '{}', '{}', 'pending', CURRENT_TIMESTAMP),
			(10003, 10, 4, 'later', 1, 'later', 'engine.lunafox.later', '{}', '{}', 'running', CURRENT_TIMESTAMP);
	`).Error; err != nil {
		t.Fatalf("insert cancel unstarted fixture failed: %v", err)
	}
	repo := NewScanTaskRepository(db)

	affected, err := repo.CancelUnstartedTasksByScanID(context.Background(), 10)
	if err != nil {
		t.Fatalf("CancelUnstartedTasksByScanID failed: %v", err)
	}
	if affected != 2 {
		t.Fatalf("expected two unstarted tasks cancelled, got %d", affected)
	}

	var rows []struct {
		ID          int
		Status      string
		CompletedAt *string
	}
	if err := db.Table("scan_task").Select("id, status, completed_at").Where("scan_id = ?", 10).Order("id ASC").Scan(&rows).Error; err != nil {
		t.Fatalf("query cancel unstarted rows failed: %v", err)
	}
	got := map[int]struct {
		status      string
		completedAt *string
	}{}
	for _, row := range rows {
		got[row.ID] = struct {
			status      string
			completedAt *string
		}{status: row.Status, completedAt: row.CompletedAt}
	}
	if got[10001].status != taskStatusCancelled || got[10002].status != taskStatusCancelled {
		t.Fatalf("expected pending and blocked tasks cancelled, got %+v", got)
	}
	if got[10001].completedAt == nil || got[10002].completedAt == nil {
		t.Fatalf("expected cancelled tasks completed_at set, got %+v", got)
	}
	if got[10003].status != taskStatusRunning || got[10003].completedAt != nil {
		t.Fatalf("expected running task untouched, got %+v", got[10003])
	}
}

func TestScanTaskRepositoryUnlockNextStageOrderSkipsTerminalOnlyStages(t *testing.T) {
	db := setupScanTaskDB(t)
	if err := db.Exec(`
		INSERT INTO scan (id, target_id, scan_workflow_id, status, configuration, created_at)
		VALUES (10, 20, 'staged_port_scan', 'running', '{}', CURRENT_TIMESTAMP);
		INSERT INTO scan_task (id, scan_id, stage_order, stage_id, step_order, step_id, engine_id, engine_config, task_execution_config, status, created_at)
		VALUES
			(10000, 10, 1, 'discovery', 1, 'subdomain_discovery', 'engine.lunafox.subdomain_discovery', '{}', '{}', 'succeeded', CURRENT_TIMESTAMP),
			(10001, 10, 2, 'unsupported', 1, 'unsupported', 'engine.lunafox.unsupported', '{}', '{}', 'skipped', CURRENT_TIMESTAMP),
			(10002, 10, 3, 'ports', 1, 'port_scan', 'engine.lunafox.port_scan', '{}', '{}', 'blocked', CURRENT_TIMESTAMP),
			(10003, 10, 4, 'later', 1, 'later', 'engine.lunafox.later', '{}', '{}', 'blocked', CURRENT_TIMESTAMP);
	`).Error; err != nil {
		t.Fatalf("insert staged unlock fixture failed: %v", err)
	}
	repo := NewScanTaskRepository(db)

	affected, err := repo.UnlockNextStageOrder(context.Background(), 10, 1)
	if err != nil {
		t.Fatalf("UnlockNextStageOrder failed: %v", err)
	}
	if affected != 1 {
		t.Fatalf("expected one unlocked task, got %d", affected)
	}

	var rows []struct {
		ID     int
		Status string
	}
	if err := db.Table("scan_task").Select("id, status").Where("id IN ?", []int{10001, 10002, 10003}).Order("id ASC").Scan(&rows).Error; err != nil {
		t.Fatalf("query staged unlock rows failed: %v", err)
	}
	got := map[int]string{}
	for _, row := range rows {
		got[row.ID] = row.Status
	}
	if got[10001] != taskStatusSkipped || got[10002] != taskStatusPending || got[10003] != taskStatusBlocked {
		t.Fatalf("unexpected staged unlock statuses: %+v", got)
	}
}

func TestScanTaskRepositoryUnlockNextStageOrderDoesNotSkipFailedStages(t *testing.T) {
	db := setupScanTaskDB(t)
	if err := db.Exec(`
		INSERT INTO scan (id, target_id, scan_workflow_id, status, configuration, created_at)
		VALUES (10, 20, 'staged_port_scan', 'running', '{}', CURRENT_TIMESTAMP);
		INSERT INTO scan_task (id, scan_id, stage_order, stage_id, step_order, step_id, engine_id, engine_config, task_execution_config, status, created_at)
		VALUES
			(10000, 10, 1, 'discovery', 1, 'subdomain_discovery', 'engine.lunafox.subdomain_discovery', '{}', '{}', 'succeeded', CURRENT_TIMESTAMP),
			(10001, 10, 2, 'failed_stage', 1, 'failed_step', 'engine.lunafox.failed', '{}', '{}', 'failed', CURRENT_TIMESTAMP),
			(10002, 10, 3, 'ports', 1, 'port_scan', 'engine.lunafox.port_scan', '{}', '{}', 'blocked', CURRENT_TIMESTAMP);
	`).Error; err != nil {
		t.Fatalf("insert failed-stage unlock fixture failed: %v", err)
	}
	repo := NewScanTaskRepository(db)

	affected, err := repo.UnlockNextStageOrder(context.Background(), 10, 1)
	if err != nil {
		t.Fatalf("UnlockNextStageOrder failed: %v", err)
	}
	if affected != 0 {
		t.Fatalf("expected failed intermediate stage to block unlock, got %d affected rows", affected)
	}

	var status string
	if err := db.Table("scan_task").Select("status").Where("id = ?", 10002).Scan(&status).Error; err != nil {
		t.Fatalf("query port scan status failed: %v", err)
	}
	if status != taskStatusBlocked {
		t.Fatalf("expected later stage to remain blocked, got %q", status)
	}
}

func TestScanTaskRepositoryUnlockNextStageOrderIsIdempotent(t *testing.T) {
	db := setupScanTaskDB(t)
	if err := db.Exec(`
		INSERT INTO scan (id, target_id, scan_workflow_id, status, configuration, created_at)
		VALUES (10, 20, 'staged_port_scan', 'running', '{}', CURRENT_TIMESTAMP);
		INSERT INTO scan_task (id, scan_id, stage_order, stage_id, step_order, step_id, engine_id, engine_config, task_execution_config, status, created_at)
		VALUES
			(10000, 10, 1, 'discovery', 1, 'subdomain_discovery', 'engine.lunafox.subdomain_discovery', '{}', '{}', 'succeeded', CURRENT_TIMESTAMP),
			(10001, 10, 2, 'ports', 1, 'port_scan', 'engine.lunafox.port_scan', '{}', '{}', 'blocked', CURRENT_TIMESTAMP),
			(10002, 10, 3, 'later', 1, 'later', 'engine.lunafox.later', '{}', '{}', 'blocked', CURRENT_TIMESTAMP);
	`).Error; err != nil {
		t.Fatalf("insert staged unlock fixture failed: %v", err)
	}
	repo := NewScanTaskRepository(db)

	affected, err := repo.UnlockNextStageOrder(context.Background(), 10, 1)
	if err != nil {
		t.Fatalf("first UnlockNextStageOrder failed: %v", err)
	}
	if affected != 1 {
		t.Fatalf("expected one task unlocked on first call, got %d", affected)
	}
	affected, err = repo.UnlockNextStageOrder(context.Background(), 10, 1)
	if err != nil {
		t.Fatalf("second UnlockNextStageOrder failed: %v", err)
	}
	if affected != 0 {
		t.Fatalf("expected idempotent retry to affect zero rows, got %d", affected)
	}

	var rows []struct {
		ID     int
		Status string
	}
	if err := db.Table("scan_task").Select("id, status").Where("id IN ?", []int{10001, 10002}).Order("id ASC").Scan(&rows).Error; err != nil {
		t.Fatalf("query staged unlock rows failed: %v", err)
	}
	got := map[int]string{}
	for _, row := range rows {
		got[row.ID] = row.Status
	}
	if got[10001] != taskStatusPending || got[10002] != taskStatusBlocked {
		t.Fatalf("unexpected idempotent unlock statuses: %+v", got)
	}
}

func insertPendingScanTask(t *testing.T, db *gorm.DB) {
	t.Helper()
	if err := db.Exec(`
		INSERT INTO scan (id, target_id, scan_workflow_id, status, configuration, created_at)
		VALUES (10, 20, 'subdomain_discovery', 'pending', '{}', CURRENT_TIMESTAMP);
		INSERT INTO scan_task (id, scan_id, stage_order, stage_id, step_order, step_id, engine_id, engine_config, task_execution_config, status, created_at)
		VALUES (10000, 10, 1, 'discovery', 1, 'subdomain_discovery', 'engine.lunafox.subdomain_discovery', '{"threads":10}', '{"workflowStepExecution":{"schemaVersion":1,"scanWorkflowId":"subdomain_discovery","step":{"stageId":"discovery","stepId":"subdomain_discovery","engineId":"engine.lunafox.subdomain_discovery","engineConfig":{"threads":10}}}}', 'pending', CURRENT_TIMESTAMP);
	`).Error; err != nil {
		t.Fatalf("insert scan task fixture failed: %v", err)
	}
}

func TestScanTaskRepositoryListsAndClearsPendingTerminalReconciliation(t *testing.T) {
	db := setupScanTaskDB(t)
	if err := db.Exec(`
		INSERT INTO scan (id, target_id, scan_workflow_id, status, agent_id, configuration, created_at)
		VALUES (10, 20, 'subdomain_discovery', 'running', 77, '{}', CURRENT_TIMESTAMP);
		INSERT INTO scan_task (id, scan_id, stage_order, stage_id, step_order, step_id, engine_id, status, engine_diagnostics, resolved_execution_plan, assigned_session_epoch, terminal_reconciliation_pending, created_at, completed_at)
		VALUES
			(10000, 10, 1, 'discovery', 1, 'one', 'engine.lunafox.subdomain_discovery', 'succeeded', '{"compatibilityRevision":"engine-execution-diagnostics-r1","availability":"unavailable","resultState":"unknown"}', X'01', 8, TRUE, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
			(10001, 10, 1, 'discovery', 2, 'two', 'engine.lunafox.subdomain_discovery', 'succeeded', '{"compatibilityRevision":"engine-execution-diagnostics-r1","availability":"unavailable","resultState":"unknown"}', X'01', 12, TRUE, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP);
	`).Error; err != nil {
		t.Fatalf("insert pending reconciliation tasks: %v", err)
	}
	repo := NewScanTaskRepository(db)
	tasks, err := repo.ListTerminalTasksPendingReconciliation(context.Background(), 0, 1)
	if err != nil || len(tasks) != 1 || tasks[0].ID != 10000 {
		t.Fatalf("first reconciliation batch = %#v, %v", tasks, err)
	}
	superseded, err := repo.ListSupersededSessionTerminalTasksPendingReconciliation(context.Background(), 77, 10, 2)
	if err != nil || len(superseded) != 1 || superseded[0].ID != 10000 {
		t.Fatalf("superseded-session reconciliation batch = %#v, %v", superseded, err)
	}
	if err := repo.ClearTerminalTaskReconciliationPending(context.Background(), 10000); err != nil {
		t.Fatalf("clear terminal marker: %v", err)
	}
	tasks, err = repo.ListTerminalTasksPendingReconciliation(context.Background(), 0, 2)
	if err != nil || len(tasks) != 1 || tasks[0].ID != 10001 {
		t.Fatalf("remaining reconciliation batch = %#v, %v", tasks, err)
	}
}

func TestScanTaskRepositorySupersededSessionReconciliationScopePrecedesBatchLimit(t *testing.T) {
	const batchSize = 256
	db := setupScanTaskDB(t)
	if err := db.Exec(`
		INSERT INTO scan (id, target_id, scan_workflow_id, status, agent_id, configuration, created_at)
		VALUES
			(10, 20, 'default', 'running', 77, '{}', CURRENT_TIMESTAMP),
			(11, 21, 'default', 'running', 88, '{}', CURRENT_TIMESTAMP)
	`).Error; err != nil {
		t.Fatalf("insert scans: %v", err)
	}
	for taskID := 1; taskID <= batchSize; taskID++ {
		if err := db.Exec(`
			INSERT INTO scan_task (
				id, scan_id, stage_order, stage_id, step_order, step_id, engine_id, status,
				engine_diagnostics, resolved_execution_plan, assigned_session_epoch, terminal_reconciliation_pending, created_at, completed_at
			) VALUES (?, 11, 1, 'ports', ?, ?, 'engine.lunafox.port_scan', 'succeeded', '{"compatibilityRevision":"engine-execution-diagnostics-r1","availability":"unavailable","resultState":"unknown"}', X'01', 1, TRUE, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
		`, taskID, taskID, "other").Error; err != nil {
			t.Fatalf("insert unrelated marker %d: %v", taskID, err)
		}
	}
	const scopedTaskID = 1000
	if err := db.Exec(`
		INSERT INTO scan_task (
			id, scan_id, stage_order, stage_id, step_order, step_id, engine_id, status,
			engine_diagnostics, resolved_execution_plan, assigned_session_epoch, terminal_reconciliation_pending, created_at, completed_at
		) VALUES (?, 10, 1, 'ports', 1, 'scoped', 'engine.lunafox.port_scan', 'succeeded', '{"compatibilityRevision":"engine-execution-diagnostics-r1","availability":"unavailable","resultState":"unknown"}', X'01', 1, TRUE, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
	`, scopedTaskID).Error; err != nil {
		t.Fatalf("insert scoped marker: %v", err)
	}

	repo := NewScanTaskRepository(db)
	tasks, err := repo.ListSupersededSessionTerminalTasksPendingReconciliation(context.Background(), 77, 2, batchSize)
	if err != nil || len(tasks) != 1 || tasks[0].ID != scopedTaskID {
		t.Fatalf("scoped reconciliation batch = %#v, %v", tasks, err)
	}
}

func TestScanTaskRepositoryTerminalSessionCASChecksCurrentAndAssignedFullTuple(t *testing.T) {
	db := setupScanTaskDB(t)
	insertPendingScanTask(t, db)
	if err := db.Exec(`CREATE TABLE agent_runtime_status (agent_id INTEGER PRIMARY KEY, session_id TEXT NOT NULL, session_epoch INTEGER NOT NULL)`).Error; err != nil {
		t.Fatalf("create agent runtime status: %v", err)
	}
	if err := db.Exec(`INSERT INTO agent_runtime_status (agent_id, session_id, session_epoch) VALUES (77, 'session-other', 9)`).Error; err != nil {
		t.Fatalf("insert agent runtime status: %v", err)
	}
	if err := db.Exec(`UPDATE scan SET agent_id = 77 WHERE id = 10`).Error; err != nil {
		t.Fatalf("assign scan owner: %v", err)
	}
	if err := db.Exec(`UPDATE scan_task SET status = ?, assigned_agent_id = 77, assigned_session_id = 'session-a', assigned_session_epoch = 9 WHERE id = 10000`, taskStatusRunning).Error; err != nil {
		t.Fatalf("assign task lease: %v", err)
	}
	repo := NewScanTaskRepository(db)

	committed, err := repo.CommitScanTaskTerminalStatusForSession(context.Background(), 10000, 77, "session-other", 9, taskStatusSucceeded, nil)
	if err != nil || committed {
		t.Fatalf("same-epoch wrong assigned session CAS = %v, %v; want false, nil", committed, err)
	}
	if err := db.Table("agent_runtime_status").Where("agent_id = 77").Updates(map[string]any{"session_id": "session-a"}).Error; err != nil {
		t.Fatalf("restore assigned process session: %v", err)
	}
	committed, err = repo.CommitScanTaskTerminalStatusForSession(context.Background(), 10000, 77, "session-a", 9, taskStatusSucceeded, nil)
	if err != nil || !committed {
		t.Fatalf("exact terminal tuple CAS = %v, %v; want true, nil", committed, err)
	}

	if err := repo.RequireCurrentAgentExecutionSession(context.Background(), 77, "session-other", 9); !errors.Is(err, scandomain.ErrAgentExecutionSessionFenced) {
		t.Fatalf("stale persisted process session verification = %v", err)
	}
}

func TestScanTaskRepositorySupersededSessionFenceReplaysUnreconciledScan(t *testing.T) {
	db := setupScanTaskDB(t)
	insertPendingScanTask(t, db)
	if err := db.Exec(`UPDATE scan SET agent_id = 77 WHERE id = 10`).Error; err != nil {
		t.Fatalf("assign scan owner: %v", err)
	}
	if err := db.Exec(`
		UPDATE scan_task
		SET status = ?,
			assigned_agent_id = 77,
			assigned_session_id = 'session-7',
			assigned_session_epoch = 9,
			started_at = CURRENT_TIMESTAMP
		WHERE id = 10000
	`, taskStatusRunning).Error; err != nil {
		t.Fatalf("assign task lease: %v", err)
	}
	repo := NewScanTaskRepository(db)

	first, err := repo.FailTasksForSupersededAgentSession(context.Background(), 77, 10)
	if err != nil || len(first) != 1 || first[0] != 10 {
		t.Fatalf("first fence scans = %v, %v", first, err)
	}
	second, err := repo.FailTasksForSupersededAgentSession(context.Background(), 77, 10)
	if err != nil || len(second) != 1 || second[0] != 10 {
		t.Fatalf("replayed fence scans = %v, %v", second, err)
	}
	if err := db.Table("scan").Where("id = ?", 10).Update("status", taskStatusFailed).Error; err != nil {
		t.Fatalf("mark scan reconciled: %v", err)
	}
	third, err := repo.FailTasksForSupersededAgentSession(context.Background(), 77, 10)
	if err != nil || len(third) != 0 {
		t.Fatalf("reconciled fence scans = %v, %v; want empty", third, err)
	}
}
