package repository

import (
	"reflect"
	"testing"
	"time"

	model "github.com/yyhuni/lunafox/server/internal/modules/scan/repository/persistence"
	"gorm.io/datatypes"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestScanHistoryReadPreservesPersistedConfigurationWithoutEnablementInference(t *testing.T) {
	persisted := map[string]any{
		"steps": map[string]any{
			"discover": map[string]any{"enabled": false},
			"ports": map[string]any{
				"enabled":      true,
				"engineConfig": map[string]any{"scan": map[string]any{"enabled": true, "timeout": 60}},
			},
		},
	}
	record, err := scanModelToRecord(&model.Scan{
		ID:             7,
		ScanWorkflowID: "default",
		Configuration:  datatypes.JSON([]byte(`{"steps":{"discover":{"enabled":false},"ports":{"enabled":true,"engineConfig":{"scan":{"enabled":true,"timeout":60}}}}}`)),
		InputSource:    "scan_snapshot",
		TriggerType:    "manual",
	})
	if err != nil {
		t.Fatalf("scanModelToRecord() error = %v", err)
	}
	persisted["steps"].(map[string]any)["ports"].(map[string]any)["engineConfig"].(map[string]any)["scan"].(map[string]any)["timeout"] = float64(60)
	if record == nil || !reflect.DeepEqual(record.Configuration, persisted) {
		t.Fatalf("history read changed persisted configuration: got=%#v want=%#v", record, persisted)
	}
}

func TestScanRepositoryGetDetailByIDReturnsRuntimeTasksWithoutStageProgressMutation(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:scan_detail_stage_progress_task_ids?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite failed: %v", err)
	}
	if err := db.Exec(`
		CREATE TABLE target (
			id INTEGER PRIMARY KEY,
			name TEXT NOT NULL,
			type TEXT NOT NULL,
			created_at DATETIME,
			last_scanned_at DATETIME,
			deleted_at DATETIME
		);
		CREATE TABLE agent (
			id INTEGER PRIMARY KEY,
			display_name TEXT,
			status TEXT
		);
		CREATE TABLE agent_runtime_status (agent_id INTEGER PRIMARY KEY, health_state TEXT);
	` + createWithScanTasksScanDDL + createWithScanTasksScanTaskDDL).Error; err != nil {
		t.Fatalf("setup tables failed: %v", err)
	}

	startedAt := time.Date(2026, 6, 19, 1, 2, 3, 0, time.UTC)
	completedAt := startedAt.Add(73 * time.Second)
	if err := db.Exec(`
		INSERT INTO target (id, name, type, created_at) VALUES (99, 'example.com', 'domain', '2026-06-19 01:02:03');
			INSERT INTO scan (id, target_id, scan_workflow_id, configuration, input_source, trigger_type, status, container_ids, created_at, stage_progress)
			VALUES (7, 99, 'full_scan', '{}', 'scan_snapshot', 'manual', 'running', '{}', '2026-06-19 01:02:03', '{"subdomain_discovery":{"detail":"existing detail"}}');
		INSERT INTO scan_task (id, scan_id, stage_order, stage_id, step_order, step_id, engine_id, engine_config, task_execution_config, status, failure_kind, skip_reason, engine_diagnostics, resolved_execution_plan, started_at, completed_at, created_at)
		VALUES (70001, 7, 1, 'discovery', 1, 'subdomain_discovery', 'engine.lunafox.subdomain_discovery', '{}', '{}', 'succeeded', '', '', '{"compatibilityRevision":"engine-execution-diagnostics-r1","availability":"unavailable","resultState":"unknown"}', X'01', '2026-06-19 01:02:03', '2026-06-19 01:03:16', '2026-06-19 01:02:03');
	`).Error; err != nil {
		t.Fatalf("insert fixtures failed: %v", err)
	}

	repo := NewScanRepository(db)
	scan, err := repo.GetDetailByID(7)
	if err != nil {
		t.Fatalf("GetDetailByID failed: %v", err)
	}

	var storedStageProgress string
	if err := db.Table("scan").Select("stage_progress").Where("id = ?", 7).Scan(&storedStageProgress).Error; err != nil {
		t.Fatalf("query stage_progress failed: %v", err)
	}
	if storedStageProgress != `{"subdomain_discovery":{"detail":"existing detail"}}` {
		t.Fatalf("stage_progress should remain unchanged, got %s", storedStageProgress)
	}
	if len(scan.RuntimeTasks) != 1 {
		t.Fatalf("expected one runtime task, got %+v", scan.RuntimeTasks)
	}
	task := scan.RuntimeTasks[0]
	if task.ID != 70001 || task.StepID != "subdomain_discovery" || task.StageID != "discovery" || task.EngineID != "engine.lunafox.subdomain_discovery" {
		t.Fatalf("unexpected runtime task identity: %+v", task)
	}
	if task.Status != "succeeded" || task.Order != 1 {
		t.Fatalf("expected task status/order in runtimeTasks, got %+v", task)
	}
	if task.SkipReason != "" {
		t.Fatalf("succeeded task must not expose a skip reason, got %q", task.SkipReason)
	}
	if task.StartedAt == nil || task.StartedAt.Format(time.RFC3339Nano) != startedAt.Format(time.RFC3339Nano) {
		t.Fatalf("expected runtime task startedAt %s, got %+v", startedAt.Format(time.RFC3339Nano), task.StartedAt)
	}
	if task.CompletedAt == nil || task.CompletedAt.Format(time.RFC3339Nano) != completedAt.Format(time.RFC3339Nano) {
		t.Fatalf("expected runtime task completedAt %s, got %+v", completedAt.Format(time.RFC3339Nano), task.CompletedAt)
	}
	if task.Duration == nil || *task.Duration != 73 {
		t.Fatalf("expected runtime task duration 73 seconds, got %+v", task.Duration)
	}
	if got := scan.PlannedEngineIDs; len(got) != 1 || got[0] != "engine.lunafox.subdomain_discovery" {
		t.Fatalf("expected detail planned engine IDs from saved-plan task snapshots, got %+v", got)
	}
}

