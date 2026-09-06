package resultingestwiring

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	assetdomain "github.com/yyhuni/lunafox/server/internal/modules/asset/domain"
	assetrepo "github.com/yyhuni/lunafox/server/internal/modules/asset/repository"
	resultingestapp "github.com/yyhuni/lunafox/server/internal/modules/resultingest/application"
	scanrepo "github.com/yyhuni/lunafox/server/internal/modules/scan/repository"
	snapshotdomain "github.com/yyhuni/lunafox/server/internal/modules/snapshot/domain"
	snapshotrepo "github.com/yyhuni/lunafox/server/internal/modules/snapshot/repository"
	"github.com/yyhuni/lunafox/server/internal/pkg/dbtx"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/logger"
)

const (
	scanOperationsPostgresDSNEnv = "LUNAFOX_SCAN_OPERATIONS_POSTGRES_DSN"
	scanOperationsPostgresSchema = "scan_operations_contract"
)

// TestScanOperationsPostgresGate covers lock behavior that SQLite and mock
// repositories cannot reproduce: a root write waits on the caller's own
// parent-row lock, while dbtx.Resolve keeps the materialization on one tx.
func TestScanOperationsPostgresGate(t *testing.T) {
	db := openScanOperationsPostgres(t)

	t.Run("result materialization reuses the caller transaction", func(t *testing.T) {
		for _, resultType := range []struct {
			name        string
			materialize func(context.Context, *gorm.DB, int, int) error
			legacyWrite func(context.Context, *gorm.DB, int) error
			snapshot    string
			asset       string
		}{
			{
				name:        "subdomain",
				materialize: scanOperationsMaterializeSubdomain,
				legacyWrite: func(ctx context.Context, db *gorm.DB, scanID int) error {
					return db.WithContext(ctx).Exec("INSERT INTO subdomain_snapshot (scan_id, dns_name) VALUES (?, ?)", scanID, "legacy.example.com").Error
				},
				snapshot: "subdomain_snapshot",
				asset:    "subdomain",
			},
			{
				name:        "host port",
				materialize: scanOperationsMaterializeHostPort,
				legacyWrite: func(ctx context.Context, db *gorm.DB, scanID int) error {
					return db.WithContext(ctx).Exec("INSERT INTO host_port_mapping_snapshot (scan_id, host, ip, port) VALUES (?, ?, ?::inet, ?)", scanID, "legacy.example.com", "192.0.2.99", 443).Error
				},
				snapshot: "host_port_mapping_snapshot",
				asset:    "host_port_mapping",
			},
		} {
			resultType := resultType
			t.Run(resultType.name, func(t *testing.T) {
				resetScanOperationsPostgresFixture(t, db)
				targetID, scanID := seedScanOperationsTargetAndScan(t, db, "running")
				ctx, cancel := context.WithTimeout(context.Background(), time.Second)
				defer cancel()
				if err := resultType.materialize(ctx, db, scanID, targetID); err != nil {
					t.Fatalf("materialize %s: %v", resultType.name, err)
				}
				if ctx.Err() != nil {
					t.Fatalf("materialize %s reached caller deadline: %v", resultType.name, ctx.Err())
				}
				assertScanOperationsCount(t, db, resultType.snapshot, 1)
				assertScanOperationsCount(t, db, resultType.asset, 1)

				// The pre-fix root-connection shape must be interruptible by the
				// caller deadline rather than turning into an unbounded test wait.
				resetScanOperationsPostgresFixture(t, db)
				targetID, scanID = seedScanOperationsTargetAndScan(t, db, "running")
				assertScanOperationsLegacyRootWriteTimesOut(t, db, targetID, scanID, resultType.legacyWrite)
			})
		}
	})

	t.Run("materialization rolls back snapshot asset and summary together", func(t *testing.T) {
		for _, resultType := range []struct {
			name        string
			materialize func(context.Context, *gorm.DB, int, int) error
			assetTable  string
			snapshot    string
			asset       string
		}{
			{name: "subdomain", materialize: scanOperationsMaterializeSubdomain, assetTable: "subdomain", snapshot: "subdomain_snapshot", asset: "subdomain"},
			{name: "host port", materialize: scanOperationsMaterializeHostPort, assetTable: "host_port_mapping", snapshot: "host_port_mapping_snapshot", asset: "host_port_mapping"},
		} {
			resultType := resultType
			t.Run(resultType.name+" asset failure", func(t *testing.T) {
				resetScanOperationsPostgresFixture(t, db)
				targetID, scanID := seedScanOperationsTargetAndScan(t, db, "running")
				removeFailure := installScanOperationsFailureTrigger(t, db, resultType.assetTable, "INSERT")
				err := resultType.materialize(context.Background(), db, scanID, targetID)
				removeFailure()
				if err == nil {
					t.Fatal("asset failure trigger did not fail materialization")
				}
				assertScanOperationsCount(t, db, resultType.snapshot, 0)
				assertScanOperationsCount(t, db, resultType.asset, 0)
				assertScanOperationsSummaryCount(t, db, scanID, 0)
			})

			t.Run(resultType.name+" summary failure", func(t *testing.T) {
				resetScanOperationsPostgresFixture(t, db)
				targetID, scanID := seedScanOperationsTargetAndScan(t, db, "running")
				removeFailure := installScanOperationsFailureTrigger(t, db, "scan", "UPDATE")
				err := resultType.materialize(context.Background(), db, scanID, targetID)
				removeFailure()
				if err == nil {
					t.Fatal("summary failure trigger did not fail materialization")
				}
				assertScanOperationsCount(t, db, resultType.snapshot, 0)
				assertScanOperationsCount(t, db, resultType.asset, 0)
				assertScanOperationsSummaryCount(t, db, scanID, 0)
			})
		}
	})

	t.Run("scan stop is atomic and ordered against result ingest", func(t *testing.T) {
		resetScanOperationsPostgresFixture(t, db)
		targetID, scanID := seedScanOperationsTargetAndScan(t, db, "running")
		_, completeTaskID := seedScanOperationsRunningTask(t, db, scanID, true)
		seedScanOperationsTask(t, db, scanID, "blocked", false)
		seedScanOperationsTask(t, db, scanID, "pending", false)
		seedScanOperationsTask(t, db, scanID, "running", false)
		stoppedAt := time.Date(2026, time.August, 9, 1, 2, 3, 0, time.UTC)
		outcome, err := scanrepo.NewScanRepository(db).StopActiveScan(context.Background(), scanID, stoppedAt)
		if err != nil {
			t.Fatalf("StopActiveScan: %v", err)
		}
		if outcome.CancelledTaskCount != 4 || len(outcome.NotificationCandidates) != 1 || outcome.NotificationCandidates[0].TaskID != completeTaskID || outcome.NotificationCandidates[0].AgentID != 17 {
			t.Fatalf("unexpected Stop outcome: %+v", outcome)
		}
		assertScanOperationsStoppedState(t, db, scanID, 4, stoppedAt)

		resetScanOperationsPostgresFixture(t, db)
		targetID, scanID = seedScanOperationsTargetAndScan(t, db, "running")
		seedScanOperationsTask(t, db, scanID, "pending", false)
		removeFailure := installScanOperationsFailureTrigger(t, db, "scan", "UPDATE")
		_, err = scanrepo.NewScanRepository(db).StopActiveScan(context.Background(), scanID, stoppedAt)
		removeFailure()
		if err == nil {
			t.Fatal("Scan update failure trigger did not fail Stop")
		}
		assertScanOperationsActiveState(t, db, scanID, "running", "pending")

		resetScanOperationsPostgresFixture(t, db)
		targetID, scanID = seedScanOperationsTargetAndScan(t, db, "running")
		_, taskID := seedScanOperationsRunningTask(t, db, scanID, true)
		coordinator := newResultIngestMaterializationCoordinator(db)
		started := make(chan struct{})
		release := make(chan struct{})
		resultDone := make(chan error, 1)
		go func() {
			resultDone <- coordinator.Materialize(context.Background(), resultingestapp.ResultMaterializationScope{
				TaskID: taskID, ScanID: scanID, TargetID: targetID, AgentID: 17, SessionID: "session-17", SessionEpoch: 23,
			}, func(context.Context) error {
				close(started)
				<-release
				return nil
			})
		}()
		select {
		case <-started:
		case <-time.After(time.Second):
			t.Fatal("result-ingest did not acquire its transaction locks")
		}
		stopDone := make(chan error, 1)
		go func() {
			_, stopErr := scanrepo.NewScanRepository(db).StopActiveScan(context.Background(), scanID, stoppedAt)
			stopDone <- stopErr
		}()
		select {
		case err := <-stopDone:
			t.Fatalf("Stop crossed the uncommitted result-ingest lock: %v", err)
		case <-time.After(100 * time.Millisecond):
		}
		close(release)
		if err := <-resultDone; err != nil {
			t.Fatalf("result-ingest transaction: %v", err)
		}
		select {
		case err := <-stopDone:
			if err != nil {
				t.Fatalf("Stop after result-ingest commit: %v", err)
			}
		case <-time.After(time.Second):
			t.Fatal("Stop and result-ingest lock order did not converge")
		}
		assertScanOperationsStoppedState(t, db, scanID, 1, stoppedAt)
		callbackCalled := false
		err = coordinator.Materialize(context.Background(), resultingestapp.ResultMaterializationScope{
			TaskID: taskID, ScanID: scanID, TargetID: targetID, AgentID: 17, SessionID: "session-17", SessionEpoch: 23,
		}, func(context.Context) error {
			callbackCalled = true
			return nil
		})
		if !errors.Is(err, resultingestapp.ErrResultExecutionFenceRejected) || callbackCalled {
			t.Fatalf("late result after cancelled Task = callback=%t err=%v", callbackCalled, err)
		}
	})

	t.Run("caller cancellation leaves no orphan stop", func(t *testing.T) {
		resetScanOperationsPostgresFixture(t, db)
		_, scanID := seedScanOperationsTargetAndScan(t, db, "running")
		seedScanOperationsTask(t, db, scanID, "pending", false)
		holder := db.Begin()
		if holder.Error != nil {
			t.Fatalf("begin lock holder: %v", holder.Error)
		}
		defer holder.Rollback()
		lockScanOperationsScan(t, holder, scanID)
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()
		_, err := scanrepo.NewScanRepository(db).StopActiveScan(ctx, scanID, time.Now().UTC())
		if err == nil || ctx.Err() == nil {
			t.Fatalf("Stop under caller cancellation = err=%v ctx=%v", err, ctx.Err())
		}
		if err := holder.Rollback().Error; err != nil {
			t.Fatalf("release Scan lock: %v", err)
		}
		assertScanOperationsActiveState(t, db, scanID, "running", "pending")
	})
}

