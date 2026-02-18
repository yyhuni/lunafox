package workerwiring

import (
	catalogapp "github.com/yyhuni/lunafox/server/internal/modules/catalog/application"
	catalogrepo "github.com/yyhuni/lunafox/server/internal/modules/catalog/repository"
	scanrepo "github.com/yyhuni/lunafox/server/internal/modules/scan/repository"
)

func NewWorkerProviderConfigScanGuardAdapter(scanRepo *scanrepo.ScanRepository) catalogapp.WorkerProviderConfigScanGuard {
	return newWorkerScanGuardAdapter(scanRepo)
}

func NewWorkerProviderConfigSettingsStoreAdapter(settingsRepo *catalogrepo.SubfinderProviderSettingsRepository) catalogapp.WorkerProviderConfigSettingsStore {
	return newWorkerSettingsStoreAdapter(settingsRepo)
}

func NewWorkerProviderConfigApplicationService(scanGuard catalogapp.WorkerProviderConfigScanGuard, settingsStore catalogapp.WorkerProviderConfigSettingsStore) *catalogapp.WorkerProviderConfigService {
	return catalogapp.NewWorkerProviderConfigService(scanGuard, settingsStore)
}
