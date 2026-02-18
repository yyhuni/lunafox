package dto

import "time"

type ScanListQuery struct {
	PaginationQuery
	TargetID int    `form:"target" binding:"omitempty"`
	Status   string `form:"status" binding:"omitempty"`
	Search   string `form:"search" binding:"omitempty"`
}

type ScanResponse struct {
	ID           int              `json:"id"`
	TargetID     int              `json:"targetId"`
	EngineIDs    []int64          `json:"engineIds"`
	EngineNames  []string         `json:"engineNames"`
	ScanMode     string           `json:"scanMode"`
	Status       string           `json:"status"`
	Progress     int              `json:"progress"`
	CurrentStage string           `json:"currentStage"`
	ErrorMessage string           `json:"errorMessage,omitempty"`
	CreatedAt    time.Time        `json:"createdAt"`
	StoppedAt    *time.Time       `json:"stoppedAt,omitempty"`
	Target       *TargetBrief     `json:"target,omitempty"`
	CachedStats  *ScanCachedStats `json:"cachedStats,omitempty"`
}

type TargetBrief struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
	Type string `json:"type"`
}

type ScanCachedStats struct {
	SubdomainsCount  int `json:"subdomainsCount"`
	WebsitesCount    int `json:"websitesCount"`
	EndpointsCount   int `json:"endpointsCount"`
	IPsCount         int `json:"ipsCount"`
	DirectoriesCount int `json:"directoriesCount"`
	ScreenshotsCount int `json:"screenshotsCount"`
	VulnsTotal       int `json:"vulnsTotal"`
	VulnsCritical    int `json:"vulnsCritical"`
	VulnsHigh        int `json:"vulnsHigh"`
	VulnsMedium      int `json:"vulnsMedium"`
	VulnsLow         int `json:"vulnsLow"`
}

type ScanDetailResponse struct {
	ScanResponse
	YamlConfiguration string                 `json:"yamlConfiguration,omitempty"`
	ResultsDir        string                 `json:"resultsDir,omitempty"`
	WorkerID          *int                   `json:"workerId,omitempty"`
	StageProgress     map[string]interface{} `json:"stageProgress,omitempty"`
}

type InitiateScanRequest struct {
	OrganizationID *int     `json:"organizationId" binding:"omitempty"`
	TargetID       *int     `json:"targetId" binding:"omitempty"`
	EngineIDs      []int    `json:"engineIds" binding:"required,min=1"`
	EngineNames    []string `json:"engineNames" binding:"required,min=1"`
	Configuration  string   `json:"configuration" binding:"required"`
}

type QuickScanRequest struct {
	Targets       []QuickScanTarget `json:"targets" binding:"required,min=1"`
	EngineIDs     []int             `json:"engineIds"`
	EngineNames   []string          `json:"engineNames"`
	Configuration string            `json:"configuration" binding:"required"`
}

type CreateScanRequest struct {
	Mode string `json:"mode" binding:"omitempty,oneof=normal quick"`

	TargetID int      `json:"targetId" binding:"omitempty"`
	Targets  []string `json:"targets" binding:"omitempty"`

	EngineIDs     []int    `json:"engineIds" binding:"omitempty"`
	EngineNames   []string `json:"engineNames" binding:"required,min=1,max=1"`
	Configuration string   `json:"configuration" binding:"omitempty"`
}

type QuickScanTarget struct {
	Name string `json:"name" binding:"required"`
}

type QuickScanResponse struct {
	Count       int            `json:"count"`
	TargetStats map[string]int `json:"targetStats"`
	AssetStats  map[string]int `json:"assetStats"`
	Errors      []string       `json:"errors,omitempty"`
	Scans       []ScanResponse `json:"scans"`
}

type StopScanResponse struct {
	RevokedTaskCount int `json:"revokedTaskCount"`
}

type ScanStatisticsResponse struct {
	Total           int64 `json:"total"`
	Running         int64 `json:"running"`
	Completed       int64 `json:"completed"`
	Failed          int64 `json:"failed"`
	TotalVulns      int64 `json:"totalVulns"`
	TotalSubdomains int64 `json:"totalSubdomains"`
	TotalEndpoints  int64 `json:"totalEndpoints"`
	TotalWebsites   int64 `json:"totalWebsites"`
	TotalAssets     int64 `json:"totalAssets"`
}

type BulkDeleteRequest struct {
	IDs []int `json:"ids" binding:"required,min=1"`
}
