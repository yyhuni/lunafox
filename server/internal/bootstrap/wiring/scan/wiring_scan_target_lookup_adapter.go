package scanwiring

import (
	"context"

	catalogapp "github.com/yyhuni/lunafox/server/internal/modules/catalog/application"
	catalogrepo "github.com/yyhuni/lunafox/server/internal/modules/catalog/repository"
	identityrepo "github.com/yyhuni/lunafox/server/internal/modules/identity/repository"
	scanapp "github.com/yyhuni/lunafox/server/internal/modules/scan/application"
)

type scanTargetLookupAdapter struct {
	repo             *catalogrepo.TargetRepository
	targetCommander  *catalogapp.TargetCommandService
	organizationRepo *identityrepo.OrganizationRepository
}

func newScanTargetLookupAdapter(repo *catalogrepo.TargetRepository, targetCommander *catalogapp.TargetCommandService, organizationRepo ...*identityrepo.OrganizationRepository) *scanTargetLookupAdapter {
	adapter := &scanTargetLookupAdapter{repo: repo, targetCommander: targetCommander}
	if len(organizationRepo) > 0 {
		adapter.organizationRepo = organizationRepo[0]
	}
	return adapter
}

func (adapter *scanTargetLookupAdapter) GetTargetRefByID(ctx context.Context, id int) (*scanapp.TargetRef, error) {
	target, err := adapter.repo.GetActiveByIDContext(ctx, id)
	if err != nil {
		return nil, err
	}
	return catalogDomainTargetToScanAppTargetRef(target), nil
}

func (adapter *scanTargetLookupAdapter) EnsureQuickTargets(ctx context.Context, names []string) (*scanapp.QuickTargetResolution, error) {
	if adapter.targetCommander == nil {
		return nil, scanapp.ErrCreateTargetLookupNotReady
	}
	result, err := adapter.targetCommander.EnsureTargets(ctx, names)
	if err != nil {
		return nil, err
	}
	resolved := make([]scanapp.TargetRef, 0, len(result.Targets))
	for index := range result.Targets {
		target := result.Targets[index]
		ref := catalogDomainTargetToScanAppTargetRef(&target)
		if ref != nil {
			resolved = append(resolved, *ref)
		}
	}
	errors := make([]scanapp.QuickTargetError, 0, len(result.Errors))
	for _, item := range result.Errors {
		errors = append(errors, scanapp.QuickTargetError{Input: item.Input, Error: item.Error})
	}

	return &scanapp.QuickTargetResolution{
		Targets: resolved,
		TargetStats: scanapp.QuickTargetStats{
			Created: result.TargetStats.Created,
			Skipped: result.TargetStats.Skipped,
			Failed:  result.TargetStats.Failed,
		},
		Errors: errors,
	}, nil
}

func (adapter *scanTargetLookupAdapter) ListOrganizationTargetRefs(ctx context.Context, organizationID int) ([]scanapp.TargetRef, error) {
	if adapter.organizationRepo == nil {
		return nil, scanapp.ErrCreateTargetLookupNotReady
	}
	exists, err := adapter.organizationRepo.ExistsContext(ctx, organizationID)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, scanapp.ErrCreateTargetNotFound
	}
	targets, _, err := adapter.organizationRepo.ListTargetsByOrganizationIDContext(ctx, organizationID, 0, 0, "", "")
	if err != nil {
		return nil, err
	}
	results := make([]scanapp.TargetRef, 0, len(targets))
	for index := range targets {
		target := targets[index]
		results = append(results, scanapp.TargetRef{
			ID:            target.ID,
			Name:          target.Name,
			Type:          target.Type,
			CreatedAt:     target.CreatedAt,
			LastScannedAt: target.LastScannedAt,
			DeletedAt:     target.DeletedAt,
		})
	}
	return results, nil
}
