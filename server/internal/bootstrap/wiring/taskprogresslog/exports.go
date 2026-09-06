package taskprogresslogwiring

import (
	scanapp "github.com/yyhuni/lunafox/server/internal/modules/scan/application"
	scanrepo "github.com/yyhuni/lunafox/server/internal/modules/scan/repository"
)

func NewTaskProgressLogQueryStoreAdapter(repo *scanrepo.TaskProgressLogRepository) scanapp.TaskProgressLogQueryStore {
	return newTaskProgressLogQueryStoreAdapter(repo)
}

func NewTaskProgressLogCommandStoreAdapter(repo *scanrepo.TaskProgressLogRepository) scanapp.TaskProgressLogCommandStore {
	return newTaskProgressLogCommandStoreAdapter(repo)
}

func NewTaskProgressLogScanLookupAdapter(repo *scanrepo.ScanRepository) scanapp.TaskProgressLogScanLookup {
	return newTaskProgressLogScanLookupAdapter(repo)
}

func NewTaskProgressLogApplicationService(
	queryStore scanapp.TaskProgressLogQueryStore,
	commandStore scanapp.TaskProgressLogCommandStore,
	scanLookup scanapp.TaskProgressLogScanLookup,
) scanapp.TaskProgressLogApplicationService {
	return scanapp.NewTaskProgressLogApplicationService(queryStore, commandStore, scanLookup)
}
