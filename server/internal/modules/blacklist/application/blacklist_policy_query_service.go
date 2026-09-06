package application

import (
	"context"
	"fmt"

	"github.com/yyhuni/lunafox/contracts/resourcenames"
	blacklistdomain "github.com/yyhuni/lunafox/server/internal/modules/blacklist/domain"
)

// BlacklistPolicyQueryService reads local singleton resources and resolves the
// immutable effective pattern input used at Scan creation.
type BlacklistPolicyQueryService struct {
	store BlacklistPolicyStore
}

// NewBlacklistPolicyQueryService constructs a query service with its required
// persistence boundary.
func NewBlacklistPolicyQueryService(store BlacklistPolicyStore) (*BlacklistPolicyQueryService, error) {
	if store == nil {
		return nil, ErrBlacklistPolicyDependency
	}
	return &BlacklistPolicyQueryService{store: store}, nil
}

// GetGlobal returns the global singleton, including a valid empty policy.
func (service *BlacklistPolicyQueryService) GetGlobal(ctx context.Context) (*BlacklistPolicy, error) {
	if err := validateBlacklistPolicyContext(ctx); err != nil {
		return nil, err
	}
	record, err := service.store.GetGlobal(ctx)
	if err != nil {
		return nil, err
	}
	return projectBlacklistPolicy(record, BlacklistPolicyScopeGlobal, 0)
}

// GetTarget returns one existing Target-local singleton.
func (service *BlacklistPolicyQueryService) GetTarget(ctx context.Context, targetID int) (*BlacklistPolicy, error) {
	if err := validateBlacklistPolicyTarget(ctx, targetID); err != nil {
		return nil, err
	}
	record, err := service.store.GetTarget(ctx, targetID)
	if err != nil {
		return nil, err
	}
	return projectBlacklistPolicy(record, BlacklistPolicyScopeTarget, targetID)
}

// ResolveEffectivePatternsForScan obtains the locked global-plus-local union
// and validates it again at the application boundary before Scan persistence.
func (service *BlacklistPolicyQueryService) ResolveEffectivePatternsForScan(ctx context.Context, targetID int) ([]string, error) {
	if err := validateBlacklistPolicyTarget(ctx, targetID); err != nil {
		return nil, err
	}
	patterns, err := service.store.ReadEffectivePatternsForScan(ctx, targetID)
	if err != nil {
		return nil, err
	}
	patterns = cloneBlacklistPatterns(patterns)
	if err := blacklistdomain.ValidateCanonicalEffectivePatterns(patterns); err != nil {
		return nil, fmt.Errorf("%w: effective patterns are invalid", ErrBlacklistPolicyDataIntegrity)
	}
	return patterns, nil
}

func projectBlacklistPolicy(record *BlacklistPolicyRecord, expectedScope BlacklistPolicyScope, targetID int) (*BlacklistPolicy, error) {
	if record == nil || record.Scope != expectedScope {
		return nil, ErrBlacklistPolicyDataIntegrity
	}
	name := resourcenames.BlacklistPolicy()
	if expectedScope == BlacklistPolicyScopeGlobal {
		if record.TargetID != nil {
			return nil, ErrBlacklistPolicyDataIntegrity
		}
	} else {
		if targetID <= 0 || record.TargetID == nil || *record.TargetID != targetID {
			return nil, ErrBlacklistPolicyDataIntegrity
		}
		name = resourcenames.TargetBlacklistPolicy(targetID)
	}
	patterns := cloneBlacklistPatterns(record.Patterns)
	if err := blacklistdomain.ValidateCanonicalPolicyPatterns(patterns); err != nil {
		return nil, fmt.Errorf("%w: local patterns are invalid", ErrBlacklistPolicyDataIntegrity)
	}
	etag, err := blacklistdomain.ETag(patterns)
	if err != nil {
		return nil, fmt.Errorf("%w: derive etag", ErrBlacklistPolicyDataIntegrity)
	}
	return &BlacklistPolicy{
		Name:       name,
		Patterns:   patterns,
		ETag:       etag,
		UpdateTime: record.UpdatedAt.UTC(),
	}, nil
}

func cloneBlacklistPatterns(patterns []string) []string {
	cloned := make([]string, len(patterns))
	copy(cloned, patterns)
	return cloned
}

func validateBlacklistPolicyContext(ctx context.Context) error {
	if ctx == nil {
		return fmt.Errorf("%w: context is required", ErrBlacklistPolicyInvalidArgument)
	}
	return nil
}

func validateBlacklistPolicyTarget(ctx context.Context, targetID int) error {
	if err := validateBlacklistPolicyContext(ctx); err != nil {
		return err
	}
	if targetID <= 0 {
		return fmt.Errorf("%w: target id is required", ErrBlacklistPolicyInvalidArgument)
	}
	return nil
}
