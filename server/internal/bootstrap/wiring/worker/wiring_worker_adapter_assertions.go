package workerwiring

import catalogapp "github.com/yyhuni/lunafox/server/internal/modules/catalog/application"

var _ catalogapp.WorkerProviderConfigScanGuard = (*workerScanGuardAdapter)(nil)
var _ catalogapp.WorkerProviderConfigSettingsStore = (*workerSettingsStoreAdapter)(nil)
