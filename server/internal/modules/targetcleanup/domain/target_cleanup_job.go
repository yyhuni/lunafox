package domain

import "time"

type CleanupJobStatus string

const (
	CleanupJobPending   CleanupJobStatus = "pending"
	CleanupJobCompleted CleanupJobStatus = "completed"
)

// CleanupJob is the durable, internal obligation for one deleted Target. It
// intentionally has no phase, cursor, owner, or lease: every run reconciles
// the database state again from its first step.
type CleanupJob struct {
	ID          int
	TargetID    int
	Status      CleanupJobStatus
	RetryCount  int
	NextRetryAt time.Time
	LastError   string
	CompletedAt *time.Time
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type CleanupBacklog struct {
	UnfinishedCount int64
	OldestCreatedAt *time.Time
}

type AssetResource string

const (
	AssetResourceSubdomain       AssetResource = "subdomain"
	AssetResourceHostPortMapping AssetResource = "host_port_mapping"
	AssetResourceWebsite         AssetResource = "website"
	AssetResourceEndpoint        AssetResource = "endpoint"
	AssetResourceDirectory       AssetResource = "directory"
	AssetResourceScreenshot      AssetResource = "screenshot"
	AssetResourceVulnerability   AssetResource = "vulnerability"
)

var orderedAssetResources = []AssetResource{
	AssetResourceSubdomain,
	AssetResourceHostPortMapping,
	AssetResourceWebsite,
	AssetResourceEndpoint,
	AssetResourceDirectory,
	AssetResourceScreenshot,
	AssetResourceVulnerability,
}

func OrderedAssetResources() []AssetResource {
	return append([]AssetResource(nil), orderedAssetResources...)
}

type CleanupCounts struct {
	Schedules              int
	OrganizationRelations  int64
	TargetPolicies         int64
	Scans                  int
	Tasks                  int
	AssetRows              map[AssetResource]int64
	AgentNotifications     int
	AgentNotificationFails int
}

func (counts *CleanupCounts) AddAssetRows(resource AssetResource, rows int64) {
	if rows <= 0 {
		return
	}
	if counts.AssetRows == nil {
		counts.AssetRows = make(map[AssetResource]int64)
	}
	counts.AssetRows[resource] += rows
}

func (counts CleanupCounts) TotalAssetRows() int64 {
	var total int64
	for _, rows := range counts.AssetRows {
		total += rows
	}
	return total
}
