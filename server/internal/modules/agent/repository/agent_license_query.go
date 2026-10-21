package repository

import (
	"context"
	"errors"
	"time"

	agentdomain "github.com/yyhuni/lunafox/server/internal/modules/agent/domain"
	model "github.com/yyhuni/lunafox/server/internal/modules/agent/repository/persistence"
	"github.com/yyhuni/lunafox/server/internal/pkg/dbtx"
	"gorm.io/gorm"
)

// CurrentAt returns effective entitlements while preserving expired signed metadata.
func (repository *AgentLicenseRepository) CurrentAt(ctx context.Context, now time.Time) (agentdomain.AgentLicense, error) {
	if err := repository.validate(); err != nil {
		return agentdomain.AgentLicense{}, err
	}
	if now.IsZero() {
		return agentdomain.AgentLicense{}, errors.New("agent license observation time is required")
	}
	return currentAgentLicenseAt(dbtx.Resolve(ctx, repository.db).WithContext(ctx), repository.verifier, now.UTC())
}

func currentAgentLicenseAt(db *gorm.DB, verifier AgentLicenseVerifier, now time.Time) (agentdomain.AgentLicense, error) {
	var records []model.AgentLicense
	if err := db.Where("singleton = ?", true).Limit(1).Find(&records).Error; err != nil {
		return agentdomain.AgentLicense{}, err
	}
	if len(records) == 0 {
		return agentdomain.AgentLicense{State: "inactive", AgentLimit: agentdomain.DeploymentAgentLimit}, nil
	}
	claims, err := verifier.Verify(records[0].SignedKey)
	if err != nil {
		// A corrupt persisted Key is an operational failure, not a free-tier
		// license. Never hide storage or verification faults behind a quota.
		return agentdomain.AgentLicense{}, errors.New("stored Agent license failed verification")
	}
	if now.Before(time.Unix(claims.IssuedAt, 0)) {
		return agentdomain.AgentLicense{}, agentdomain.ErrAgentLicenseKeyNotYetValid
	}
	return agentLicenseProjection(claims.AgentLimit, claims.Months, claims.LicenseID, time.Unix(claims.ExpiresAt, 0).UTC(), now), nil
}

func agentLicenseProjection(limit, months int, licenseID string, expiresAt, now time.Time) agentdomain.AgentLicense {
	license := agentdomain.AgentLicense{State: "active", AgentLimit: limit, Months: &months, LicenseID: &licenseID, ExpiresAt: &expiresAt}
	if !now.Before(expiresAt) {
		license.State = "expired"
		license.AgentLimit = agentdomain.DeploymentAgentLimit
	}
	return license
}
