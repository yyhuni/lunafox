package scanwiring

import scanapp "github.com/yyhuni/lunafox/server/internal/modules/scan/application"

var _ scanapp.ScanQueryStore = (*scanQueryStoreAdapter)(nil)
var _ scanapp.ScanCommandStore = (*scanCommandStoreAdapter)(nil)
var _ scanapp.ScanCreateTargetLookup = (*scanTargetLookupAdapter)(nil)
var _ scanapp.ScanTaskCanceller = (*scanTaskCancellerAdapter)(nil)
var _ scanapp.TaskStore = (*scanTaskStoreAdapter)(nil)
var _ scanapp.TaskRuntimeScanStore = (*scanTaskRuntimeStoreAdapter)(nil)
