package repository

import (
	"context"
	"errors"
	"time"

	agentdomain "github.com/yyhuni/lunafox/server/internal/modules/agent/domain"
)

type agentLocationMapRow struct {
	AgentID           int       `gorm:"column:agent_id"`
	DisplayName       string    `gorm:"column:display_name"`
	Status            string    `gorm:"column:status"`
	HealthState       string    `gorm:"column:health_state"`
	RuntimeAgentID    *int      `gorm:"column:runtime_agent_id"`
	TaskSlotsUsed     *int      `gorm:"column:task_slots_used"`
	Latitude          float64   `gorm:"column:latitude"`
	Longitude         float64   `gorm:"column:longitude"`
	AccuracyRadiusKM  *float64  `gorm:"column:accuracy_radius_km"`
	SourceObservedIP  string    `gorm:"column:source_observed_ip"`
	ProviderKey       string    `gorm:"column:provider_key"`
	ResolvedAt        time.Time `gorm:"column:resolved_at"`
	ForcedExpired     bool      `gorm:"column:forced_expired"`
	LocationUpdatedAt time.Time `gorm:"column:location_updated_at"`
}

// ListLocationMapAgents returns every valid positioned Agent without collection pagination.
func (repository *agentRepository) ListLocationMapAgents(ctx context.Context) ([]agentdomain.AgentLocationMapRecord, error) {
	if repository == nil || repository.db == nil {
		return nil, errors.New("Agent location map repository is required")
	}
	var rows []agentLocationMapRow
	err := repository.db.WithContext(ctx).
		Table("agent").
		Select(`
agent.id AS agent_id,
agent.display_name,
agent.status,
agent_runtime_status.health_state,
agent_runtime_status.agent_id AS runtime_agent_id,
agent_runtime_status.task_slots_used,
agent_location.latitude,
agent_location.longitude,
agent_location.accuracy_radius_km,
agent_location.source_observed_ip,
agent_location.provider_key,
agent_location.resolved_at,
agent_location.forced_expired,
agent_location.updated_at AS location_updated_at`).
		Joins("JOIN agent_location ON agent_location.agent_id = agent.id").
		Joins("LEFT JOIN agent_runtime_status ON agent_runtime_status.agent_id = agent.id").
		Where("agent_location.latitude >= ? AND agent_location.latitude <= ?", -90, 90).
		Where("agent_location.longitude >= ? AND agent_location.longitude <= ?", -180, 180).
		Where("agent_location.accuracy_radius_km IS NULL OR agent_location.accuracy_radius_km >= 0").
		Where("agent_location.source_observed_ip <> ''").
		Where("agent_location.provider_key <> ''").
		Where("agent_location.resolved_at IS NOT NULL").
		Order("agent.id ASC").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	records := make([]agentdomain.AgentLocationMapRecord, 0, len(rows))
	for _, row := range rows {
		var taskSlotsUsed *int
		if row.RuntimeAgentID != nil && row.TaskSlotsUsed != nil {
			value := *row.TaskSlotsUsed
			taskSlotsUsed = &value
		}
		records = append(records, agentdomain.AgentLocationMapRecord{
			AgentID:       row.AgentID,
			DisplayName:   row.DisplayName,
			Status:        row.Status,
			HealthState:   row.HealthState,
			TaskSlotsUsed: taskSlotsUsed,
			Location: agentdomain.AgentLocationSnapshot{
				AgentID:          row.AgentID,
				Latitude:         row.Latitude,
				Longitude:        row.Longitude,
				AccuracyRadiusKM: copyFloat64Ptr(row.AccuracyRadiusKM),
				SourceObservedIP: row.SourceObservedIP,
				ProviderKey:      row.ProviderKey,
				ResolvedAt:       row.ResolvedAt.UTC(),
				ForcedExpired:    row.ForcedExpired,
				UpdatedAt:        row.LocationUpdatedAt.UTC(),
			},
		})
	}
	return records, nil
}
