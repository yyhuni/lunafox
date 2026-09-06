package application

import (
	"context"
	"fmt"
	"strings"

	blacklistdomain "github.com/yyhuni/lunafox/server/internal/modules/blacklist/domain"
)

// BlacklistPolicyUpdateService canonicalizes complete local replacements before
// delegating the locked etag comparison to the store.
type BlacklistPolicyUpdateService struct {
	store BlacklistPolicyStore
}

// NewBlacklistPolicyUpdateService constructs an update service with its
// required persistence boundary.
func NewBlacklistPolicyUpdateService(store BlacklistPolicyStore) (*BlacklistPolicyUpdateService, error) {
	if store == nil {
		return nil, ErrBlacklistPolicyDependency
	}
	return &BlacklistPolicyUpdateService{store: store}, nil
}

// ReplaceGlobal replaces all global local-scope patterns under the supplied
// current etag.
func (service *BlacklistPolicyUpdateService) ReplaceGlobal(ctx context.Context, input ReplaceBlacklistPolicyInput) (*BlacklistPolicy, error) {
	if err := validateBlacklistPolicyContext(ctx); err != nil {
		return nil, err
	}
	patterns, err := canonicalBlacklistPolicyReplacement(input)
	if err != nil {
		return nil, err
	}
	record, err := service.store.ReplaceGlobal(ctx, input.ETag, patterns)
	if err != nil {
		return nil, err
	}
	return projectBlacklistPolicy(record, BlacklistPolicyScopeGlobal, 0)
}

// ReplaceTarget replaces all Target-local patterns under the supplied current
// etag. It cannot alter the inherited global policy.
func (service *BlacklistPolicyUpdateService) ReplaceTarget(ctx context.Context, targetID int, input ReplaceBlacklistPolicyInput) (*BlacklistPolicy, error) {
	if err := validateBlacklistPolicyTarget(ctx, targetID); err != nil {
		return nil, err
	}
	patterns, err := canonicalBlacklistPolicyReplacement(input)
	if err != nil {
		return nil, err
	}
	record, err := service.store.ReplaceTarget(ctx, targetID, input.ETag, patterns)
	if err != nil {
		return nil, err
	}
	return projectBlacklistPolicy(record, BlacklistPolicyScopeTarget, targetID)
}

func canonicalBlacklistPolicyReplacement(input ReplaceBlacklistPolicyInput) ([]string, error) {
	if strings.TrimSpace(input.ETag) == "" {
		return nil, fmt.Errorf("%w: etag is required", ErrBlacklistPolicyInvalidArgument)
	}
	patterns, err := blacklistdomain.CanonicalizePolicyPatterns(input.Patterns)
	if err != nil {
		return nil, fmt.Errorf("%w: patterns are invalid", ErrBlacklistPolicyInvalidArgument)
	}
	return patterns, nil
}
