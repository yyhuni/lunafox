package repository

import (
	"database/sql"
	"errors"

	"github.com/yyhuni/lunafox/server/internal/agentlicense"
	"gorm.io/gorm"
)

// AgentLicenseVerifier verifies signed claims without interpreting the system clock.
type AgentLicenseVerifier interface {
	Verify(key string) (agentlicense.Claims, error)
}

// AgentLicenseRepository stores only the current signed deployment entitlement.
type AgentLicenseRepository struct {
	db          *gorm.DB
	verifier    AgentLicenseVerifier
	verifierErr error
}

// NewAgentLicenseRepository uses the same product trust root in every creation path.
func NewAgentLicenseRepository(db *gorm.DB) *AgentLicenseRepository {
	verifier, err := agentlicense.NewDefaultVerifier()
	return &AgentLicenseRepository{db: db, verifier: verifier, verifierErr: err}
}

// NewAgentLicenseRepositoryWithVerifier explicitly injects a trust root for isolated tests.
func NewAgentLicenseRepositoryWithVerifier(db *gorm.DB, verifier AgentLicenseVerifier) (*AgentLicenseRepository, error) {
	if db == nil {
		return nil, errors.New("agent license database is required")
	}
	if verifier == nil {
		return nil, errors.New("agent license verifier is required")
	}
	return &AgentLicenseRepository{db: db, verifier: verifier}, nil
}

func (repository *AgentLicenseRepository) validate() error {
	if repository == nil || repository.db == nil {
		return errors.New("agent license database is required")
	}
	if repository.verifierErr != nil || repository.verifier == nil {
		return errors.New("agent license verifier is not configured")
	}
	return nil
}

func agentQuotaTransactionOptions(db *gorm.DB) *sql.TxOptions {
	if db.Name() == "postgres" {
		return &sql.TxOptions{Isolation: sql.LevelReadCommitted}
	}
	return nil
}

func lockAgentQuota(tx *gorm.DB) error {
	if tx.Name() != "postgres" {
		return nil
	}
	// Entitlement and identity writers share this exclusive boundary with
	// claim readers. The next READ COMMITTED statement must observe the
	// previous writer's commit before deriving a quota or identity rank.
	return tx.Exec("SELECT pg_advisory_xact_lock(?)", agentQuotaAdvisoryLock).Error
}
