package domain

import "time"

// AssetStatistics is the current aggregate projection used by the Overview report view.
type AssetStatistics struct {
	TotalTargets    int64
	TotalSubdomains int64
	TotalIPs        int64
	TotalEndpoints  int64
	TotalWebsites   int64
	TotalVulns      int64
	TotalAssets     int64
	RunningScans    int64
	UpdatedAt       time.Time

	ChangeTargets    int64
	ChangeSubdomains int64
	ChangeIPs        int64
	ChangeEndpoints  int64
	ChangeWebsites   int64
	ChangeVulns      int64
	ChangeAssets     int64

	VulnsBySeverity VulnerabilitySeverityCounts
}

// VulnerabilitySeverityCounts represents the fixed severity buckets displayed by Overview.
type VulnerabilitySeverityCounts struct {
	Critical int64
	High     int64
	Medium   int64
	Low      int64
	Info     int64
}

// AssetStatisticsHistoryItem is a reconstructed UTC end-of-day current-state projection.
type AssetStatisticsHistoryItem struct {
	Date            time.Time
	TotalTargets    int64
	TotalSubdomains int64
	TotalIPs        int64
	TotalEndpoints  int64
	TotalWebsites   int64
	TotalVulns      int64
	TotalAssets     int64
}