// TestScanOperationsRecoveryFixtures exercises the runbook's two decisions in
// an isolated database: a fresh scope exits without Stop, while an old lock is
// released by a Server restart before the normal Stop command is allowed.
func TestScanOperationsRecoveryFixtures(t *testing.T) {
	db := openScanOperationsPostgres(t)

	t.Run("fresh or unaffected scope exits without Stop", func(t *testing.T) {
		resetScanOperationsPostgresFixture(t, db)
		var activeScans int64
		if err := db.Table("scan").Where("status IN ?", []string{"pending", "running"}).Count(&activeScans).Error; err != nil {
			t.Fatalf("count active Scans: %v", err)
		}
		if activeScans != 0 {
			t.Fatalf("fresh fixture active Scans = %d, want 0", activeScans)
		}
		t.Log("recovery decision=skip reason=no affected Scan or old lock chain")
	})

	t.Run("old lock releases before normal Stop", func(t *testing.T) {
		resetScanOperationsPostgresFixture(t, db)
		_, scanID := seedScanOperationsTargetAndScan(t, db, "running")
		seedScanOperationsTask(t, db, scanID, "running", false)

		holder := db.Begin()
		if holder.Error != nil {
			t.Fatalf("begin old Server lock holder: %v", holder.Error)
		}
		defer holder.Rollback()
		var holderPID int
		if err := holder.Raw("SELECT pg_backend_pid()").Scan(&holderPID).Error; err != nil {
			t.Fatalf("get old Server PID: %v", err)
		}
		lockScanOperationsScan(t, holder, scanID)
		if got := scanOperationsRelationLockCount(t, db, holderPID); got == 0 {
			t.Fatal("old lock fixture did not retain a Scan relation lock")
		}
		t.Logf("affected_scan_id=%d old_server_pid=%d lock_chain=present", scanID, holderPID)

		// Closing the old transaction models a real Server process restart. A
		// page refresh or hot replacement would leave this connection alive.
		if err := holder.Rollback().Error; err != nil {
			t.Fatalf("restart releases old Server transaction: %v", err)
		}
		if got := scanOperationsRelationLockCount(t, db, holderPID); got != 0 {
			t.Fatalf("old Scan relation locks after restart = %d, want 0", got)
		}

		stoppedAt := time.Date(2026, time.August, 9, 2, 3, 4, 0, time.UTC)
		outcome, err := scanrepo.NewScanRepository(db).StopActiveScan(context.Background(), scanID, stoppedAt)
		if err != nil {
			t.Fatalf("normal Stop after restart: %v", err)
		}
		if outcome.CancelledTaskCount != 1 {
			t.Fatalf("normal Stop cancelled tasks = %d, want 1", outcome.CancelledTaskCount)
		}
		assertScanOperationsStoppedState(t, db, scanID, 1, stoppedAt)
		t.Logf("affected_scan_id=%d scan_state=cancelled task_state=cancelled lock_chain=released", scanID)
	})
}

