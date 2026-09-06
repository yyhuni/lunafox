package repository

import (
	agentdomain "github.com/yyhuni/lunafox/server/internal/modules/agent/domain"
	"gorm.io/gorm"
)

type agentRepository struct {
	db                      *gorm.DB
	offlineNotificationSink AgentOfflineNotificationSink
}

// NewAgentRepository creates a new agent repository.
func NewAgentRepository(db *gorm.DB, notificationSinks ...AgentOfflineNotificationSink) agentdomain.AgentOperationalRepository {
	if len(notificationSinks) > 1 {
		panic("only one agent offline notification sink is supported")
	}
	repository := &agentRepository{db: db}
	if len(notificationSinks) == 1 {
		repository.offlineNotificationSink = notificationSinks[0]
	}
	return repository
}
