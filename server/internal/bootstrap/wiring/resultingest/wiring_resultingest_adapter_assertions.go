package resultingestwiring

import resultingestapp "github.com/yyhuni/lunafox/server/internal/modules/resultingest/application"

var _ resultingestapp.ScanResultSummaryUpdater = (*resultIngestScanResultSummaryUpdaterAdapter)(nil)
var _ resultingestapp.ResultMaterializationCoordinator = (*resultIngestMaterializationCoordinator)(nil)
