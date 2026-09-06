package application

import (
	"context"
	"time"

	"github.com/yyhuni/lunafox/server/internal/pkg"
	"go.uber.org/zap"
)

const RegistrationTokenNeverAttributedRetention = 24 * time.Hour

// RegistrationTokenCleanupJob keeps cleanup off token creation and rechecks
// eligibility in the repository's conditional delete.
type RegistrationTokenCleanupJob struct {
	store    RegistrationTokenStore
	clock    Clock
	interval time.Duration
}

func NewRegistrationTokenCleanupJob(store RegistrationTokenStore, clock Clock, interval time.Duration) *RegistrationTokenCleanupJob {
	if store == nil {
		panic("registration token store is required")
	}
	if clock == nil {
		panic("clock is required")
	}
	if interval <= 0 {
		panic("registration token cleanup interval must be positive")
	}
	return &RegistrationTokenCleanupJob{store: store, clock: clock, interval: interval}
}

func (job *RegistrationTokenCleanupJob) Start(ctx context.Context) {
	if job == nil {
		return
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

func (job *RegistrationTokenCleanupJob) runOnce(ctx context.Context) {
	cutoff := job.clock.NowUTC().Add(-RegistrationTokenNeverAttributedRetention)
	if err := job.store.DeleteNeverAttributedBefore(ctx, cutoff); err != nil && ctx.Err() == nil {
		pkg.Warn("Failed to clean never-attributed registration tokens", zap.Error(err))
	}
}
