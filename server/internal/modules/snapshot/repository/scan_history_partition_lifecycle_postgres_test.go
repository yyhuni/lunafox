package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

const scanHistoryLifecycleTestSchema = "scan_history_partition_lifecycle"

func TestScanHistoryPartitionLifecyclePostgres(t *testing.T) {
	dsn := strings.TrimSpace(os.Getenv("LUNAFOX_POSTGRES_PARTITION_DSN"))
	if dsn == "" {
		t.Skip("set LUNAFOX_POSTGRES_PARTITION_DSN to run PostgreSQL scan-history partition lifecycle verification")
	}
	if !strings.Contains(dsn, "dbname=lunafox_partition_check") || !strings.Contains(dsn, "search_path="+scanHistoryLifecycleTestSchema) {
		t.Fatalf("partition lifecycle DSN must use lunafox_partition_check and search_path=%s", scanHistoryLifecycleTestSchema)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatalf("open PostgreSQL partition lifecycle database: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("open PostgreSQL partition lifecycle SQL handle: %v", err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })

	resetScanHistoryLifecycleSchema(t, ctx, db)
	t.Cleanup(func() {
		_ = db.WithContext(context.Background()).Exec("DROP SCHEMA IF EXISTS " + scanHistoryLifecycleTestSchema + " CASCADE").Error
	})

	lifecycle := NewScanHistoryPartitionLifecycle(db)
	lockConn, err := sqlDB.Conn(ctx)
	if err != nil {
		t.Fatalf("open lock-holder connection: %v", err)
	}
	if _, err := lockConn.ExecContext(ctx, `SELECT pg_advisory_lock($1)`, scanHistoryLifecycleLock); err != nil {
		t.Fatalf("hold lifecycle advisory lock: %v", err)
	}
	err = lifecycle.EnsureCoverage(ctx, 40_001)
	if _, unlockErr := lockConn.ExecContext(ctx, `SELECT pg_advisory_unlock($1)`, scanHistoryLifecycleLock); unlockErr != nil {
		t.Fatalf("release lifecycle advisory lock: %v", unlockErr)
	}
	if closeErr := lockConn.Close(); closeErr != nil {
		t.Fatalf("close lock-holder connection: %v", closeErr)
	}
	if !errors.Is(err, ErrScanHistoryLifecycleLockHeld) {
		t.Fatalf("coverage must report advisory-lock contention, got %v", err)
	}
	if err := lifecycle.EnsureCoverage(ctx, 1); err != nil {
		t.Fatalf("ensure initial and future coverage: %v", err)
	}
	if err := lifecycle.EnsureCoverage(ctx, 1); err != nil {
		t.Fatalf("reconcile existing coverage: %v", err)
	}
	for _, parent := range ScanHistoryParents() {
		assertScanHistoryChildCount(t, ctx, db, parent, 2)
	}
	initialStates, err := lifecycle.ChildStates(ctx, ScanHistoryRange{Start: 0, End: ScanHistoryPartitionSpan})
	if err != nil || len(initialStates) != len(ScanHistoryParents()) {
		t.Fatalf("read initial child states: states=%v err=%v", initialStates, err)
	}
	for _, state := range initialStates {
		if !strings.HasSuffix(state, ":attached") {
			t.Fatalf("initial child state=%q, want attached", state)
		}
	}

	if err := db.WithContext(ctx).Exec(`INSERT INTO scan (id, status, stopped_at, created_at) VALUES (1, 'succeeded', CURRENT_TIMESTAMP - INTERVAL '31 days', CURRENT_TIMESTAMP - INTERVAL '31 days')`).Error; err != nil {
		t.Fatalf("insert eligible scan: %v", err)
	}
	if err := db.WithContext(ctx).Exec(`INSERT INTO scan_operation (id, scan_id, target_id, request_fingerprint) VALUES ('00000000-0000-0000-0000-000000000001', 1, 1, '0000000000000000000000000000000000000000000000000000000000000000')`).Error; err != nil {
		t.Fatalf("insert scan-backed operation: %v", err)
	}
	if err := db.WithContext(ctx).Exec(`INSERT INTO scan (id, status, created_at) VALUES (10001, 'running', CURRENT_TIMESTAMP - INTERVAL '31 days')`).Error; err != nil {
		t.Fatalf("insert blocked scan: %v", err)
	}
	if err := db.WithContext(ctx).Exec(`INSERT INTO scan_task (id, scan_id) VALUES (1, 1)`).Error; err != nil {
		t.Fatalf("insert scan task: %v", err)
	}
	for _, parent := range ScanHistoryParents() {
		if parent == "task_progress_log" {
			if err := db.WithContext(ctx).Exec(`INSERT INTO task_progress_log (scan_id, task_id, request_id, sequence) VALUES (1, 1, 'request-1', 1)`).Error; err != nil {
				t.Fatalf("route task progress log: %v", err)
			}
		} else if parent == "screenshot_snapshot" {
			if err := db.WithContext(ctx).Exec(`INSERT INTO screenshot_snapshot (scan_id, url) VALUES (1, 'https://example.test/')`).Error; err != nil {
				t.Fatalf("route screenshot history: %v", err)
			}
		} else if err := db.WithContext(ctx).Exec(fmt.Sprintf("INSERT INTO %s (scan_id, payload) VALUES (1, 'history')", parent)).Error; err != nil {
			t.Fatalf("route %s history: %v", parent, err)
		}
		assertScanHistoryRowRoutedToRange(t, ctx, db, parent, 1, 0)
	}
	if err := db.WithContext(ctx).Exec(`INSERT INTO screenshot_snapshot (scan_id, url) VALUES (1, 'https://example.test/') ON CONFLICT (scan_id, url) DO UPDATE SET url = EXCLUDED.url`).Error; err != nil {
		t.Fatalf("replay screenshot conflict: %v", err)
	}
	var replayedRows int64
	if err := db.WithContext(ctx).Table("screenshot_snapshot").Where("scan_id = ?", 1).Count(&replayedRows).Error; err != nil || replayedRows != 1 {
		t.Fatalf("screenshot replay must retain one row: count=%d err=%v", replayedRows, err)
	}
	if err := db.WithContext(ctx).Exec(`INSERT INTO task_progress_log (scan_id, task_id, request_id, sequence) VALUES (1, 1, 'request-1', 1) ON CONFLICT (scan_id, task_id, request_id, sequence) DO NOTHING`).Error; err != nil {
		t.Fatalf("replay task-progress-log idempotency key: %v", err)
	}
	if err := db.WithContext(ctx).Exec(`INSERT INTO task_progress_log (scan_id, task_id, request_id, sequence) VALUES (2, 1, 'wrong-scan', 1)`).Error; err == nil {
		t.Fatal("task-progress-log composite task reference must reject a mismatched scan")
	}
	if err := db.WithContext(ctx).Exec(`INSERT INTO website_snapshot (scan_id, payload) VALUES (20000, 'missing-range')`).Error; err == nil {
		t.Fatal("missing range must reject history writes; no default partition may accept them")
	}

	cutoff := time.Now().UTC().Add(-30 * 24 * time.Hour)
	if blocked, err := lifecycle.OldestBlockedRange(ctx, cutoff); err != nil || blocked == nil || blocked.Range.Start != 10_000 {
		t.Fatalf("oldest blocked range=%+v err=%v, want [10000,20000)", blocked, err)
	}
	lockConn, err = sqlDB.Conn(ctx)
	if err != nil {
		t.Fatalf("open retention lock-holder connection: %v", err)
	}
	if _, err := lockConn.ExecContext(ctx, `SELECT pg_advisory_lock($1)`, scanHistoryLifecycleLock); err != nil {
		t.Fatalf("hold retention lifecycle advisory lock: %v", err)
	}
	_, lockOutcomes, retentionErr := lifecycle.RunRetention(ctx, ScanHistoryRetentionEnforce, cutoff, retentionRunOptions(1, 10, 1))
	if _, unlockErr := lockConn.ExecContext(ctx, `SELECT pg_advisory_unlock($1)`, scanHistoryLifecycleLock); unlockErr != nil {
		t.Fatalf("release retention lifecycle advisory lock: %v", unlockErr)
	}
	if closeErr := lockConn.Close(); closeErr != nil {
		t.Fatalf("close retention lock-holder connection: %v", closeErr)
	}
	if retentionErr != nil || len(lockOutcomes) != 1 || lockOutcomes[0].Outcome != scanHistoryRetentionOutcomeLockContended {
		t.Fatalf("retention lock contention outcomes=%+v err=%v, want one lock-contended range", lockOutcomes, retentionErr)
	}
	var scanHidden bool
	if err := db.WithContext(ctx).Raw(`SELECT deleted_at IS NOT NULL FROM scan WHERE id = 1`).Scan(&scanHidden).Error; err != nil || scanHidden {
		t.Fatalf("lock-contended retention must not soft-delete scan: hidden=%t err=%v", scanHidden, err)
	}
	if ranges, outcomes, err := lifecycle.RunRetention(ctx, ScanHistoryRetentionDisabled, cutoff, retentionRunOptions(10, 10, 1)); err != nil || len(ranges) != 0 || len(outcomes) != 0 {
		t.Fatalf("disabled retention must be read-only and skip candidates: ranges=%v outcomes=%v err=%v", ranges, outcomes, err)
	}
	if ranges, outcomes, err := lifecycle.RunRetention(ctx, ScanHistoryRetentionReport, cutoff, retentionRunOptions(10, 10, 1)); err != nil || len(ranges) != 1 || ranges[0].Start != 0 || len(outcomes) != 0 {
		t.Fatalf("report retention must expose one candidate without mutation: ranges=%v outcomes=%v err=%v", ranges, outcomes, err)
	}
	var retainedOperationCount int64
	if err := db.WithContext(ctx).Table("scan_operation").Where("scan_id = ?", 1).Count(&retainedOperationCount).Error; err != nil || retainedOperationCount != 1 {
		t.Fatalf("retained scan must retain its operation: count=%d err=%v", retainedOperationCount, err)
	}
	assertScanHistoryChildCount(t, ctx, db, "website_snapshot", 2)

	if _, outcomes, err := lifecycle.RunRetention(ctx, ScanHistoryRetentionEnforce, cutoff, retentionRunOptions(1, 10, 1)); err != nil {
		t.Fatalf("enforce retention: %v", err)
	} else if len(outcomes) != 1 || outcomes[0].Outcome != "completed" {
		t.Fatalf("enforce retention outcome=%+v, want one completed range", outcomes)
	}
	for _, parent := range ScanHistoryParents() {
		assertScanHistoryPartitionAbsent(t, ctx, db, scanHistoryPartitionName(parent, 0))
	}
	completedStates, err := lifecycle.ChildStates(ctx, ScanHistoryRange{Start: 0, End: ScanHistoryPartitionSpan})
	if err != nil || len(completedStates) != len(ScanHistoryParents()) {
		t.Fatalf("read completed child states: states=%v err=%v", completedStates, err)
	}
	for _, state := range completedStates {
		if !strings.HasSuffix(state, ":dropped") {
			t.Fatalf("completed child state=%q, want dropped", state)
		}
	}
	var count int64
	if err := db.WithContext(ctx).Table("scan").Where("id = ?", 1).Count(&count).Error; err != nil || count != 0 {
		t.Fatalf("eligible scan must be physically removed after history cleanup: count=%d err=%v", count, err)
	}
	if err := db.WithContext(ctx).Table("scan_task").Where("scan_id = ?", 1).Count(&count).Error; err != nil || count != 0 {
		t.Fatalf("scan tasks must be bounded-deleted before final scan delete: count=%d err=%v", count, err)
	}
	if err := db.WithContext(ctx).Table("scan_operation").Where("scan_id = ?", 1).Count(&count).Error; err != nil || count != 0 {
		t.Fatalf("scan operation must cascade with its retained Scan owner: count=%d err=%v", count, err)
	}
	if err := db.WithContext(ctx).Table("current_asset").Where("id = ?", 1).Count(&count).Error; err != nil || count != 1 {
		t.Fatalf("current asset projection must remain untouched: count=%d err=%v", count, err)
	}

	assertPartialCleanupRetry(t, ctx, db, lifecycle, cutoff)
	assertCommittedTaskBatchRetry(t, ctx, db, lifecycle, cutoff)
	assertTaskBudgetResume(t, ctx, db, lifecycle, cutoff)
	assertRangeBudget(t, ctx, db, lifecycle, cutoff)
	assertEligibilityRevalidationLocksScan(t, ctx, db, sqlDB, lifecycle, cutoff)
	assertProvisioningFollowsScanSequence(t, ctx, db, lifecycle)
}

func assertPartialCleanupRetry(t *testing.T, ctx context.Context, db *gorm.DB, lifecycle *ScanHistoryPartitionLifecycle, cutoff time.Time) {
	t.Helper()
	const scanID = 20_001
	const taskID = 2
	const rangeStart = 20_000
	if err := lifecycle.EnsureCoverage(ctx, scanID); err != nil {
		t.Fatalf("ensure partial-cleanup coverage: %v", err)
	}
	if err := db.WithContext(ctx).Exec(`INSERT INTO scan (id, status, stopped_at, created_at, deleted_at) VALUES (?, 'succeeded', CURRENT_TIMESTAMP - INTERVAL '31 days', CURRENT_TIMESTAMP - INTERVAL '31 days', CURRENT_TIMESTAMP)`, scanID).Error; err != nil {
		t.Fatalf("insert partial-cleanup scan: %v", err)
	}
	if err := db.WithContext(ctx).Exec(`INSERT INTO scan_task (id, scan_id) VALUES (?, ?)`, taskID, scanID).Error; err != nil {
		t.Fatalf("insert partial-cleanup task: %v", err)
	}
	for _, parent := range ScanHistoryParents() {
		if parent == "task_progress_log" {
			if err := db.WithContext(ctx).Exec(`INSERT INTO task_progress_log (scan_id, task_id, request_id, sequence) VALUES (?, ?, 'partial-retry', 1)`, scanID, taskID).Error; err != nil {
				t.Fatalf("insert partial-cleanup progress log: %v", err)
			}
		} else if parent == "screenshot_snapshot" {
			if err := db.WithContext(ctx).Exec(`INSERT INTO screenshot_snapshot (scan_id, url) VALUES (?, 'https://partial.example.test/')`, scanID).Error; err != nil {
				t.Fatalf("insert partial-cleanup screenshot: %v", err)
			}
		} else if err := db.WithContext(ctx).Exec(fmt.Sprintf("INSERT INTO %s (scan_id, payload) VALUES (?, 'partial-retry')", parent), scanID).Error; err != nil {
			t.Fatalf("insert partial-cleanup %s evidence: %v", parent, err)
		}
	}
	for _, parent := range []string{"website_snapshot", "task_progress_log"} {
		if err := detachAndDropPartition(db.WithContext(ctx), parent, rangeStart); err != nil {
			t.Fatalf("inject partial cleanup for %s: %v", parent, err)
		}
	}
	if _, outcomes, err := lifecycle.RunRetention(ctx, ScanHistoryRetentionEnforce, cutoff, retentionRunOptions(1, 10, 1)); err != nil {
		t.Fatalf("retry partial cleanup: %v", err)
	} else if len(outcomes) != 1 || outcomes[0].Range.Start != rangeStart || outcomes[0].Outcome != "completed" {
		t.Fatalf("partial cleanup retry outcomes=%+v, want one completed [%d,%d)", outcomes, rangeStart, rangeStart+ScanHistoryPartitionSpan)
	}
	for _, parent := range ScanHistoryParents() {
		assertScanHistoryPartitionAbsent(t, ctx, db, scanHistoryPartitionName(parent, rangeStart))
	}
	var count int64
	if err := db.WithContext(ctx).Table("scan").Where("id = ?", scanID).Count(&count).Error; err != nil || count != 0 {
		t.Fatalf("partial cleanup retry must remove scan: count=%d err=%v", count, err)
	}
}

func assertCommittedTaskBatchRetry(t *testing.T, ctx context.Context, db *gorm.DB, lifecycle *ScanHistoryPartitionLifecycle, cutoff time.Time) {
	t.Helper()
	const scanID = 30_001
	const rangeStart = 30_000
	prepareEligibleRetentionRange(t, ctx, db, lifecycle, scanID, 30_001, 30_002, 30_003)
	if err := db.WithContext(ctx).Exec(`
		CREATE FUNCTION fail_second_retention_task_delete() RETURNS trigger LANGUAGE plpgsql AS $$
		BEGIN
			IF OLD.id = 30002 THEN RAISE EXCEPTION 'injected second task batch failure'; END IF;
			RETURN OLD;
		END
		$$`).Error; err != nil {
		t.Fatalf("create second task batch failure function: %v", err)
	}
	if err := db.WithContext(ctx).Exec(`
		CREATE TRIGGER scan_task_fail_second_retention_batch
		BEFORE DELETE ON scan_task
		FOR EACH ROW EXECUTE FUNCTION fail_second_retention_task_delete()`).Error; err != nil {
		t.Fatalf("create second task batch failure trigger: %v", err)
	}

	if _, outcomes, err := lifecycle.RunRetention(ctx, ScanHistoryRetentionEnforce, cutoff, retentionRunOptions(1, 3, 1)); err != nil {
		t.Fatalf("run injected task batch failure: %v", err)
	} else if len(outcomes) != 1 || outcomes[0].Range.Start != rangeStart || outcomes[0].Outcome != scanHistoryRetentionOutcomeFailed || outcomes[0].TaskRowsDeleted != 1 || outcomes[0].TaskDeleteBatches != 1 || outcomes[0].Err == nil {
		t.Fatalf("injected task batch failure outcomes=%+v, want one failed range with one committed batch", outcomes)
	}
	assertScanHistoryRangeSoftDeleted(t, ctx, db, scanID)
	assertScanTaskCount(t, ctx, db, scanID, 2)
	for _, parent := range ScanHistoryParents() {
		assertScanHistoryPartitionAbsent(t, ctx, db, scanHistoryPartitionName(parent, rangeStart))
	}

	if err := db.WithContext(ctx).Exec(`DROP TRIGGER scan_task_fail_second_retention_batch ON scan_task`).Error; err != nil {
		t.Fatalf("drop second task batch failure trigger: %v", err)
	}
	if err := db.WithContext(ctx).Exec(`DROP FUNCTION fail_second_retention_task_delete()`).Error; err != nil {
		t.Fatalf("drop second task batch failure function: %v", err)
	}
	if _, outcomes, err := lifecycle.RunRetention(ctx, ScanHistoryRetentionEnforce, cutoff, retentionRunOptions(1, 3, 1)); err != nil {
		t.Fatalf("retry committed task batches: %v", err)
	} else if len(outcomes) != 1 || outcomes[0].Outcome != scanHistoryRetentionOutcomeCompleted || outcomes[0].TaskRowsDeleted != 2 || outcomes[0].TaskDeleteBatches != 2 {
		t.Fatalf("committed task batch retry outcomes=%+v, want completed range with two remaining task batches", outcomes)
	}
	assertScanHistoryRangeHardDeleted(t, ctx, db, scanID)
}

func assertTaskBudgetResume(t *testing.T, ctx context.Context, db *gorm.DB, lifecycle *ScanHistoryPartitionLifecycle, cutoff time.Time) {
	t.Helper()
	const scanID = 40_001
	const rangeStart = 40_000
	prepareEligibleRetentionRange(t, ctx, db, lifecycle, scanID, 40_001, 40_002, 40_003)
	options := retentionRunOptions(1, 2, 1)
	if _, outcomes, err := lifecycle.RunRetention(ctx, ScanHistoryRetentionEnforce, cutoff, options); err != nil {
		t.Fatalf("run task cleanup budget: %v", err)
	} else if len(outcomes) != 1 || outcomes[0].Range.Start != rangeStart || outcomes[0].Outcome != scanHistoryRetentionOutcomeTaskBudgetExhausted || outcomes[0].TaskRowsDeleted != 2 || outcomes[0].TaskDeleteBatches != 2 || outcomes[0].Err != nil {
		t.Fatalf("task cleanup budget outcomes=%+v, want one deferred range after two batches", outcomes)
	}
	assertScanHistoryRangeSoftDeleted(t, ctx, db, scanID)
	assertScanTaskCount(t, ctx, db, scanID, 1)
	for _, parent := range ScanHistoryParents() {
		assertScanHistoryPartitionAbsent(t, ctx, db, scanHistoryPartitionName(parent, rangeStart))
	}

	if _, outcomes, err := lifecycle.RunRetention(ctx, ScanHistoryRetentionEnforce, cutoff, options); err != nil {
		t.Fatalf("resume task cleanup budget: %v", err)
	} else if len(outcomes) != 1 || outcomes[0].Outcome != scanHistoryRetentionOutcomeCompleted || outcomes[0].TaskRowsDeleted != 1 || outcomes[0].TaskDeleteBatches != 1 {
		t.Fatalf("task cleanup budget resume outcomes=%+v, want completed range with the remaining batch", outcomes)
	}
	assertScanHistoryRangeHardDeleted(t, ctx, db, scanID)
}

func assertRangeBudget(t *testing.T, ctx context.Context, db *gorm.DB, lifecycle *ScanHistoryPartitionLifecycle, cutoff time.Time) {
	t.Helper()
	const firstScanID = 50_001
	const secondScanID = 60_001
	prepareEligibleRetentionRange(t, ctx, db, lifecycle, firstScanID)
	prepareEligibleRetentionRange(t, ctx, db, lifecycle, secondScanID)
	options := retentionRunOptions(1, 1, 1)
	if ranges, outcomes, err := lifecycle.RunRetention(ctx, ScanHistoryRetentionEnforce, cutoff, options); err != nil {
		t.Fatalf("run range cleanup budget: %v", err)
	} else if len(ranges) != 1 || len(outcomes) != 1 || outcomes[0].Range.Start != 50_000 || outcomes[0].Outcome != scanHistoryRetentionOutcomeCompleted {
		t.Fatalf("range cleanup budget ranges=%v outcomes=%+v, want only the oldest completed range", ranges, outcomes)
	}
	assertScanHistoryRangeHardDeleted(t, ctx, db, firstScanID)
	assertScanHistoryRangeVisible(t, ctx, db, secondScanID)

	if _, outcomes, err := lifecycle.RunRetention(ctx, ScanHistoryRetentionEnforce, cutoff, options); err != nil {
		t.Fatalf("resume range cleanup budget: %v", err)
	} else if len(outcomes) != 1 || outcomes[0].Range.Start != 60_000 || outcomes[0].Outcome != scanHistoryRetentionOutcomeCompleted {
		t.Fatalf("range cleanup budget resume outcomes=%+v, want second range completed", outcomes)
	}
	assertScanHistoryRangeHardDeleted(t, ctx, db, secondScanID)
}

func assertEligibilityRevalidationLocksScan(t *testing.T, ctx context.Context, db *gorm.DB, sqlDB *sql.DB, lifecycle *ScanHistoryPartitionLifecycle, cutoff time.Time) {
	t.Helper()
	const scanID = 70_001
	partitionRange := ScanHistoryRange{Start: 70_000, End: 80_000}
	prepareEligibleRetentionRange(t, ctx, db, lifecycle, scanID)

	tx := db.WithContext(ctx).Begin()
	if tx.Error != nil {
		t.Fatalf("begin eligibility revalidation transaction: %v", tx.Error)
	}
	defer func() { _ = tx.Rollback().Error }()
	eligible, err := eligibleRange(tx, partitionRange, cutoff)
	if err != nil || !eligible {
		t.Fatalf("revalidate eligible range with lock: eligible=%t err=%v", eligible, err)
	}

	updateConn, err := sqlDB.Conn(ctx)
	if err != nil {
		t.Fatalf("open concurrent status-update connection: %v", err)
	}
	defer func() { _ = updateConn.Close() }()
	if _, err := updateConn.ExecContext(ctx, `SET lock_timeout = '100ms'`); err != nil {
		t.Fatalf("set concurrent status-update lock timeout: %v", err)
	}
	if _, err := updateConn.ExecContext(ctx, `UPDATE scan SET status = 'running', stopped_at = NULL WHERE id = $1`, scanID); err == nil {
		t.Fatal("eligibility lock must block a concurrent scan status transition")
	}
	if err := tx.Rollback().Error; err != nil {
		t.Fatalf("release eligibility revalidation transaction: %v", err)
	}

	var status string
	if err := db.WithContext(ctx).Raw(`SELECT status FROM scan WHERE id = ?`, scanID).Scan(&status).Error; err != nil || status != "succeeded" {
		t.Fatalf("concurrent status update must not cross eligibility lock: status=%q err=%v", status, err)
	}
}

func assertProvisioningFollowsScanSequence(t *testing.T, ctx context.Context, db *gorm.DB, lifecycle *ScanHistoryPartitionLifecycle) {
	t.Helper()
	const uncalledLastValue = 9_999
	uncalledRange := ScanHistoryRange{Start: 0, End: 10_000}
	if err := db.WithContext(ctx).Exec(`SELECT setval('scan_id_seq', ?, false)`, uncalledLastValue).Error; err != nil {
		t.Fatalf("reset uncalled scan allocation sequence: %v", err)
	}
	current, err := lifecycle.EnsureCurrentAndFuture(ctx)
	if err != nil || current != uncalledRange {
		t.Fatalf("provision from uncalled scan allocation sequence: range=%+v err=%v, want %+v", current, err, uncalledRange)
	}
	var uncalledScanID int
	if err := db.WithContext(ctx).Raw(`
		INSERT INTO scan (status, stopped_at, created_at)
		VALUES ('succeeded', CURRENT_TIMESTAMP - INTERVAL '31 days', CURRENT_TIMESTAMP - INTERVAL '31 days')
		RETURNING id`).Scan(&uncalledScanID).Error; err != nil || uncalledScanID != uncalledLastValue {
		t.Fatalf("allocate from uncalled scan sequence: id=%d err=%v, want %d", uncalledScanID, err, uncalledLastValue)
	}
	if err := db.WithContext(ctx).Exec(`INSERT INTO website_snapshot (scan_id, payload) VALUES (?, 'uncalled-sequence-covered')`, uncalledScanID).Error; err != nil {
		t.Fatalf("write history for uncalled sequence-allocated scan: %v", err)
	}
	assertScanHistoryRowRoutedToRange(t, ctx, db, "website_snapshot", uncalledScanID, uncalledRange.Start)

	const lastAllocatedID = 100_001
	const expectedNextID = lastAllocatedID + 1
	wantRange := ScanHistoryRange{Start: 100_000, End: 110_000}
	if err := db.WithContext(ctx).Exec(`SELECT setval('scan_id_seq', ?, true)`, lastAllocatedID).Error; err != nil {
		t.Fatalf("advance scan allocation sequence: %v", err)
	}

	current, err = lifecycle.EnsureCurrentAndFuture(ctx)
	if err != nil || current != wantRange {
		t.Fatalf("provision from scan allocation sequence: range=%+v err=%v, want %+v", current, err, wantRange)
	}
	for _, parent := range ScanHistoryParents() {
		for _, start := range []int{wantRange.Start, wantRange.End} {
			partition := scanHistoryPartitionName(parent, start)
			var exists bool
			if err := db.WithContext(ctx).Raw(`SELECT to_regclass(?) IS NOT NULL`, partition).Scan(&exists).Error; err != nil || !exists {
				t.Fatalf("sequence-driven provisioning must create %s: exists=%t err=%v", partition, exists, err)
			}
		}
	}

	var scanID int
	if err := db.WithContext(ctx).Raw(`
		INSERT INTO scan (status, stopped_at, created_at)
		VALUES ('succeeded', CURRENT_TIMESTAMP - INTERVAL '31 days', CURRENT_TIMESTAMP - INTERVAL '31 days')
		RETURNING id`).Scan(&scanID).Error; err != nil || scanID != expectedNextID {
		t.Fatalf("allocate scan after sequence-driven provisioning: id=%d err=%v, want %d", scanID, err, expectedNextID)
	}
	if err := db.WithContext(ctx).Exec(`INSERT INTO website_snapshot (scan_id, payload) VALUES (?, 'sequence-covered')`, scanID).Error; err != nil {
		t.Fatalf("write history for sequence-allocated scan: %v", err)
	}
	assertScanHistoryRowRoutedToRange(t, ctx, db, "website_snapshot", scanID, wantRange.Start)
}

func prepareEligibleRetentionRange(t *testing.T, ctx context.Context, db *gorm.DB, lifecycle *ScanHistoryPartitionLifecycle, scanID int, taskIDs ...int) {
	t.Helper()
	if err := lifecycle.EnsureCoverage(ctx, scanID); err != nil {
		t.Fatalf("ensure retention test coverage for scan %d: %v", scanID, err)
	}
	if err := db.WithContext(ctx).Exec(`
		INSERT INTO scan (id, status, stopped_at, created_at)
		VALUES (?, 'succeeded', CURRENT_TIMESTAMP - INTERVAL '31 days', CURRENT_TIMESTAMP - INTERVAL '31 days')`, scanID).Error; err != nil {
		t.Fatalf("insert retention test scan %d: %v", scanID, err)
	}
	for _, taskID := range taskIDs {
		if err := db.WithContext(ctx).Exec(`INSERT INTO scan_task (id, scan_id) VALUES (?, ?)`, taskID, scanID).Error; err != nil {
			t.Fatalf("insert retention test task %d: %v", taskID, err)
		}
	}
}

func retentionRunOptions(taskDeleteBatchSize, maxTaskDeleteBatchesPerRange, maxRangesPerRun int) ScanHistoryRetentionRunOptions {
	return ScanHistoryRetentionRunOptions{
		TaskDeleteBatchSize:          taskDeleteBatchSize,
		MaxTaskDeleteBatchesPerRange: maxTaskDeleteBatchesPerRange,
		MaxRangesPerRun:              maxRangesPerRun,
		MaxRunDuration:               time.Minute,
	}
}

func assertScanHistoryRangeSoftDeleted(t *testing.T, ctx context.Context, db *gorm.DB, scanID int) {
	t.Helper()
	var softDeleted bool
	if err := db.WithContext(ctx).Raw(`SELECT deleted_at IS NOT NULL FROM scan WHERE id = ?`, scanID).Scan(&softDeleted).Error; err != nil || !softDeleted {
		t.Fatalf("scan %d must remain soft-deleted after partial cleanup: soft_deleted=%t err=%v", scanID, softDeleted, err)
	}
}

func assertScanHistoryRangeHardDeleted(t *testing.T, ctx context.Context, db *gorm.DB, scanID int) {
	t.Helper()
	var count int64
	if err := db.WithContext(ctx).Table("scan").Where("id = ?", scanID).Count(&count).Error; err != nil || count != 0 {
		t.Fatalf("scan %d must be hard-deleted: count=%d err=%v", scanID, count, err)
	}
	assertScanTaskCount(t, ctx, db, scanID, 0)
}

func assertScanHistoryRangeVisible(t *testing.T, ctx context.Context, db *gorm.DB, scanID int) {
	t.Helper()
	var softDeleted bool
	if err := db.WithContext(ctx).Raw(`SELECT deleted_at IS NOT NULL FROM scan WHERE id = ?`, scanID).Scan(&softDeleted).Error; err != nil || softDeleted {
		t.Fatalf("scan %d must remain visible before its range budget is scheduled: soft_deleted=%t err=%v", scanID, softDeleted, err)
	}
}

func assertScanTaskCount(t *testing.T, ctx context.Context, db *gorm.DB, scanID int, want int64) {
	t.Helper()
	var count int64
	if err := db.WithContext(ctx).Table("scan_task").Where("scan_id = ?", scanID).Count(&count).Error; err != nil || count != want {
		t.Fatalf("scan %d task count=%d, want %d: %v", scanID, count, want, err)
	}
}

func resetScanHistoryLifecycleSchema(t *testing.T, ctx context.Context, db *gorm.DB) {
	t.Helper()
	statements := []string{
		"DROP SCHEMA IF EXISTS " + scanHistoryLifecycleTestSchema + " CASCADE",
		"CREATE SCHEMA " + scanHistoryLifecycleTestSchema,
		`CREATE TABLE scan (id SERIAL PRIMARY KEY, status TEXT NOT NULL, stopped_at TIMESTAMPTZ, deleted_at TIMESTAMPTZ, created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP)`,
		`CREATE TABLE scan_operation (id UUID PRIMARY KEY, scan_id INTEGER NOT NULL UNIQUE REFERENCES scan(id) ON DELETE CASCADE, target_id INTEGER NOT NULL, request_fingerprint VARCHAR(64) NOT NULL CHECK (request_fingerprint ~ '^[0-9a-f]{64}$'))`,
		`CREATE TABLE scan_task (id INTEGER PRIMARY KEY, scan_id INTEGER NOT NULL REFERENCES scan(id) ON DELETE CASCADE, UNIQUE (scan_id, id))`,
		`CREATE TABLE current_asset (id INTEGER PRIMARY KEY)`,
		`INSERT INTO current_asset (id) VALUES (1)`,
		`CREATE FUNCTION assert_scan_soft_deleted_before_hard_delete() RETURNS trigger LANGUAGE plpgsql AS $$
			BEGIN
				IF OLD.deleted_at IS NULL THEN RAISE EXCEPTION 'scan must be soft-deleted before physical cleanup'; END IF;
				RETURN OLD;
			END
		$$`,
		`CREATE TRIGGER scan_requires_soft_delete BEFORE DELETE ON scan FOR EACH ROW EXECUTE FUNCTION assert_scan_soft_deleted_before_hard_delete()`,
	}
	for _, parent := range ScanHistoryParents() {
		if parent == "task_progress_log" {
			statements = append(statements, `CREATE TABLE task_progress_log (
				id BIGSERIAL NOT NULL, scan_id INTEGER NOT NULL, task_id INTEGER NOT NULL, request_id TEXT NOT NULL, sequence INTEGER NOT NULL,
				PRIMARY KEY (scan_id, id), UNIQUE (scan_id, task_id, request_id, sequence),
				FOREIGN KEY (scan_id, task_id) REFERENCES scan_task(scan_id, id)
			) PARTITION BY RANGE (scan_id)`)
			continue
		}
		if parent == "screenshot_snapshot" {
			statements = append(statements, `CREATE TABLE screenshot_snapshot (
				id BIGSERIAL NOT NULL, scan_id INTEGER NOT NULL, url TEXT NOT NULL,
				PRIMARY KEY (scan_id, id), UNIQUE (scan_id, url)
			) PARTITION BY RANGE (scan_id)`)
			continue
		}
		statements = append(statements, fmt.Sprintf(`CREATE TABLE %s (
			id BIGSERIAL NOT NULL, scan_id INTEGER NOT NULL, payload TEXT NOT NULL DEFAULT '', PRIMARY KEY (scan_id, id)
		) PARTITION BY RANGE (scan_id)`, parent))
	}
	for _, statement := range statements {
		if err := db.WithContext(ctx).Exec(statement).Error; err != nil {
			t.Fatalf("prepare PostgreSQL scan-history lifecycle schema: %v", err)
		}
	}
}

func assertScanHistoryChildCount(t *testing.T, ctx context.Context, db *gorm.DB, parent string, want int) {
	t.Helper()
	var count int
	if err := db.WithContext(ctx).Raw(`SELECT COUNT(*) FROM pg_inherits WHERE inhparent = ?::regclass`, parent).Scan(&count).Error; err != nil || count != want {
		t.Fatalf("%s child count=%d, want %d: %v", parent, count, want, err)
	}
}

func assertScanHistoryRowRoutedToRange(t *testing.T, ctx context.Context, db *gorm.DB, parent string, scanID, rangeStart int) {
	t.Helper()
	var actual string
	if err := db.WithContext(ctx).Raw(fmt.Sprintf("SELECT tableoid::regclass::text FROM %s WHERE scan_id = ? LIMIT 1", parent), scanID).Scan(&actual).Error; err != nil {
		t.Fatalf("read routed %s row: %v", parent, err)
	}
	if actual != scanHistoryPartitionName(parent, rangeStart) {
		t.Fatalf("%s row routed to %q, want %q", parent, actual, scanHistoryPartitionName(parent, rangeStart))
	}
}

func assertScanHistoryPartitionAbsent(t *testing.T, ctx context.Context, db *gorm.DB, partition string) {
	t.Helper()
	var exists bool
	if err := db.WithContext(ctx).Raw(`SELECT to_regclass(?) IS NOT NULL`, partition).Scan(&exists).Error; err != nil || exists {
		t.Fatalf("partition %s must be dropped: exists=%t err=%v", partition, exists, err)
	}
}
