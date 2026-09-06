package scanwiring

import (
	"context"
	"fmt"

	blacklistapp "github.com/yyhuni/lunafox/server/internal/modules/blacklist/application"
	scanapp "github.com/yyhuni/lunafox/server/internal/modules/scan/application"
)

// NewEffectiveBlacklistPolicyResolverAdapter keeps Scan dependent on its own
// narrow port while bootstrap owns the blacklist-to-Scan application bridge.
func NewEffectiveBlacklistPolicyResolverAdapter(policyService blacklistapp.BlacklistPolicyApplicationService) (scanapp.EffectiveBlacklistPolicyResolver, error) {
	if policyService == nil {
		return nil, fmt.Errorf("blacklist policy application service is required")
	}
	return effectiveBlacklistPolicyResolverAdapter{policyService: policyService}, nil
}

type effectiveBlacklistPolicyResolverAdapter struct {
	policyService blacklistapp.BlacklistPolicyApplicationService
}

func (adapter effectiveBlacklistPolicyResolverAdapter) ResolveEffectivePatternsForScan(ctx context.Context, targetID int) ([]string, error) {
	patterns, err := adapter.policyService.ResolveEffectivePatternsForScan(ctx, targetID)
	if err != nil {
		return nil, err
	}
	return append([]string{}, patterns...), nil
}

var _ scanapp.EffectiveBlacklistPolicyResolver = (*effectiveBlacklistPolicyResolverAdapter)(nil)
