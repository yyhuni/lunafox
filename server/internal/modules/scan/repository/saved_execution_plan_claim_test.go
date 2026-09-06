package repository

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/yyhuni/lunafox/contracts/agentexecution"
	"github.com/yyhuni/lunafox/contracts/resourcenames"
	scandomain "github.com/yyhuni/lunafox/server/internal/modules/scan/domain"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/durationpb"
)

func TestSavedPlanClaimSkipsIncompatibleCandidateAndReplaysExactPlan(t *testing.T) {
	db := openSavedExecutionPlanTestDB(t)
	if err := db.Exec(`CREATE TABLE scan (
		id INTEGER PRIMARY KEY, status TEXT NOT NULL, agent_id INTEGER, deleted_at DATETIME
	)`).Error; err != nil {
		t.Fatalf("create scan: %v", err)
	}
	if err := db.Exec(`CREATE TABLE scan_task (
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
	)`).Error; err != nil {
		t.Fatalf("create scan_task: %v", err)
	}
	if err := db.Exec(`CREATE UNIQUE INDEX claim_request ON scan_task(assigned_agent_id, assigned_session_epoch, assigned_request_id) WHERE assigned_request_id IS NOT NULL`).Error; err != nil {
		t.Fatalf("create claim index: %v", err)
	}
	if err := db.Exec(`CREATE TABLE agent_runtime_status (agent_id INTEGER PRIMARY KEY, session_id TEXT NOT NULL, session_epoch INTEGER NOT NULL)`).Error; err != nil {
		t.Fatalf("create agent_runtime_status: %v", err)
	}
	if err := db.Exec(`CREATE TABLE agent (id INTEGER PRIMARY KEY)`).Error; err != nil {
		t.Fatalf("create agent: %v", err)
	}
	if err := db.Exec(`INSERT INTO agent (id) VALUES (42)`).Error; err != nil {
		t.Fatalf("insert agent: %v", err)
	}
	if err := db.Exec(`INSERT INTO agent_runtime_status (agent_id, session_id, session_epoch) VALUES (42, 'session-a', 9)`).Error; err != nil {
		t.Fatalf("insert agent runtime session: %v", err)
	}
	if err := db.Exec(`INSERT INTO scan (id, status) VALUES (7, 'pending')`).Error; err != nil {
		t.Fatalf("insert scan: %v", err)
	}

	incompatible := mustSavedPlanForTask(t, resourcenames.Task(7, 1), resourcenames.Scan(7), "engine.lunafox.port_scan", "ports", "old")
	incompatiblePlan, err := agentexecution.UnmarshalResolvedEngineExecutionPlan(incompatible)
	if err != nil {
		t.Fatalf("decode incompatible fixture: %v", err)
	}
	incompatiblePlan.EngineRelease.EngineApiMajor = 3
	incompatible, err = agentexecution.MarshalResolvedEngineExecutionPlan(incompatiblePlan)
	if err != nil {
		t.Fatalf("encode incompatible fixture: %v", err)
	}
	compatible := mustSavedPlanForTask(t, resourcenames.Task(7, 2), resourcenames.Scan(7), "engine.lunafox.port_scan", "ports", "current")
	compatiblePlan, err := agentexecution.UnmarshalResolvedEngineExecutionPlan(compatible)
	if err != nil {
		t.Fatalf("decode compatible fixture: %v", err)
	}
	compatiblePlan.Limits.MaxExecutionDuration = durationpb.New(7 * 24 * time.Hour)
	compatible, err = agentexecution.MarshalResolvedEngineExecutionPlan(compatiblePlan)
	if err != nil {
		t.Fatalf("encode frozen seven-day fixture: %v", err)
	}
	if err := db.Exec(`INSERT INTO scan_task (id, scan_id, stage_order, created_at, status, resolved_execution_plan) VALUES
		(1, 7, 2, '2026-01-01T00:00:00Z', 'pending', ?),
		(2, 7, 1, '2026-01-01T00:00:01Z', 'pending', ?)`, incompatible, compatible).Error; err != nil {
		t.Fatalf("insert tasks: %v", err)
	}

	repository := &scanTaskRepository{db: db}
	requestID := "018f6f22-0d23-7b4a-a508-8f4c8abf16d1"
	claimed, err := repository.ClaimNextCompatibleSavedExecutionPlan(context.Background(), 42, "session-a", 9, requestID, []uint32{2})
	if err != nil {
		t.Fatalf("claim compatible saved plan: %v", err)
	}
	want, err := agentexecution.UnmarshalResolvedEngineExecutionPlan(compatible)
	if err != nil {
		t.Fatalf("decode expected plan: %v", err)
	}
	if !proto.Equal(claimed, want) {
		t.Fatalf("claimed plan = %#v, want exact persisted plan %#v", claimed, want)
	}
	if got := claimed.GetLimits().GetMaxExecutionDuration().AsDuration(); got != 7*24*time.Hour {
		t.Fatalf("claim rewrote frozen seven-day budget to %s", got)
	}

	var rows []struct {
		ID                   int
		Status               string
		AssignedAgentID      *int
		AssignedSessionID    *string
		AssignedSessionEpoch *int64
		AssignedRequestID    *string
	}
	if err := db.Table("scan_task").Order("id").Find(&rows).Error; err != nil {
		t.Fatalf("read task states: %v", err)
	}
	if rows[0].Status != "pending" || rows[0].AssignedAgentID != nil {
		t.Fatalf("incompatible task was mutated: %#v", rows[0])
	}
	if rows[1].Status != "running" || rows[1].AssignedAgentID == nil || *rows[1].AssignedAgentID != 42 || rows[1].AssignedSessionID == nil || *rows[1].AssignedSessionID != "session-a" || rows[1].AssignedSessionEpoch == nil || *rows[1].AssignedSessionEpoch != 9 || rows[1].AssignedRequestID == nil || *rows[1].AssignedRequestID != requestID {
		t.Fatalf("compatible task claim was not atomically fenced: %#v", rows[1])
	}

	replayed, err := repository.ClaimNextCompatibleSavedExecutionPlan(context.Background(), 42, "session-a", 9, requestID, []uint32{2})
	if err != nil {
		t.Fatalf("replay saved claim: %v", err)
	}
	if !proto.Equal(replayed, want) {
		t.Fatalf("replayed plan differs from persisted assignment: %#v", replayed)
	}
	if got := replayed.GetLimits().GetMaxExecutionDuration().AsDuration(); got != 7*24*time.Hour {
		t.Fatalf("claim replay rewrote frozen seven-day budget to %s", got)
	}

	miss, err := repository.ClaimNextCompatibleSavedExecutionPlan(context.Background(), 42, "session-a", 9, "018f6f22-0d23-7b4a-a508-8f4c8abf16d2", []uint32{2})
	if err != nil || miss != nil {
		t.Fatalf("incompatible-only scheduler miss = %#v, %v; want no task", miss, err)
	}
	later := mustSavedPlanForTask(t, resourcenames.Task(7, 3), resourcenames.Scan(7), "engine.lunafox.port_scan", "ports", "later")
	if err := db.Exec(`INSERT INTO scan_task (id, scan_id, stage_order, created_at, status, resolved_execution_plan) VALUES (3, 7, 3, '2026-01-01T00:00:02Z', 'pending', ?)`, later).Error; err != nil {
		t.Fatalf("insert later-ready task: %v", err)
	}
	reobserved, err := repository.ClaimNextCompatibleSavedExecutionPlan(context.Background(), 42, "session-a", 9, "018f6f22-0d23-7b4a-a508-8f4c8abf16d2", []uint32{2})
	if err != nil || reobserved == nil || reobserved.GetTask() != resourcenames.Task(7, 3) {
		t.Fatalf("same request after no_task = %#v, %v; want later-ready task", reobserved, err)
	}
}