func openScanOperationsPostgres(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := strings.TrimSpace(os.Getenv(scanOperationsPostgresDSNEnv))
	if dsn == "" {
		t.Skip("set " + scanOperationsPostgresDSNEnv + " with search_path=" + scanOperationsPostgresSchema + " to run scan operations PostgreSQL verification")
	}
	if !strings.Contains(dsn, "search_path="+scanOperationsPostgresSchema) {
		t.Fatalf("scan operations PostgreSQL DSN must pin search_path=%s", scanOperationsPostgresSchema)
	}
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatalf("open scan operations PostgreSQL database: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("open scan operations PostgreSQL SQL handle: %v", err)
	}
	sqlDB.SetMaxOpenConns(8)
	t.Cleanup(func() { _ = sqlDB.Close() })
	if err := db.Exec("DROP SCHEMA IF EXISTS " + scanOperationsPostgresSchema + " CASCADE").Error; err != nil {
		t.Fatalf("drop scan operations schema: %v", err)
	}
	if err := db.Exec("CREATE SCHEMA " + scanOperationsPostgresSchema).Error; err != nil {
		t.Fatalf("create scan operations schema: %v", err)
	}
	t.Cleanup(func() {
		cleanupCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		_ = db.WithContext(cleanupCtx).Exec("DROP SCHEMA IF EXISTS " + scanOperationsPostgresSchema + " CASCADE").Error
	})
	for _, statement := range scanOperationsPostgresSchemaStatements() {
		if err := db.Exec(statement).Error; err != nil {
			t.Fatalf("prepare scan operations PostgreSQL schema: %v", err)
		}
	}
	return db
}

