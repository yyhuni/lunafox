package scanwiring

import scanapp "github.com/yyhuni/lunafox/server/internal/modules/scan/application"

var _ scanapp.ScanQueryStore = (*scanQueryStoreAdapter)(nil)
var _ scanapp.ScanCommandStore = (*scanCommandStoreAdapter)(nil)
var _ scanapp.MCPScanOperationStore = (*scanCommandStoreAdapter)(nil)
var _ scanapp.ScanCreateTargetLookup = (*scanTargetLookupAdapter)(nil)
var _ scanapp.ScanStopStore = (*scanStopStoreAdapter)(nil)
var _ scanapp.ScanTaskStore = (*scanTaskStoreAdapter)(nil)
var _ scanapp.EngineDiagnosticTerminalTaskStore = (*scanTaskStoreAdapter)(nil)
var _ scanapp.ScanTaskRuntimeScanStore = (*scanTaskRuntimeScanStoreAdapter)(nil)
