package targetcleanupwiring

import (
	"context"

	scheduledscanrepo "github.com/yyhuni/lunafox/server/internal/modules/scheduledscan/repository"
)

type targetCleanupScheduleCleanerAdapter struct {
	repo *scheduledscanrepo.ScheduledScanRepository
}

func newTargetCleanupScheduleCleanerAdapter(repo *scheduledscanrepo.ScheduledScanRepository) *targetCleanupScheduleCleanerAdapter {
	return &targetCleanupScheduleCleanerAdapter{repo: repo}
}

func (adapter *targetCleanupScheduleCleanerAdapter) DeleteTargetScopedSchedules(ctx context.Context, targetID int) (int, error) {
	return adapter.repo.DeleteTargetScopedForCleanup(ctx, targetID)
}