func TestScanRepositoryGetDetailByIDProjectsAssignedAgentWithoutCollectionWindow(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:scan_detail_agent_projection?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite failed: %v", err)
	}
	if err := db.Exec(`
		CREATE TABLE target (id INTEGER PRIMARY KEY, name TEXT NOT NULL, type TEXT NOT NULL, created_at DATETIME, deleted_at DATETIME);
		CREATE TABLE agent (id INTEGER PRIMARY KEY, display_name TEXT NOT NULL, status TEXT NOT NULL);
		CREATE TABLE agent_runtime_status (agent_id INTEGER PRIMARY KEY, health_state TEXT);
	` + createWithScanTasksScanDDL + createWithScanTasksScanTaskDDL).Error; err != nil {
		t.Fatalf("setup tables failed: %v", err)
	}
	if err := db.Exec(`
		INSERT INTO target (id, name, type, created_at) VALUES (1, 'example.com', 'domain', CURRENT_TIMESTAMP);
		WITH RECURSIVE agent_ids(id) AS (
			SELECT 1
			UNION ALL
			SELECT id + 1 FROM agent_ids WHERE id < 1001
		)
		INSERT INTO agent (id, display_name, status)
		SELECT id, printf('Agent %04d', id), 'offline' FROM agent_ids;
		UPDATE agent SET display_name = 'Pinned Agent', status = 'online' WHERE id = 1001;
		INSERT INTO agent_runtime_status (agent_id, health_state) VALUES (1001, 'paused');
			INSERT INTO scan (id, target_id, scan_workflow_id, configuration, input_source, trigger_type, status, container_ids, agent_id, assignment_mode, created_at)
			VALUES
				(7, 1, 'default', '{}', 'scan_snapshot', 'manual', 'running', '{}', 1001, 'pinned', CURRENT_TIMESTAMP),
				(8, 1, 'default', '{}', 'scan_snapshot', 'manual', 'pending', '{}', 1000, 'automatic', CURRENT_TIMESTAMP),
				(9, 1, 'default', '{}', 'scan_snapshot', 'manual', 'pending', '{}', NULL, 'automatic', CURRENT_TIMESTAMP);
	`).Error; err != nil {
		t.Fatalf("insert fixtures failed: %v", err)
	}

	repository := NewScanRepository(db)
	pinned, err := repository.GetDetailByID(7)
	if err != nil {
		t.Fatalf("GetDetailByID pinned Agent failed: %v", err)
	}
	if pinned.AgentID == nil || *pinned.AgentID != 1001 || pinned.AgentName != "Pinned Agent" || pinned.AgentStatus != "online" || pinned.AgentHealthState != "paused" || pinned.AgentDeleted || pinned.AssignmentMode != "pinned" {
		t.Fatalf("pinned Agent projection = %+v", pinned)
	}

	if err := db.Table("agent").Where("id = ?", 1001).Update("display_name", "Renamed Agent").Error; err != nil {
		t.Fatalf("rename Agent: %v", err)
	}
	renamed, err := repository.GetDetailByID(7)
	if err != nil {
		t.Fatalf("GetDetailByID renamed Agent failed: %v", err)
	}
	if renamed.AgentName != "Renamed Agent" {
		t.Fatalf("detail retained an assignment-time name: %+v", renamed)
	}

	if err := db.Exec("DELETE FROM agent WHERE id = ?", 1001).Error; err != nil {
		t.Fatalf("delete Agent: %v", err)
	}
	deleted, err := repository.GetDetailByID(7)
	if err != nil {
		t.Fatalf("GetDetailByID deleted Agent failed: %v", err)
	}
	if deleted.AgentID == nil || *deleted.AgentID != 1001 || !deleted.AgentDeleted || deleted.AgentName != "" || deleted.AgentStatus != "" || deleted.AgentHealthState != "" || deleted.AssignmentMode != "pinned" {
		t.Fatalf("deleted Agent projection = %+v", deleted)
	}

	withoutRuntime, err := repository.GetDetailByID(8)
	if err != nil {
		t.Fatalf("GetDetailByID Agent without runtime failed: %v", err)
	}
	if withoutRuntime.AgentID == nil || *withoutRuntime.AgentID != 1000 || withoutRuntime.AgentName != "Agent 1000" || withoutRuntime.AgentStatus != "offline" || withoutRuntime.AgentHealthState != "" || withoutRuntime.AgentDeleted || withoutRuntime.AssignmentMode != "automatic" {
		t.Fatalf("Agent without runtime projection = %+v", withoutRuntime)
	}

	unassigned, err := repository.GetDetailByID(9)
	if err != nil {
		t.Fatalf("GetDetailByID unassigned Scan failed: %v", err)
	}
	if unassigned.AgentID != nil || unassigned.AgentName != "" || unassigned.AgentStatus != "" || unassigned.AgentHealthState != "" || unassigned.AgentDeleted || unassigned.AssignmentMode != "automatic" {
		t.Fatalf("unassigned Scan projection = %+v", unassigned)
	}
}

