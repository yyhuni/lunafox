package domain

import "time"

// QueryTargetRef is the lightweight target projection for scan query read models.
type QueryTargetRef struct {
	ID        int
	Name      string
	Type      string
	CreatedAt time.Time
}

// QueryScan is the read-model projection returned by scan query use-cases.
type QueryScan struct {
	ID                     int
	TargetID               int
	EngineIDs              []int64
	EngineNames            []byte
	YamlConfiguration      string
	ScanMode               string
	Status                 string
	ResultsDir             string
	WorkerID               *int
	ErrorMessage           string
	Progress               int
	CurrentStage           string
	StageProgress          []byte
	CreatedAt              time.Time
	StoppedAt              *time.Time
	CachedSubdomainsCount  int
	CachedWebsitesCount    int
	CachedEndpointsCount   int
	CachedIPsCount         int
	CachedDirectoriesCount int
	CachedScreenshotsCount int
	CachedVulnsTotal       int
	CachedVulnsCritical    int
	CachedVulnsHigh        int
	CachedVulnsMedium      int
	CachedVulnsLow         int
	Target                 *QueryTargetRef
}

// QueryStatistics is the aggregate projection used by scan statistics endpoints.
type QueryStatistics struct {
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
