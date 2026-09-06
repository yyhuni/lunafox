package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	resultingestwiring "github.com/yyhuni/lunafox/server/internal/bootstrap/wiring/resultingest"
	serverdatabase "github.com/yyhuni/lunafox/server/internal/database"
	catalogdomain "github.com/yyhuni/lunafox/server/internal/modules/catalog/domain"
	catalogrepository "github.com/yyhuni/lunafox/server/internal/modules/catalog/repository"
	identityrepository "github.com/yyhuni/lunafox/server/internal/modules/identity/repository"
	resultingestapp "github.com/yyhuni/lunafox/server/internal/modules/resultingest/application"
	scandomain "github.com/yyhuni/lunafox/server/internal/modules/scan/domain"
	scanrepository "github.com/yyhuni/lunafox/server/internal/modules/scan/repository"
	scheduledapp "github.com/yyhuni/lunafox/server/internal/modules/scheduledscan/application"
	scheduledrepository "github.com/yyhuni/lunafox/server/internal/modules/scheduledscan/repository"
	targetcleanupapp "github.com/yyhuni/lunafox/server/internal/modules/targetcleanup/application"
	cleanupdomain "github.com/yyhuni/lunafox/server/internal/modules/targetcleanup/domain"
	targetcleanuprepository "github.com/yyhuni/lunafox/server/internal/modules/targetcleanup/repository"
	"github.com/yyhuni/lunafox/server/internal/pkg/dbtx"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

const targetCleanupPostgresGateDSNEnv = "LUNAFOX_TARGET_CLEANUP_POSTGRES_DSN"

// TestTargetCleanupPostgresGate is intentionally opt-in. The release command
// always supplies a freshly provisioned PostgreSQL 18 DSN; ordinary Go tests
// skip it so a developer does not accidentally run destructive fixtures.
func TestTargetCleanupPostgresGate(t *testing.T) {
	db := openTargetCleanupPostgresGate(t)

	t.Run("restart reconciliation preserves committed effects and retries rolled back effects", func(t *testing.T) {
		targetCleanupPostgresRestartScenario(t, db)
	})
	t.Run("completion conditions are exhaustive while retained history is ignored", func(t *testing.T) {
		targetCleanupPostgresCompletionConditions(t, db)
	})
	t.Run("transaction-local timeout settings do not leak", func(t *testing.T) {
		targetCleanupPostgresTimeoutScope(t, db)
	})
	t.Run("Target writers and result materialization serialize with delete in both commit orders", func(t *testing.T) {
		targetCleanupPostgresWriteDeleteCommitOrders(t, db)
	})
	t.Run("cancellation remains the database fact when Agent delivery or result arrival is late", func(t *testing.T) {
		targetCleanupPostgresCancellationOutcomes(t, db)
	})
	t.Run("commit-boundary faults converge after restart from database truth", func(t *testing.T) {
		targetCleanupPostgresCommitBoundaryRecovery(t, db)
	})
}

func openTargetCleanupPostgresGate(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := strings.TrimSpace(os.Getenv(targetCleanupPostgresGateDSNEnv))
	if dsn == "" {
		t.Skip("set " + targetCleanupPostgresGateDSNEnv + " to run the PostgreSQL 18 Target cleanup gate")
	}
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: gormlogger.Default.LogMode(gormlogger.Silent)})
	if err != nil {
		t.Fatalf("open Target cleanup PostgreSQL gate database: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("open Target cleanup PostgreSQL SQL handle: %v", err)
	}
	sqlDB.SetMaxOpenConns(16)
	t.Cleanup(func() { _ = sqlDB.Close() })

	var databaseName string
	if err := db.Raw("SELECT current_database()").Scan(&databaseName).Error; err != nil {
		t.Fatalf("read Target cleanup gate database name: %v", err)
	}
	if !strings.HasPrefix(databaseName, "lunafox_target_cleanup_") {
		t.Fatalf("refusing destructive Target cleanup gate database %q", databaseName)
	}
	var publicTables int64
	if err := db.Raw(`SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = 'public'`).Scan(&publicTables).Error; err != nil {
		t.Fatalf("inspect Target cleanup gate database: %v", err)
	}
	if publicTables != 0 {
		t.Fatalf("Target cleanup gate must start from an empty public schema, found %d tables", publicTables)
	}

	previousFS, previousPath := serverdatabase.MigrationsFS, serverdatabase.MigrationsPath
	serverdatabase.MigrationsFS = migrationsFS
	serverdatabase.MigrationsPath = "migrations"
	t.Cleanup(func() {
		serverdatabase.MigrationsFS = previousFS
		serverdatabase.MigrationsPath = previousPath
	})
	if err := serverdatabase.RunMigrations(sqlDB); err != nil {
		t.Fatalf("apply empty 000001 baseline for Target cleanup gate: %v", err)
	}
	t.Cleanup(func() {
		if err := serverdatabase.MigrateDown(sqlDB); err != nil {
			t.Errorf("tear down Target cleanup gate baseline: %v", err)
		}
	})
	return db
}

func targetCleanupPostgresRestartScenario(t *testing.T, db *gorm.DB) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	now := time.Now().UTC().Truncate(time.Microsecond)
	oldTargetID := targetCleanupGateCreateTarget(t, db, "restart-reconcile.example")
	if _, err := catalogrepository.NewTargetRepository(db).TombstoneAndEnsureCleanup(ctx, oldTargetID); err != nil {
		t.Fatalf("tombstone Target: %v", err)
	}
	newTargetID := targetCleanupGateCreateTarget(t, db, "restart-reconcile.example")
	targetCleanupGateSeedOwnedState(t, db, oldTargetID)
	if err := db.Exec("INSERT INTO subdomain (target_id, dns_name) VALUES (?, ?)", newTargetID, "replacement.restart-reconcile.example").Error; err != nil {
		t.Fatalf("seed replacement Target asset: %v", err)
	}

	cleanupStore := targetcleanuprepository.NewTargetCleanupRepository(db)
	job := targetCleanupGateJob(t, cleanupStore, oldTargetID)
	reconcilerDependencies := newTargetCleanupGateReconcilerDependencies(db)
	firstStore := &targetCleanupGateFailAfterAssetStore{TargetCleanupDataStore: cleanupStore, resource: cleanupdomain.AssetResourceSubdomain}
	firstService, err := targetcleanupapp.NewTargetCleanupReconciliationService(firstStore, reconcilerDependencies.schedules, reconcilerDependencies.scans, &targetCleanupGatePublisher{deliver: false})
	if err != nil {
		t.Fatalf("create first reconciliation service: %v", err)
	}
	firstService.WithClock(func() time.Time { return now })
	if _, err := firstService.Reconcile(ctx, job, targetcleanupapp.TargetCleanupRunOptions{AssetBatchSize: 1, MaxAssetBatches: 100, MaxRunDuration: time.Minute}); err == nil {
		t.Fatal("first reconciliation crossed an injected post-commit failure")
	}
	if remaining := targetCleanupGateCount(t, db, "subdomain", "target_id = ?", oldTargetID); remaining != 0 {
		t.Fatalf("committed first asset batch was not preserved, remaining=%d", remaining)
	}
	if remaining := targetCleanupGateCount(t, db, "website", "target_id = ?", oldTargetID); remaining == 0 {
		t.Fatal("first failure unexpectedly removed later asset tables")
	}

	secondPublisher := &targetCleanupGatePublisher{deliver: false}
	secondService, err := targetcleanupapp.NewTargetCleanupReconciliationService(cleanupStore, reconcilerDependencies.schedules, reconcilerDependencies.scans, secondPublisher)
	if err != nil {
		t.Fatalf("create restarted reconciliation service: %v", err)
	}
	secondService.WithClock(func() time.Time { return now.Add(time.Minute) })
	result, err := secondService.Reconcile(ctx, job, targetcleanupapp.TargetCleanupRunOptions{AssetBatchSize: 2, MaxAssetBatches: 100, MaxRunDuration: time.Minute})
	if err != nil || !result.Completed {
		t.Fatalf("restarted reconciliation = %+v, %v", result, err)
	}
	for _, resource := range cleanupdomain.OrderedAssetResources() {
		table := targetCleanupGateAssetTable(resource)
		if remaining := targetCleanupGateCount(t, db, table, "target_id = ?", oldTargetID); remaining != 0 {
			t.Fatalf("restarted cleanup left %s rows=%d", table, remaining)
		}
	}
	if got := targetCleanupGateCount(t, db, "subdomain", "target_id = ?", newTargetID); got != 1 {
		t.Fatalf("replacement Target asset count=%d, want 1", got)
	}
	if targetCleanupGateCount(t, db, "scheduled_scan", "target_id = ?", oldTargetID) != 0 || targetCleanupGateCount(t, db, "scheduled_scan_occurrence", "scheduled_scan_id NOT IN (SELECT id FROM scheduled_scan)") != 0 {
		t.Fatal("Target-scoped Schedule or occurrence survived cleanup")
	}
	if targetCleanupGateCount(t, db, "organization_target", "target_id = ?", oldTargetID) != 0 {
		t.Fatal("Organization relationship survived cleanup")
	}
	if targetCleanupGateCount(t, db, "blacklist_policy", "scope = 'target' AND target_id = ?", oldTargetID) != 0 {
		t.Fatal("Target blacklist policy survived cleanup")
	}
	if targetCleanupGateCount(t, db, "blacklist_policy", "scope = 'global' AND target_id IS NULL") != 1 {
		t.Fatal("global blacklist policy was not preserved")
	}
	if targetCleanupGateCount(t, db, "scan", "id = (SELECT MIN(id) FROM scan WHERE target_id = ?)", oldTargetID) != 1 {
		t.Fatal("retained Scan history was deleted by Target cleanup")
	}
	var scanStatus struct {
		Status    string
		StoppedAt *time.Time `gorm:"column:stopped_at"`
	}
	if err := db.Table("scan").Where("target_id = ?", oldTargetID).Take(&scanStatus).Error; err != nil {
		t.Fatalf("read retained Scan history: %v", err)
	}
	if scanStatus.Status != "cancelled" || scanStatus.StoppedAt == nil {
		t.Fatalf("retained Scan state = %+v, want cancelled with stopped_at", scanStatus)
	}
	if secondPublisher.calls != 0 {
		t.Fatalf("restart resent an already committed Agent notification: calls=%d", secondPublisher.calls)
	}
	var jobStatus string
	if err := db.Table("target_cleanup_job").Select("status").Where("target_id = ?", oldTargetID).Scan(&jobStatus).Error; err != nil {
		t.Fatalf("read completed Target cleanup Job: %v", err)
	}
	if jobStatus != "completed" {
		t.Fatalf("Target cleanup Job status=%q, want completed", jobStatus)
	}
}

