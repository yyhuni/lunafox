package repository

import (
	"context"
	"errors"

	agentdomain "github.com/yyhuni/lunafox/server/internal/modules/agent/domain"
	model "github.com/yyhuni/lunafox/server/internal/modules/agent/repository/persistence"
	"gorm.io/gorm"
)

func (repository *agentLocationRepository) GetLocation(ctx context.Context, agentID int) (*agentdomain.AgentLocationSnapshot, error) {
	if repository == nil || repository.db == nil {
		return nil, errors.New("Agent location repository is required")
	}
	if agentID <= 0 {
		return nil, errors.New("Agent identity is required")
	}
	var location model.AgentLocation
	err := repository.db.WithContext(ctx).First(&location, "agent_id = ?", agentID).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return modelAgentLocationToDomain(&location), nil
}
