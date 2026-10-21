package repository

import (
	"context"
	"errors"
	"time"

	agentdomain "github.com/yyhuni/lunafox/server/internal/modules/agent/domain"
	"github.com/yyhuni/lunafox/server/internal/pkg/dbtx"
	"gorm.io/gorm"
)

// WithinClaim holds a shared entitlement boundary for one atomic task claim.
func (repository *AgentLicenseRepository) WithinClaim(ctx context.Context, callback func(context.Context) error) error {
	return repository.withEntitlementBoundary(ctx, callback, true)
}

// WithinIdentityChange serializes identity changes with entitlement reads and claims.
func (repository *AgentLicenseRepository) WithinIdentityChange(ctx context.Context, callback func(context.Context) error) error {
	return repository.withEntitlementBoundary(ctx, callback, false)
}

func (repository *AgentLicenseRepository) withEntitlementBoundary(ctx context.Context, callback func(context.Context) error, shared bool) error {
	if err := repository.validate(); err != nil {
		return err
	}
	if ctx == nil || callback == nil {
		return errors.New("agent entitlement transaction scope is required")
	}
	return dbtx.Resolve(ctx, repository.db).WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if shared && tx.Name() == "postgres" {
			// Claims may run concurrently, but registration, replacement and deletion
			// must wait until the claim commits its authoritative rank decision.
			if err := tx.Exec("SELECT pg_advisory_xact_lock_shared(?)", agentQuotaAdvisoryLock).Error; err != nil {
				return err
			}
		} else if err := lockAgentQuota(tx); err != nil {
			return err
		}
		return callback(dbtx.WithTransaction(ctx, tx))
	}, agentQuotaTransactionOptions(repository.db))
}

// EligibleAgentIDsAt returns the stable oldest identities covered by current entitlement.
func (repository *AgentLicenseRepository) EligibleAgentIDsAt(ctx context.Context, now time.Time) ([]int, error) {
	license, err := repository.CurrentAt(ctx, now)
	if err != nil {
		return nil, err
	}
	var ids []int
	// Connection and runtime state cannot rotate quota ownership. Offline rows
	// still occupy their original place until the identity is explicitly deleted.
	if err := dbtx.Resolve(ctx, repository.db).WithContext(ctx).Table("agent").
		Order("created_at ASC, id ASC").Limit(license.AgentLimit).Pluck("id", &ids).Error; err != nil {
		return nil, err
	}
	return ids, nil
}

// ExecutionEligibilityAt derives an identity's eligibility without persisting stale status.
func (repository *AgentLicenseRepository) ExecutionEligibilityAt(ctx context.Context, agentID int, now time.Time) (string, error) {
	if agentID <= 0 {
		return "", errors.New("agent identity is required")
	}
	ids, err := repository.EligibleAgentIDsAt(ctx, now)
	if err != nil {
		return "", err
	}
	for _, id := range ids {
		if id == agentID {
			return agentdomain.AgentExecutionEligible, nil
		}
	}
	var count int64
	if err := dbtx.Resolve(ctx, repository.db).WithContext(ctx).Table("agent").Where("id = ?", agentID).Count(&count).Error; err != nil {
		return "", err
	}
	if count == 0 {
		return "", gorm.ErrRecordNotFound
	}
	return agentdomain.AgentExecutionLicenseRestricted, nil
}