func scanOperationsPostgresSchemaStatements() []string {
	return []string{
		`CREATE TABLE target (
			id SERIAL PRIMARY KEY,
			name TEXT NOT NULL DEFAULT '',
			type TEXT NOT NULL DEFAULT 'domain',
			deleted_at TIMESTAMPTZ
		)`,
		`CREATE TABLE scan (
			id SERIAL PRIMARY KEY,
			target_id INTEGER NOT NULL REFERENCES target(id),
			status TEXT NOT NULL,
			deleted_at TIMESTAMPTZ,
			stopped_at TIMESTAMPTZ,
			cached_subdomains_count INTEGER NOT NULL DEFAULT 0,
			cached_websites_count INTEGER NOT NULL DEFAULT 0,
			cached_endpoints_count INTEGER NOT NULL DEFAULT 0,
			cached_ips_count INTEGER NOT NULL DEFAULT 0,
			cached_directories_count INTEGER NOT NULL DEFAULT 0,
			cached_screenshots_count INTEGER NOT NULL DEFAULT 0,
			cached_vulns_total INTEGER NOT NULL DEFAULT 0,
			cached_vulns_critical INTEGER NOT NULL DEFAULT 0,
			cached_vulns_high INTEGER NOT NULL DEFAULT 0,
			cached_vulns_medium INTEGER NOT NULL DEFAULT 0,
			cached_vulns_low INTEGER NOT NULL DEFAULT 0,
			stats_updated_at TIMESTAMPTZ
		)`,
		`CREATE TABLE agent_runtime_status (
			agent_id INTEGER PRIMARY KEY,
			session_id TEXT NOT NULL,
			session_epoch BIGINT NOT NULL
		)`,
		`CREATE TABLE scan_task (
			id SERIAL PRIMARY KEY,
			scan_id INTEGER NOT NULL REFERENCES scan(id),
			-- Stop cancellation derives diagnostics from the saved-plan byte length and writes them here.
			resolved_execution_plan BYTEA NOT NULL DEFAULT ''::bytea,
			engine_diagnostics JSONB,
			status TEXT NOT NULL,
			assigned_agent_id INTEGER,
			assigned_session_id TEXT,
			assigned_session_epoch BIGINT,
			assigned_request_id TEXT,
			completed_at TIMESTAMPTZ
		)`,
		`CREATE TABLE subdomain_snapshot (
			id SERIAL PRIMARY KEY,
			scan_id INTEGER NOT NULL REFERENCES scan(id),
			dns_name TEXT NOT NULL,
			created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
			UNIQUE (scan_id, dns_name)
		)`,
		`CREATE TABLE host_port_mapping_snapshot (
			id SERIAL PRIMARY KEY,
			scan_id INTEGER NOT NULL REFERENCES scan(id),
			host TEXT NOT NULL,
			ip INET NOT NULL,
			port INTEGER NOT NULL,
			created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
			UNIQUE (scan_id, host, ip, port)
		)`,
		`CREATE TABLE website_snapshot (scan_id INTEGER NOT NULL REFERENCES scan(id))`,
		`CREATE TABLE endpoint_snapshot (scan_id INTEGER NOT NULL REFERENCES scan(id))`,
		`CREATE TABLE directory_snapshot (scan_id INTEGER NOT NULL REFERENCES scan(id))`,
		`CREATE TABLE screenshot_snapshot (scan_id INTEGER NOT NULL REFERENCES scan(id))`,
		`CREATE TABLE vulnerability_snapshot (scan_id INTEGER NOT NULL REFERENCES scan(id), severity TEXT NOT NULL DEFAULT 'unknown')`,
		`CREATE TABLE subdomain (
			id SERIAL PRIMARY KEY,
			target_id INTEGER NOT NULL REFERENCES target(id),
			dns_name TEXT NOT NULL,
			created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
			UNIQUE (target_id, dns_name)
		)`,
		`CREATE TABLE host_port_mapping (
			id SERIAL PRIMARY KEY,
			target_id INTEGER NOT NULL REFERENCES target(id),
			host TEXT NOT NULL,
			ip INET NOT NULL,
			port INTEGER NOT NULL,
			created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
			UNIQUE (target_id, host, ip, port)
		)`,
	}
}

