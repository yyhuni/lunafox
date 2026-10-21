package repository

import (
	"context"
	"strings"
	"time"

	agentdomain "github.com/yyhuni/lunafox/server/internal/modules/agent/domain"
	model "github.com/yyhuni/lunafox/server/internal/modules/agent/repository/persistence"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/logger"
)

// ActivateAt atomically imports a currently valid signed Key without extending it.
func (repository *AgentLicenseRepository) ActivateAt(ctx context.Context, key string, now time.Time) (agentdomain.AgentLicense, error) {
	startedAt := time.Now()
	if err := repository.validate(); err != nil {
		return agentdomain.AgentLicense{}, err
	}
	key = strings.TrimSpace(key)
	if key == "" {
		return agentdomain.AgentLicense{}, agentdomain.ErrAgentLicenseKeyRequired
	}
	if len(key) > 4096 || now.IsZero() {
		return agentdomain.AgentLicense{}, agentdomain.ErrAgentLicenseKeyInvalid
	}
	now = now.UTC()
	var license agentdomain.AgentLicense
	err := repository.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := lockAgentQuota(tx); err != nil {
			return err
		}
		// Keep the caller's testable observation instant, but advance it by the
		// monotonic lock-wait duration so a waiting activation cannot revive an
		// entitlement that expired before it acquired the registration boundary.
		activationAt := now.Add(time.Since(startedAt))
		claims, err := repository.verifier.Verify(key)
		if err != nil {
			return agentdomain.ErrAgentLicenseKeyInvalid
		}
		if activationAt.Before(time.Unix(claims.IssuedAt, 0)) {
			return agentdomain.ErrAgentLicenseKeyNotYetValid
		}
		expiresAt := time.Unix(claims.ExpiresAt, 0).UTC()
		if !activationAt.Before(expiresAt) {
			return agentdomain.ErrAgentLicenseKeyExpired
		}
		license = agentLicenseProjection(claims.AgentLimit, claims.Months, claims.LicenseID, expiresAt, activationAt)
		// The signed bytes are the sole source of dates and limits. Reimporting
		// them changes no entitlement, and sensitive Key values must not enter SQL logs.
		return tx.Session(&gorm.Session{Logger: tx.Logger.LogMode(logger.Silent)}).Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "singleton"}},
			DoUpdates: clause.AssignmentColumns([]string{"signed_key"}),
		}).Create(&model.AgentLicense{Singleton: true, SignedKey: key}).Error
	}, agentQuotaTransactionOptions(repository.db))
	if err != nil {
		return agentdomain.AgentLicense{}, err
	}
	return license, nil
}
