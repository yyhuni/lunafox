package application

import (
	scandomain "github.com/yyhuni/lunafox/server/internal/modules/scan/domain"
)

type QueryTargetRef = scandomain.QueryTargetRef

type QueryScan = scandomain.QueryScan

type QueryStatistics = scandomain.QueryStatistics

type ScanStatistics struct {
	Total           int64
	Running         int64
	Completed       int64
	Failed          int64
	TotalVulns      int64
	TotalSubdomains int64
	TotalEndpoints  int64
	TotalWebsites   int64
	TotalAssets     int64
}
