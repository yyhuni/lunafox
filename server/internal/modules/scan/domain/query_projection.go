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
	ScanWorkflowID         string
	PlannedEngineIDs       []string
	Configuration          map[string]any
	InputSource            InputSource
	TriggerType            ScanTriggerType
	Status                 string
	ResultsDir             string
	AgentID                *int
	AgentName              string
	AgentStatus            string
	AgentHealthState       string
	AgentDeleted           bool
	AssignmentMode         string
	ErrorMessage           string
	Failure                *FailureDetail
	Progress               int
	CurrentStage           string
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
	RuntimeTasks           []QueryRuntimeTask
}

// QueryRuntimeTask is the scan detail projection for persisted scan task rows.
type QueryRuntimeTask struct {
	ID            int
	StepID        string
	StageID       string
	EngineID      string
	Status        string
	SkipReason    string
	Order         int
	StartedAt     *time.Time
	CompletedAt   *time.Time
	Duration      *float64
	Error         string
	FailureKind   string
	FailureDetail string
	Diagnostics   *EngineExecutionDiagnostics
}

// QueryStatistics is the aggregate projection used by scan statistics endpoints.
type QueryStatistics struct {
	Total           int64
	Pending         int64
	Running         int64
	Completed       int64
	Failed          int64
	Cancelled       int64
	TotalVulns      int64
	TotalSubdomains int64
	TotalEndpoints  int64
	TotalWebsites   int64
	TotalAssets     int64
}