func targetCleanupPostgresCompletionConditions(t *testing.T, db *gorm.DB) {
	t.Helper()
	ctx := context.Background()
	store := targetcleanuprepository.NewTargetCleanupRepository(db)
	conditionCases := []struct {
		name string
		seed func(*testing.T, *gorm.DB, int)
	}{
		{name: "target schedule", seed: func(t *testing.T, db *gorm.DB, targetID int) {
			targetCleanupGateSeedSchedule(t, db, targetID)
		}},
		{name: "schedule occurrence", seed: func(t *testing.T, db *gorm.DB, targetID int) {
			targetCleanupGateSeedSchedule(t, db, targetID)
		}},
		{name: "Organization relationship", seed: func(t *testing.T, db *gorm.DB, targetID int) {
			var organizationID int
			if err := db.Raw("INSERT INTO organization (name) VALUES (?) RETURNING id", "condition-org").Scan(&organizationID).Error; err != nil {
				t.Fatalf("seed Organization: %v", err)
			}
			if err := db.Exec("INSERT INTO organization_target (organization_id, target_id) VALUES (?, ?)", organizationID, targetID).Error; err != nil {
				t.Fatalf("seed Organization relationship: %v", err)
			}
		}},
		{name: "Target policy", seed: func(t *testing.T, db *gorm.DB, targetID int) {
			if err := db.Exec("INSERT INTO blacklist_policy (scope, target_id, patterns) VALUES ('target', ?, '[]'::jsonb)", targetID).Error; err != nil {
				t.Fatalf("seed Target policy: %v", err)
			}
		}},
		{name: "active Scan", seed: func(t *testing.T, db *gorm.DB, targetID int) {
			if err := db.Exec("INSERT INTO scan (target_id, scan_workflow_id, input_source, trigger_type, status) VALUES (?, 'condition-check', 'scan_snapshot', 'manual', 'pending')", targetID).Error; err != nil {
				t.Fatalf("seed active Scan: %v", err)
			}
		}},
	}
	for _, resource := range cleanupdomain.OrderedAssetResources() {
		resource := resource
		conditionCases = append(conditionCases, struct {
			name string
			seed func(*testing.T, *gorm.DB, int)
		}{name: string(resource), seed: func(t *testing.T, db *gorm.DB, targetID int) {
			targetCleanupGateSeedOneAsset(t, db, targetID, resource)
		}})
	}
	for _, condition := range conditionCases {
		condition := condition
		t.Run(condition.name, func(t *testing.T) {
			targetID, jobID := targetCleanupGateCreateTombstoneJob(t, db, "condition-"+strings.ReplaceAll(condition.name, " ", "-"))
			condition.seed(t, db, targetID)
			completed, err := store.MarkCompletedIfClear(ctx, jobID, targetID, time.Now().UTC())
			if err != nil {
				t.Fatalf("MarkCompletedIfClear(%s): %v", condition.name, err)
			}
			if completed {
				t.Fatalf("false completion for condition %s", condition.name)
			}
		})
	}

	historyTargetID, historyJobID := targetCleanupGateCreateTombstoneJob(t, db, "history-is-not-a-condition")
	var historyScanID int
	if err := db.Raw("INSERT INTO scan (target_id, scan_workflow_id, input_source, trigger_type, status) VALUES (?, 'history-check', 'scan_snapshot', 'manual', 'cancelled') RETURNING id", historyTargetID).Scan(&historyScanID).Error; err != nil {
		t.Fatalf("seed terminal history Scan: %v", err)
	}
	if err := db.Exec(`INSERT INTO scan_task (scan_id, stage_id, step_id, engine_id, status) VALUES (?, 'history', 'step', 'engine.history', 'succeeded')`, historyScanID).Error; err != nil {
		t.Fatalf("seed terminal history Task: %v", err)
	}
	if err := db.Exec("INSERT INTO scan_blacklist_snapshot (scan_id, patterns) VALUES (?, '[]'::jsonb)", historyScanID).Error; err != nil {
		t.Fatalf("seed retained blacklist snapshot: %v", err)
	}
	completed, err := store.MarkCompletedIfClear(ctx, historyJobID, historyTargetID, time.Now().UTC())
	if err != nil || !completed {
		t.Fatalf("retained history must not block completion: completed=%t err=%v", completed, err)
	}
	if targetCleanupGateCount(t, db, "scan", "id = ?", historyScanID) != 1 || targetCleanupGateCount(t, db, "scan_task", "scan_id = ?", historyScanID) != 1 {
		t.Fatal("completion verification touched retained Scan history")
	}
}

func targetCleanupPostgresTimeoutScope(t *testing.T, db *gorm.DB) {
	t.Helper()
	targetID, jobID := targetCleanupGateCreateTombstoneJob(t, db, "timeout-scope.example")
	targetCleanupGateSeedSchedule(t, db, targetID)
	targetCleanupGateSeedOwnedStateWithoutSchedule(t, db, targetID)
	trace := &targetCleanupGateSQLTrace{}
	traced := db.Session(&gorm.Session{Logger: trace})
	store := targetcleanuprepository.NewTargetCleanupRepository(traced)
	dependencies := newTargetCleanupGateReconcilerDependencies(traced)
	service, err := targetcleanupapp.NewTargetCleanupReconciliationService(store, dependencies.schedules, dependencies.scans, &targetCleanupGatePublisher{deliver: false})
	if err != nil {
		t.Fatalf("create timeout-scope reconciliation service: %v", err)
	}
	result, err := service.Reconcile(context.Background(), cleanupdomain.CleanupJob{ID: jobID, TargetID: targetID}, targetcleanupapp.TargetCleanupRunOptions{
		AssetBatchSize: 1, MaxAssetBatches: 100, MaxRunDuration: time.Minute,
	})
	if err != nil || !result.Completed {
		t.Fatalf("timeout-scope Reconcile() = %+v, %v", result, err)
	}
	lockTimeoutSets := trace.Count("set_config('lock_timeout', '5s', true)")
	statementTimeoutSets := trace.Count("set_config('statement_timeout', '30s', true)")
	// The run opens one confirm transaction, two Schedule transactions, one
	// control-plane transaction, two Scan-cancellation transactions, fourteen
	// asset transactions, and one final verification transaction.
	const expectedCleanupTransactions = 21
	if lockTimeoutSets != expectedCleanupTransactions || statementTimeoutSets != expectedCleanupTransactions {
		t.Fatalf("cleanup transactions did not configure matching local timeouts: lock=%d statement=%d", lockTimeoutSets, statementTimeoutSets)
	}
	assertTargetCleanupGateTimeoutDefaults(t, traced)

	// A deliberately short local timeout demonstrates rollback scope without
	// waiting five seconds for the production cleanup setting.
	if err := traced.Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec("SELECT set_config('statement_timeout', '10ms', true)").Error; err != nil {
			return err
		}
		if err := tx.Exec("SELECT pg_sleep(0.1)").Error; err == nil {
			return errors.New("statement timeout did not fire")
		}
		return errors.New("rollback local timeout test")
	}); err == nil {
		t.Fatal("local timeout transaction unexpectedly committed")
	}
	assertTargetCleanupGateTimeoutDefaults(t, traced)
}

func targetCleanupPostgresWriteDeleteCommitOrders(t *testing.T, db *gorm.DB) {
	t.Helper()
	t.Run("Target update", func(t *testing.T) {
		targetCleanupPostgresTargetUpdateCommitOrders(t, db)
	})
	t.Run("Organization relationship", func(t *testing.T) {
		targetCleanupPostgresOrganizationRelationshipCommitOrders(t, db)
	})
	t.Run("Target-scoped Schedule", func(t *testing.T) {
		targetCleanupPostgresScheduledScanCommitOrders(t, db)
	})
	t.Run("ordinary Scan creation", func(t *testing.T) {
		targetCleanupPostgresScanCreateCommitOrders(t, db)
	})
	t.Run("result materialization", func(t *testing.T) {
		targetCleanupPostgresResultMaterializationCommitOrders(t, db)
	})
}

