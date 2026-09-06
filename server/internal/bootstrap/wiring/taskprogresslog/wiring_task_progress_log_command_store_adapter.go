package taskprogresslogwiring

import (
	"context"

	scanapp "github.com/yyhuni/lunafox/server/internal/modules/scan/application"
	scanrepo "github.com/yyhuni/lunafox/server/internal/modules/scan/repository"
)

type taskProgressLogCommandStoreAdapter struct {
	repo *scanrepo.TaskProgressLogRepository
}

func newTaskProgressLogCommandStoreAdapter(repo *scanrepo.TaskProgressLogRepository) *taskProgressLogCommandStoreAdapter {
	return &taskProgressLogCommandStoreAdapter{repo: repo}
}

func (adapter *taskProgressLogCommandStoreAdapter) BatchCreateTaskProgressLogs(ctx context.Context, logs []scanapp.TaskProgressLogEntry) (int, int, error) {
	return adapter.repo.BatchCreateTaskProgressLogs(ctx, logs)
}
