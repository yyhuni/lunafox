package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"
)

const (
	// ScanHistoryPartitionSpan is intentionally shared by every historical
	// parent so one eligible range can be reclaimed without row deletion.
	ScanHistoryPartitionSpan    = 10_000
	scanHistoryLifecycleLock    = 8_186_240
	scanHistoryLockTimeout      = "5s"
	scanHistoryStatementTimeout = "30s"
	scanHistoryUnlockTimeout    = 5 * time.Second
)

var ErrScanHistoryLifecycleLockHeld = errors.New("scan history lifecycle lock is held")

type ScanHistoryRetentionMode string

const (
	ScanHistoryRetentionDisabled ScanHistoryRetentionMode = "disabled"
	ScanHistoryRetentionReport   ScanHistoryRetentionMode = "report"
	ScanHistoryRetentionEnforce  ScanHistoryRetentionMode = "enforce"
)

const (
	scanHistoryRetentionOutcomeCompleted              = "completed"
	scanHistoryRetentionOutcomeNoLongerEligible       = "no_longer_eligible"
	scanHistoryRetentionOutcomeTaskBudgetExhausted    = "task_budget_exhausted"
	scanHistoryRetentionOutcomeRunTimeBudgetExhausted = "run_time_budget_exhausted"
	scanHistoryRetentionOutcomeLockContended          = "lock_contended"
	scanHistoryRetentionOutcomeFailed                 = "failed"
	scanHistoryRetentionOutcomeCancelled              = "cancelled"
)

// ScanHistoryRetentionRunOptions bounds one enforce-mode retention pass. The
// limits keep unpartitioned task cleanup from turning into an unbounded
// transaction or a long-running maintenance job.
type ScanHistoryRetentionRunOptions struct {
	TaskDeleteBatchSize          int
	MaxTaskDeleteBatchesPerRange int
	MaxRangesPerRun              int
	MaxRunDuration               time.Duration
}

// ValidateScanHistoryRetentionRunOptions rejects zero or negative cleanup
// budgets before a retention run begins.
func ValidateScanHistoryRetentionRunOptions(options ScanHistoryRetentionRunOptions) error {
	return options.validate()
}

func (options ScanHistoryRetentionRunOptions) validate() error {
	if options.TaskDeleteBatchSize <= 0 {
		return fmt.Errorf("task delete batch size must be positive")
	}
	if options.MaxTaskDeleteBatchesPerRange <= 0 {
		return fmt.Errorf("maximum task delete batches per range must be positive")
	}
	if options.MaxRangesPerRun <= 0 {
		return fmt.Errorf("maximum ranges per run must be positive")
	}
	if options.MaxRunDuration <= 0 {
		return fmt.Errorf("maximum retention run duration must be positive")
	}
	return nil
}

var scanHistoryParents = []string{
	"subdomain_snapshot",
	"website_snapshot",
	"endpoint_snapshot",
	"directory_snapshot",
	"host_port_mapping_snapshot",
	"screenshot_snapshot",
	"vulnerability_snapshot",
	"task_progress_log",
}

type ScanHistoryRange struct {
	Start int `gorm:"column:start_id"`
	End   int `gorm:"column:end_id"`
}

// ScanHistoryRetentionOutcome records one range's enforce result. Operations
// use this to distinguish a retryable range failure from a completed cleanup.
type ScanHistoryRetentionOutcome struct {
	Range             ScanHistoryRange
	Outcome           string
	TaskRowsDeleted   int64
	TaskDeleteBatches int
	Duration          time.Duration
	Err               error
}

type scanHistoryRangeCleanupResult struct {
	Outcome           string
	TaskRowsDeleted   int64
	TaskDeleteBatches int
	Err               error
}

// ScanHistoryBlockedRange identifies the oldest complete range whose lifecycle
// facts still prevent retention. It is an operations signal, never a cleanup
// eligibility shortcut.
type ScanHistoryBlockedRange struct {
	Range    ScanHistoryRange `gorm:"embedded"`
	OldestAt time.Time        `gorm:"column:oldest_at"`
}

func (r ScanHistoryRange) valid() bool {
	return r.Start >= 0 && r.End-r.Start == ScanHistoryPartitionSpan && r.Start%ScanHistoryPartitionSpan == 0
}