func targetCleanupPostgresTargetUpdateCommitOrders(t *testing.T, db *gorm.DB) {
	t.Helper()
	repo := catalogrepository.NewTargetRepository(db)
	t.Run("DELETE first rejects update", func(t *testing.T) {
		targetID := targetCleanupGateCreateTarget(t, db, "update-delete-first.example")
		if _, err := repo.TombstoneAndEnsureCleanup(context.Background(), targetID); err != nil {
			t.Fatalf("tombstone Target: %v", err)
		}
		err := repo.Update(&catalogdomain.Target{ID: targetID, Name: "update-rejected.example", Type: catalogdomain.TargetTypeDomain})
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			t.Fatalf("Update() error = %v, want unavailable Target", err)
		}
		targetCleanupGateAssertTargetState(t, db, targetID, "update-delete-first.example", true, 1)
	})

	t.Run("update first commits before DELETE", func(t *testing.T) {
		targetID := targetCleanupGateCreateTarget(t, db, "update-first.example")
		blocker := newTargetCleanupGateBlocker()
		targetCleanupGateRegisterUpdateBlocker(t, db, blocker)

		updateDone := make(chan error, 1)
		go func() {
			updateDone <- repo.Update(&catalogdomain.Target{ID: targetID, Name: "update-first-committed.example", Type: catalogdomain.TargetTypeDomain})
		}()
		targetCleanupGateAwaitSignal(t, blocker.reached, "Target update lock")

		deleteDone := make(chan targetCleanupGateDeleteResult, 1)
		go func() {
			deleted, err := repo.TombstoneAndEnsureCleanup(context.Background(), targetID)
			deleteDone <- targetCleanupGateDeleteResult{deleted: deleted, err: err}
		}()
		targetCleanupGateAssertBlocked(t, deleteDone, "Target DELETE behind update")
		blocker.Release()

		if err := targetCleanupGateAwaitResult(t, updateDone, "Target update"); err != nil {
			t.Fatalf("Update() after lock = %v", err)
		}
		deleted := targetCleanupGateAwaitResult(t, deleteDone, "Target DELETE")
		if deleted.err != nil || !deleted.deleted {
			t.Fatalf("TombstoneAndEnsureCleanup() = %+v", deleted)
		}
		targetCleanupGateAssertTargetState(t, db, targetID, "update-first-committed.example", true, 1)
	})
}

func targetCleanupPostgresOrganizationRelationshipCommitOrders(t *testing.T, db *gorm.DB) {
	t.Helper()
	repo := identityrepository.NewOrganizationRepository(db)
	t.Run("DELETE first rejects relationship", func(t *testing.T) {
		targetID := targetCleanupGateCreateTarget(t, db, "relationship-delete-first.example")
		organizationID := targetCleanupGateCreateOrganization(t, db, "relationship-delete-first")
		if _, err := catalogrepository.NewTargetRepository(db).TombstoneAndEnsureCleanup(context.Background(), targetID); err != nil {
			t.Fatalf("tombstone Target: %v", err)
		}
		err := repo.BatchAddTargets(organizationID, []int{targetID})
		if !errors.Is(err, identityrepository.ErrTargetNotFound) {
			t.Fatalf("BatchAddTargets() error = %v, want unavailable Target", err)
		}
		if got := targetCleanupGateCount(t, db, "organization_target", "organization_id = ? AND target_id = ?", organizationID, targetID); got != 0 {
			t.Fatalf("DELETE-first relationship rows = %d, want 0", got)
		}
	})

	t.Run("relationship first commits before DELETE", func(t *testing.T) {
		targetID := targetCleanupGateCreateTarget(t, db, "relationship-first.example")
		organizationID := targetCleanupGateCreateOrganization(t, db, "relationship-first")
		blocker := newTargetCleanupGateBlocker()
		targetCleanupGateRegisterQueryBlocker(t, db, blocker)

		relationshipDone := make(chan error, 1)
		go func() {
			relationshipDone <- repo.BatchAddTargets(organizationID, []int{targetID})
		}()
		targetCleanupGateAwaitSignal(t, blocker.reached, "Organization relationship Target lock")

		deleteDone := make(chan targetCleanupGateDeleteResult, 1)
		go func() {
			deleted, err := catalogrepository.NewTargetRepository(db).TombstoneAndEnsureCleanup(context.Background(), targetID)
			deleteDone <- targetCleanupGateDeleteResult{deleted: deleted, err: err}
		}()
		targetCleanupGateAssertBlocked(t, deleteDone, "Target DELETE behind Organization relationship")
		blocker.Release()

		if err := targetCleanupGateAwaitResult(t, relationshipDone, "Organization relationship"); err != nil {
			t.Fatalf("BatchAddTargets() = %v", err)
		}
		deleted := targetCleanupGateAwaitResult(t, deleteDone, "Target DELETE")
		if deleted.err != nil || !deleted.deleted {
			t.Fatalf("TombstoneAndEnsureCleanup() = %+v", deleted)
		}
		if got := targetCleanupGateCount(t, db, "organization_target", "organization_id = ? AND target_id = ?", organizationID, targetID); got != 1 {
			t.Fatalf("relationship-first rows = %d, want 1 before asynchronous cleanup", got)
		}
	})
}

func targetCleanupPostgresScheduledScanCommitOrders(t *testing.T, db *gorm.DB) {
	t.Helper()
	t.Run("DELETE first rejects creation", func(t *testing.T) {
		targetID := targetCleanupGateCreateTarget(t, db, "schedule-delete-first.example")
		if _, err := catalogrepository.NewTargetRepository(db).TombstoneAndEnsureCleanup(context.Background(), targetID); err != nil {
			t.Fatalf("tombstone Target: %v", err)
		}
		_, err := scheduledrepository.NewScheduledScanRepository(db).Create(context.Background(), targetCleanupGateScheduleCreateInput(targetID, "schedule-delete-first"))
		if !errors.Is(err, scheduledapp.ErrScheduledScanInvalidArgument) {
			t.Fatalf("Create() error = %v, want unavailable Target", err)
		}
		if got := targetCleanupGateCount(t, db, "scheduled_scan", "target_id = ?", targetID); got != 0 {
			t.Fatalf("DELETE-first Target Schedule rows = %d, want 0", got)
		}
	})

	t.Run("Schedule creation first commits before DELETE", func(t *testing.T) {
		targetID := targetCleanupGateCreateTarget(t, db, "schedule-first.example")
		blocker := newTargetCleanupGateBlocker()
		calculator := &targetCleanupGateBlockingScheduleCalculator{
			delegate: scheduledapp.NewCronScheduleCalculator(),
			blocker:  blocker,
		}
		repo := scheduledrepository.NewScheduledScanRepository(db, calculator)
		createDone := make(chan targetCleanupGateScheduleCreateResult, 1)
		go func() {
			item, err := repo.Create(context.Background(), targetCleanupGateScheduleCreateInput(targetID, "schedule-first"))
			createDone <- targetCleanupGateScheduleCreateResult{item: item, err: err}
		}()
		targetCleanupGateAwaitSignal(t, blocker.reached, "Target Schedule lock")

		deleteDone := make(chan targetCleanupGateDeleteResult, 1)
		go func() {
			deleted, err := catalogrepository.NewTargetRepository(db).TombstoneAndEnsureCleanup(context.Background(), targetID)
			deleteDone <- targetCleanupGateDeleteResult{deleted: deleted, err: err}
		}()
		targetCleanupGateAssertBlocked(t, deleteDone, "Target DELETE behind Schedule creation")
		blocker.Release()

		created := targetCleanupGateAwaitResult(t, createDone, "Target Schedule creation")
		if created.err != nil || created.item == nil || created.item.ID <= 0 {
			t.Fatalf("Create() = %+v, %v", created.item, created.err)
		}
		deleted := targetCleanupGateAwaitResult(t, deleteDone, "Target DELETE")
		if deleted.err != nil || !deleted.deleted {
			t.Fatalf("TombstoneAndEnsureCleanup() = %+v", deleted)
		}
		if got := targetCleanupGateCount(t, db, "scheduled_scan", "id = ? AND target_id = ?", created.item.ID, targetID); got != 1 {
			t.Fatalf("Schedule creation-first rows = %d, want 1 before asynchronous cleanup", got)
		}
	})
}