func TestScanRepositoryGetDetailByIDProjectsStableSkipReason(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:scan_detail_skip_reason?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite failed: %v", err)
	}
	if err := db.Exec(`
		CREATE TABLE target (id INTEGER PRIMARY KEY, name TEXT NOT NULL, type TEXT NOT NULL, created_at DATETIME, deleted_at DATETIME);
		CREATE TABLE agent (id INTEGER PRIMARY KEY, display_name TEXT, status TEXT);
		CREATE TABLE agent_runtime_status (agent_id INTEGER PRIMARY KEY, health_state TEXT);
	` + createWithScanTasksScanDDL + createWithScanTasksScanTaskDDL).Error; err != nil {
		t.Fatalf("setup tables failed: %v", err)
	}
	if err := db.Exec(`
		INSERT INTO target (id, name, type, created_at) VALUES (1, 'example.com', 'domain', CURRENT_TIMESTAMP);
			INSERT INTO scan (id, target_id, scan_workflow_id, configuration, input_source, trigger_type, status, container_ids, created_at)
			VALUES (2, 1, 'default', '{}', 'scan_snapshot', 'manual', 'succeeded', '{}', CURRENT_TIMESTAMP);
		INSERT INTO scan_task (id, scan_id, stage_order, stage_id, step_order, step_id, engine_id, status, skip_reason, created_at)
		VALUES (3, 2, 1, 'discovery', 1, 'subdomain_discovery', 'engine.lunafox.subdomain_discovery', 'skipped', 'user_disabled', CURRENT_TIMESTAMP);
	`).Error; err != nil {
		t.Fatalf("insert fixtures failed: %v", err)
	}

	scan, err := NewScanRepository(db).GetDetailByID(2)
	if err != nil {
		t.Fatalf("GetDetailByID failed: %v", err)
	}
	if len(scan.RuntimeTasks) != 1 || scan.RuntimeTasks[0].SkipReason != "user_disabled" {
		t.Fatalf("runtime skip reason = %+v, want user_disabled", scan.RuntimeTasks)
	}
}