func scanHistoryRangeForID(scanID int) (ScanHistoryRange, error) {
	if scanID < 0 {
		return ScanHistoryRange{}, fmt.Errorf("scan id must be non-negative")
	}
	start := scanID / ScanHistoryPartitionSpan * ScanHistoryPartitionSpan
	return ScanHistoryRange{Start: start, End: start + ScanHistoryPartitionSpan}, nil
}

func scanHistoryPartitionName(parent string, start int) string {
	return fmt.Sprintf("%s_p%08d", parent, start)
}

// ScanHistoryPartitionLifecycle owns PostgreSQL catalog operations for the
// eight scan-history parents. It is deliberately separate from result writes:
// result ingestion must fail rather than create DDL on its hot path.
type ScanHistoryPartitionLifecycle struct {
	db *gorm.DB
}

func NewScanHistoryPartitionLifecycle(db *gorm.DB) *ScanHistoryPartitionLifecycle {
	return &ScanHistoryPartitionLifecycle{db: db}
}

func ValidateScanHistoryRetentionMode(mode ScanHistoryRetentionMode) error {
	switch mode {
	case ScanHistoryRetentionDisabled, ScanHistoryRetentionReport, ScanHistoryRetentionEnforce:
		return nil
	default:
		return fmt.Errorf("invalid scan history retention mode %q", mode)
	}
}