func targetCleanupPostgresScanCreateCommitOrders(t *testing.T, db *gorm.DB) {
	t.Helper()
	t.Run("DELETE first rejects Scan aggregate before dependent work", func(t *testing.T) {
		targetID := targetCleanupGateCreateTarget(t, db, "scan-delete-first.example")
		if _, err := catalogrepository.NewTargetRepository(db).TombstoneAndEnsureCleanup(context.Background(), targetID); err != nil {
			t.Fatalf("tombstone Target: %v", err)
		}
		resolverCalled := false
		err := scanrepository.NewScanRepository(db).CreateWithScanTasksAndPlans(
			context.Background(),
			targetCleanupGateScanCreateInput(targetID),
			func(context.Context, int) ([]string, error) {
				resolverCalled = true
				return []string{}, nil
			},
			func(int, int, *scandomain.CreateScanTask) error { return nil },
		)
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			t.Fatalf("CreateWithScanTasksAndPlans() error = %v, want unavailable Target", err)
		}
		if resolverCalled {
			t.Fatal("deleted Target reached dependent Scan creation work")
		}
		if got := targetCleanupGateCount(t, db, "scan", "target_id = ?", targetID); got != 0 {
			t.Fatalf("DELETE-first Scan rows = %d, want 0", got)
		}
	})

	t.Run("Scan creation first commits before DELETE", func(t *testing.T) {
		targetID := targetCleanupGateCreateTarget(t, db, "scan-first.example")
		blocker := newTargetCleanupGateBlocker()
		created := targetCleanupGateScanCreateInput(targetID)
		createDone := make(chan error, 1)
		go func() {
			createDone <- scanrepository.NewScanRepository(db).CreateWithScanTasksAndPlans(
				context.Background(),
				created,
				func(context.Context, int) ([]string, error) {
					blocker.Block()
					return []string{}, nil
				},
				func(int, int, *scandomain.CreateScanTask) error { return nil },
			)
		}()
		targetCleanupGateAwaitSignal(t, blocker.reached, "ordinary Scan Target lock")

		deleteDone := make(chan targetCleanupGateDeleteResult, 1)
		go func() {
			deleted, err := catalogrepository.NewTargetRepository(db).TombstoneAndEnsureCleanup(context.Background(), targetID)
			deleteDone <- targetCleanupGateDeleteResult{deleted: deleted, err: err}
		}()
		targetCleanupGateAssertBlocked(t, deleteDone, "Target DELETE behind Scan creation")
		blocker.Release()

		if err := targetCleanupGateAwaitResult(t, createDone, "ordinary Scan creation"); err != nil {
			t.Fatalf("CreateWithScanTasksAndPlans() = %v", err)
		}
		deleted := targetCleanupGateAwaitResult(t, deleteDone, "Target DELETE")
		if deleted.err != nil || !deleted.deleted {
			t.Fatalf("TombstoneAndEnsureCleanup() = %+v", deleted)
		}
		if created.ID <= 0 || targetCleanupGateCount(t, db, "scan", "id = ? AND target_id = ?", created.ID, targetID) != 1 {
			t.Fatalf("Scan creation did not commit its aggregate before DELETE: %+v", created)
		}
	})
}

func targetCleanupPostgresResultMaterializationCommitOrders(t *testing.T, db *gorm.DB) {
	t.Helper()
	t.Run("persisted process takeover rejects a late result before Task fencing", func(t *testing.T) {
		targetID := targetCleanupGateCreateTarget(t, db, "result-session-takeover.example")
		_, _, scope := targetCleanupGateSeedResultExecution(t, db, targetID, "result-session-takeover")
		if err := db.Exec(
			"UPDATE agent_runtime_status SET session_id = ?, session_epoch = ? WHERE agent_id = ?",
			"replacement-"+scope.SessionID,
			scope.SessionEpoch+1,
			scope.AgentID,
		).Error; err != nil {
			t.Fatalf("persist replacement result session: %v", err)
		}
		persisted := false
		err := resultingestwiring.NewResultIngestMaterializationCoordinator(db).Materialize(context.Background(), scope, func(context.Context) error {
			persisted = true
			return nil
		})
		if !errors.Is(err, resultingestapp.ErrResultExecutionFenceRejected) || persisted {
			t.Fatalf("late result after persisted takeover = %v, persisted=%t", err, persisted)
		}
	})

	t.Run("DELETE first rejects the whole result batch", func(t *testing.T) {
		targetID := targetCleanupGateCreateTarget(t, db, "result-delete-first.example")
		scanID, taskID, scope := targetCleanupGateSeedResultExecution(t, db, targetID, "result-delete-first")
		if _, err := catalogrepository.NewTargetRepository(db).TombstoneAndEnsureCleanup(context.Background(), targetID); err != nil {
			t.Fatalf("tombstone Target: %v", err)
		}
		persisted := false
		err := resultingestwiring.NewResultIngestMaterializationCoordinator(db).Materialize(context.Background(), scope, func(context.Context) error {
			persisted = true
			return nil
		})
		if !errors.Is(err, resultingestapp.ErrResultExecutionFenceRejected) {
			t.Fatalf("Materialize() error = %v, want execution fence rejection", err)
		}
		if persisted || targetCleanupGateCount(t, db, "subdomain", "target_id = ?", targetID) != 0 {
			t.Fatalf("DELETE-first result materialization wrote state: persisted=%t target=%d scan=%d task=%d", persisted, targetID, scanID, taskID)
		}
	})

	t.Run("materialization first commits before DELETE", func(t *testing.T) {
		targetID := targetCleanupGateCreateTarget(t, db, "result-first.example")
		_, _, scope := targetCleanupGateSeedResultExecution(t, db, targetID, "result-first")
		blocker := newTargetCleanupGateBlocker()
		materializeDone := make(chan error, 1)
		go func() {
			materializeDone <- resultingestwiring.NewResultIngestMaterializationCoordinator(db).Materialize(context.Background(), scope, func(ctx context.Context) error {
				blocker.Block()
				return dbtx.Resolve(ctx, db).WithContext(ctx).Exec(
					"INSERT INTO subdomain (target_id, dns_name) VALUES (?, ?)",
					targetID,
					"materialized-result-first.example",
				).Error
			})
		}()
		targetCleanupGateAwaitSignal(t, blocker.reached, "result materialization execution locks")

		deleteDone := make(chan targetCleanupGateDeleteResult, 1)
		go func() {
			deleted, err := catalogrepository.NewTargetRepository(db).TombstoneAndEnsureCleanup(context.Background(), targetID)
			deleteDone <- targetCleanupGateDeleteResult{deleted: deleted, err: err}
		}()
		targetCleanupGateAssertBlocked(t, deleteDone, "Target DELETE behind result materialization")
		blocker.Release()

		if err := targetCleanupGateAwaitResult(t, materializeDone, "result materialization"); err != nil {
			t.Fatalf("Materialize() = %v", err)
		}
		deleted := targetCleanupGateAwaitResult(t, deleteDone, "Target DELETE")
		if deleted.err != nil || !deleted.deleted {
			t.Fatalf("TombstoneAndEnsureCleanup() = %+v", deleted)
		}
		if got := targetCleanupGateCount(t, db, "subdomain", "target_id = ?", targetID); got != 1 {
			t.Fatalf("materialization-first current assets = %d, want 1 before asynchronous cleanup", got)
		}
	})
}

func targetCleanupPostgresCancellationOutcomes(t *testing.T, db *gorm.DB) {
	t.Helper()
	t.Run("offline Agent delivery does not prevent completion", func(t *testing.T) {
		targetID := targetCleanupGateCreateTarget(t, db, "offline-agent.example")
		if _, err := catalogrepository.NewTargetRepository(db).TombstoneAndEnsureCleanup(context.Background(), targetID); err != nil {
			t.Fatalf("tombstone Target: %v", err)
		}
		targetCleanupGateSeedOwnedStateWithoutSchedule(t, db, targetID)
		store := targetcleanuprepository.NewTargetCleanupRepository(db)
		job := targetCleanupGateJob(t, store, targetID)
		dependencies := newTargetCleanupGateReconcilerDependencies(db)
		publisher := &targetCleanupGatePublisher{deliver: false}
		service, err := targetcleanupapp.NewTargetCleanupReconciliationService(store, dependencies.schedules, dependencies.scans, publisher)
		if err != nil {
			t.Fatalf("create reconciliation service: %v", err)
		}
		result, err := service.Reconcile(context.Background(), job, targetCleanupGateRunOptions())
		if err != nil || !result.Completed || result.Counts.AgentNotifications != 1 || result.Counts.AgentNotificationFails != 1 || publisher.calls != 1 {
			t.Fatalf("offline-Agent reconciliation = %+v, err=%v calls=%d", result, err, publisher.calls)
		}
		var state struct {
			Status    string     `gorm:"column:status"`
			StoppedAt *time.Time `gorm:"column:stopped_at"`
		}
		if err := db.Table("scan").Where("target_id = ?", targetID).Take(&state).Error; err != nil {
			t.Fatalf("read cancelled Scan: %v", err)
		}
		if state.Status != "cancelled" || state.StoppedAt == nil {
			t.Fatalf("offline-Agent Scan state = %+v", state)
		}
		targetCleanupGateAssertJobStatus(t, db, job.ID, "completed")
	})

	t.Run("late result after cancellation is rejected", func(t *testing.T) {
		targetID := targetCleanupGateCreateTarget(t, db, "late-result.example")
		scanID, taskID, scope := targetCleanupGateSeedResultExecution(t, db, targetID, "late-result")
		if _, err := catalogrepository.NewTargetRepository(db).TombstoneAndEnsureCleanup(context.Background(), targetID); err != nil {
			t.Fatalf("tombstone Target: %v", err)
		}
		cancellation, err := scanrepository.NewScanRepository(db).CancelNextActiveForDeletedTarget(context.Background(), targetID, time.Now().UTC())
		if err != nil || cancellation == nil || cancellation.ScanID != scanID {
			t.Fatalf("CancelNextActiveForDeletedTarget() = %+v, %v", cancellation, err)
		}
		persisted := false
		err = resultingestwiring.NewResultIngestMaterializationCoordinator(db).Materialize(context.Background(), scope, func(context.Context) error {
			persisted = true
			return nil
		})
		if !errors.Is(err, resultingestapp.ErrResultExecutionFenceRejected) || persisted {
			t.Fatalf("late Materialize() = %v persisted=%t, want rejection", err, persisted)
		}
		var states struct {
			ScanStatus string `gorm:"column:scan_status"`
			TaskStatus string `gorm:"column:task_status"`
		}
		if err := db.Raw(`SELECT scan.status AS scan_status, scan_task.status AS task_status
			FROM scan JOIN scan_task ON scan_task.scan_id = scan.id
			WHERE scan.id = ? AND scan_task.id = ?`, scanID, taskID).Scan(&states).Error; err != nil {
			t.Fatalf("read cancelled result execution: %v", err)
		}
		if states.ScanStatus != "cancelled" || states.TaskStatus != "cancelled" {
			t.Fatalf("late result changed cancellation facts: %+v", states)
		}
	})
}

