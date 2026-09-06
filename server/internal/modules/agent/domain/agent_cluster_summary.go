package domain

import "time"

const AgentExecutionFreshness = 15 * time.Second

// AgentClusterAggregate contains complete-fleet facts before policy mapping.
type AgentClusterAggregate struct {
	TotalNodes         int64
	HealthyCount       int64
	WarningCount       int64
	OfflineCount       int64
	UnknownCount       int64
	StaleAgentCount    int64
	ConfiguredSlots    int64
	OccupiedSlots      int64
	AvailableSlots     int64
	UnavailableSlots   int64
	OvercommittedSlots int64
	PositionedCount    int64
	UnpositionedCount  int64
}
