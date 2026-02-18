package repository

import (
	agentdomain "github.com/yyhuni/lunafox/server/internal/modules/agent/domain"
	"gorm.io/gorm"
)

type agentRepository struct {
	db *gorm.DB
}

// NewAgentRepository creates a new agent repository.
func NewAgentRepository(db *gorm.DB) agentdomain.AgentRepository {
	return &agentRepository{db: db}
}
