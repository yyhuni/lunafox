package application

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/yyhuni/lunafox/server/internal/modules/notification/domain"
	pkg "github.com/yyhuni/lunafox/server/internal/pkg"
	"go.uber.org/zap"
)

const (
	defaultNotificationRetentionInterval = time.Hour
	defaultNotificationRetentionBatch    = 500
	defaultNotificationRetentionTimeout  = 5 * time.Minute
)

// RetentionJobOptions bounds periodic, retryable notification cleanup work.
// A run deletes at most one bounded batch from each independently retained
// projection, so a transient failure can be retried without touching active
// outbox or delivery work.
type RetentionJobOptions struct {
	Interval   time.Duration
	BatchSize  int
	RunTimeout time.Duration
}

// RetentionResult reports the rows removed by one bounded cleanup cycle.
type RetentionResult struct {
	PublishedOutboxRows int64
	ExpiredInboxRows    int64
	TerminalDeliveries  int64
}

// RetentionJob removes only records whose independent retention lifecycle is
// complete. Canonical facts and non-terminal work remain outside this job.
type RetentionJob struct {
	outbox     OutboxStore
	inbox      InboxStore
	deliveries DeliveryStore
	options    RetentionJobOptions
	now        func() time.Time
	startOnce  sync.Once
	done       chan struct{}
}

// NewRetentionJob creates the managed 90-day notification cleanup worker.
func NewRetentionJob(outbox OutboxStore, inbox InboxStore, deliveries DeliveryStore, options RetentionJobOptions) *RetentionJob {
	if outbox == nil || inbox == nil || deliveries == nil {
		panic("notification retention stores are required")
	}
	if options.Interval <= 0 {
		options.Interval = defaultNotificationRetentionInterval
	}
	if options.BatchSize <= 0 {
		options.BatchSize = defaultNotificationRetentionBatch
	}
	if options.RunTimeout <= 0 {
		options.RunTimeout = defaultNotificationRetentionTimeout
	}
	return &RetentionJob{
		outbox:     outbox,
		inbox:      inbox,
		deliveries: deliveries,
		options:    options,
		now:        func() time.Time { return time.Now().UTC() },
		done:       make(chan struct{}),
	}
}

// Start runs an initial bounded cleanup and then repeats until cancellation.
func (job *RetentionJob) Start(ctx context.Context) {
	if job == nil {
		return
	}
	job.startOnce.Do(func() { go job.run(ctx) })
}

// Done closes after the job has stopped scheduling and running cleanup.
func (job *RetentionJob) Done() <-chan struct{} {
	if job == nil {
		closed := make(chan struct{})
		close(closed)
		return closed
	}
	return job.done
}

func (job *RetentionJob) run(ctx context.Context) {
	defer close(job.done)
	job.runAndLog(ctx)
	ticker := time.NewTicker(job.options.Interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			job.runAndLog(ctx)
		}
	}
}

func (job *RetentionJob) runAndLog(ctx context.Context) {
	result, err := job.RunOnce(ctx)
	if err != nil {
		if ctx.Err() == nil {
			pkg.Warn("Notification retention run deferred", zap.Error(err))
		}
		return
	}
	if result.PublishedOutboxRows+result.ExpiredInboxRows+result.TerminalDeliveries == 0 {
		return
	}
	pkg.Info("Notification retention run completed",
		zap.Int64("notification.outbox.deleted", result.PublishedOutboxRows),
		zap.Int64("notification.inbox.deleted", result.ExpiredInboxRows),
		zap.Int64("notification.delivery.deleted", result.TerminalDeliveries),
	)
}

// RunOnce executes one bounded cleanup cycle. Each repository rechecks its
// status predicate in SQL, which keeps the work safe to retry after failures.
func (job *RetentionJob) RunOnce(ctx context.Context) (RetentionResult, error) {
	if job == nil || job.outbox == nil || job.inbox == nil || job.deliveries == nil || job.now == nil {
		return RetentionResult{}, fmt.Errorf("notification retention is not configured")
	}
	if err := ctx.Err(); err != nil {
		return RetentionResult{}, err
	}
	runCtx, cancel := context.WithTimeout(ctx, job.options.RunTimeout)
	defer cancel()
	batch := domain.RetentionBatch{
		Limit:  job.options.BatchSize,
		Before: job.now().UTC().Add(-domain.NotificationRetention),
	}
	result := RetentionResult{}
	var err error
	if result.PublishedOutboxRows, err = job.outbox.DeletePublishedBefore(runCtx, batch); err != nil {
		return result, fmt.Errorf("delete retained notification outbox rows: %w", err)
	}
	if result.ExpiredInboxRows, err = job.inbox.DeleteExpiredInbox(runCtx, batch); err != nil {
		return result, fmt.Errorf("delete expired notification inbox rows: %w", err)
	}
	if result.TerminalDeliveries, err = job.deliveries.DeleteTerminalBefore(runCtx, batch); err != nil {
		return result, fmt.Errorf("delete retained notification deliveries: %w", err)
	}
	if err := runCtx.Err(); err != nil {
		return result, err
	}
	return result, nil
}
