package dto

import "time"

type AssetStatisticsResponse struct {
	TotalTargets    int64     `json:"totalTargets"`
	TotalSubdomains int64     `json:"totalSubdomains"`
	TotalIPs        int64     `json:"totalIps"`
	TotalEndpoints  int64     `json:"totalEndpoints"`
	TotalWebsites   int64     `json:"totalWebsites"`
	TotalVulns      int64     `json:"totalVulns"`
	TotalAssets     int64     `json:"totalAssets"`
	RunningScans    int64     `json:"runningScans"`
	UpdatedAt       time.Time `json:"updatedAt"`

	ChangeTargets    int64                               `json:"changeTargets"`
	ChangeSubdomains int64                               `json:"changeSubdomains"`
	ChangeIPs        int64                               `json:"changeIps"`
	ChangeEndpoints  int64                               `json:"changeEndpoints"`
	ChangeWebsites   int64                               `json:"changeWebsites"`
	ChangeVulns      int64                               `json:"changeVulns"`
	ChangeAssets     int64                               `json:"changeAssets"`
	VulnsBySeverity  VulnerabilitySeverityCountsResponse `json:"vulnBySeverity"`
}

type VulnerabilitySeverityCountsResponse struct {
	Critical int64 `json:"critical"`
	High     int64 `json:"high"`
	Medium   int64 `json:"medium"`
	Low      int64 `json:"low"`
	Info     int64 `json:"info"`
}

type AssetStatisticsHistoryItemResponse struct {
	Date            string `json:"date"`
	TotalTargets    int64  `json:"totalTargets"`
	TotalSubdomains int64  `json:"totalSubdomains"`
	TotalIPs        int64  `json:"totalIps"`
	TotalEndpoints  int64  `json:"totalEndpoints"`
	TotalWebsites   int64  `json:"totalWebsites"`
	TotalVulns      int64  `json:"totalVulns"`
	TotalAssets     int64  `json:"totalAssets"`
}
