package repository

import (
	"context"
	"time"

	agentdomain "github.com/yyhuni/lunafox/server/internal/modules/agent/domain"
	"github.com/yyhuni/lunafox/server/internal/modules/agent/repository/persistence"
	"github.com/yyhuni/lunafox/server/internal/pkg/scope"
)

// FindByID finds an agent by ID.
func (r *agentRepository) GetByID(ctx context.Context, id int) (*agentdomain.Agent, error) {
	var agent model.Agent
	err := r.db.WithContext(ctx).First(&agent, id).Error
	if err != nil {
		return nil, err
	}
	return modelAgentToDomain(&agent), nil
}

// FindByAPIKey finds an agent by API key.
func (r *agentRepository) FindByAPIKey(ctx context.Context, apiKey string) (*agentdomain.Agent, error) {
	var agent model.Agent
	err := r.db.WithContext(ctx).Where("api_key = ?", apiKey).First(&agent).Error
	if err != nil {
		return nil, err
	}
	return modelAgentToDomain(&agent), nil
}

// List returns agents with pagination and optional status filter.
func (r *agentRepository) List(ctx context.Context, page, pageSize int, status string) ([]*agentdomain.Agent, int64, error) {
	var items []model.Agent
	var total int64

	query := r.db.WithContext(ctx).Model(&model.Agent{})
	if status != "" {
		query = query.Where("status = ?", status)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	err := query.Scopes(
		scope.WithPagination(page, pageSize),
		scope.OrderBy("id", true),
	).Find(&items).Error
	if err != nil {
		return nil, 0, err
	}

	agents := make([]*agentdomain.Agent, 0, len(items))
	for index := range items {
		agents = append(agents, modelAgentToDomain(&items[index]))
	}
	return agents, total, nil
}

// FindStaleOnline returns online agents whose heartbeat is older than the cutoff.
func (r *agentRepository) FindStaleOnline(ctx context.Context, before time.Time) ([]*agentdomain.Agent, error) {
	var items []model.Agent
	err := r.db.WithContext(ctx).
		Where("status = ? AND last_heartbeat IS NOT NULL AND last_heartbeat < ?", "online", before).
		Find(&items).Error
	if err != nil {
		return nil, err
	}

	agents := make([]*agentdomain.Agent, 0, len(items))
	for index := range items {
		agents = append(agents, modelAgentToDomain(&items[index]))
	}
	return agents, nil
}
