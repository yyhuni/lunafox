package repository

import (
	"context"
	"time"

	agentdomain "github.com/yyhuni/lunafox/server/internal/modules/agent/domain"
	model "github.com/yyhuni/lunafox/server/internal/modules/agent/repository/persistence"
	"gorm.io/gorm"
)

// FindValid returns a non-expired token by value.
func (r *registrationTokenRepository) FindValid(ctx context.Context, token string, now time.Time) (*agentdomain.RegistrationToken, error) {
	var item model.RegistrationToken
	err := r.db.WithContext(ctx).
		Where("token = ? AND expires_at > ?", token, now).
		First(&item).Error
	if err != nil {
		return nil, err
	}
	return modelTokenToDomain(&item), nil
}

// GetResourceByID returns the non-secret token resource and every currently
// attributed Agent, independent of token expiry.
func (r *registrationTokenRepository) GetResourceByID(ctx context.Context, id int) (*agentdomain.RegistrationTokenResource, error) {
	var item model.RegistrationToken
	err := r.db.WithContext(ctx).
		Preload("Agents", func(db *gorm.DB) *gorm.DB {
			return db.Order("agent.created_at ASC").Order("agent.id ASC")
		}).
		Preload("Agents.RuntimeStatus").
		First(&item, id).Error
	if err != nil {
		return nil, err
	}
	agents := make([]*agentdomain.Agent, 0, len(item.Agents))
	for index := range item.Agents {
		agents = append(agents, modelAgentToDomain(&item.Agents[index]))
	}
	return &agentdomain.RegistrationTokenResource{
		ID:        item.ID,
		ExpiresAt: item.ExpiresAt.UTC(),
		Agents:    agents,
	}, nil
}