func resetScanOperationsPostgresFixture(t *testing.T, db *gorm.DB) {
	t.Helper()
	if err := db.Exec(`TRUNCATE host_port_mapping, subdomain, host_port_mapping_snapshot,
		subdomain_snapshot, website_snapshot, endpoint_snapshot, directory_snapshot,
		screenshot_snapshot, vulnerability_snapshot, scan_task, agent_runtime_status, scan,
		target RESTART IDENTITY CASCADE`).Error; err != nil {
		t.Fatalf("reset scan operations fixture: %v", err)
	}
}

func seedScanOperationsTargetAndScan(t *testing.T, db *gorm.DB, status string) (int, int) {
	t.Helper()
	var targetID, scanID int
	if err := db.Raw("INSERT INTO target (name, type) VALUES (?, 'domain') RETURNING id", "example.com").Scan(&targetID).Error; err != nil {
		t.Fatalf("seed Target: %v", err)
	}
	if err := db.Raw("INSERT INTO scan (target_id, status) VALUES (?, ?) RETURNING id", targetID, status).Scan(&scanID).Error; err != nil {
		t.Fatalf("seed Scan: %v", err)
	}
	return targetID, scanID
}

func seedScanOperationsTask(t *testing.T, db *gorm.DB, scanID int, status string, completeAssignment bool) int {
	t.Helper()
	var taskID int
	if completeAssignment {
		if err := db.Raw(`INSERT INTO scan_task (scan_id, status, assigned_agent_id, assigned_session_id, assigned_session_epoch, assigned_request_id)
			VALUES (?, ?, 17, 'session-17', 23, 'request-17') RETURNING id`, scanID, status).Scan(&taskID).Error; err != nil {
			t.Fatalf("seed assigned Scan Task: %v", err)
		}
		return taskID
	}
	if err := db.Raw("INSERT INTO scan_task (scan_id, status) VALUES (?, ?) RETURNING id", scanID, status).Scan(&taskID).Error; err != nil {
		t.Fatalf("seed Scan Task: %v", err)
	}
	return taskID
}

