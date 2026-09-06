package repository

import (
	agentdomain "github.com/yyhuni/lunafox/server/internal/modules/agent/domain"
	"gorm.io/gorm"
)

type agentLocationRepository struct {
	db *gorm.DB
}

// NewAgentLocationRepository creates the persistence boundary for Agent GeoIP snapshots.
func NewAgentLocationRepository(db *gorm.DB) agentdomain.AgentLocationRepository {
	return &agentLocationRepository{db: db}
}
