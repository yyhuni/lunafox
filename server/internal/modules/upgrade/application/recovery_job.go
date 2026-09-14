package application

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/yyhuni/lunafox/server/internal/modules/upgrade/domain"
)

const defaultUpgradeRecoveryInterval = 2 * time.Second

// RecoveryJob replays the host checkpoint into the durable Server operation.
// It is deliberately independent from the HTTP request and keeps polling after
// a Server restart because the host upgrader may still be running.
type RecoveryJob struct {
	service  *Service
	reader   JournalEventReader
	interval time.Duration

	startOnce sync.Once
	done      chan struct{}
}

func NewRecoveryJob(service *Service, reader JournalEventReader, interval time.Duration) (*RecoveryJob, error) {
	if service == nil || reader == nil {
		return nil, fmt.Errorf("upgrade recovery dependencies are required")
	}
	if interval <= 0 {
		interval = defaultUpgradeRecoveryInterval
	}
	return &RecoveryJob{service: service, reader: reader, interval: interval, done: make(chan struct{})}, nil
}

// Start launches the supervised replay loop. Repeated Start calls are
// idempotent so bootstrap shutdown/reload paths cannot create duplicate readers.
func (job *RecoveryJob) Start(ctx context.Context) {
	if job == nil {
		return
	}
	if ctx == nil {
		ctx = context.Background()
	}
	job.startOnce.Do(func() { go job.run(ctx) })
}

func (job *RecoveryJob) Done() <-chan struct{} {
	if job == nil {
		closed := make(chan struct{})
		close(closed)
		return closed
	}
	return job.done
}

// RunOnce performs one deterministic startup/recovery pass. Missing journal is
// a normal no-upgrade state; malformed or otherwise unreadable journal state is
// converted to a durable failure/recovery outcome when an active operation exists.
func (job *RecoveryJob) RunOnce(ctx context.Context) (*domain.Operation, error) {
	if job == nil || job.service == nil || job.reader == nil {
		return nil, ErrUpgradeDependency
	}
	if ctx == nil {
		ctx = context.Background()
	}
	event, err := job.reader.ReadCurrent(ctx)
	if err == nil {
		return job.service.ReconcileHostEvent(ctx, event)
	}
	if errors.Is(err, os.ErrNotExist) {
		return job.service.ReconcileJournalUnavailable(ctx, "host upgrade journal is not present")
	}
	return job.service.ReconcileJournalUnavailable(ctx, "host upgrade journal is unreadable or corrupt")
}

func (job *RecoveryJob) run(ctx context.Context) {
	defer close(job.done)
	ticker := time.NewTicker(job.interval)
	defer ticker.Stop()
	for {
		_, _ = job.RunOnce(ctx)
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}