func seedScanOperationsRunningTask(t *testing.T, db *gorm.DB, scanID int, completeAssignment bool) (int, int) {
	t.Helper()
	if err := db.Exec("INSERT INTO agent_runtime_status (agent_id, session_id, session_epoch) VALUES (17, 'session-17', 23)").Error; err != nil {
		t.Fatalf("seed Agent runtime status: %v", err)
	}
	return 17, seedScanOperationsTask(t, db, scanID, "running", completeAssignment)
}

func scanOperationsMaterializeSubdomain(ctx context.Context, db *gorm.DB, scanID int, targetID int) error {
	snapshots := snapshotrepo.NewSubdomainSnapshotRepository(db)
	assets := assetrepo.NewSubdomainRepository(db)
	scans := scanrepo.NewScanRepository(db)
	return db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := lockScanOperationsResultScope(tx, targetID, scanID); err != nil {
			return err
		}
		txCtx := dbtx.WithTransaction(ctx, tx)
		if _, err := snapshots.BatchCreateContext(txCtx, []snapshotdomain.SubdomainSnapshot{{ScanID: scanID, DNSName: "api.example.com"}}); err != nil {
			return err
		}
		if _, err := assets.BatchCreateContext(txCtx, []assetdomain.Subdomain{{TargetID: targetID, DNSName: "api.example.com"}}); err != nil {
			return err
		}
		return scans.RefreshScanResultSummary(txCtx, scanID, targetID)
	})
}

