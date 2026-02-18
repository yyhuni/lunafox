package scanlogwiring

import scanapp "github.com/yyhuni/lunafox/server/internal/modules/scan/application"

var _ scanapp.ScanLogQueryStore = (*scanLogQueryStoreAdapter)(nil)
var _ scanapp.ScanLogCommandStore = (*scanLogCommandStoreAdapter)(nil)
var _ scanapp.ScanLogScanLookup = (*scanLogScanLookupAdapter)(nil)
