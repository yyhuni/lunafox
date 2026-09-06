package taskprogresslogwiring

import (
	scanapp "github.com/yyhuni/lunafox/server/internal/modules/scan/application"
	scanrepo "github.com/yyhuni/lunafox/server/internal/modules/scan/repository"
)

type taskProgressLogQueryStoreAdapter struct {
	repo *scanrepo.TaskProgressLogRepository
}

func newTaskProgressLogQueryStoreAdapter(repo *scanrepo.TaskProgressLogRepository) *taskProgressLogQueryStoreAdapter {
	return &taskProgressLogQueryStoreAdapter{repo: repo}
}

func (adapter *taskProgressLogQueryStoreAdapter) FindByScanIDWithCursor(scanID int, afterID int64, limit int) ([]scanapp.TaskProgressLogEntry, error) {
	return adapter.repo.FindByScanIDWithCursor(scanID, afterID, limit)
}
