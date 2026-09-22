package application

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/yyhuni/lunafox/server/internal/modules/upgrade/domain"
	"github.com/yyhuni/lunafox/server/internal/pkg"
	"go.uber.org/zap"
)

const defaultUpgradeRecoveryInterval = 2 * time.Second
const defaultUpgradeRecoveryStallTimeout = 15 * time.Minute
const upgradeRecoveryGracePeriod = 2 * time.Minute
const minimumUpgradeRecoveryStallTimeout = 5 * time.Minute
const maximumUpgradeRecoveryStallTimeout = 60 * time.Minute

// RecoveryJob replays the host checkpoint into the durable Server operation.
// It is deliberately independent from the HTTP request and keeps polling after
// a Server restart because the host upgrader may still be running.
type RecoveryJob struct {
	service      *Service
	reader       JournalEventReader
	interval     time.Duration
	stallTimeout time.Duration

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

// SetStallTimeout is an explicit deployment/test override. Leaving it unset
// makes the service derive the watchdog window from each operation's validated
// maintenance window instead of applying one global timeout to every release.
func (job *RecoveryJob) SetStallTimeout(timeout time.Duration) {
	if job == nil {
		return
	}
	if timeout < 0 {
		timeout = 0
	}
	job.stallTimeout = timeout
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
		// The recovery boundary, rather than a pluggable reader, owns the
		// provenance marker that enables journal replay validation.
		event.FromJournal = true
		operation, reconcileErr := job.service.ReconcileHostEvent(ctx, event)
		if reconcileErr != nil {
			return operation, reconcileErr
		}
		return job.service.ReconcileStalledOperation(ctx, job.stallTimeout)
	}
	if errors.Is(err, os.ErrNotExist) {
		return job.service.ReconcileStalledOperation(ctx, job.stallTimeout)
	}
	// A corrupt checkpoint is actionable immediately because it cannot be
	// replayed safely. A missing checkpoint is handled by the bounded watchdog
	// above so a short write/replace window does not fail a fresh operation.
	if _, reconcileErr := job.service.ReconcileJournalUnavailable(ctx, "host upgrade journal is unreadable or corrupt"); reconcileErr != nil {
		return nil, reconcileErr
	}
	return job.service.ReconcileStalledOperation(ctx, job.stallTimeout)
}

func (job *RecoveryJob) run(ctx context.Context) {
	defer close(job.done)
	ticker := time.NewTicker(job.interval)
	defer ticker.Stop()
	for {
		if _, err := job.RunOnce(ctx); err != nil {
			// A single corrupt checkpoint or transient database failure must not
			// terminate the supervisor; the next tick retries and the error is
			// visible through the server's structured logger hook.
			job.reportError(err)
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func (job *RecoveryJob) reportError(err error) {
	if err == nil {
		return
	}
	pkg.Error("Upgrade recovery poll failed", zap.String("component", "upgrade-recovery"), zap.Error(err))
}
