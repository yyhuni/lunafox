package job

import (
	"context"
	"errors"
	"time"

	snapshotrepo "github.com/yyhuni/lunafox/server/internal/modules/snapshot/repository"
	"github.com/yyhuni/lunafox/server/internal/pkg"
	"go.uber.org/zap"
)

type ScanHistoryRetentionJob struct {
	lifecycle  *snapshotrepo.ScanHistoryPartitionLifecycle
	mode       snapshotrepo.ScanHistoryRetentionMode
	interval   time.Duration
	retention  time.Duration
	runOptions snapshotrepo.ScanHistoryRetentionRunOptions
}

func NewScanHistoryRetentionJob(
	lifecycle *snapshotrepo.ScanHistoryPartitionLifecycle,
	mode snapshotrepo.ScanHistoryRetentionMode,
	interval time.Duration,
	retention time.Duration,
	runOptions snapshotrepo.ScanHistoryRetentionRunOptions,
) *ScanHistoryRetentionJob {
	return &ScanHistoryRetentionJob{
		lifecycle:  lifecycle,
		mode:       mode,
		interval:   interval,
		retention:  retention,
		runOptions: runOptions,
	}
}

func (job *ScanHistoryRetentionJob) Start(ctx context.Context) {
	if job == nil || job.lifecycle == nil {
		return
	}
	if job.interval <= 0 {
		job.interval = time.Hour
	}
	go func() {
		job.runOnce(ctx)
		ticker := time.NewTicker(job.interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				job.runOnce(ctx)
			}
		}
	}()
}

func (job *ScanHistoryRetentionJob) runOnce(ctx context.Context) {
	startedAt := time.Now()
	coverageRange, err := job.lifecycle.EnsureCurrentAndFuture(ctx)
	if err != nil {
		pkg.Error("Ensure scan history partition coverage failed", zap.String("outcome", "failed"), zap.Duration("duration", time.Since(startedAt)), zap.Error(err))
		return
	}
	pkg.Info("Scan history partition coverage reconciled",
		zap.Int("range_start", coverageRange.Start),
		zap.Int("range_end", coverageRange.End),
		zap.Int("expected_children", len(snapshotrepo.ScanHistoryParents())),
		zap.String("outcome", "completed"),
		zap.Duration("duration", time.Since(startedAt)),
	)
	cutoff := time.Now().UTC().Add(-job.retention)
	ranges, outcomes, err := job.lifecycle.RunRetention(ctx, job.mode, cutoff, job.runOptions)
	if err != nil {
		pkg.Error("Run scan history retention failed", zap.String("mode", string(job.mode)), zap.Error(err))
		return
	}
	for _, outcome := range outcomes {
		childStates, childStateErr := job.lifecycle.ChildStates(ctx, outcome.Range)
		scanCount, scanCountErr := job.lifecycle.RangeScanCount(ctx, outcome.Range)
		fields := []zap.Field{
			zap.String("mode", string(job.mode)),
			zap.Int("range_start", outcome.Range.Start),
			zap.Int("range_end", outcome.Range.End),
			zap.String("outcome", outcome.Outcome),
			zap.Duration("duration", outcome.Duration),
			zap.Int("expected_children", len(snapshotrepo.ScanHistoryParents())),
			zap.Strings("child_states", childStates),
			zap.Int64("scan_count", scanCount),
			zap.Int64("task_rows_deleted", outcome.TaskRowsDeleted),
			zap.Int("task_delete_batches", outcome.TaskDeleteBatches),
		}
		if childStateErr != nil {
			fields = append(fields, zap.Error(childStateErr))
		}
		if scanCountErr != nil {
			fields = append(fields, zap.Error(scanCountErr))
		}
		if outcome.Err != nil {
			if errors.Is(outcome.Err, snapshotrepo.ErrScanHistoryLifecycleLockHeld) {
				pkg.Info("Scan history retention range skipped because another replica owns the lifecycle lock", append(fields, zap.Error(outcome.Err))...)
				continue
			}
			pkg.Error("Scan history retention range requires retry", append(fields, zap.Error(outcome.Err))...)
			continue
		}
		if outcome.Outcome == "completed" {
			pkg.Info("Scan history retention range completed", fields...)
			continue
		}
		pkg.Info("Scan history retention range deferred", fields...)
	}
	blocked, err := job.lifecycle.OldestBlockedRange(ctx, cutoff)
	if err != nil {
		pkg.Error("Read oldest blocked scan history range failed", zap.String("mode", string(job.mode)), zap.Error(err))
	} else if blocked != nil {
		pkg.Info("Scan history retention boundary remains retained",
			zap.String("mode", string(job.mode)),
			zap.Int("range_start", blocked.Range.Start),
			zap.Int("range_end", blocked.Range.End),
			zap.Duration("oldest_blocked_boundary_age", time.Since(blocked.OldestAt)),
		)
	}
	pkg.Info("Scan history retention run completed",
		zap.String("mode", string(job.mode)),
		zap.Int("eligible_ranges", len(ranges)),
		zap.Int("range_outcomes", len(outcomes)),
		zap.Int("max_ranges_per_run", job.runOptions.MaxRangesPerRun),
		zap.Int("max_task_delete_batches_per_range", job.runOptions.MaxTaskDeleteBatchesPerRange),
		zap.Duration("max_run_duration", job.runOptions.MaxRunDuration),
		zap.Time("cutoff", cutoff),
	)
}