func TestSavedPlanClaimRejectsPersistedSessionTakeoverBeforeCAS(t *testing.T) {
	db := openSavedExecutionPlanTestDB(t)
	for _, statement := range []string{
		`CREATE TABLE scan (id INTEGER PRIMARY KEY, status TEXT NOT NULL, agent_id INTEGER, deleted_at DATETIME)`,
		`CREATE TABLE scan_task (id INTEGER PRIMARY KEY, scan_id INTEGER NOT NULL, stage_order INTEGER NOT NULL, created_at DATETIME NOT NULL, status TEXT NOT NULL, assigned_agent_id INTEGER, assigned_session_id TEXT, assigned_session_epoch INTEGER, assigned_request_id TEXT, terminal_reconciliation_pending BOOLEAN NOT NULL DEFAULT FALSE, started_at DATETIME, resolved_execution_plan BLOB NOT NULL)`,
		`CREATE TABLE agent_runtime_status (agent_id INTEGER PRIMARY KEY, session_id TEXT NOT NULL, session_epoch INTEGER NOT NULL)`,
		`CREATE UNIQUE INDEX claim_request_takeover ON scan_task(assigned_agent_id, assigned_session_epoch, assigned_request_id) WHERE assigned_request_id IS NOT NULL`,
		`INSERT INTO scan (id, status) VALUES (7, 'pending')`,
		`INSERT INTO agent_runtime_status (agent_id, session_id, session_epoch) VALUES (42, 'session-new', 10)`,
	} {
		if err := db.Exec(statement).Error; err != nil {
			t.Fatalf("setup takeover fixture: %v", err)
		}
	}
	plan := mustSavedPlanForTask(t, resourcenames.Task(7, 1), resourcenames.Scan(7), "engine.lunafox.port_scan", "ports", "current")
	if err := db.Exec(`INSERT INTO scan_task (id, scan_id, stage_order, created_at, status, resolved_execution_plan) VALUES (1, 7, 1, CURRENT_TIMESTAMP, 'pending', ?)`, plan).Error; err != nil {
		t.Fatalf("insert pending plan: %v", err)
	}
	repository := &scanTaskRepository{db: db}

	claimed, err := repository.ClaimNextCompatibleSavedExecutionPlan(context.Background(), 42, "session-old", 9, "018f6f22-0d23-7b4a-a508-8f4c8abf16d3", []uint32{2})
	if !errors.Is(err, scandomain.ErrAgentExecutionSessionFenced) || claimed != nil {
		t.Fatalf("stale persisted session claim = %#v, %v", claimed, err)
	}
	var status string
	if err := db.Table("scan_task").Select("status").Where("id = 1").Scan(&status).Error; err != nil || status != taskStatusPending {
		t.Fatalf("stale claim mutated task: status=%q err=%v", status, err)
	}
}
