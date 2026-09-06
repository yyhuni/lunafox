package application

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	cleanupdomain "github.com/yyhuni/lunafox/server/internal/modules/targetcleanup/domain"
)

const (
	targetCleanupDueJobBatchSize = 100
	targetCleanupPollInterval    = 15 * time.Second
	targetCleanupDeferInterval   = time.Second
	targetCleanupRetryCap        = time.Hour
)

type TargetCleanupRunner struct {
	jobs       TargetCleanupJobStore
	reconciler TargetCleanupReconciler
	options    TargetCleanupRunOptions
	observer   TargetCleanupRunObserver
	now        func() time.Time
	startOnce  sync.Once
	done       chan struct{}
	wake       chan struct{}
}

func NewTargetCleanupRunner(
	jobs TargetCleanupJobStore,
	reconciler TargetCleanupReconciler,
	options TargetCleanupRunOptions,
	observer TargetCleanupRunObserver,
) (*TargetCleanupRunner, error) {
	if jobs == nil || reconciler == nil {
		return nil, fmt.Errorf("target cleanup runner dependencies are required")
	}
	if err := validateTargetCleanupRunOptions(options); err != nil {
		return nil, err
	}
	if observer == nil {
		observer = noopTargetCleanupRunObserver{}
	}
	return &TargetCleanupRunner{
		jobs: jobs, reconciler: reconciler, options: options, observer: observer, now: time.Now,
		done: make(chan struct{}), wake: make(chan struct{}, 1),
	}, nil
}

func (runner *TargetCleanupRunner) WithClock(now func() time.Time) *TargetCleanupRunner {
	if runner != nil && now != nil {
		runner.now = now
	}
	return runner
}

func (runner *TargetCleanupRunner) Start(ctx context.Context) {
	if runner == nil {
		return
	}
	runner.startOnce.Do(func() {
		go runner.run(ctx)
	})
}

func (runner *TargetCleanupRunner) Done() <-chan struct{} {
	if runner == nil {
		closed := make(chan struct{})
		close(closed)
		return closed
	}
	return runner.done
}

// Wake is deliberately lossy. Persistent due-job polling remains the recovery
// path after a process restart or a dropped in-memory signal.
func (runner *TargetCleanupRunner) Wake() {
	if runner == nil {
		return
	}
	select {
	case runner.wake <- struct{}{}:
	default:
	}
}

func (runner *TargetCleanupRunner) run(ctx context.Context) {
	defer close(runner.done)
	for ctx.Err() == nil {
		_ = runner.RunPass(ctx)
		timer := time.NewTimer(targetCleanupPollInterval)
		select {
		case <-ctx.Done():
			if !timer.Stop() {
				select {
				case <-timer.C:
				default:
				}
			}
			return
		case <-runner.wake:
			if !timer.Stop() {
				select {
				case <-timer.C:
				default:
				}
			}
		case <-timer.C:
		}
	}
}

func (runner *TargetCleanupRunner) RunPass(ctx context.Context) error {
	if runner == nil || runner.jobs == nil || runner.reconciler == nil {
		return fmt.Errorf("target cleanup runner is not configured")
	}
	now := runner.now().UTC()
	jobs, err := runner.jobs.ListDue(ctx, now, targetCleanupDueJobBatchSize)
	if err != nil {
		return err
	}
	for _, job := range jobs {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		runner.runJob(ctx, job)
	}
	backlog, err := runner.jobs.InspectBacklog(ctx)
	if err != nil {
		return err
	}
	runner.observer.BacklogObserved(backlog)
	return nil
}

func (runner *TargetCleanupRunner) runJob(ctx context.Context, job cleanupdomain.CleanupJob) {
	startedAt := runner.now().UTC()
	runner.observer.RunStarted(job, startedAt)
	result, err := runner.reconciler.Reconcile(ctx, job, runner.options)
	finishedAt := runner.now().UTC()
	event := TargetCleanupRunEvent{Job: job, StartedAt: startedAt, FinishedAt: finishedAt, Result: result, RetryCount: job.RetryCount}
	if err != nil {
		if ctx.Err() != nil {
			return
		}
		nextRetry := finishedAt.Add(targetCleanupRetryDelay(job.RetryCount + 1))
		failureClass := targetCleanupFailureClass(err)
		if persistErr := runner.jobs.RecordFailure(ctx, job.ID, job.RetryCount+1, nextRetry, failureClass); persistErr != nil {
			failureClass = "diagnostic_persist_failed"
		}
		event.Outcome = "failed"
		event.RetryCount = job.RetryCount + 1
		event.NextRetryAt = &nextRetry
		event.FailureClass = failureClass
		runner.observer.RunFinished(event)
		return
	}
	if result.Completed {
		event.Outcome = "completed"
		runner.observer.RunFinished(event)
		return
	}

	nextRetry := finishedAt.Add(targetCleanupDeferInterval)
	if persistErr := runner.jobs.Defer(ctx, job.ID, nextRetry); persistErr != nil {
		event.Outcome = "failed"
		event.FailureClass = "diagnostic_persist_failed"
	} else {
		event.Outcome = "deferred"
		event.NextRetryAt = &nextRetry
	}
	runner.observer.RunFinished(event)
}

func targetCleanupRetryDelay(retryCount int) time.Duration {
	if retryCount <= 0 {
		return time.Second
	}
	delay := time.Second
	for attempt := 1; attempt < retryCount && delay < targetCleanupRetryCap; attempt++ {
		delay *= 2
		if delay > targetCleanupRetryCap {
			return targetCleanupRetryCap
		}
	}
	return delay
}

func targetCleanupFailureClass(err error) string {
	if err == nil {
		return ""
	}
	if err == context.DeadlineExceeded {
		return "timeout"
	}
	if err == context.Canceled {
		return "canceled"
	}
	message := strings.ToLower(err.Error())
	if strings.Contains(message, "lock timeout") || strings.Contains(message, "deadlock") {
		return "database_lock"
	}
	if strings.Contains(message, "statement timeout") || strings.Contains(message, "timeout") {
		return "database_timeout"
	}
	return "database_or_reconciliation"
}

type noopTargetCleanupRunObserver struct{}

func (noopTargetCleanupRunObserver) RunStarted(cleanupdomain.CleanupJob, time.Time) {}
func (noopTargetCleanupRunObserver) RunFinished(TargetCleanupRunEvent)              {}
func (noopTargetCleanupRunObserver) BacklogObserved(cleanupdomain.CleanupBacklog) {
}
