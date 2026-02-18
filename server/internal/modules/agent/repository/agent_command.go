package repository

import (
	"context"
	"time"

	agentdomain "github.com/yyhuni/lunafox/server/internal/modules/agent/domain"
	"github.com/yyhuni/lunafox/server/internal/modules/agent/repository/persistence"
)

// Create creates a new agent.
func (r *agentRepository) Create(ctx context.Context, agent *agentdomain.Agent) error {
	return r.db.WithContext(ctx).Create(domainAgentToModel(agent)).Error
}

// Update updates an agent.
func (r *agentRepository) Update(ctx context.Context, agent *agentdomain.Agent) error {
	return r.db.WithContext(ctx).Save(domainAgentToModel(agent)).Error
}

// UpdateStatus updates agent status.
func (r *agentRepository) UpdateStatus(ctx context.Context, id int, status string) error {
	return r.db.WithContext(ctx).Model(&model.Agent{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"status":     status,
			"updated_at": time.Now().UTC(),
		}).Error
}

// Delete deletes an agent.
func (r *agentRepository) Delete(ctx context.Context, id int) error {
	return r.db.WithContext(ctx).Delete(&model.Agent{}, id).Error
}

// UpdateHeartbeat updates agent last heartbeat time and runtime fields.
func (r *agentRepository) UpdateHeartbeat(ctx context.Context, id int, update agentdomain.AgentHeartbeatUpdate) error {
	now := update.LastHeartbeat
	updates := map[string]interface{}{
		"last_heartbeat": now,
		"status":         "online",
		"updated_at":     now,
	}

	if update.Version != "" {
		updates["version"] = update.Version
	}
	if update.Hostname != "" {
		updates["hostname"] = update.Hostname
	}
	if update.HasHealth {
		updates["health_state"] = update.HealthState
		updates["health_reason"] = update.HealthReason
		updates["health_message"] = update.HealthMessage
		updates["health_since"] = update.HealthSince
	}

	return r.db.WithContext(ctx).Model(&model.Agent{}).
		Where("id = ?", id).
		Updates(updates).Error
}