func TestScanRepositoryRetainsCancelledHistoryAndTombstonedTargetContext(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:scan_deleted_target_history_visibility?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite failed: %v", err)
	}
	if err := db.Exec(`
		CREATE TABLE target (
			id INTEGER PRIMARY KEY,
			name TEXT NOT NULL,
			type TEXT NOT NULL,
			created_at DATETIME,
			last_scanned_at DATETIME,
			deleted_at DATETIME
		);
		CREATE TABLE agent (id INTEGER PRIMARY KEY, display_name TEXT, status TEXT);
		CREATE TABLE agent_runtime_status (agent_id INTEGER PRIMARY KEY, health_state TEXT);
	` + createWithScanTasksScanDDL + createWithScanTasksScanTaskDDL).Error; err != nil {
		t.Fatalf("setup history visibility tables: %v", err)
	}
	stoppedAt := time.Date(2026, 8, 6, 12, 0, 0, 0, time.UTC)
	if err := db.Exec(`
		INSERT INTO target (id, name, type, created_at, deleted_at)
		VALUES (1, 'retained-target.example', 'domain', ?, ?);
			INSERT INTO scan (id, target_id, scan_workflow_id, configuration, input_source, trigger_type, status, container_ids, created_at, stopped_at)
			VALUES (7, 1, 'default', '{}', 'scan_snapshot', 'manual', 'cancelled', '{}', ?, ?);
			INSERT INTO scan_task (id, scan_id, stage_order, stage_id, step_order, step_id, engine_id, status, created_at, completed_at)
			VALUES (70, 7, 1, 'discovery', 1, 'subdomain_discovery', 'engine.lunafox.subdomain_discovery', 'cancelled', ?, ?);
	`, stoppedAt.Add(-time.Hour), stoppedAt, stoppedAt.Add(-time.Hour), stoppedAt, stoppedAt.Add(-time.Hour), stoppedAt).Error; err != nil {
		t.Fatalf("seed cancelled retained history: %v", err)
	}

	repository := NewScanRepository(db)
	detail, err := repository.GetDetailByID(7)
	if err != nil {
		t.Fatalf("GetDetailByID(): %v", err)
	}
	if detail.Status != "cancelled" || detail.StoppedAt == nil || !detail.StoppedAt.Equal(stoppedAt) || detail.Target == nil || detail.Target.Name != "retained-target.example" {
		t.Fatalf("cancelled history projection = %+v", detail)
	}

	items, total, err := repository.List(1, 10, 0, "", "", "createdAt desc")
	if err != nil {
		t.Fatalf("List(): %v", err)
	}
	if total != 1 || len(items) != 1 || items[0].Status != "cancelled" || items[0].StoppedAt == nil || items[0].Target == nil || items[0].Target.Name != "retained-target.example" {
		t.Fatalf("cancelled history list projection = total=%d items=%+v", total, items)
	}
}