func scanOperationsMaterializeHostPort(ctx context.Context, db *gorm.DB, scanID int, targetID int) error {
	snapshots := snapshotrepo.NewHostPortSnapshotRepository(db)
	assets := assetrepo.NewHostPortRepository(db)
	scans := scanrepo.NewScanRepository(db)
	return db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := lockScanOperationsResultScope(tx, targetID, scanID); err != nil {
			return err
		}
		txCtx := dbtx.WithTransaction(ctx, tx)
		if _, err := snapshots.BatchCreateContext(txCtx, []snapshotdomain.HostPortSnapshot{{ScanID: scanID, Host: "api.example.com", IP: "192.0.2.10", Port: 443}}); err != nil {
			return err
		}
		if _, err := assets.BatchUpsertContext(txCtx, []assetdomain.HostPort{{TargetID: targetID, Host: "api.example.com", IP: "192.0.2.10", Port: 443}}); err != nil {
			return err
		}
		return scans.RefreshScanResultSummary(txCtx, scanID, targetID)
	})
}

func lockScanOperationsResultScope(tx *gorm.DB, targetID int, scanID int) error {
	var target struct{ ID int }
	if err := tx.Table("target").Clauses(clause.Locking{Strength: "UPDATE"}).Select("id").Where("id = ?", targetID).Take(&target).Error; err != nil {
		return err
	}
	var scan struct{ ID int }
	return tx.Table("scan").Clauses(clause.Locking{Strength: "UPDATE"}).Select("id").Where("id = ?", scanID).Take(&scan).Error
}

func assertScanOperationsLegacyRootWriteTimesOut(t *testing.T, db *gorm.DB, targetID int, scanID int, legacyWrite func(context.Context, *gorm.DB, int) error) {
	t.Helper()
	holder := db.Begin()
	if holder.Error != nil {
		t.Fatalf("begin legacy lock holder: %v", holder.Error)
	}
	lockScanOperationsTarget(t, holder, targetID)
	lockScanOperationsScan(t, holder, scanID)
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	err := legacyWrite(ctx, db, scanID)
	if err == nil || ctx.Err() != context.DeadlineExceeded {
		_ = holder.Rollback().Error
		t.Fatalf("legacy root write = err=%v caller=%v, want caller deadline", err, ctx.Err())
	}
	if err := holder.Rollback().Error; err != nil {
		t.Fatalf("release legacy lock holder: %v", err)
	}
}

func lockScanOperationsTarget(t *testing.T, tx *gorm.DB, targetID int) {
	t.Helper()
	var id int
	if err := tx.Raw("SELECT id FROM target WHERE id = ? FOR UPDATE", targetID).Scan(&id).Error; err != nil || id != targetID {
		t.Fatalf("lock Target %d: id=%d err=%v", targetID, id, err)
	}
}

func lockScanOperationsScan(t *testing.T, tx *gorm.DB, scanID int) {
	t.Helper()
	var id int
	if err := tx.Raw("SELECT id FROM scan WHERE id = ? FOR UPDATE", scanID).Scan(&id).Error; err != nil || id != scanID {
		t.Fatalf("lock Scan %d: id=%d err=%v", scanID, id, err)
	}
}

func scanOperationsRelationLockCount(t *testing.T, db *gorm.DB, pid int) int64 {
	t.Helper()
	var count int64
	if err := db.Raw(`SELECT COUNT(*)
		FROM pg_locks
		WHERE pid = ?
		  AND relation = 'scan'::regclass
		  AND mode = 'RowShareLock'`, pid).Scan(&count).Error; err != nil {
		t.Fatalf("count Scan relation locks for pid %d: %v", pid, err)
	}
	return count
}

