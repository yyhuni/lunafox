package scanwiring

import (
	catalogrepo "github.com/yyhuni/lunafox/server/internal/modules/catalog/repository"
	scanapp "github.com/yyhuni/lunafox/server/internal/modules/scan/application"
	scandomain "github.com/yyhuni/lunafox/server/internal/modules/scan/domain"
	scanrepo "github.com/yyhuni/lunafox/server/internal/modules/scan/repository"
)

func NewScanQueryStoreAdapter(repo *scanrepo.ScanRepository) scanapp.ScanQueryStore {
	return newScanQueryStoreAdapter(repo)
}

func NewScanCommandStoreAdapter(repo *scanrepo.ScanRepository) scanapp.ScanCommandStore {
	return newScanCommandStoreAdapter(repo)
}

func NewScanDomainRepositoryAdapter(repo *scanrepo.ScanRepository) scandomain.ScanRepository {
	return newScanDomainRepositoryAdapter(repo)
}

func NewScanTaskCancellerAdapter(repo scanrepo.ScanTaskRepository) scanapp.ScanTaskCanceller {
	return newScanTaskCancellerAdapter(repo)
}

func NewScanTargetLookupAdapter(repo *catalogrepo.TargetRepository) scanapp.ScanCreateTargetLookup {
	return newScanTargetLookupAdapter(repo)
}

func NewScanTaskStoreAdapter(repo scanrepo.ScanTaskRepository) scanapp.TaskStore {
	return newScanTaskStoreAdapter(repo)
}

func NewScanTaskRuntimeStoreAdapter(repo *scanrepo.ScanRepository) scanapp.TaskRuntimeScanStore {
	return newScanTaskRuntimeStoreAdapter(repo)
}

func NewScanApplicationService(
	queryStore scanapp.ScanQueryStore,
	commandStore scanapp.ScanCommandStore,
	domainRepository scandomain.ScanRepository,
	taskCanceller scanapp.ScanTaskCanceller,
	notifier scanapp.TaskCancelNotifier,
	targetLookup scanapp.ScanCreateTargetLookup,
) *scanapp.ScanFacade {
	return scanapp.NewScanFacade(queryStore, commandStore, domainRepository, taskCanceller, notifier, targetLookup)
}

func NewScanTaskApplicationService(
	taskStore scanapp.TaskStore,
	runtimeStore scanapp.TaskRuntimeScanStore,
) *scanapp.ScanTaskFacade {
	return scanapp.NewScanTaskFacade(taskStore, runtimeStore)
}