// EnsureCoverage pre-creates the range containing scanID and the next range.
func (l *ScanHistoryPartitionLifecycle) EnsureCoverage(ctx context.Context, scanID int) error {
	if l == nil || l.db == nil {
		return fmt.Errorf("scan history partition lifecycle database is required")
	}
	current, err := scanHistoryRangeForID(scanID)
	if err != nil {
		return err
	}
	if err := l.withLock(ctx, func(tx *gorm.DB) error {
		for _, partitionRange := range []ScanHistoryRange{
			current,
			{Start: current.End, End: current.End + ScanHistoryPartitionSpan},
		} {
			if err := ensureScanHistoryRange(tx, partitionRange); err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		return fmt.Errorf("ensure scan history partition coverage: %w", err)
	}
	return nil
}

// EnsureCurrentAndFuture derives the provisioning point from the allocated
// scan sequence. It does not inspect evidence rows, so startup remains cheap.
func (l *ScanHistoryPartitionLifecycle) EnsureCurrentAndFuture(ctx context.Context) (ScanHistoryRange, error) {
	if l == nil || l.db == nil {
		return ScanHistoryRange{}, fmt.Errorf("scan history partition lifecycle database is required")
	}
	var sequence struct {
		LastValue int
		IsCalled  bool
	}
	if err := l.db.WithContext(ctx).Raw(`SELECT last_value, is_called FROM scan_id_seq`).Scan(&sequence).Error; err != nil {
		return ScanHistoryRange{}, fmt.Errorf("read next scan id from allocation sequence: %w", err)
	}
	nextScanID := sequence.LastValue
	if sequence.IsCalled {
		nextScanID++
	}
	current, err := scanHistoryRangeForID(nextScanID)
	if err != nil {
		return ScanHistoryRange{}, err
	}
	// Retention can remove every row in the highest allocated range; table data
	// then no longer identifies the next write's partition, but the sequence does.
	if err := l.EnsureCoverage(ctx, nextScanID); err != nil {
		return ScanHistoryRange{}, err
	}
	return current, nil
}

func ensureScanHistoryRange(db *gorm.DB, partitionRange ScanHistoryRange) error {
	if !partitionRange.valid() {
		return fmt.Errorf("invalid scan history partition range [%d,%d)", partitionRange.Start, partitionRange.End)
	}
	for _, parent := range scanHistoryParents {
		partition := scanHistoryPartitionName(parent, partitionRange.Start)
		statement := fmt.Sprintf(
			"CREATE TABLE IF NOT EXISTS %s PARTITION OF %s FOR VALUES FROM (%d) TO (%d)",
			partition,
			parent,
			partitionRange.Start,
			partitionRange.End,
		)
		if err := db.Exec(statement).Error; err != nil {
			return fmt.Errorf("create partition %s: %w", partition, err)
		}
	}
	return nil
}

// EligibleRanges uses lifecycle facts rather than ID order. A range remains
// retained when any of its scans is unfinished or inside the 30-day window.
func (l *ScanHistoryPartitionLifecycle) EligibleRanges(ctx context.Context, cutoff time.Time) ([]ScanHistoryRange, error) {
	return l.eligibleRanges(ctx, cutoff, 0)
}

func (l *ScanHistoryPartitionLifecycle) eligibleRanges(ctx context.Context, cutoff time.Time, maxRanges int) ([]ScanHistoryRange, error) {
	if l == nil || l.db == nil {
		return nil, fmt.Errorf("scan history partition lifecycle database is required")
	}
	if cutoff.IsZero() {
		return nil, fmt.Errorf("retention cutoff is required")
	}
	if maxRanges < 0 {
		return nil, fmt.Errorf("maximum eligible ranges must not be negative")
	}
	var ranges []ScanHistoryRange
	query := `
		WITH ranged_scans AS (
			SELECT (id / ?) * ? AS start_id, status, stopped_at
			FROM scan
		)
		SELECT start_id, start_id + ? AS end_id
		FROM ranged_scans
		GROUP BY start_id
		HAVING COUNT(*) > 0
		   AND BOOL_AND(
				status IN ('succeeded', 'failed', 'cancelled')
				AND stopped_at IS NOT NULL
				AND stopped_at <= ?
		   )
		ORDER BY start_id ASC`
	args := []any{
		ScanHistoryPartitionSpan,
		ScanHistoryPartitionSpan,
		ScanHistoryPartitionSpan,
		cutoff.UTC(),
	}
	if maxRanges > 0 {
		query += ` LIMIT ?`
		args = append(args, maxRanges)
	}
	err := l.db.WithContext(ctx).Raw(query, args...).Scan(&ranges).Error
	if err != nil {
		return nil, fmt.Errorf("find eligible scan history ranges: %w", err)
	}
	return ranges, nil
}

// OldestBlockedRange reports the oldest populated range that cannot be
// reclaimed at cutoff. It makes delayed boundary retention observable without
// weakening the requirement that every scan in a range be terminal and expired.
func (l *ScanHistoryPartitionLifecycle) OldestBlockedRange(ctx context.Context, cutoff time.Time) (*ScanHistoryBlockedRange, error) {
	if l == nil || l.db == nil {
		return nil, fmt.Errorf("scan history partition lifecycle database is required")
	}
	if cutoff.IsZero() {
		return nil, fmt.Errorf("retention cutoff is required")
	}
	var blocked ScanHistoryBlockedRange
	result := l.db.WithContext(ctx).Raw(`
		WITH ranged_scans AS (
			SELECT (id / ?) * ? AS start_id, status, stopped_at, created_at
			FROM scan
		)
		SELECT start_id, start_id + ? AS end_id, MIN(created_at) AS oldest_at
		FROM ranged_scans
		GROUP BY start_id
		HAVING COUNT(*) > 0
		   AND NOT BOOL_AND(
				status IN ('succeeded', 'failed', 'cancelled')
				AND stopped_at IS NOT NULL
				AND stopped_at <= ?
			)
		ORDER BY oldest_at ASC
		LIMIT 1`,
		ScanHistoryPartitionSpan,
		ScanHistoryPartitionSpan,
		ScanHistoryPartitionSpan,
		cutoff.UTC(),
	).Scan(&blocked)
	if result.Error != nil {
		return nil, fmt.Errorf("find oldest blocked scan history range: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return nil, nil
	}
	return &blocked, nil
}

// ChildStates returns catalog-backed state for every history child in range.
// It is used for lifecycle observability and intentionally carries no evidence
// payload or scan content.
func (l *ScanHistoryPartitionLifecycle) ChildStates(ctx context.Context, partitionRange ScanHistoryRange) ([]string, error) {
	if l == nil || l.db == nil {
		return nil, fmt.Errorf("scan history partition lifecycle database is required")
	}
	if !partitionRange.valid() {
		return nil, fmt.Errorf("invalid scan history partition range [%d,%d)", partitionRange.Start, partitionRange.End)
	}
	states := make([]string, 0, len(scanHistoryParents))
	for _, parent := range scanHistoryParents {
		partition := scanHistoryPartitionName(parent, partitionRange.Start)
		var exists bool
		if err := l.db.WithContext(ctx).Raw(`SELECT to_regclass(?) IS NOT NULL`, partition).Scan(&exists).Error; err != nil {
			return nil, fmt.Errorf("inspect partition state %s: %w", partition, err)
		}
		state := "dropped"
		if exists {
			var attached bool
			if err := l.db.WithContext(ctx).Raw(`
				SELECT EXISTS (
					SELECT 1 FROM pg_inherits
					WHERE inhparent = ?::regclass AND inhrelid = ?::regclass
				)`, parent, partition).Scan(&attached).Error; err != nil {
				return nil, fmt.Errorf("inspect attachment state %s: %w", partition, err)
			}
			if attached {
				state = "attached"
			} else {
				state = "detached"
			}
		}
		states = append(states, parent+":"+state)
	}
	return states, nil
}

// RangeScanCount returns only metadata cardinality for retention logs.
func (l *ScanHistoryPartitionLifecycle) RangeScanCount(ctx context.Context, partitionRange ScanHistoryRange) (int64, error) {
	if l == nil || l.db == nil {
		return 0, fmt.Errorf("scan history partition lifecycle database is required")
	}
	var count int64
	if err := l.db.WithContext(ctx).Raw(`SELECT COUNT(*) FROM scan WHERE id >= ? AND id < ?`, partitionRange.Start, partitionRange.End).Scan(&count).Error; err != nil {
		return 0, fmt.Errorf("count scans in history range [%d,%d): %w", partitionRange.Start, partitionRange.End, err)
	}
	return count, nil
}

// RunRetention executes one bounded lifecycle pass. Report mode has no
// mutations; enforce mode revalidates each range while holding one session
// lock across independently committed cleanup phases.
func (l *ScanHistoryPartitionLifecycle) RunRetention(ctx context.Context, mode ScanHistoryRetentionMode, cutoff time.Time, options ScanHistoryRetentionRunOptions) ([]ScanHistoryRange, []ScanHistoryRetentionOutcome, error) {
	if err := ValidateScanHistoryRetentionMode(mode); err != nil {
		return nil, nil, err
	}
	if mode == ScanHistoryRetentionDisabled {
		return nil, nil, nil
	}
	if err := ValidateScanHistoryRetentionRunOptions(options); err != nil {
		return nil, nil, err
	}
	if mode == ScanHistoryRetentionReport {
		ranges, err := l.EligibleRanges(ctx, cutoff)
		return ranges, nil, err
	}

	runCtx, cancel := context.WithTimeout(ctx, options.MaxRunDuration)
	defer cancel()
	ranges, err := l.eligibleRanges(runCtx, cutoff, options.MaxRangesPerRun)
	if err != nil {
		return nil, nil, err
	}
	if len(ranges) == 0 {
		return ranges, nil, nil
	}

	outcomes := make([]ScanHistoryRetentionOutcome, 0, len(ranges))
	lockErr := l.withLifecycleSessionLock(runCtx, func(conn *gorm.DB) error {
		for _, partitionRange := range ranges {
			startedAt := time.Now()
			result := l.cleanupRange(runCtx, conn, partitionRange, cutoff, options)
			outcomes = append(outcomes, ScanHistoryRetentionOutcome{
				Range:             partitionRange,
				Outcome:           result.Outcome,
				TaskRowsDeleted:   result.TaskRowsDeleted,
				TaskDeleteBatches: result.TaskDeleteBatches,
				Duration:          time.Since(startedAt),
				Err:               result.Err,
			})
			if runCtx.Err() != nil {
				break
			}
		}
		return nil
	})
	if lockErr == nil {
		return ranges, outcomes, nil
	}
	if errors.Is(lockErr, ErrScanHistoryLifecycleLockHeld) {
		for _, partitionRange := range ranges {
			outcomes = append(outcomes, ScanHistoryRetentionOutcome{
				Range:   partitionRange,
				Outcome: scanHistoryRetentionOutcomeLockContended,
				Err:     lockErr,
			})
		}
		return ranges, outcomes, nil
	}
	if errors.Is(lockErr, context.DeadlineExceeded) {
		if len(outcomes) == 0 {
			outcomes = append(outcomes, ScanHistoryRetentionOutcome{
				Range:   ranges[0],
				Outcome: scanHistoryRetentionOutcomeRunTimeBudgetExhausted,
				Err:     lockErr,
			})
		}
		return ranges, outcomes, nil
	}
	return ranges, outcomes, fmt.Errorf("run scan history retention lifecycle: %w", lockErr)
}

func (l *ScanHistoryPartitionLifecycle) cleanupRange(ctx context.Context, conn *gorm.DB, partitionRange ScanHistoryRange, cutoff time.Time, options ScanHistoryRetentionRunOptions) scanHistoryRangeCleanupResult {
	eligible, err := l.prepareRangeCleanup(ctx, conn, partitionRange, cutoff)
	if err != nil {
		return failedRangeCleanupResult(err)
	}
	if !eligible {
		return scanHistoryRangeCleanupResult{Outcome: scanHistoryRetentionOutcomeNoLongerEligible}
	}

	completed, err := l.finalizeRangeCleanup(ctx, conn, partitionRange)
	if err != nil {
		return failedRangeCleanupResult(err)
	}
	if completed {
		return scanHistoryRangeCleanupResult{Outcome: scanHistoryRetentionOutcomeCompleted}
	}

	result := scanHistoryRangeCleanupResult{}
	for result.TaskDeleteBatches < options.MaxTaskDeleteBatchesPerRange {
		deleted, err := l.deleteScanTaskBatch(ctx, conn, partitionRange, options.TaskDeleteBatchSize)
		if err != nil {
			result.Err = err
			result.Outcome = scanHistoryRetentionOutcomeForError(err)
			return result
		}
		result.TaskRowsDeleted += deleted
		result.TaskDeleteBatches++

		completed, err = l.finalizeRangeCleanup(ctx, conn, partitionRange)
		if err != nil {
			result.Err = err
			result.Outcome = scanHistoryRetentionOutcomeForError(err)
			return result
		}
		if completed {
			result.Outcome = scanHistoryRetentionOutcomeCompleted
			return result
		}
	}
	result.Outcome = scanHistoryRetentionOutcomeTaskBudgetExhausted
	return result
}

func (l *ScanHistoryPartitionLifecycle) prepareRangeCleanup(ctx context.Context, conn *gorm.DB, partitionRange ScanHistoryRange, cutoff time.Time) (bool, error) {
	eligible := false
	err := l.withLifecycleTransaction(ctx, conn, func(tx *gorm.DB) error {
		var err error
		eligible, err = eligibleRange(tx, partitionRange, cutoff)
		if err != nil || !eligible {
			return err
		}
		if err := tx.Exec(`UPDATE scan SET deleted_at = COALESCE(deleted_at, CURRENT_TIMESTAMP) WHERE id >= ? AND id < ?`, partitionRange.Start, partitionRange.End).Error; err != nil {
			return fmt.Errorf("soft-delete scan history range [%d,%d): %w", partitionRange.Start, partitionRange.End, err)
		}
		for _, parent := range scanHistoryParents {
			if err := detachAndDropPartition(tx, parent, partitionRange.Start); err != nil {
				return err
			}
		}
		return nil
	})
	return eligible, err
}

func (l *ScanHistoryPartitionLifecycle) deleteScanTaskBatch(ctx context.Context, conn *gorm.DB, partitionRange ScanHistoryRange, batchSize int) (int64, error) {
	var deleted int64
	err := l.withLifecycleTransaction(ctx, conn, func(tx *gorm.DB) error {
		result := tx.Exec(`
			WITH task_batch AS (
				SELECT id
				FROM scan_task
				WHERE scan_id >= ? AND scan_id < ?
				ORDER BY id
				LIMIT ?
			)
			DELETE FROM scan_task AS task
			USING task_batch
			WHERE task.id = task_batch.id`, partitionRange.Start, partitionRange.End, batchSize)
		if result.Error != nil {
			return fmt.Errorf("batch-delete scan tasks for [%d,%d): %w", partitionRange.Start, partitionRange.End, result.Error)
		}
		deleted = result.RowsAffected
		return nil
	})
	return deleted, err
}

func (l *ScanHistoryPartitionLifecycle) finalizeRangeCleanup(ctx context.Context, conn *gorm.DB, partitionRange ScanHistoryRange) (bool, error) {
	completed := false
	err := l.withLifecycleTransaction(ctx, conn, func(tx *gorm.DB) error {
		var scanCount int64
		// This row lock closes the gap between checking tasks and hard deletion:
		// a concurrent foreign-key insert cannot turn the final delete into a cascade.
		if err := tx.Raw(`
			SELECT COUNT(*)
			FROM (
				SELECT id FROM scan
				WHERE id >= ? AND id < ?
				FOR UPDATE
			) AS locked_scans`, partitionRange.Start, partitionRange.End).Scan(&scanCount).Error; err != nil {
			return fmt.Errorf("lock scans before final cleanup [%d,%d): %w", partitionRange.Start, partitionRange.End, err)
		}
		if scanCount == 0 {
			completed = true
			return nil
		}

		var tasksRemain bool
		if err := tx.Raw(`SELECT EXISTS (SELECT 1 FROM scan_task WHERE scan_id >= ? AND scan_id < ?)`, partitionRange.Start, partitionRange.End).Scan(&tasksRemain).Error; err != nil {
			return fmt.Errorf("check remaining scan tasks for [%d,%d): %w", partitionRange.Start, partitionRange.End, err)
		}
		if tasksRemain {
			return nil
		}
		if err := scanHistoryPartitionsDropped(tx, partitionRange); err != nil {
			return err
		}

		if err := tx.Exec(`
			DELETE FROM scan
			WHERE id >= ? AND id < ?
			  AND NOT EXISTS (
				SELECT 1 FROM scan_task
				WHERE scan_id >= ? AND scan_id < ?
			  )`, partitionRange.Start, partitionRange.End, partitionRange.Start, partitionRange.End).Error; err != nil {
			return fmt.Errorf("hard-delete scan history range [%d,%d): %w", partitionRange.Start, partitionRange.End, err)
		}
		completed = true
		return nil
	})
	return completed, err
}

func failedRangeCleanupResult(err error) scanHistoryRangeCleanupResult {
	return scanHistoryRangeCleanupResult{Outcome: scanHistoryRetentionOutcomeForError(err), Err: err}
}

func scanHistoryRetentionOutcomeForError(err error) string {
	switch {
	case errors.Is(err, context.DeadlineExceeded):
		return scanHistoryRetentionOutcomeRunTimeBudgetExhausted
	case errors.Is(err, context.Canceled):
		return scanHistoryRetentionOutcomeCancelled
	default:
		return scanHistoryRetentionOutcomeFailed
	}
}

func eligibleRange(db *gorm.DB, partitionRange ScanHistoryRange, cutoff time.Time) (bool, error) {
	// Raw SQL deliberately includes soft-deleted scans. Their deleted_at marker
	// is what lets a later bounded run resume from catalog and task-row state.
	// The locked read keeps a concurrent status transition from crossing the
	// eligibility check and causing a newly active range to be hidden.
	var eligible bool
	err := db.Raw(`
		SELECT COUNT(*) > 0 AND BOOL_AND(
			status IN ('succeeded', 'failed', 'cancelled')
			AND stopped_at IS NOT NULL
			AND stopped_at <= ?
		)
		FROM (
			SELECT status, stopped_at
			FROM scan
			WHERE id >= ? AND id < ?
			FOR UPDATE
		) AS locked_scans`, cutoff.UTC(), partitionRange.Start, partitionRange.End).
		Scan(&eligible).Error
	if err != nil {
		return false, fmt.Errorf("revalidate scan history range [%d,%d): %w", partitionRange.Start, partitionRange.End, err)
	}
	return eligible, nil
}

func detachAndDropPartition(db *gorm.DB, parent string, rangeStart int) error {
	partition := scanHistoryPartitionName(parent, rangeStart)
	var exists bool
	if err := db.Raw(`SELECT to_regclass(?) IS NOT NULL`, partition).Scan(&exists).Error; err != nil {
		return fmt.Errorf("inspect partition %s: %w", partition, err)
	}
	if !exists {
		return nil
	}
	var attached bool
	if err := db.Raw(`
		SELECT EXISTS (
			SELECT 1 FROM pg_inherits
			WHERE inhparent = ?::regclass AND inhrelid = ?::regclass
		)`, parent, partition).Scan(&attached).Error; err != nil {
		return fmt.Errorf("inspect partition %s: %w", partition, err)
	}
	if attached {
		if err := db.Exec(fmt.Sprintf("ALTER TABLE %s DETACH PARTITION %s", parent, partition)).Error; err != nil {
			return fmt.Errorf("detach partition %s: %w", partition, err)
		}
	}
	if err := db.Exec(fmt.Sprintf("DROP TABLE %s", partition)).Error; err != nil {
		return fmt.Errorf("drop partition %s: %w", partition, err)
	}
	return nil
}

func scanHistoryPartitionsDropped(db *gorm.DB, partitionRange ScanHistoryRange) error {
	for _, parent := range scanHistoryParents {
		partition := scanHistoryPartitionName(parent, partitionRange.Start)
		var exists bool
		if err := db.Raw(`SELECT to_regclass(?) IS NOT NULL`, partition).Scan(&exists).Error; err != nil {
			return fmt.Errorf("inspect final partition state %s: %w", partition, err)
		}
		if exists {
			return fmt.Errorf("cannot hard-delete scan history range [%d,%d): partition %s still exists", partitionRange.Start, partitionRange.End, partition)
		}
	}
	return nil
}

// withLifecycleSessionLock holds one PostgreSQL session lock across committed
// cleanup phases. Transaction locks would release between task batches and let
// another server become a concurrent lifecycle owner.
func (l *ScanHistoryPartitionLifecycle) withLifecycleSessionLock(ctx context.Context, operation func(conn *gorm.DB) error) error {
	if l == nil || l.db == nil {
		return fmt.Errorf("scan history partition lifecycle database is required")
	}
	return l.db.WithContext(ctx).Connection(func(conn *gorm.DB) (err error) {
		var acquired bool
		if err := conn.Raw(`SELECT pg_try_advisory_lock(?)`, int64(scanHistoryLifecycleLock)).Scan(&acquired).Error; err != nil {
			return fmt.Errorf("acquire scan history lifecycle session lock: %w", err)
		}
		if !acquired {
			return ErrScanHistoryLifecycleLockHeld
		}
		defer func() {
			unlockCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), scanHistoryUnlockTimeout)
			defer cancel()
			var released bool
			unlockErr := conn.WithContext(unlockCtx).Raw(`SELECT pg_advisory_unlock(?)`, int64(scanHistoryLifecycleLock)).Scan(&released).Error
			if unlockErr == nil && !released {
				unlockErr = errors.New("scan history lifecycle session lock was not held by pinned connection")
			}
			if unlockErr != nil {
				err = errors.Join(err, fmt.Errorf("release scan history lifecycle session lock: %w", unlockErr))
			}
		}()
		return operation(conn)
	})
}

func (l *ScanHistoryPartitionLifecycle) withLifecycleTransaction(ctx context.Context, conn *gorm.DB, operation func(*gorm.DB) error) error {
	return conn.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := configureScanHistoryTransaction(tx); err != nil {
			return err
		}
		return operation(tx)
	})
}