func TestScanRepositoryListReturnsPlannedEngineIDsForPageScans(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:scan_list_engine_names?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite failed: %v", err)
	}
	if err := db.Exec(`
		CREATE TABLE target (
			id INTEGER PRIMARY KEY,
			name TEXT NOT NULL,
			type TEXT NOT NULL,
			created_at DATETIME,
			last_scanned_at DATETIME,
			deleted_at DATETIME
		);
		CREATE TABLE agent (
			id INTEGER PRIMARY KEY,
			display_name TEXT
		);
	` + createWithScanTasksScanDDL + createWithScanTasksScanTaskDDL).Error; err != nil {
		t.Fatalf("setup tables failed: %v", err)
	}

	if err := db.Exec(`
		INSERT INTO target (id, name, type, created_at) VALUES (10, 'example.com', 'domain', '2026-07-03 01:02:03');
			INSERT INTO scan (id, target_id, scan_workflow_id, configuration, input_source, trigger_type, status, container_ids, created_at)
			VALUES (101, 10, 'full_recon', '{}', 'scan_snapshot', 'manual', 'running', '{}', '2026-07-03 01:02:03');
			INSERT INTO scan (id, target_id, scan_workflow_id, configuration, input_source, trigger_type, status, container_ids, created_at)
			VALUES (102, 10, 'empty_recon', '{}', 'scan_snapshot', 'manual', 'pending', '{}', '2026-07-02 01:02:03');
		INSERT INTO scan_task (id, scan_id, stage_order, stage_id, step_order, step_id, engine_id, engine_config, task_execution_config, status, resolved_execution_plan, created_at)
		VALUES
			(1001, 101, 1, 'discovery', 1, 'subdomain_discovery', 'engine.lunafox.subdomain_discovery', '{}', '{}', 'succeeded', X'01', '2026-07-03 01:02:03'),
			(1002, 101, 2, 'ports', 1, 'port_scan', 'engine.lunafox.port_scan', '{}', '{}', 'running', X'02', '2026-07-03 01:03:03'),
			(1003, 101, 2, 'ports', 2, 'port_scan', 'engine.lunafox.port_scan', '{}', '{}', 'skipped', X'', '2026-07-03 01:04:03');
	`).Error; err != nil {
		t.Fatalf("insert fixtures failed: %v", err)
	}

	repo := NewScanRepository(db)
	scans, total, err := repo.List(1, 10, 0, "", "", "createdAt desc")
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if total != 2 || len(scans) != 2 {
		t.Fatalf("expected two scans, total=%d rows=%d", total, len(scans))
	}
	if scans[0].ID != 101 {
		t.Fatalf("expected newest scan first, got %+v", scans[0])
	}
	if got := scans[0].PlannedEngineIDs; len(got) != 2 || got[0] != "engine.lunafox.subdomain_discovery" || got[1] != "engine.lunafox.port_scan" {
		t.Fatalf("unexpected planned engine IDs for scan 101: %+v", got)
	}
	if got := scans[1].PlannedEngineIDs; len(got) != 0 {
		t.Fatalf("expected empty planned engine IDs for scan without tasks, got %+v", got)
	}
}

