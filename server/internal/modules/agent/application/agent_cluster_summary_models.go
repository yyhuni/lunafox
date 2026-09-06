package application

import "time"

type AgentClusterNodeCounts struct {
	Total   int64
	Healthy int64
	Warning int64
	Offline int64
	Unknown int64
	Stale   int64
}

type AgentClusterExecutionCapacity struct {
	Configured    int64
	Occupied      int64
	Available     int64
	Unavailable   int64
	Overcommitted int64
}

type AgentClusterLocationCoverage struct {
	Positioned   int64
	Unpositioned int64
}

type AgentClusterSummary struct {
	GeneratedAt               time.Time
	ExecutionFreshnessSeconds int
	Nodes                     AgentClusterNodeCounts
	ExecutionCapacity         AgentClusterExecutionCapacity
	State                     AgentClusterState
	ReasonCodes               []AgentClusterReasonCode
	LocationCoverage          AgentClusterLocationCoverage
}
