package resultingestwiring

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	resultingestapp "github.com/yyhuni/lunafox/server/internal/modules/resultingest/application"
	"github.com/yyhuni/lunafox/server/internal/pkg/dbtx"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestResultIngestMaterializationCoordinatorRejectsStaleExecutionBeforeAnyWrite(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(t *testing.T, db *gorm.DB)
	}{
		{
			name: "DELETE first tombstones Target",
			mutate: func(t *testing.T, db *gorm.DB) {
				t.Helper()
				if err := db.Exec("UPDATE target SET deleted_at = ? WHERE id = 34", time.Now().UTC()).Error; err != nil {
					t.Fatalf("tombstone Target: %v", err)
				}
			},
		},
		{
			name: "cancellation first terminates Scan",
			mutate: func(t *testing.T, db *gorm.DB) {
				t.Helper()
				if err := db.Exec("UPDATE scan SET status = 'cancelled' WHERE id = 12").Error; err != nil {
					t.Fatalf("cancel Scan: %v", err)
				}
			},
		},
		{
			name: "cancellation first cancels Task",
			mutate: func(t *testing.T, db *gorm.DB) {
				t.Helper()
				if err := db.Exec("UPDATE scan_task SET status = 'cancelled' WHERE id = 101").Error; err != nil {
					t.Fatalf("cancel task: %v", err)
				}
			},
		},
		{
			name: "lease change reassigns Agent",
			mutate: func(t *testing.T, db *gorm.DB) {
				t.Helper()
				if err := db.Exec("UPDATE scan_task SET assigned_agent_id = 18 WHERE id = 101").Error; err != nil {
					t.Fatalf("reassign Agent: %v", err)
				}
			},
		},
		{
			name: "lease change replaces Session",
			mutate: func(t *testing.T, db *gorm.DB) {
				t.Helper()
				if err := db.Exec("UPDATE scan_task SET assigned_session_id = 'session-new' WHERE id = 101").Error; err != nil {
					t.Fatalf("replace session: %v", err)
				}
			},
		},
		{
			name: "lease change replaces Epoch",
			mutate: func(t *testing.T, db *gorm.DB) {
				t.Helper()
				if err := db.Exec("UPDATE scan_task SET assigned_session_epoch = 24 WHERE id = 101").Error; err != nil {
					t.Fatalf("replace epoch: %v", err)
				}
			},
		},
		{
			name: "process takeover replaces persisted Session",
			mutate: func(t *testing.T, db *gorm.DB) {
				t.Helper()
				if err := db.Exec("UPDATE agent_runtime_status SET session_id = 'session-new' WHERE agent_id = 17").Error; err != nil {
					t.Fatalf("replace persisted session: %v", err)
				}
			},
		},
		{
			name: "process takeover replaces persisted Epoch",
			mutate: func(t *testing.T, db *gorm.DB) {
				t.Helper()
				if err := db.Exec("UPDATE agent_runtime_status SET session_epoch = 24 WHERE agent_id = 17").Error; err != nil {
					t.Fatalf("replace persisted epoch: %v", err)
				}
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			db := newResultIngestMaterializationDB(t)
			test.mutate(t, db)
			coordinator := newResultIngestMaterializationCoordinator(db)
			persisted := false

			err := coordinator.Materialize(context.Background(), testResultMaterializationScope(), func(ctx context.Context) error {
				persisted = true
				return insertResultMaterializationEffects(ctx, db)
			})
			if !errors.Is(err, resultingestapp.ErrResultExecutionFenceRejected) {
				t.Fatalf("Materialize() error = %v, want execution fence rejection", err)
			}
			if persisted {
				t.Fatal("stale execution reached result writes")
			}
			assertResultMaterializationEffects(t, db, 0)
		})
	}
}

func TestResultIngestMaterializationCoordinatorMaterializationFirstCommitsBeforeLaterDelete(t *testing.T) {
	db := newResultIngestMaterializationDB(t)
	coordinator := newResultIngestMaterializationCoordinator(db)

	err := coordinator.Materialize(context.Background(), testResultMaterializationScope(), func(ctx context.Context) error {
		return insertResultMaterializationEffects(ctx, db)
	})
	if err != nil {
		t.Fatalf("materialization-first transaction: %v", err)
	}
	assertResultMaterializationEffects(t, db, 1)

	if err := db.Exec("UPDATE target SET deleted_at = ? WHERE id = 34", time.Now().UTC()).Error; err != nil {
		t.Fatalf("delete after materialization: %v", err)
	}
	writesReached := false
	err = coordinator.Materialize(context.Background(), testResultMaterializationScope(), func(context.Context) error {
		writesReached = true
		return nil
	})
	if !errors.Is(err, resultingestapp.ErrResultExecutionFenceRejected) {
		t.Fatalf("late materialization error = %v, want execution fence rejection", err)
	}
	if writesReached {
		t.Fatal("late materialization reached result writes after Target deletion")
	}
	assertResultMaterializationEffects(t, db, 1)
}

