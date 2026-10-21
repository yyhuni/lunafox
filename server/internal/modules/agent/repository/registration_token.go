package repository

import (
	agentdomain "github.com/yyhuni/lunafox/server/internal/modules/agent/domain"
	"gorm.io/gorm"
)

type registrationTokenRepository struct {
	db       *gorm.DB
	licenses *AgentLicenseRepository
}

// NewRegistrationTokenRepository creates a new registration token repository.
func NewRegistrationTokenRepository(db *gorm.DB) agentdomain.RegistrationTokenRepository {
	return &registrationTokenRepository{db: db, licenses: NewAgentLicenseRepository(db)}
}
