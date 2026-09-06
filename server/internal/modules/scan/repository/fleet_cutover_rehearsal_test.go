package repository

import (
	"context"
	"testing"

	"github.com/yyhuni/lunafox/contracts/resourcenames"
)

func TestFleetCutoverOldAgentCannotClaimPackagePlan(t *testing.T) {
	db := openSavedExecutionPlanTestDB(t)
	statements := []string{
		`CREATE TABLE scan (
			id INTEGER PRIMARY KEY,
			status TEXT NOT NULL,
			agent_id INTEGER,
			deleted_at DATETIME
		)`,
		`CREATE TABLE scan_task (
			id INTEGER PRIMARY KEY,
			scan_id INTEGER NOT NULL,
			stage_order INTEGER NOT NULL,
			created_at DATETIME NOT NULL,
			status TEXT NOT NULL,
			assigned_agent_id INTEGER,
			assigned_session_id TEXT,
			assigned_session_epoch INTEGER,
			assigned_request_id TEXT,
			terminal_reconciliation_pending BOOLEAN NOT NULL DEFAULT FALSE,
			started_at DATETIME,
			resolved_execution_plan BLOB NOT NULL
		)`,
		`CREATE UNIQUE INDEX fleet_cutover_claim_request ON scan_task(assigned_agent_id, assigned_session_epoch, assigned_request_id) WHERE assigned_request_id IS NOT NULL`,
		`CREATE TABLE agent_runtime_status (
			agent_id INTEGER PRIMARY KEY,
			session_id TEXT NOT NULL,
			session_epoch INTEGER NOT NULL
		)`,
		`INSERT INTO agent_runtime_status (agent_id, session_id, session_epoch) VALUES
			(41, 'legacy-session-1', 41),
			(42, 'engine-container-session-1', 42)`,
		`INSERT INTO scan (id, status) VALUES (12, 'pending')`,
	}
	for _, statement := range statements {
		if err := db.Exec(statement).Error; err != nil {
			t.Fatalf("set up fleet cutover claim fixture: %v", err)
		}
	}

	// Package v2 is already verified by the Server compiler; the saved claim
	// boundary receives only its exact packageDigest and required API major.
	plan := mustSavedPlanForTask(
		t,
		resourcenames.Task(12, 1),
		resourcenames.Scan(12),
		"engine.lunafox.port_scan",
		"ports",
		"run",
	)
	if err := db.Exec(`INSERT INTO scan_task (
		id, scan_id, stage_order, created_at, status, resolved_execution_plan
	) VALUES (1, 12, 1, '2026-07-24T00:00:00Z', 'pending', ?)`, plan).Error; err != nil {
		t.Fatalf("insert package v2/API v2 saved plan: %v", err)
	}

	repository := &scanTaskRepository{db: db}
	legacyClaim, err := repository.ClaimNextCompatibleSavedExecutionPlan(
		context.Background(),
		41,
		"legacy-session-1",
		41,
		"018f6f22-0d23-7b4a-a508-8f4c8abf1201",
		[]uint32{1},
	)
	if err != nil {
		t.Fatalf("probe legacy Agent claim: %v", err)
	}
	if legacyClaim != nil {
		t.Fatal("legacy Agent claimed an Engine API v2 saved plan")
	}
	var afterLegacy struct {
		Status            string
		AssignedAgentID   *int
		AssignedSessionID *string
	}
	if err := db.Table("scan_task").Select("status, assigned_agent_id, assigned_session_id").Where("id = 1").Take(&afterLegacy).Error; err != nil {
		t.Fatalf("read task after legacy claim probe: %v", err)
	}
	if afterLegacy.Status != taskStatusPending || afterLegacy.AssignedAgentID != nil || afterLegacy.AssignedSessionID != nil {
		t.Fatalf("legacy capability probe mutated the package v2 task: %#v", afterLegacy)
	}

	newClaim, err := repository.ClaimNextCompatibleSavedExecutionPlan(
		context.Background(),
		42,
		"engine-container-session-1",
		42,
		"018f6f22-0d23-7b4a-a508-8f4c8abf1202",
		[]uint32{2},
	)
	if err != nil {
		t.Fatalf("claim package v2/API v2 plan with new Agent: %v", err)
	}
	if newClaim == nil || newClaim.GetEngineRelease().GetEngineApiMajor() != 2 || newClaim.GetEngineRelease().GetPackageDigest() == "" {
		t.Fatalf("new Agent did not receive the exact package v2/API v2 plan: %#v", newClaim)
	}

	t.Log("FLEET_CUTOVER_CLAIM_PROBE old_agent_claimed=false new_agent_claimed=true required_engine_api_major=2")
}
