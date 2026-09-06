package application

import "context"

// BlacklistPolicyFacade is the application entry point for singleton HTTP
// resources and the later Scan effective-policy resolver adapter.
type BlacklistPolicyFacade struct {
	query  *BlacklistPolicyQueryService
	update *BlacklistPolicyUpdateService
}

// NewBlacklistPolicyFacade joins the independently testable query and update
// services and rejects incomplete bootstrap construction.
func NewBlacklistPolicyFacade(query *BlacklistPolicyQueryService, update *BlacklistPolicyUpdateService) (*BlacklistPolicyFacade, error) {
	if query == nil || update == nil {
		return nil, ErrBlacklistPolicyDependency
	}
	return &BlacklistPolicyFacade{query: query, update: update}, nil
}

// GetGlobal returns the global singleton resource.
func (facade *BlacklistPolicyFacade) GetGlobal(ctx context.Context) (*BlacklistPolicy, error) {
	return facade.query.GetGlobal(ctx)
}

// GetTarget returns a Target-local singleton resource.
func (facade *BlacklistPolicyFacade) GetTarget(ctx context.Context, targetID int) (*BlacklistPolicy, error) {
	return facade.query.GetTarget(ctx, targetID)
}

// ReplaceGlobal performs an etag-guarded full replacement of global patterns.
func (facade *BlacklistPolicyFacade) ReplaceGlobal(ctx context.Context, input ReplaceBlacklistPolicyInput) (*BlacklistPolicy, error) {
	return facade.update.ReplaceGlobal(ctx, input)
}

// ReplaceTarget performs an etag-guarded full replacement of Target-local patterns.
func (facade *BlacklistPolicyFacade) ReplaceTarget(ctx context.Context, targetID int, input ReplaceBlacklistPolicyInput) (*BlacklistPolicy, error) {
	return facade.update.ReplaceTarget(ctx, targetID, input)
}

// ResolveEffectivePatternsForScan returns the validated locked union for one
// Scan creation transaction.
func (facade *BlacklistPolicyFacade) ResolveEffectivePatternsForScan(ctx context.Context, targetID int) ([]string, error) {
	return facade.query.ResolveEffectivePatternsForScan(ctx, targetID)
}

var _ BlacklistPolicyApplicationService = (*BlacklistPolicyFacade)(nil)