func TestResultIngestMaterializationCoordinatorCommitsOnlyCompleteAuthorizedBatch(t *testing.T) {
	db := newResultIngestMaterializationDB(t)
	coordinator := newResultIngestMaterializationCoordinator(db)
	persistErr := errors.New("summary refresh failed")

	err := coordinator.Materialize(context.Background(), testResultMaterializationScope(), func(ctx context.Context) error {
		if err := insertResultMaterializationEffects(ctx, db); err != nil {
			return err
		}
		return persistErr
	})
	if !errors.Is(err, persistErr) {
		t.Fatalf("Materialize() callback error = %v, want %v", err, persistErr)
	}
	assertResultMaterializationEffects(t, db, 0)

	err = coordinator.Materialize(context.Background(), testResultMaterializationScope(), func(ctx context.Context) error {
		return insertResultMaterializationEffects(ctx, db)
	})
	if err != nil {
		t.Fatalf("Materialize() valid batch: %v", err)
	}
	assertResultMaterializationEffects(t, db, 1)
}

func TestResultIngestMaterializationCoordinatorRejectsMalformedLeaseBeforeTransaction(t *testing.T) {
	db := newResultIngestMaterializationDB(t)
	coordinator := newResultIngestMaterializationCoordinator(db)
	scope := testResultMaterializationScope()
	scope.SessionID = " session-17 "
	called := false

	err := coordinator.Materialize(context.Background(), scope, func(context.Context) error {
		called = true
		return nil
	})
	if !errors.Is(err, resultingestapp.ErrResultExecutionFenceRejected) {
		t.Fatalf("Materialize() malformed lease error = %v", err)
	}
	if called {
		t.Fatal("malformed lease reached persistence callback")
	}
}

func testResultMaterializationScope() resultingestapp.ResultMaterializationScope {
	return resultingestapp.ResultMaterializationScope{
		TaskID:       101,
		ScanID:       12,
		TargetID:     34,
		AgentID:      17,
		SessionID:    "session-17",
		SessionEpoch: 23,
	}
}

func insertResultMaterializationEffects(ctx context.Context, root *gorm.DB) error {
	tx := dbtx.Resolve(ctx, root).WithContext(ctx)
	for _, table := range []string{"result_current_asset", "result_snapshot", "result_summary"} {
		if err := tx.Exec(fmt.Sprintf("INSERT INTO %s (id) VALUES (1)", table)).Error; err != nil {
			return err
		}
	}
	return nil
}

func assertResultMaterializationEffects(t *testing.T, db *gorm.DB, want int64) {
	t.Helper()
	for _, table := range []string{"result_current_asset", "result_snapshot", "result_summary"} {
		var got int64
		if err := db.Table(table).Count(&got).Error; err != nil {
			t.Fatalf("count %s: %v", table, err)
		}
		if got != want {
			t.Fatalf("%s rows = %d, want %d", table, got, want)
		}
	}
}

func newResultIngestMaterializationDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := fmt.Sprintf("file:%s-%d?mode=memory&cache=shared", strings.ReplaceAll(t.Name(), "/", "_"), time.Now().UnixNano())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	statements := []string{
		`CREATE TABLE target (id INTEGER PRIMARY KEY, deleted_at DATETIME)`,
		`CREATE TABLE scan (id INTEGER PRIMARY KEY, target_id INTEGER NOT NULL, status TEXT NOT NULL, deleted_at DATETIME)`,
		`CREATE TABLE scan_task (id INTEGER PRIMARY KEY, scan_id INTEGER NOT NULL, status TEXT NOT NULL, assigned_agent_id INTEGER, assigned_session_id TEXT, assigned_session_epoch INTEGER)`,
		`CREATE TABLE agent_runtime_status (agent_id INTEGER PRIMARY KEY, session_id TEXT NOT NULL, session_epoch INTEGER NOT NULL)`,
		`CREATE TABLE result_current_asset (id INTEGER PRIMARY KEY)`,
		`CREATE TABLE result_snapshot (id INTEGER PRIMARY KEY)`,
		`CREATE TABLE result_summary (id INTEGER PRIMARY KEY)`,
		`INSERT INTO target (id) VALUES (34)`,
		`INSERT INTO scan (id, target_id, status) VALUES (12, 34, 'running')`,
		`INSERT INTO scan_task (id, scan_id, status, assigned_agent_id, assigned_session_id, assigned_session_epoch) VALUES (101, 12, 'running', 17, 'session-17', 23)`,
		`INSERT INTO agent_runtime_status (agent_id, session_id, session_epoch) VALUES (17, 'session-17', 23)`,
	}
	for _, statement := range statements {
		if err := db.Exec(statement).Error; err != nil {
			t.Fatalf("prepare result materialization schema: %v", err)
		}
	}
	return db
}
