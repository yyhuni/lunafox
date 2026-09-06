package application

import (
	"context"
	"fmt"
	"sync"
	"time"

	"go.uber.org/zap"
)

const (
	occurrenceRetentionAge      = 7 * 24 * time.Hour
	occurrenceRetentionInterval = time.Hour
	occurrenceRetentionDeadline = 5 * time.Minute
	occurrenceRetentionBatches  = 100
)

type OccurrenceRetentionJob struct {
	repository OccurrenceRetentionRepository
	clock      RuntimeClock
	waiter     RuntimeWaiter
	logger     EventLogger
	startOnce  sync.Once
	done       chan struct{}
}

func NewOccurrenceRetentionJob(repository OccurrenceRetentionRepository) *OccurrenceRetentionJob {
	return &OccurrenceRetentionJob{
		repository: repository,
		clock:      systemRuntimeClock{},
		waiter:     systemRuntimeWaiter{},
		logger:     packageEventLogger{},
		done:       make(chan struct{}),
	}
}

func (job *OccurrenceRetentionJob) WithRuntime(clock RuntimeClock, waiter RuntimeWaiter, logger EventLogger) *OccurrenceRetentionJob {
	if job == nil {
		return nil
	}
	if clock != nil {
		job.clock = clock
	}
	if waiter != nil {
		job.waiter = waiter
	}
	if logger != nil {
		job.logger = logger
	}
	return job
}

func (job *OccurrenceRetentionJob) Start(ctx context.Context) {
	if job == nil {
		return
	}
	job.startOnce.Do(func() { go job.run(ctx) })
}

func (job *OccurrenceRetentionJob) Done() <-chan struct{} {
	if job == nil {
		closed := make(chan struct{})
		close(closed)
		return closed
	}
	return job.done
}

func (job *OccurrenceRetentionJob) run(ctx context.Context) {
	defer close(job.done)
	for ctx.Err() == nil {
		job.RunOnce(ctx)
		if err := job.waiter.Wait(ctx, occurrenceRetentionInterval); err != nil {
			return
		}
	}
}

func (job *OccurrenceRetentionJob) RunOnce(ctx context.Context) (int64, error) {
	if job == nil || job.repository == nil || job.clock == nil {
		return 0, fmt.Errorf("scheduled scan occurrence retention is not configured")
	}
	startedAt := job.clock.Now().UTC()
	cutoff := startedAt.Add(-occurrenceRetentionAge)
	runCtx, cancel := context.WithTimeout(ctx, occurrenceRetentionDeadline)
	defer cancel()

	var deleted int64
	for batch := 0; batch < occurrenceRetentionBatches; batch++ {
		count, err := job.repository.DeleteOccurrenceBatch(runCtx, cutoff)
		deleted += count
		if err != nil {
			job.logger.Error("Scheduled scan occurrence retention failed",
				zap.Int64("scheduled_scan.occurrence.deleted_count", deleted),
				zap.Time("scheduled_scan.occurrence.cutoff", cutoff),
				zap.String("outcome", "failed"),
				zap.String("error_kind", runtimeErrorKind(err)),
			)
			return deleted, err
		}
		if count < OccurrenceDeleteBatchSize {
			break
		}
	}
	if err := runCtx.Err(); err != nil {
		job.logger.Error("Scheduled scan occurrence retention failed",
			zap.Int64("scheduled_scan.occurrence.deleted_count", deleted),
			zap.Time("scheduled_scan.occurrence.cutoff", cutoff),
			zap.String("outcome", "failed"),
			zap.String("error_kind", runtimeErrorKind(err)),
		)
		return deleted, err
	}
	if deleted > 0 {
		duration := job.clock.Now().UTC().Sub(startedAt)
		if duration < 0 {
			duration = 0
		}
		job.logger.Info("Scheduled scan occurrence retention completed",
			zap.Int64("scheduled_scan.occurrence.deleted_count", deleted),
			zap.Time("scheduled_scan.occurrence.cutoff", cutoff),
			zap.String("outcome", "completed"),
			zap.Duration("duration", duration),
		)
	}
	return deleted, nil
}
