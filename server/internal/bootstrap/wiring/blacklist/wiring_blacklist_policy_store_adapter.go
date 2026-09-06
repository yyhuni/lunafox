package blacklistwiring

import (
	"context"
	"errors"

	blacklistapp "github.com/yyhuni/lunafox/server/internal/modules/blacklist/application"
	blacklistdomain "github.com/yyhuni/lunafox/server/internal/modules/blacklist/domain"
	blacklistrepo "github.com/yyhuni/lunafox/server/internal/modules/blacklist/repository"
)

type blacklistPolicyStoreAdapter struct {
	repo *blacklistrepo.PolicyRepository
}

func newBlacklistPolicyStoreAdapter(repo *blacklistrepo.PolicyRepository) *blacklistPolicyStoreAdapter {
	return &blacklistPolicyStoreAdapter{repo: repo}
}

func (adapter *blacklistPolicyStoreAdapter) GetGlobal(ctx context.Context) (*blacklistapp.BlacklistPolicyRecord, error) {
	policy, err := adapter.repo.GetGlobal(ctx)
	if err != nil {
		return nil, mapBlacklistPolicyRepositoryError(err)
	}
	return toBlacklistPolicyRecord(policy)
}

func (adapter *blacklistPolicyStoreAdapter) GetTarget(ctx context.Context, targetID int) (*blacklistapp.BlacklistPolicyRecord, error) {
	policy, err := adapter.repo.GetTarget(ctx, targetID)
	if err != nil {
		return nil, mapBlacklistPolicyRepositoryError(err)
	}
	return toBlacklistPolicyRecord(policy)
}

func (adapter *blacklistPolicyStoreAdapter) ReplaceGlobal(ctx context.Context, etag string, patterns []string) (*blacklistapp.BlacklistPolicyRecord, error) {
	policy, _, err := adapter.repo.ReplacePatterns(ctx, blacklistdomain.ScopeGlobal, nil, etag, patterns)
	if err != nil {
		return nil, mapBlacklistPolicyRepositoryError(err)
	}
	return toBlacklistPolicyRecord(policy)
}

func (adapter *blacklistPolicyStoreAdapter) ReplaceTarget(ctx context.Context, targetID int, etag string, patterns []string) (*blacklistapp.BlacklistPolicyRecord, error) {
	policy, _, err := adapter.repo.ReplacePatterns(ctx, blacklistdomain.ScopeTarget, &targetID, etag, patterns)
	if err != nil {
		return nil, mapBlacklistPolicyRepositoryError(err)
	}
	return toBlacklistPolicyRecord(policy)
}

func (adapter *blacklistPolicyStoreAdapter) ReadEffectivePatternsForScan(ctx context.Context, targetID int) ([]string, error) {
	patterns, err := adapter.repo.ReadEffectivePatternsForScan(ctx, targetID)
	if err != nil {
		return nil, mapBlacklistPolicyRepositoryError(err)
	}
	return append([]string{}, patterns...), nil
}

func toBlacklistPolicyRecord(policy *blacklistdomain.Policy) (*blacklistapp.BlacklistPolicyRecord, error) {
	if policy == nil {
		return nil, blacklistapp.ErrBlacklistPolicyDataIntegrity
	}
	record := &blacklistapp.BlacklistPolicyRecord{
		Patterns:  append([]string{}, policy.Patterns...),
		UpdatedAt: policy.UpdatedAt,
	}
	switch policy.Scope {
	case blacklistdomain.ScopeGlobal:
		record.Scope = blacklistapp.BlacklistPolicyScopeGlobal
	case blacklistdomain.ScopeTarget:
		record.Scope = blacklistapp.BlacklistPolicyScopeTarget
	default:
		return nil, blacklistapp.ErrBlacklistPolicyDataIntegrity
	}
	if policy.TargetID != nil {
		targetID := *policy.TargetID
		record.TargetID = &targetID
	}
	return record, nil
}

func mapBlacklistPolicyRepositoryError(err error) error {
	switch {
	case err == nil:
		return nil
	case errors.Is(err, blacklistrepo.ErrPolicyNotFound):
		return blacklistapp.ErrBlacklistPolicyNotFound
	case errors.Is(err, blacklistrepo.ErrPolicyETagConflict):
		return blacklistapp.ErrBlacklistPolicyConflict
	case errors.Is(err, blacklistrepo.ErrPolicyDataIntegrity), errors.Is(err, blacklistdomain.ErrNonCanonicalPatterns), errors.Is(err, blacklistdomain.ErrEffectiveLimitExceeded):
		return blacklistapp.ErrBlacklistPolicyDataIntegrity
	case errors.Is(err, blacklistrepo.ErrInvalidPolicyUpdate), errors.Is(err, blacklistdomain.ErrInvalidPattern), errors.Is(err, blacklistdomain.ErrPolicyLimitExceeded):
		return blacklistapp.ErrBlacklistPolicyInvalidArgument
	default:
		return err
	}
}