func targetCleanupPostgresCommitBoundaryRecovery(t *testing.T, db *gorm.DB) {
	t.Helper()
	t.Run("tombstone and Job registration roll back together", func(t *testing.T) {
		targetID := targetCleanupGateCreateTarget(t, db, "job-registration-rollback.example")
		functionName := fmt.Sprintf("target_cleanup_gate_fail_job_%d", targetID)
		triggerName := fmt.Sprintf("target_cleanup_gate_fail_job_trigger_%d", targetID)
		dropFailureTrigger := func() {
			if err := db.Exec("DROP TRIGGER IF EXISTS " + triggerName + " ON target_cleanup_job").Error; err != nil {
				t.Errorf("drop injected Job trigger: %v", err)
			}
			if err := db.Exec("DROP FUNCTION IF EXISTS " + functionName + "()").Error; err != nil {
				t.Errorf("drop injected Job function: %v", err)
			}
		}
		if err := db.Exec("CREATE FUNCTION " + functionName + "() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'injected cleanup Job registration failure'; END; $$").Error; err != nil {
			t.Fatalf("create injected Job failure function: %v", err)
		}
		if err := db.Exec("CREATE TRIGGER " + triggerName + " BEFORE INSERT ON target_cleanup_job FOR EACH ROW EXECUTE FUNCTION " + functionName + "()").Error; err != nil {
			dropFailureTrigger()
			t.Fatalf("create injected Job failure trigger: %v", err)
		}

		if _, err := catalogrepository.NewTargetRepository(db).TombstoneAndEnsureCleanup(context.Background(), targetID); err == nil {
			dropFailureTrigger()
			t.Fatal("tombstone unexpectedly committed without durable Job")
		}
		var deletedRows, jobs int64
		if err := db.Table("target").Where("id = ? AND deleted_at IS NOT NULL", targetID).Count(&deletedRows).Error; err != nil {
			dropFailureTrigger()
			t.Fatalf("inspect rolled-back Target tombstone: %v", err)
		}
		if err := db.Table("target_cleanup_job").Where("target_id = ?", targetID).Count(&jobs).Error; err != nil {
			dropFailureTrigger()
			t.Fatalf("inspect rolled-back Target cleanup Job: %v", err)
		}
		if deletedRows != 0 || jobs != 0 {
			dropFailureTrigger()
			t.Fatalf("failed registration leaked state: tombstones=%d jobs=%d", deletedRows, jobs)
		}
		dropFailureTrigger()

		deleted, err := catalogrepository.NewTargetRepository(db).TombstoneAndEnsureCleanup(context.Background(), targetID)
		if err != nil || !deleted {
			t.Fatalf("retry TombstoneAndEnsureCleanup() = deleted=%t err=%v", deleted, err)
		}
		targetCleanupGateAssertTargetState(t, db, targetID, "job-registration-rollback.example", true, 1)
	})

	t.Run("committed reconciliation side effects are rediscovered from step one", func(t *testing.T) {
		cases := []struct {
			name     string
			decorate func(
				targetcleanupapp.TargetCleanupDataStore,
				targetcleanupapp.TargetScheduleCleaner,
				targetcleanupapp.TargetScanCanceller,
			) (targetcleanupapp.TargetCleanupDataStore, targetcleanupapp.TargetScheduleCleaner, targetcleanupapp.TargetScanCanceller)
		}{
			{
				name: "Schedule deletion",
				decorate: func(store targetcleanupapp.TargetCleanupDataStore, schedules targetcleanupapp.TargetScheduleCleaner, scans targetcleanupapp.TargetScanCanceller) (targetcleanupapp.TargetCleanupDataStore, targetcleanupapp.TargetScheduleCleaner, targetcleanupapp.TargetScanCanceller) {
					return store, &targetCleanupGateFailAfterScheduleCleaner{TargetScheduleCleaner: schedules}, scans
				},
			},
			{
				name: "control-plane deletion",
				decorate: func(store targetcleanupapp.TargetCleanupDataStore, schedules targetcleanupapp.TargetScheduleCleaner, scans targetcleanupapp.TargetScanCanceller) (targetcleanupapp.TargetCleanupDataStore, targetcleanupapp.TargetScheduleCleaner, targetcleanupapp.TargetScanCanceller) {
					return &targetCleanupGateFailAfterControlStore{TargetCleanupDataStore: store}, schedules, scans
				},
			},
			{
				name: "Scan cancellation",
				decorate: func(store targetcleanupapp.TargetCleanupDataStore, schedules targetcleanupapp.TargetScheduleCleaner, scans targetcleanupapp.TargetScanCanceller) (targetcleanupapp.TargetCleanupDataStore, targetcleanupapp.TargetScheduleCleaner, targetcleanupapp.TargetScanCanceller) {
					return store, schedules, &targetCleanupGateFailAfterScanCanceller{TargetScanCanceller: scans}
				},
			},
			{
				name: "asset batch",
				decorate: func(store targetcleanupapp.TargetCleanupDataStore, schedules targetcleanupapp.TargetScheduleCleaner, scans targetcleanupapp.TargetScanCanceller) (targetcleanupapp.TargetCleanupDataStore, targetcleanupapp.TargetScheduleCleaner, targetcleanupapp.TargetScanCanceller) {
					return &targetCleanupGateFailAfterAssetStore{TargetCleanupDataStore: store, resource: cleanupdomain.AssetResourceSubdomain}, schedules, scans
				},
			},
		}
		for _, test := range cases {
			test := test
			t.Run(test.name, func(t *testing.T) {
				targetCleanupGateRestartAfterCommittedFailure(t, db, "restart-"+strings.ReplaceAll(strings.ToLower(test.name), " ", "-"), test.decorate)
			})
		}
	})

	t.Run("final completion remains durable when the caller fails after commit", func(t *testing.T) {
		targetID, jobID := targetCleanupGateCreateTombstoneJob(t, db, "completion-commit.example")
		targetCleanupGateSeedOwnedState(t, db, targetID)
		store := targetcleanuprepository.NewTargetCleanupRepository(db)
		dependencies := newTargetCleanupGateReconcilerDependencies(db)
		first, err := targetcleanupapp.NewTargetCleanupReconciliationService(
			&targetCleanupGateFailAfterCompletionStore{TargetCleanupDataStore: store},
			dependencies.schedules,
			dependencies.scans,
			&targetCleanupGatePublisher{deliver: false},
		)
		if err != nil {
			t.Fatalf("create completion-failure reconciliation service: %v", err)
		}
		if _, err := first.Reconcile(context.Background(), cleanupdomain.CleanupJob{ID: jobID, TargetID: targetID}, targetCleanupGateRunOptions()); err == nil {
			t.Fatal("post-completion injected failure did not surface")
		}
		targetCleanupGateAssertJobStatus(t, db, jobID, "completed")
		due, err := store.ListDue(context.Background(), time.Now().UTC().Add(time.Hour), 100)
		if err != nil {
			t.Fatalf("list Jobs after restart boundary: %v", err)
		}
		for _, job := range due {
			if job.ID == jobID {
				t.Fatal("completed Job remained due after caller-side failure")
			}
		}
	})
}

