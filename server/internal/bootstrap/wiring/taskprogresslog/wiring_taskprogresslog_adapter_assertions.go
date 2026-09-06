package taskprogresslogwiring

import scanapp "github.com/yyhuni/lunafox/server/internal/modules/scan/application"

var _ scanapp.TaskProgressLogQueryStore = (*taskProgressLogQueryStoreAdapter)(nil)
var _ scanapp.TaskProgressLogCommandStore = (*taskProgressLogCommandStoreAdapter)(nil)
var _ scanapp.TaskProgressLogScanLookup = (*taskProgressLogScanLookupAdapter)(nil)
