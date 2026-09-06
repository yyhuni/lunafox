package repository

import (
	"context"
	"database/sql"
	"errors"
	"time"

	agentdomain "github.com/yyhuni/lunafox/server/internal/modules/agent/domain"
)

type agentClusterAggregateRow struct {
	TotalNodes         int64 `gorm:"column:total_nodes"`
	HealthyCount       int64 `gorm:"column:healthy_count"`
	WarningCount       int64 `gorm:"column:warning_count"`
	OfflineCount       int64 `gorm:"column:offline_count"`
	UnknownCount       int64 `gorm:"column:unknown_count"`
	StaleAgentCount    int64 `gorm:"column:stale_agent_count"`
	ConfiguredSlots    int64 `gorm:"column:configured_slots"`
	OccupiedSlots      int64 `gorm:"column:occupied_slots"`
	AvailableSlots     int64 `gorm:"column:available_slots"`
	UnavailableSlots   int64 `gorm:"column:unavailable_slots"`
	OvercommittedSlots int64 `gorm:"column:overcommitted_slots"`
	PositionedCount    int64 `gorm:"column:positioned_count"`
	UnpositionedCount  int64 `gorm:"column:unpositioned_count"`
}

const agentClusterAggregateQuery = `
WITH agent_facts AS (
    SELECT
        CASE
            WHEN agent.status = 'offline' THEN 'offline'
            WHEN agent.status = 'online' AND agent_runtime_status.health_state = 'healthy' THEN 'healthy'
            WHEN agent.status = 'online' AND agent_runtime_status.health_state = 'paused' THEN 'warning'
            ELSE 'unknown'
        END AS node_bucket,
        CASE WHEN COALESCE(agent.max_tasks, 0) > 0 THEN agent.max_tasks ELSE 0 END AS configured_slots,
        COALESCE(agent_runtime_status.task_slots_used, 0) AS task_slots_used,
        CASE
            WHEN agent_runtime_status.last_heartbeat > @fresh_after
             AND agent_runtime_status.last_heartbeat <= @generated_at
            THEN 1 ELSE 0
        END AS execution_fresh,
        CASE
            WHEN agent_location.agent_id IS NOT NULL
             AND agent_location.latitude >= -90 AND agent_location.latitude <= 90
             AND agent_location.longitude >= -180 AND agent_location.longitude <= 180
             AND (agent_location.accuracy_radius_km IS NULL OR agent_location.accuracy_radius_km >= 0)
             AND COALESCE(agent_location.provider_key, '') <> ''
             AND agent_location.resolved_at IS NOT NULL
            THEN 1 ELSE 0
        END AS positioned
    FROM agent
    LEFT JOIN agent_runtime_status ON agent_runtime_status.agent_id = agent.id
    LEFT JOIN agent_location ON agent_location.agent_id = agent.id
)
SELECT
    COUNT(*) AS total_nodes,
    COALESCE(SUM(CASE WHEN node_bucket = 'healthy' THEN 1 ELSE 0 END), 0) AS healthy_count,
    COALESCE(SUM(CASE WHEN node_bucket = 'warning' THEN 1 ELSE 0 END), 0) AS warning_count,
    COALESCE(SUM(CASE WHEN node_bucket = 'offline' THEN 1 ELSE 0 END), 0) AS offline_count,
    COALESCE(SUM(CASE WHEN node_bucket = 'unknown' THEN 1 ELSE 0 END), 0) AS unknown_count,
    COALESCE(SUM(CASE WHEN execution_fresh = 0 THEN 1 ELSE 0 END), 0) AS stale_agent_count,
    COALESCE(SUM(configured_slots), 0) AS configured_slots,
    COALESCE(SUM(CASE
        WHEN node_bucket = 'healthy' AND execution_fresh = 1 THEN
            CASE
                WHEN task_slots_used <= 0 THEN 0
                WHEN task_slots_used >= configured_slots THEN configured_slots
                ELSE task_slots_used
            END
        ELSE 0
    END), 0) AS occupied_slots,
    COALESCE(SUM(CASE
        WHEN node_bucket = 'healthy' AND execution_fresh = 1 THEN
            CASE
                WHEN task_slots_used <= 0 THEN configured_slots
                WHEN task_slots_used >= configured_slots THEN 0
                ELSE configured_slots - task_slots_used
            END
        ELSE 0
    END), 0) AS available_slots,
    COALESCE(SUM(CASE
        WHEN node_bucket = 'healthy' AND execution_fresh = 1 THEN 0
        ELSE configured_slots
    END), 0) AS unavailable_slots,
    COALESCE(SUM(CASE
        WHEN node_bucket = 'healthy' AND execution_fresh = 1 AND task_slots_used > configured_slots
        THEN task_slots_used - configured_slots
        ELSE 0
    END), 0) AS overcommitted_slots,
    COALESCE(SUM(positioned), 0) AS positioned_count,
    COUNT(*) - COALESCE(SUM(positioned), 0) AS unpositioned_count
FROM agent_facts`

// GetClusterAggregate keeps fleet completeness and every freshness predicate inside one database query.
func (repository *agentRepository) GetClusterAggregate(ctx context.Context, generatedAt time.Time) (agentdomain.AgentClusterAggregate, error) {
	if repository == nil || repository.db == nil {
		return agentdomain.AgentClusterAggregate{}, errors.New("Agent cluster summary repository is required")
	}
	if generatedAt.IsZero() {
		return agentdomain.AgentClusterAggregate{}, errors.New("Agent cluster summary generation time is required")
	}
	generatedAt = generatedAt.UTC()
	var row agentClusterAggregateRow
	if err := repository.db.WithContext(ctx).Raw(
		agentClusterAggregateQuery,
		sql.Named("fresh_after", generatedAt.Add(-agentdomain.AgentExecutionFreshness)),
		sql.Named("generated_at", generatedAt),
	).Scan(&row).Error; err != nil {
		return agentdomain.AgentClusterAggregate{}, err
	}
	return agentdomain.AgentClusterAggregate{
		TotalNodes:         row.TotalNodes,
		HealthyCount:       row.HealthyCount,
		WarningCount:       row.WarningCount,
		OfflineCount:       row.OfflineCount,
		UnknownCount:       row.UnknownCount,
		StaleAgentCount:    row.StaleAgentCount,
		ConfiguredSlots:    row.ConfiguredSlots,
		OccupiedSlots:      row.OccupiedSlots,
		AvailableSlots:     row.AvailableSlots,
		UnavailableSlots:   row.UnavailableSlots,
		OvercommittedSlots: row.OvercommittedSlots,
		PositionedCount:    row.PositionedCount,
		UnpositionedCount:  row.UnpositionedCount,
	}, nil
}