func installScanOperationsFailureTrigger(t *testing.T, db *gorm.DB, table string, operation string) func() {
	t.Helper()
	triggerName := "scan_operations_injected_failure"
	functionName := "scan_operations_injected_failure_fn"
	if err := db.Exec(fmt.Sprintf(`CREATE FUNCTION %s() RETURNS trigger LANGUAGE plpgsql AS $$
		BEGIN
			RAISE EXCEPTION 'scan operations injected failure';
		END;
		$$`, functionName)).Error; err != nil {
		t.Fatalf("create injected failure function: %v", err)
	}
	if err := db.Exec(fmt.Sprintf("CREATE TRIGGER %s BEFORE %s ON %s FOR EACH ROW EXECUTE FUNCTION %s()", triggerName, operation, table, functionName)).Error; err != nil {
		t.Fatalf("create injected failure trigger: %v", err)
	}
	return func() {
		if err := db.Exec(fmt.Sprintf("DROP TRIGGER IF EXISTS %s ON %s", triggerName, table)).Error; err != nil {
			t.Fatalf("drop injected failure trigger: %v", err)
		}
		if err := db.Exec("DROP FUNCTION IF EXISTS " + functionName + "()").Error; err != nil {
			t.Fatalf("drop injected failure function: %v", err)
		}
	}
}

func assertScanOperationsCount(t *testing.T, db *gorm.DB, table string, want int64) {
	t.Helper()
	var count int64
	if err := db.Table(table).Count(&count).Error; err != nil {
		t.Fatalf("count %s: %v", table, err)
	}
	if count != want {
		t.Fatalf("%s rows = %d, want %d", table, count, want)
	}
}

func assertScanOperationsSummaryCount(t *testing.T, db *gorm.DB, scanID int, want int) {
	t.Helper()
	var count int
	if err := db.Raw("SELECT cached_subdomains_count + cached_ips_count FROM scan WHERE id = ?", scanID).Scan(&count).Error; err != nil {
		t.Fatalf("read Scan summary: %v", err)
	}
	if count != want {
		t.Fatalf("Scan summary count = %d, want %d", count, want)
	}
}

func assertScanOperationsStoppedState(t *testing.T, db *gorm.DB, scanID int, wantTaskCount int, stoppedAt time.Time) {
	t.Helper()
	var scan struct {
		Status    string
		StoppedAt *time.Time
	}
	if err := db.Table("scan").Select("status", "stopped_at").Where("id = ?", scanID).Take(&scan).Error; err != nil {
		t.Fatalf("read stopped Scan: %v", err)
	}
	if scan.Status != "cancelled" || scan.StoppedAt == nil || !scan.StoppedAt.Equal(stoppedAt) {
		t.Fatalf("stopped Scan = %+v, want cancelled at %s", scan, stoppedAt)
	}
	var tasks []struct {
		Status      string
		CompletedAt *time.Time
	}
	if err := db.Table("scan_task").Select("status", "completed_at").Where("scan_id = ?", scanID).Order("id ASC").Find(&tasks).Error; err != nil {
		t.Fatalf("read stopped Tasks: %v", err)
	}
	if len(tasks) != wantTaskCount {
		t.Fatalf("stopped Task count = %d, want %d", len(tasks), wantTaskCount)
	}
	for _, task := range tasks {
		if task.Status != "cancelled" || task.CompletedAt == nil || !task.CompletedAt.Equal(stoppedAt) {
			t.Fatalf("stopped Task = %+v, want cancelled at %s", task, stoppedAt)
		}
	}
}

func assertScanOperationsActiveState(t *testing.T, db *gorm.DB, scanID int, wantScanStatus string, wantTaskStatus string) {
	t.Helper()
	var scanStatus string
	if err := db.Raw("SELECT status FROM scan WHERE id = ?", scanID).Scan(&scanStatus).Error; err != nil {
		t.Fatalf("read active Scan: %v", err)
	}
	var taskStatus string
	if err := db.Raw("SELECT status FROM scan_task WHERE scan_id = ? ORDER BY id ASC LIMIT 1", scanID).Scan(&taskStatus).Error; err != nil {
		t.Fatalf("read active Task: %v", err)
	}
	if scanStatus != wantScanStatus || taskStatus != wantTaskStatus {
		t.Fatalf("active state = scan:%s task:%s, want scan:%s task:%s", scanStatus, taskStatus, wantScanStatus, wantTaskStatus)
	}
}