func TestScanRepositoryListAppliesTargetNameAndStatusFilter(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:scan_list_target_name_status_filter?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite failed: %v", err)
	}
	if err := db.Exec(`
		CREATE TABLE target (
			id INTEGER PRIMARY KEY,
			name TEXT NOT NULL,
			type TEXT NOT NULL,
			created_at DATETIME,
			last_scanned_at DATETIME,
			deleted_at DATETIME
		);
		CREATE TABLE agent (
			id INTEGER PRIMARY KEY,
			display_name TEXT
		);
	` + createWithScanTasksScanDDL + createWithScanTasksScanTaskDDL).Error; err != nil {
		t.Fatalf("setup tables failed: %v", err)
	}

	if err := db.Exec(`
		INSERT INTO target (id, name, type, created_at) VALUES
			(10, 'Acme.com', 'domain', '2026-07-03 01:02:03'),
			(11, 'example.com', 'domain', '2026-07-03 01:02:03');
			INSERT INTO scan (id, target_id, scan_workflow_id, configuration, input_source, trigger_type, status, container_ids, created_at)
			VALUES
				(101, 10, 'full_recon', '{}', 'scan_snapshot', 'manual', 'running', '{}', '2026-07-03 01:02:03'),
				(102, 10, 'full_recon', '{}', 'scan_snapshot', 'manual', 'failed', '{}', '2026-07-02 01:02:03'),
				(103, 10, 'full_recon', '{}', 'scan_snapshot', 'manual', 'succeeded', '{}', '2026-07-01 01:02:03'),
				(104, 11, 'full_recon', '{}', 'scan_snapshot', 'manual', 'running', '{}', '2026-07-04 01:02:03');
	`).Error; err != nil {
		t.Fatalf("insert fixtures failed: %v", err)
	}

	repo := NewScanRepository(db)
	scans, total, err := repo.List(1, 10, 0, "", `(status=="running" || status=="failed") && targetName="acme"`, "createdAt desc")
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if total != 2 || len(scans) != 2 {
		t.Fatalf("expected two filtered scans, total=%d rows=%d %+v", total, len(scans), scans)
	}
	if scans[0].ID != 101 || scans[1].ID != 102 {
		t.Fatalf("expected matching scans ordered by createdAt desc, got ids %d,%d", scans[0].ID, scans[1].ID)
	}
}

func TestScanRepositoryListSupportsCreatedAtAscendingOrder(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:scan_list_created_at_asc?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite failed: %v", err)
	}
	if err := db.Exec(`
		CREATE TABLE target (
			id INTEGER PRIMARY KEY,
			name TEXT NOT NULL,
			type TEXT NOT NULL,
			created_at DATETIME,
			last_scanned_at DATETIME,
			deleted_at DATETIME
		);
		CREATE TABLE agent (
			id INTEGER PRIMARY KEY,
			display_name TEXT
		);
	` + createWithScanTasksScanDDL + createWithScanTasksScanTaskDDL).Error; err != nil {
		t.Fatalf("setup tables failed: %v", err)
	}

	if err := db.Exec(`
		INSERT INTO target (id, name, type, created_at) VALUES (10, 'acme.com', 'domain', '2026-07-03 01:02:03');
			INSERT INTO scan (id, target_id, scan_workflow_id, configuration, input_source, trigger_type, status, container_ids, created_at)
			VALUES
				(101, 10, 'full_recon', '{}', 'scan_snapshot', 'manual', 'running', '{}', '2026-07-03 01:02:03'),
				(102, 10, 'full_recon', '{}', 'scan_snapshot', 'manual', 'failed', '{}', '2026-07-02 01:02:03');
	`).Error; err != nil {
		t.Fatalf("insert fixtures failed: %v", err)
	}

	repo := NewScanRepository(db)
	scans, total, err := repo.List(1, 10, 0, "", "", "createdAt asc")
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if total != 2 || len(scans) != 2 {
		t.Fatalf("expected two scans, total=%d rows=%d", total, len(scans))
	}
	if scans[0].ID != 102 || scans[1].ID != 101 {
		t.Fatalf("expected oldest scan first, got ids %d,%d", scans[0].ID, scans[1].ID)
	}
}