func targetCleanupGateRestartAfterCommittedFailure(
	t *testing.T,
	db *gorm.DB,
	name string,
	decorate func(
		targetcleanupapp.TargetCleanupDataStore,
		targetcleanupapp.TargetScheduleCleaner,
		targetcleanupapp.TargetScanCanceller,
	) (targetcleanupapp.TargetCleanupDataStore, targetcleanupapp.TargetScheduleCleaner, targetcleanupapp.TargetScanCanceller),
) {
	t.Helper()
	targetID, jobID := targetCleanupGateCreateTombstoneJob(t, db, name+".example")
	targetCleanupGateSeedOwnedState(t, db, targetID)
	store := targetcleanuprepository.NewTargetCleanupRepository(db)
	dependencies := newTargetCleanupGateReconcilerDependencies(db)
	firstStore, firstSchedules, firstScans := decorate(store, dependencies.schedules, dependencies.scans)
	first, err := targetcleanupapp.NewTargetCleanupReconciliationService(firstStore, firstSchedules, firstScans, &targetCleanupGatePublisher{deliver: false})
	if err != nil {
		t.Fatalf("create first reconciliation service: %v", err)
	}
	if _, err := first.Reconcile(context.Background(), cleanupdomain.CleanupJob{ID: jobID, TargetID: targetID}, targetCleanupGateRunOptions()); err == nil {
		t.Fatal("injected post-commit failure did not surface")
	}

	restarted, err := targetcleanupapp.NewTargetCleanupReconciliationService(store, dependencies.schedules, dependencies.scans, &targetCleanupGatePublisher{deliver: false})
	if err != nil {
		t.Fatalf("create restarted reconciliation service: %v", err)
	}
	result, err := restarted.Reconcile(context.Background(), cleanupdomain.CleanupJob{ID: jobID, TargetID: targetID}, targetCleanupGateRunOptions())
	if err != nil || !result.Completed {
		t.Fatalf("restart reconciliation = %+v, %v", result, err)
	}
	targetCleanupGateAssertJobStatus(t, db, jobID, "completed")
	for _, resource := range cleanupdomain.OrderedAssetResources() {
		if got := targetCleanupGateCount(t, db, targetCleanupGateAssetTable(resource), "target_id = ?", targetID); got != 0 {
			t.Fatalf("restart left %s rows=%d", resource, got)
		}
	}
}

type targetCleanupGateDeleteResult struct {
	deleted bool
	err     error
}

type targetCleanupGateScheduleCreateResult struct {
	item *scheduledapp.ScheduledScan
	err  error
}

type targetCleanupGateBlocker struct {
	reached     chan struct{}
	released    chan struct{}
	blockOnce   sync.Once
	releaseOnce sync.Once
}

func newTargetCleanupGateBlocker() *targetCleanupGateBlocker {
	return &targetCleanupGateBlocker{reached: make(chan struct{}), released: make(chan struct{})}
}

func (blocker *targetCleanupGateBlocker) Block() {
	if blocker == nil {
		return
	}
	blocker.blockOnce.Do(func() {
		close(blocker.reached)
		<-blocker.released
	})
}

func (blocker *targetCleanupGateBlocker) Release() {
	if blocker == nil {
		return
	}
	blocker.releaseOnce.Do(func() { close(blocker.released) })
}

func targetCleanupGateAwaitSignal(t *testing.T, signal <-chan struct{}, description string) {
	t.Helper()
	select {
	case <-signal:
	case <-time.After(5 * time.Second):
		t.Fatalf("timed out waiting for %s", description)
	}
}

func targetCleanupGateAssertBlocked[T any](t *testing.T, result <-chan T, description string) {
	t.Helper()
	select {
	case <-result:
		t.Fatalf("%s crossed the held Target lock", description)
	case <-time.After(100 * time.Millisecond):
	}
}

func targetCleanupGateAwaitResult[T any](t *testing.T, result <-chan T, description string) T {
	t.Helper()
	select {
	case value := <-result:
		return value
	case <-time.After(5 * time.Second):
		var zero T
		t.Fatalf("timed out waiting for %s", description)
		return zero
	}
}

func targetCleanupGateRegisterUpdateBlocker(t *testing.T, db *gorm.DB, blocker *targetCleanupGateBlocker) {
	t.Helper()
	callbackName := targetCleanupGateCallbackName(t, "update")
	if err := db.Callback().Update().Before("gorm:update").Register(callbackName, func(*gorm.DB) {
		blocker.Block()
	}); err != nil {
		t.Fatalf("register Target update blocker: %v", err)
	}
	t.Cleanup(func() {
		blocker.Release()
		if err := db.Callback().Update().Remove(callbackName); err != nil {
			t.Errorf("remove Target update blocker: %v", err)
		}
	})
}

func targetCleanupGateRegisterQueryBlocker(t *testing.T, db *gorm.DB, blocker *targetCleanupGateBlocker) {
	t.Helper()
	callbackName := targetCleanupGateCallbackName(t, "query")
	if err := db.Callback().Query().After("gorm:query").Register(callbackName, func(*gorm.DB) {
		blocker.Block()
	}); err != nil {
		t.Fatalf("register Target query blocker: %v", err)
	}
	t.Cleanup(func() {
		blocker.Release()
		if err := db.Callback().Query().Remove(callbackName); err != nil {
			t.Errorf("remove Target query blocker: %v", err)
		}
	})
}

func targetCleanupGateCallbackName(t *testing.T, kind string) string {
	name := strings.Map(func(character rune) rune {
		if character >= 'a' && character <= 'z' || character >= 'A' && character <= 'Z' || character >= '0' && character <= '9' {
			return character
		}
		return '-'
	}, t.Name())
	return fmt.Sprintf("target-cleanup-gate-%s-%s-%d", kind, name, time.Now().UnixNano())
}

func targetCleanupGateCreateOrganization(t *testing.T, db *gorm.DB, name string) int {
	t.Helper()
	var organizationID int
	if err := db.Raw("INSERT INTO organization (name) VALUES (?) RETURNING id", name).Scan(&organizationID).Error; err != nil {
		t.Fatalf("create Organization %q: %v", name, err)
	}
	return organizationID
}

func targetCleanupGateScheduleCreateInput(targetID int, name string) *scheduledapp.ScheduledScanCreate {
	return &scheduledapp.ScheduledScanCreate{
		Name:           name,
		ScanWorkflowID: "cleanup-gate",
		Configuration:  map[string]any{},
		InputSource:    scandomain.InputSourceScanSnapshot,
		TargetID:       &targetID,
		CronExpression: "0 2 * * *",
		IsEnabled:      false,
	}
}

type targetCleanupGateBlockingScheduleCalculator struct {
	delegate scheduledapp.ScheduleCalculator
	blocker  *targetCleanupGateBlocker
}

func (calculator *targetCleanupGateBlockingScheduleCalculator) Validate(cronExpression string) error {
	calculator.blocker.Block()
	return calculator.delegate.Validate(cronExpression)
}

func (calculator *targetCleanupGateBlockingScheduleCalculator) FirstAfter(cronExpression string, instant time.Time) (time.Time, error) {
	return calculator.delegate.FirstAfter(cronExpression, instant)
}

func (calculator *targetCleanupGateBlockingScheduleCalculator) LatestAtOrBefore(cronExpression string, persistedCursor, instant time.Time) (time.Time, error) {
	return calculator.delegate.LatestAtOrBefore(cronExpression, persistedCursor, instant)
}

func (calculator *targetCleanupGateBlockingScheduleCalculator) AdvanceAfter(cronExpression string, instant time.Time) (time.Time, error) {
	return calculator.delegate.AdvanceAfter(cronExpression, instant)
}

func targetCleanupGateScanCreateInput(targetID int) *scanrepository.ScanCreateRecord {
	return &scanrepository.ScanCreateRecord{
		TargetID:       targetID,
		ScanWorkflowID: "cleanup-gate",
		InputSource:    scandomain.InputSourceScanSnapshot,
		TriggerType:    scandomain.ScanTriggerTypeManual,
		Status:         "pending",
	}
}

func targetCleanupGateSeedResultExecution(t *testing.T, db *gorm.DB, targetID int, label string) (int, int, resultingestapp.ResultMaterializationScope) {
	t.Helper()
	var scanID int
	if err := db.Raw("INSERT INTO scan (target_id, scan_workflow_id, input_source, trigger_type, status) VALUES (?, 'result-gate', 'scan_snapshot', 'manual', 'running') RETURNING id", targetID).Scan(&scanID).Error; err != nil {
		t.Fatalf("seed result Scan: %v", err)
	}
	sessionID := "result-session-" + label
	requestID := fmt.Sprintf("result-request-%d", targetID)
	const agentID = 73
	const epoch = 17
	if err := db.Exec(`
		INSERT INTO registration_token (id, token, expires_at)
		VALUES (?, 'r0000073', NOW() + INTERVAL '1 day')
		ON CONFLICT (id) DO NOTHING`, agentID).Error; err != nil {
		t.Fatalf("seed result Agent registration token: %v", err)
	}
	if err := db.Exec(`
		INSERT INTO agent (id, instance_id, display_name, authentication_token, status, registration_token_id)
		VALUES (?, 'target-cleanup-result-agent', 'Target cleanup result Agent', 'a0000073', 'online', ?)
		ON CONFLICT (id) DO NOTHING`, agentID, agentID).Error; err != nil {
		t.Fatalf("seed result Agent: %v", err)
	}
	if err := db.Exec(`
		INSERT INTO agent_runtime_status (agent_id, session_id, session_epoch, last_heartbeat)
		VALUES (?, ?, ?, NOW())
		ON CONFLICT (agent_id) DO UPDATE SET
			session_id = EXCLUDED.session_id,
			session_epoch = EXCLUDED.session_epoch,
			last_heartbeat = EXCLUDED.last_heartbeat`, agentID, sessionID, epoch).Error; err != nil {
		t.Fatalf("seed result Agent runtime session: %v", err)
	}
	var taskID int
	if err := db.Raw(`INSERT INTO scan_task
		(scan_id, stage_id, step_id, engine_id, resolved_execution_plan, status,
		 assigned_agent_id, assigned_session_id, assigned_session_epoch, assigned_request_id)
		VALUES (?, 'result', 'materialize', 'engine.result', decode('01', 'hex'), 'running', ?, ?, ?, ?) RETURNING id`,
		scanID,
		agentID,
		sessionID,
		epoch,
		requestID,
	).Scan(&taskID).Error; err != nil {
		t.Fatalf("seed result Task: %v", err)
	}
	return scanID, taskID, resultingestapp.ResultMaterializationScope{
		TaskID:       taskID,
		ScanID:       scanID,
		TargetID:     targetID,
		AgentID:      agentID,
		SessionID:    sessionID,
		SessionEpoch: epoch,
	}
}

