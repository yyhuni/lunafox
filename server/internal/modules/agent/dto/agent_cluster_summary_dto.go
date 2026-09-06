package dto

import "time"

type AgentClusterExecutionCapacityResponse struct {
	ConfiguredSlots    int64 `json:"configuredSlots"`
	OccupiedSlots      int64 `json:"occupiedSlots"`
	AvailableSlots     int64 `json:"availableSlots"`
	UnavailableSlots   int64 `json:"unavailableSlots"`
	OvercommittedSlots int64 `json:"overcommittedSlots"`
}

type AgentClusterLocationCoverageResponse struct {
	PositionedCount   int64 `json:"positionedCount"`
	UnpositionedCount int64 `json:"unpositionedCount"`
}

type AgentClusterSummaryResponse struct {
	Name                      string                                `json:"name"`
	GeneratedAt               time.Time                             `json:"generatedAt"`
	ExecutionFreshnessSeconds int                                   `json:"executionFreshnessSeconds"`
	TotalNodes                int64                                 `json:"totalNodes"`
	HealthyCount              int64                                 `json:"healthyCount"`
	WarningCount              int64                                 `json:"warningCount"`
	OfflineCount              int64                                 `json:"offlineCount"`
	UnknownCount              int64                                 `json:"unknownCount"`
	StaleAgentCount           int64                                 `json:"staleAgentCount"`
	ExecutionCapacity         AgentClusterExecutionCapacityResponse `json:"executionCapacity"`
	ClusterState              string                                `json:"clusterState"`
	ReasonCodes               []string                              `json:"reasonCodes"`
	LocationCoverage          AgentClusterLocationCoverageResponse  `json:"locationCoverage"`
}
