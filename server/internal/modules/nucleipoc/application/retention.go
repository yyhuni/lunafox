package application

import (
	"context"
	"sync"
	"time"
)

const (
	SyncTaskRetention = 30 * 24 * time.Hour
	// Tombstones outlive task diagnostics so an old requestId cannot be reused
	// immediately after the visible task history expires.
	SyncRequestTombstoneRetention = 365 * 24 * time.Hour
	RetentionInterval             = time.Hour
)

// RetentionJob owns startup recovery and bounded hourly cleanup. It is kept
// separate from the HTTP handler so expired task records cannot be recreated
// by a request and workspace residuals are retried independently.
type RetentionJob struct {
	store     Store
	workspace Workspace
	interval  time.Duration
	startOnce sync.Once
	done      chan struct{}
}

func NewRetentionJob(store Store, workspace Workspace, interval time.Duration) *RetentionJob {
	if interval <= 0 {
		interval = RetentionInterval
	}
	return &RetentionJob{store: store, workspace: workspace, interval: interval, done: make(chan struct{})}
}

func (job *RetentionJob) Start(ctx context.Context) {
	if job == nil {
		return
	}
	if ctx == nil {
		ctx = context.Background()
	}
	job.startOnce.Do(func() {
		go func() {
			defer close(job.done)
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
	})
}

func (job *RetentionJob) Done() <-chan struct{} {
	if job == nil {
		closed := make(chan struct{})
		close(closed)
		return closed
	}
	return job.done
}

func (job *RetentionJob) RunOnce(ctx context.Context, now time.Time) error {
	if job == nil || job.store == nil {
		return nil
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}
	if _, err := job.store.DeleteExpiredTasks(ctx, now.Add(-SyncTaskRetention), 500); err != nil {
		return err
	}
	if _, err := job.store.DeleteExpiredTombstones(ctx, now.Add(-SyncRequestTombstoneRetention), 500); err != nil {
		return err
	}
	if job.workspace != nil {
		protected := map[string]struct{}{}
		if reader, ok := job.store.(ActiveWorkspaceKeyReader); ok {
			if keys, err := reader.ActiveWorkspaceKeys(ctx); err == nil {
				for _, key := range keys {
					if key != "" {
						protected[key] = struct{}{}
					}
				}
			}
		}
		if protectedWorkspace, ok := job.workspace.(ProtectedResidualWorkspace); ok {
			_, _ = protectedWorkspace.RemoveResidualsExcept(ctx, protected)
		} else {
			_, _ = job.workspace.RemoveResiduals(ctx)
		}
	}
	return nil
}

func (job *RetentionJob) runOnce(ctx context.Context) {
	if job == nil || job.store == nil {
		return
	}
	// Recovery runs before the runner is started by bootstrap. A bounded
	// context prevents a broken database from holding server startup forever.
	recoveryCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	_, _ = job.store.RecoverInterruptedTasks(recoveryCtx, time.Now().UTC())
	_ = job.RunOnce(recoveryCtx, time.Now().UTC())
}