func targetCleanupGateRunOptions() targetcleanupapp.TargetCleanupRunOptions {
	return targetcleanupapp.TargetCleanupRunOptions{AssetBatchSize: 1, MaxAssetBatches: 100, MaxRunDuration: time.Minute}
}

func targetCleanupGateAssertTargetState(t *testing.T, db *gorm.DB, targetID int, wantName string, wantDeleted bool, wantJobs int64) {
	t.Helper()
	var target struct {
		Name      string     `gorm:"column:name"`
		DeletedAt *time.Time `gorm:"column:deleted_at"`
	}
	if err := db.Table("target").Select("name, deleted_at").Where("id = ?", targetID).Take(&target).Error; err != nil {
		t.Fatalf("read Target %d: %v", targetID, err)
	}
	if target.Name != wantName || (target.DeletedAt != nil) != wantDeleted {
		t.Fatalf("Target %d = %+v, want name=%q deleted=%t", targetID, target, wantName, wantDeleted)
	}
	if jobs := targetCleanupGateCount(t, db, "target_cleanup_job", "target_id = ?", targetID); jobs != wantJobs {
		t.Fatalf("Target %d cleanup Jobs = %d, want %d", targetID, jobs, wantJobs)
	}
}

func targetCleanupGateAssertJobStatus(t *testing.T, db *gorm.DB, jobID int, want string) {
	t.Helper()
	var status string
	if err := db.Table("target_cleanup_job").Select("status").Where("id = ?", jobID).Scan(&status).Error; err != nil {
		t.Fatalf("read Target cleanup Job %d: %v", jobID, err)
	}
	if status != want {
		t.Fatalf("Target cleanup Job %d status = %q, want %q", jobID, status, want)
	}
}

type targetCleanupGateFailAfterScheduleCleaner struct {
	targetcleanupapp.TargetScheduleCleaner
	failed bool
}

func (cleaner *targetCleanupGateFailAfterScheduleCleaner) DeleteTargetScopedSchedules(ctx context.Context, targetID int) (int, error) {
	deleted, err := cleaner.TargetScheduleCleaner.DeleteTargetScopedSchedules(ctx, targetID)
	if err != nil || cleaner.failed {
		return deleted, err
	}
	cleaner.failed = true
	return deleted, errors.New("injected failure after committed Schedule cleanup")
}

type targetCleanupGateFailAfterControlStore struct {
	targetcleanupapp.TargetCleanupDataStore
	failed bool
}

func (store *targetCleanupGateFailAfterControlStore) DeleteTargetControlPlane(ctx context.Context, targetID int) (int64, int64, error) {
	relationships, policies, err := store.TargetCleanupDataStore.DeleteTargetControlPlane(ctx, targetID)
	if err != nil || store.failed {
		return relationships, policies, err
	}
	store.failed = true
	return relationships, policies, errors.New("injected failure after committed control-plane cleanup")
}

type targetCleanupGateFailAfterScanCanceller struct {
	targetcleanupapp.TargetScanCanceller
	failed bool
}

func (canceller *targetCleanupGateFailAfterScanCanceller) CancelNextActiveScan(ctx context.Context, targetID int, now time.Time) (*targetcleanupapp.TargetScanCancellation, error) {
	cancellation, err := canceller.TargetScanCanceller.CancelNextActiveScan(ctx, targetID, now)
	if err != nil || cancellation == nil || canceller.failed {
		return cancellation, err
	}
	canceller.failed = true
	return nil, errors.New("injected failure after committed Scan cancellation")
}

type targetCleanupGateFailAfterCompletionStore struct {
	targetcleanupapp.TargetCleanupDataStore
	failed bool
}

func (store *targetCleanupGateFailAfterCompletionStore) MarkCompletedIfClear(ctx context.Context, jobID, targetID int, completedAt time.Time) (bool, error) {
	completed, err := store.TargetCleanupDataStore.MarkCompletedIfClear(ctx, jobID, targetID, completedAt)
	if err != nil || !completed || store.failed {
		return completed, err
	}
	store.failed = true
	return false, errors.New("injected failure after committed Target cleanup completion")
}

type targetCleanupGateReconcilerDependencies struct {
	schedules targetcleanupapp.TargetScheduleCleaner
	scans     targetcleanupapp.TargetScanCanceller
}

func newTargetCleanupGateReconcilerDependencies(db *gorm.DB) targetCleanupGateReconcilerDependencies {
	return targetCleanupGateReconcilerDependencies{
		schedules: targetCleanupGateScheduleCleaner{repository: scheduledrepository.NewScheduledScanRepository(db)},
		scans:     targetCleanupGateScanCanceller{repository: scanrepository.NewScanRepository(db)},
	}
}

type targetCleanupGateScheduleCleaner struct {
	repository *scheduledrepository.ScheduledScanRepository
}

func (cleaner targetCleanupGateScheduleCleaner) DeleteTargetScopedSchedules(ctx context.Context, targetID int) (int, error) {
	return cleaner.repository.DeleteTargetScopedForCleanup(ctx, targetID)
}

type targetCleanupGateScanCanceller struct {
	repository *scanrepository.ScanRepository
}

func (canceller targetCleanupGateScanCanceller) CancelNextActiveScan(ctx context.Context, targetID int, now time.Time) (*targetcleanupapp.TargetScanCancellation, error) {
	result, err := canceller.repository.CancelNextActiveForDeletedTarget(ctx, targetID, now)
	if err != nil || result == nil {
		return nil, err
	}
	converted := &targetcleanupapp.TargetScanCancellation{ScanID: result.ScanID, CancelledTaskCount: result.CancelledTaskCount}
	for _, candidate := range result.NotificationCandidates {
		converted.NotificationCandidates = append(converted.NotificationCandidates, targetcleanupapp.TargetTaskCancelNotification{TaskID: candidate.TaskID, AgentID: candidate.AgentID})
	}
	return converted, nil
}

type targetCleanupGatePublisher struct {
	deliver bool
	calls   int
}

func (publisher *targetCleanupGatePublisher) TrySendTaskCancel(int, int, int) bool {
	publisher.calls++
	return publisher.deliver
}

type targetCleanupGateFailAfterAssetStore struct {
	targetcleanupapp.TargetCleanupDataStore
	resource cleanupdomain.AssetResource
	failOnce bool
}

func (store *targetCleanupGateFailAfterAssetStore) DeleteCurrentAssetBatch(ctx context.Context, targetID int, resource cleanupdomain.AssetResource, batchSize int) (int64, error) {
	deleted, err := store.TargetCleanupDataStore.DeleteCurrentAssetBatch(ctx, targetID, resource, batchSize)
	if err != nil || resource != store.resource || deleted == 0 || store.failOnce {
		return deleted, err
	}
	store.failOnce = true
	return deleted, errors.New("injected failure after committed asset batch")
}

type targetCleanupGateSQLTrace struct {
	mu         sync.Mutex
	statements []string
}

func (trace *targetCleanupGateSQLTrace) LogMode(gormlogger.LogLevel) gormlogger.Interface {
	return trace
}
func (trace *targetCleanupGateSQLTrace) Info(context.Context, string, ...interface{})  {}
func (trace *targetCleanupGateSQLTrace) Warn(context.Context, string, ...interface{})  {}
func (trace *targetCleanupGateSQLTrace) Error(context.Context, string, ...interface{}) {}
func (trace *targetCleanupGateSQLTrace) Trace(_ context.Context, _ time.Time, fc func() (string, int64), _ error) {
	statement, _ := fc()
	trace.mu.Lock()
	trace.statements = append(trace.statements, statement)
	trace.mu.Unlock()
}
func (trace *targetCleanupGateSQLTrace) Count(fragment string) int {
	trace.mu.Lock()
	defer trace.mu.Unlock()
	count := 0
	for _, statement := range trace.statements {
		if strings.Contains(strings.ToLower(statement), strings.ToLower(fragment)) {
			count++
		}
	}
	return count
}

func targetCleanupGateCreateTarget(t *testing.T, db *gorm.DB, name string) int {
	t.Helper()
	var targetID int
	if err := db.Raw("INSERT INTO target (name, type) VALUES (?, 'domain') RETURNING id", name).Scan(&targetID).Error; err != nil {
		t.Fatalf("create Target %q: %v", name, err)
	}
	return targetID
}

func targetCleanupGateCreateTombstoneJob(t *testing.T, db *gorm.DB, name string) (int, int) {
	t.Helper()
	targetID := targetCleanupGateCreateTarget(t, db, name)
	deletedAt := time.Now().UTC()
	if err := db.Exec("UPDATE target SET deleted_at = ? WHERE id = ?", deletedAt, targetID).Error; err != nil {
		t.Fatalf("tombstone Target %d: %v", targetID, err)
	}
	var jobID int
	if err := db.Raw("INSERT INTO target_cleanup_job (target_id) VALUES (?) RETURNING id", targetID).Scan(&jobID).Error; err != nil {
		t.Fatalf("create Target cleanup Job for %d: %v", targetID, err)
	}
	return targetID, jobID
}

