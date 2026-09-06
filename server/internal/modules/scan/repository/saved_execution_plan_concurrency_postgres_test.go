package repository

import (
	"context"
	"errors"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/yyhuni/lunafox/contracts/resourcenames"
	scandomain "github.com/yyhuni/lunafox/server/internal/modules/scan/domain"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

const savedPlanLeaseConcurrencySchema = "scan_lease_concurrency"

func TestSavedPlanLeaseConcurrencyPostgres(t *testing.T) {
	db := openSavedPlanLeaseConcurrencyPostgres(t)
	repo := NewScanTaskRepository(db)

	t.Run("same request concurrent claim replays one plan", func(t *testing.T) {
		resetSavedPlanLeaseConcurrencyFixture(t, db)
		const callers = 8
		start := make(chan struct{})
		type result struct {
			task string
			err  error
		}
		results := make(chan result, callers)
		var group sync.WaitGroup
		group.Add(callers)
		for index := 0; index < callers; index++ {
			go func() {
				defer group.Done()
				<-start
				plan, err := repo.ClaimNextCompatibleSavedExecutionPlan(
					context.Background(), 42, "session-a", 9,
					"550e8400-e29b-41d4-a716-446655440000", []uint32{2},
				)
				if plan == nil {
					results <- result{err: err}
					return
				}
				results <- result{task: plan.GetTask(), err: err}
			}()
		}
		close(start)
		group.Wait()
		close(results)
		for result := range results {
			if result.err != nil || result.task != resourcenames.Task(7, 1) {
				t.Fatalf("concurrent claim result = task %q error %v", result.task, result.err)
			}
		}
		var row struct {
			Status            string
			AssignedRequestID *string
		}
		if err := db.Table("scan_task").Select("status, assigned_request_id").Where("id = 1").Take(&row).Error; err != nil {
			t.Fatalf("read claimed row: %v", err)
		}
		if row.Status != taskStatusRunning || row.AssignedRequestID == nil || *row.AssignedRequestID != "550e8400-e29b-41d4-a716-446655440000" {
			t.Fatalf("persisted concurrent claim = %+v", row)
		}
	})

	t.Run("takeover transaction fences blocked claim", func(t *testing.T) {
		resetSavedPlanLeaseConcurrencyFixture(t, db)
		takeover := beginPersistedSessionTakeover(t, db, "session-b", 10)
		result := make(chan error, 1)
		go func() {
			_, err := repo.ClaimNextCompatibleSavedExecutionPlan(
				context.Background(), 42, "session-a", 9,
				"550e8400-e29b-41d4-a716-446655440001", []uint32{2},
			)
			result <- err
		}()
		assertPostgresOperationBlocked(t, result)
		if err := takeover.Commit().Error; err != nil {
			t.Fatalf("commit takeover: %v", err)
		}
		if err := <-result; !errors.Is(err, scandomain.ErrAgentExecutionSessionFenced) {
			t.Fatalf("claim after takeover error = %v", err)
		}
		var status string
		if err := db.Table("scan_task").Select("status").Where("id = 1").Scan(&status).Error; err != nil {
			t.Fatalf("read task after fenced claim: %v", err)
		}
		if status != taskStatusPending {
			t.Fatalf("fenced claim changed task status to %q", status)
		}
	})

	t.Run("takeover transaction fences blocked terminal CAS", func(t *testing.T) {
		resetSavedPlanLeaseConcurrencyFixture(t, db)
		if err := db.Exec("UPDATE scan SET status = 'running', agent_id = 42 WHERE id = 7").Error; err != nil {
			t.Fatalf("assign scan lease: %v", err)
		}
		if err := db.Exec("UPDATE scan_task SET status = 'running', assigned_agent_id = 42, assigned_session_id = 'session-a', assigned_session_epoch = 9, assigned_request_id = '550e8400-e29b-41d4-a716-446655440002' WHERE id = 1").Error; err != nil {
			t.Fatalf("assign task lease: %v", err)
		}
		takeover := beginPersistedSessionTakeover(t, db, "session-b", 10)
		type terminalResult struct {
			committed bool
			err       error
		}
		result := make(chan terminalResult, 1)
		go func() {
			committed, err := repo.CommitScanTaskTerminalStatusForSession(
				context.Background(), 1, 42, "session-a", 9, taskStatusSucceeded, nil,
			)
			result <- terminalResult{committed: committed, err: err}
		}()
		assertPostgresOperationBlocked(t, result)
		if err := takeover.Commit().Error; err != nil {
			t.Fatalf("commit takeover: %v", err)
		}
		terminal := <-result
		if terminal.committed || !errors.Is(terminal.err, scandomain.ErrAgentExecutionSessionFenced) {
			t.Fatalf("terminal CAS after takeover = committed %t error %v", terminal.committed, terminal.err)
		}
		var row struct {
			Status                        string
			TerminalReconciliationPending bool
		}
		if err := db.Table("scan_task").Select("status, terminal_reconciliation_pending").Where("id = 1").Take(&row).Error; err != nil {
			t.Fatalf("read task after fenced terminal CAS: %v", err)
		}
		if row.Status != taskStatusRunning || row.TerminalReconciliationPending {
			t.Fatalf("fenced terminal CAS changed task: %+v", row)
		}
	})
}

func openSavedPlanLeaseConcurrencyPostgres(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := strings.TrimSpace(os.Getenv("LUNAFOX_POSTGRES_LEASE_DSN"))
	if dsn == "" {
		t.Skip("set LUNAFOX_POSTGRES_LEASE_DSN to run the opt-in PostgreSQL lease concurrency verification")
	}
	if !strings.Contains(dsn, "search_path="+savedPlanLeaseConcurrencySchema) {
		t.Fatalf("lease concurrency DSN must pin search_path=%s", savedPlanLeaseConcurrencySchema)
	}
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatalf("open PostgreSQL lease concurrency database: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("open PostgreSQL SQL handle: %v", err)
	}
	sqlDB.SetMaxOpenConns(16)
	t.Cleanup(func() { _ = sqlDB.Close() })
	if err := db.Exec("DROP SCHEMA IF EXISTS " + savedPlanLeaseConcurrencySchema + " CASCADE").Error; err != nil {
		t.Fatalf("drop lease concurrency schema: %v", err)
	}
	if err := db.Exec("CREATE SCHEMA " + savedPlanLeaseConcurrencySchema).Error; err != nil {
		t.Fatalf("create lease concurrency schema: %v", err)
	}
	t.Cleanup(func() {
		cleanupCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		_ = db.WithContext(cleanupCtx).Exec("DROP SCHEMA IF EXISTS " + savedPlanLeaseConcurrencySchema + " CASCADE").Error
	})
	statements := []string{
		"CREATE TABLE agent_runtime_status (agent_id INTEGER PRIMARY KEY, session_id VARCHAR(64) NOT NULL, session_epoch BIGINT NOT NULL)",
		"CREATE TABLE scan (id INTEGER PRIMARY KEY, status VARCHAR(20) NOT NULL, agent_id INTEGER, deleted_at TIMESTAMPTZ)",
		"CREATE TABLE scan_task (id INTEGER PRIMARY KEY, scan_id INTEGER NOT NULL, stage_order INTEGER NOT NULL, created_at TIMESTAMPTZ NOT NULL, status VARCHAR(20) NOT NULL, assigned_agent_id INTEGER, assigned_session_id VARCHAR(64), assigned_session_epoch BIGINT, assigned_request_id VARCHAR(36), resolved_execution_plan BYTEA NOT NULL DEFAULT ''::bytea, terminal_reconciliation_pending BOOLEAN NOT NULL DEFAULT FALSE, error_message VARCHAR(4096) NOT NULL DEFAULT '', failure_kind VARCHAR(100) NOT NULL DEFAULT '', failure_detail VARCHAR(500) NOT NULL DEFAULT '', engine_diagnostics JSONB, started_at TIMESTAMPTZ, completed_at TIMESTAMPTZ)",
		"CREATE UNIQUE INDEX unique_scan_task_claim_request ON scan_task(assigned_agent_id, assigned_session_epoch, assigned_request_id) WHERE assigned_request_id IS NOT NULL",
	}
	for _, statement := range statements {
		if err := db.Exec(statement).Error; err != nil {
			t.Fatalf("prepare PostgreSQL lease concurrency schema: %v", err)
		}
	}
	return db
}

func resetSavedPlanLeaseConcurrencyFixture(t *testing.T, db *gorm.DB) {
	t.Helper()
	if err := db.Exec("TRUNCATE scan_task, scan, agent_runtime_status").Error; err != nil {
		t.Fatalf("reset lease concurrency fixture: %v", err)
	}
	if err := db.Exec("INSERT INTO agent_runtime_status (agent_id, session_id, session_epoch) VALUES (42, 'session-a', 9)").Error; err != nil {
		t.Fatalf("insert persisted session: %v", err)
	}
	if err := db.Exec("INSERT INTO scan (id, status) VALUES (7, 'pending')").Error; err != nil {
		t.Fatalf("insert scan: %v", err)
	}
	plan := mustSavedPlanForTask(t, resourcenames.Task(7, 1), resourcenames.Scan(7), "engine.lunafox.port_scan", "ports", "run")
	if err := db.Exec("INSERT INTO scan_task (id, scan_id, stage_order, created_at, status, resolved_execution_plan) VALUES (1, 7, 1, CURRENT_TIMESTAMP, 'pending', ?)", plan).Error; err != nil {
		t.Fatalf("insert saved plan: %v", err)
	}
}

func beginPersistedSessionTakeover(t *testing.T, db *gorm.DB, sessionID string, sessionEpoch int64) *gorm.DB {
	t.Helper()
	tx := db.Begin()
	if tx.Error != nil {
		t.Fatalf("begin takeover: %v", tx.Error)
	}
	t.Cleanup(func() { _ = tx.Rollback().Error })
	var agentID int
	if err := tx.Raw("SELECT agent_id FROM agent_runtime_status WHERE agent_id = 42 FOR UPDATE").Scan(&agentID).Error; err != nil {
		t.Fatalf("lock persisted session: %v", err)
	}
	if err := tx.Exec("UPDATE agent_runtime_status SET session_id = ?, session_epoch = ? WHERE agent_id = 42", sessionID, sessionEpoch).Error; err != nil {
		t.Fatalf("stage persisted session takeover: %v", err)
	}
	return tx
}

func assertPostgresOperationBlocked[T any](t *testing.T, result <-chan T) {
	t.Helper()
	select {
	case <-result:
		t.Fatal("operation crossed an uncommitted persisted-session takeover")
	case <-time.After(100 * time.Millisecond):
	}
}