func configureScanHistoryTransaction(tx *gorm.DB) error {
	if err := tx.Exec(`SELECT set_config('lock_timeout', ?, true)`, scanHistoryLockTimeout).Error; err != nil {
		return fmt.Errorf("set scan history lock timeout: %w", err)
	}
	if err := tx.Exec(`SELECT set_config('statement_timeout', ?, true)`, scanHistoryStatementTimeout).Error; err != nil {
		return fmt.Errorf("set scan history statement timeout: %w", err)
	}
	return nil
}

func (l *ScanHistoryPartitionLifecycle) withLock(ctx context.Context, operation func(*gorm.DB) error) error {
	return l.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := configureScanHistoryTransaction(tx); err != nil {
			return err
		}
		var acquired bool
		if err := tx.Raw(`SELECT pg_try_advisory_xact_lock(?)`, int64(scanHistoryLifecycleLock)).Scan(&acquired).Error; err != nil {
			return fmt.Errorf("acquire scan history lifecycle lock: %w", err)
		}
		if !acquired {
			return ErrScanHistoryLifecycleLockHeld
		}
		return operation(tx)
	})
}

func ScanHistoryParents() []string {
	return append([]string(nil), scanHistoryParents...)
}

func IsScanHistoryParent(table string) bool {
	return strings.TrimSpace(table) != "" && containsScanHistoryParent(table)
}

func containsScanHistoryParent(table string) bool {
	for _, parent := range scanHistoryParents {
		if table == parent {
			return true
		}
	}
	return false
}