func targetCleanupGateJob(t *testing.T, repository *targetcleanuprepository.TargetCleanupRepository, targetID int) cleanupdomain.CleanupJob {
	t.Helper()
	jobs, err := repository.ListDue(context.Background(), time.Now().UTC().Add(time.Minute), 100)
	if err != nil {
		t.Fatalf("list Target cleanup Jobs: %v", err)
	}
	for _, job := range jobs {
		if job.TargetID == targetID {
			return job
		}
	}
	t.Fatalf("Target cleanup Job for Target %d was not due", targetID)
	return cleanupdomain.CleanupJob{}
}

func targetCleanupGateSeedOwnedState(t *testing.T, db *gorm.DB, targetID int) {
	t.Helper()
	targetCleanupGateSeedSchedule(t, db, targetID)
	targetCleanupGateSeedOwnedStateWithoutSchedule(t, db, targetID)
}

func targetCleanupGateSeedOwnedStateWithoutSchedule(t *testing.T, db *gorm.DB, targetID int) {
	t.Helper()
	var organizationID int
	if err := db.Raw("INSERT INTO organization (name) VALUES (?) RETURNING id", fmt.Sprintf("cleanup-org-%d", targetID)).Scan(&organizationID).Error; err != nil {
		t.Fatalf("seed Organization for Target %d: %v", targetID, err)
	}
	if err := db.Exec("INSERT INTO organization_target (organization_id, target_id) VALUES (?, ?)", organizationID, targetID).Error; err != nil {
		t.Fatalf("seed Organization relationship for Target %d: %v", targetID, err)
	}
	if err := db.Exec("INSERT INTO blacklist_policy (scope, target_id, patterns) VALUES ('target', ?, '[]'::jsonb)", targetID).Error; err != nil {
		t.Fatalf("seed Target policy for Target %d: %v", targetID, err)
	}
	var scanID int
	if err := db.Raw("INSERT INTO scan (target_id, scan_workflow_id, input_source, trigger_type, status) VALUES (?, 'cleanup-gate', 'scan_snapshot', 'manual', 'running') RETURNING id", targetID).Scan(&scanID).Error; err != nil {
		t.Fatalf("seed active Scan for Target %d: %v", targetID, err)
	}
	var taskID int
	if err := db.Raw(`INSERT INTO scan_task
		(scan_id, stage_id, step_id, engine_id, resolved_execution_plan, status,
		 assigned_agent_id, assigned_session_id, assigned_session_epoch, assigned_request_id)
		VALUES (?, 'discovery', 'subdomain', 'engine.cleanup', decode('01', 'hex'), 'running', 1, ?, 1, ?) RETURNING id`,
		scanID,
		fmt.Sprintf("session-%d", targetID),
		fmt.Sprintf("request-%d", targetID),
	).Scan(&taskID).Error; err != nil {
		t.Fatalf("seed running Task for Target %d: %v", targetID, err)
	}
	if err := db.Exec("INSERT INTO task_progress_log (scan_id, task_id, request_id, sequence, content) VALUES (?, ?, 'cleanup-gate', 1, 'retained history')", scanID, taskID).Error; err != nil {
		t.Fatalf("seed progress history for Target %d: %v", targetID, err)
	}
	if err := db.Exec("INSERT INTO subdomain_snapshot (scan_id, dns_name) VALUES (?, ?)", scanID, fmt.Sprintf("history-%d.example", targetID)).Error; err != nil {
		t.Fatalf("seed Snapshot history for Target %d: %v", targetID, err)
	}
	targetCleanupGateSeedAssets(t, db, targetID, fmt.Sprintf("target-%d", targetID))
}

func targetCleanupGateSeedSchedule(t *testing.T, db *gorm.DB, targetID int) int {
	t.Helper()
	var scheduleID int
	if err := db.Raw(`INSERT INTO scheduled_scan
		(name, scan_workflow_id, input_source, target_id, cron_expression, is_enabled, next_run_time)
		VALUES (?, 'cleanup-gate', 'scan_snapshot', ?, '0 2 * * *', TRUE, CURRENT_TIMESTAMP) RETURNING id`, fmt.Sprintf("target-schedule-%d", targetID), targetID).Scan(&scheduleID).Error; err != nil {
		t.Fatalf("seed Target Schedule %d: %v", targetID, err)
	}
	if err := db.Exec("INSERT INTO scheduled_scan_occurrence (scheduled_scan_id, scheduled_for) VALUES (?, CURRENT_TIMESTAMP)", scheduleID).Error; err != nil {
		t.Fatalf("seed Schedule occurrence %d: %v", scheduleID, err)
	}
	return scheduleID
}

func targetCleanupGateSeedAssets(t *testing.T, db *gorm.DB, targetID int, prefix string) {
	t.Helper()
	statements := []struct {
		query string
		args  []any
	}{
		{query: "INSERT INTO subdomain (target_id, dns_name) VALUES (?, ?)", args: []any{targetID, "subdomain-" + prefix + ".example"}},
		{query: "INSERT INTO host_port_mapping (target_id, host, ip, port) VALUES (?, ?, '192.0.2.10', 443)", args: []any{targetID, "host-" + prefix + ".example"}},
		{query: "INSERT INTO website (target_id, url) VALUES (?, ?)", args: []any{targetID, "https://website-" + prefix + ".example/"}},
		{query: "INSERT INTO endpoint (target_id, url) VALUES (?, ?)", args: []any{targetID, "https://endpoint-" + prefix + ".example/api"}},
		{query: "INSERT INTO directory (target_id, url) VALUES (?, ?)", args: []any{targetID, "https://directory-" + prefix + ".example/"}},
		{query: "INSERT INTO screenshot (target_id, url) VALUES (?, ?)", args: []any{targetID, "https://screenshot-" + prefix + ".example/"}},
		{query: "INSERT INTO vulnerability (target_id, url, vuln_type, severity, source) VALUES (?, ?, 'test', 'low', 'cleanup-gate')", args: []any{targetID, "https://vulnerability-" + prefix + ".example/"}},
	}
	for _, statement := range statements {
		if err := db.Exec(statement.query, statement.args...).Error; err != nil {
			t.Fatalf("seed Target %d asset with %q: %v", targetID, statement.query, err)
		}
	}
}

func targetCleanupGateSeedOneAsset(t *testing.T, db *gorm.DB, targetID int, resource cleanupdomain.AssetResource) {
	t.Helper()
	targetCleanupGateSeedAssets(t, db, targetID, fmt.Sprintf("condition-%d", targetID))
	for _, candidate := range cleanupdomain.OrderedAssetResources() {
		if candidate == resource {
			continue
		}
		if err := db.Exec("DELETE FROM "+targetCleanupGateAssetTable(candidate)+" WHERE target_id = ?", targetID).Error; err != nil {
			t.Fatalf("remove non-condition %s asset: %v", candidate, err)
		}
	}
	if resource == cleanupdomain.AssetResourceVulnerability {
		if err := db.Exec("UPDATE vulnerability SET reviewed = TRUE WHERE target_id = ?", targetID).Error; err != nil {
			t.Fatalf("mark reviewed vulnerability: %v", err)
		}
	}
}

func targetCleanupGateAssetTable(resource cleanupdomain.AssetResource) string {
	switch resource {
	case cleanupdomain.AssetResourceSubdomain:
		return "subdomain"
	case cleanupdomain.AssetResourceHostPortMapping:
		return "host_port_mapping"
	case cleanupdomain.AssetResourceWebsite:
		return "website"
	case cleanupdomain.AssetResourceEndpoint:
		return "endpoint"
	case cleanupdomain.AssetResourceDirectory:
		return "directory"
	case cleanupdomain.AssetResourceScreenshot:
		return "screenshot"
	case cleanupdomain.AssetResourceVulnerability:
		return "vulnerability"
	default:
		panic("unknown Target cleanup asset resource " + string(resource))
	}
}

func targetCleanupGateCount(t *testing.T, db *gorm.DB, table, condition string, arguments ...any) int64 {
	t.Helper()
	var count int64
	if err := db.Table(table).Where(condition, arguments...).Count(&count).Error; err != nil {
		t.Fatalf("count %s: %v", table, err)
	}
	return count
}

func assertTargetCleanupGateTimeoutDefaults(t *testing.T, db *gorm.DB) {
	t.Helper()
	var settings struct {
		LockTimeout      string `gorm:"column:lock_timeout"`
		StatementTimeout string `gorm:"column:statement_timeout"`
	}
	if err := db.Raw("SELECT current_setting('lock_timeout') AS lock_timeout, current_setting('statement_timeout') AS statement_timeout").Scan(&settings).Error; err != nil {
		t.Fatalf("read PostgreSQL timeout settings: %v", err)
	}
	if settings.LockTimeout != "0" || settings.StatementTimeout != "0" {
		t.Fatalf("transaction-local cleanup timeout leaked into session: %+v", settings)
	}
}
