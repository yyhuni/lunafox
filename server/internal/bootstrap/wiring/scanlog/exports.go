package scanlogwiring

import (
	scanapp "github.com/yyhuni/lunafox/server/internal/modules/scan/application"
	scanrepo "github.com/yyhuni/lunafox/server/internal/modules/scan/repository"
)

func NewScanLogQueryStoreAdapter(repo *scanrepo.ScanLogRepository) scanapp.ScanLogQueryStore {
	return newScanLogQueryStoreAdapter(repo)
}

func NewScanLogCommandStoreAdapter(repo *scanrepo.ScanLogRepository) scanapp.ScanLogCommandStore {
	return newScanLogCommandStoreAdapter(repo)
}

func NewScanLogScanLookupAdapter(repo *scanrepo.ScanRepository) scanapp.ScanLogScanLookup {
	return newScanLogScanLookupAdapter(repo)
}

func NewScanLogApplicationService(
	queryStore scanapp.ScanLogQueryStore,
	commandStore scanapp.ScanLogCommandStore,
	scanLookup scanapp.ScanLogScanLookup,
) scanapp.ScanLogApplicationService {
	return scanapp.NewScanLogService(queryStore, commandStore, scanLookup)
}
