package repository

import (
	"context"
	"testing"

	"github.com/yyhuni/lunafox/contracts/resourcenames"
)

func TestSavedPlanClaimNeverClaimsUserDisabledOrPlanningSkippedTask(t *testing.T) {
	db := openSavedExecutionPlanTestDB(t)
	for _, statement := range []string{
		`CREATE TABLE scan (id INTEGER PRIMARY KEY, status TEXT NOT NULL, agent_id INTEGER, deleted_at DATETIME)`,
		`CREATE TABLE scan_task (
			id INTEGER PRIMARY KEY,
			scan_id INTEGER NOT NULL,
			stage_order INTEGER NOT NULL,
			created_at DATETIME NOT NULL,
			status TEXT NOT NULL,
			skip_reason TEXT NOT NULL DEFAULT '',
			assigned_agent_id INTEGER,
			assigned_session_id TEXT,
			assigned_session_epoch INTEGER,
			assigned_request_id TEXT,
			terminal_reconciliation_pending BOOLEAN NOT NULL DEFAULT FALSE,
			started_at DATETIME,
			resolved_execution_plan BLOB NOT NULL DEFAULT ''
		)`,
		`CREATE TABLE agent_runtime_status (agent_id INTEGER PRIMARY KEY, session_id TEXT NOT NULL, session_epoch INTEGER NOT NULL)`,
		`CREATE UNIQUE INDEX planning_skip_claim_request ON scan_task(assigned_agent_id, assigned_session_epoch, assigned_request_id) WHERE assigned_request_id IS NOT NULL`,
		`INSERT INTO scan (id, status) VALUES (7, 'pending')`,
		`INSERT INTO agent_runtime_status (agent_id, session_id, session_epoch) VALUES (42, 'session-a', 9)`,
		`INSERT INTO scan_task (id, scan_id, stage_order, created_at, status, skip_reason, resolved_execution_plan)
			 VALUES (1, 7, 1, '2026-01-01T00:00:00Z', 'skipped', 'target_not_applicable', '')`,
		`INSERT INTO scan_task (id, scan_id, stage_order, created_at, status, skip_reason, resolved_execution_plan)
			 VALUES (2, 7, 2, '2026-01-01T00:00:00Z', 'skipped', 'user_disabled', '')`,
	} {
		if err := db.Exec(statement).Error; err != nil {
			t.Fatalf("setup planning skip fixture: %v", err)
		}
	}
	plan := mustSavedPlanForTask(t, resourcenames.Task(7, 3), resourcenames.Scan(7), "engine.lunafox.port_scan", "ports", "port_scan")
	if err := db.Exec(
		`INSERT INTO scan_task (id, scan_id, stage_order, created_at, status, resolved_execution_plan)
			 VALUES (3, 7, 3, '2026-01-01T00:00:01Z', 'pending', ?)`,
		plan,
	).Error; err != nil {
		t.Fatalf("insert applicable saved plan: %v", err)
	}

	repository := &scanTaskRepository{db: db}
	requestID := "018f6f22-0d23-7b4a-a508-8f4c8abf16d1"
	claimed, err := repository.ClaimNextCompatibleSavedExecutionPlan(context.Background(), 42, "session-a", 9, requestID, []uint32{2})
	if err != nil {
		t.Fatalf("claim later applicable saved plan: %v", err)
	}
	if claimed == nil || claimed.GetTask() != resourcenames.Task(7, 3) {
		t.Fatalf("claimed plan = %#v, want task 3", claimed)
	}

	var rows []struct {
		ID                    int
		Status                string
		SkipReason            string
		ResolvedExecutionPlan []byte
		AssignedAgentID       *int
		AssignedSessionID     *string
		AssignedSessionEpoch  *int64
		AssignedRequestID     *string
	}
	if err := db.Table("scan_task").Order("id ASC").Find(&rows).Error; err != nil {
		t.Fatalf("read task lease states: %v", err)
	}
	if len(rows) != 3 {
		t.Fatalf("task rows = %d, want 3", len(rows))
	}
	skipped := rows[0]
	if skipped.Status != taskStatusSkipped || skipped.SkipReason == "" || len(skipped.ResolvedExecutionPlan) != 0 ||
		skipped.AssignedAgentID != nil || skipped.AssignedSessionID != nil || skipped.AssignedSessionEpoch != nil || skipped.AssignedRequestID != nil {
		t.Fatalf("planning-time skipped task was mutated or claimable: %#v", skipped)
	}
	userDisabled := rows[1]
	if userDisabled.Status != taskStatusSkipped || userDisabled.SkipReason != "user_disabled" || len(userDisabled.ResolvedExecutionPlan) != 0 ||
		userDisabled.AssignedAgentID != nil || userDisabled.AssignedSessionID != nil || userDisabled.AssignedSessionEpoch != nil || userDisabled.AssignedRequestID != nil {
		t.Fatalf("user-disabled task was mutated or claimable: %#v", userDisabled)
	}
	applicable := rows[2]
	if applicable.Status != taskStatusRunning || applicable.AssignedAgentID == nil || *applicable.AssignedAgentID != 42 ||
		applicable.AssignedSessionID == nil || *applicable.AssignedSessionID != "session-a" ||
		applicable.AssignedSessionEpoch == nil || *applicable.AssignedSessionEpoch != 9 ||
		applicable.AssignedRequestID == nil || *applicable.AssignedRequestID != requestID {
		t.Fatalf("later applicable task was not atomically claimed: %#v", applicable)
	}
}
