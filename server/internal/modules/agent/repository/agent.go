package repository

import (
	agentdomain "github.com/yyhuni/lunafox/server/internal/modules/agent/domain"
	"gorm.io/gorm"
)

type agentRepository struct {
	db                      *gorm.DB
	licenses                *AgentLicenseRepository
	offlineNotificationSink AgentOfflineNotificationSink
}

// NewAgentRepository creates a new agent repository.
func NewAgentRepository(db *gorm.DB, notificationSinks ...AgentOfflineNotificationSink) agentdomain.AgentOperationalRepository {
	if len(notificationSinks) > 1 {
		panic("only one agent offline notification sink is supported")
	}
	repository := &agentRepository{db: db, licenses: NewAgentLicenseRepository(db)}
	if len(notificationSinks) == 1 {
		repository.offlineNotificationSink = notificationSinks[0]
	}
	return repository
}

// NewAgentRepositoryWithLicenseVerifier injects an explicit trust root for isolated tests.
func NewAgentRepositoryWithLicenseVerifier(db *gorm.DB, verifier AgentLicenseVerifier, notificationSinks ...AgentOfflineNotificationSink) (agentdomain.AgentOperationalRepository, error) {
	licenses, err := NewAgentLicenseRepositoryWithVerifier(db, verifier)
	if err != nil {
		return nil, err
	}
	repository := NewAgentRepository(db, notificationSinks...).(*agentRepository)
	repository.licenses = licenses
	return repository, nil
}
