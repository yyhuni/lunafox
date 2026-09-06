package securitywiring

import (
	"context"

	catalogrepo "github.com/yyhuni/lunafox/server/internal/modules/catalog/repository"
	securitydomain "github.com/yyhuni/lunafox/server/internal/modules/security/domain"
)

type securityTargetLookupAdapter struct {
	repo *catalogrepo.TargetRepository
}

func newSecurityTargetLookupAdapter(repo *catalogrepo.TargetRepository) *securityTargetLookupAdapter {
	return &securityTargetLookupAdapter{repo: repo}
}

func (adapter *securityTargetLookupAdapter) GetActiveByID(id int) (*securitydomain.TargetRef, error) {
	return adapter.GetActiveByIDContext(context.Background(), id)
}

func (adapter *securityTargetLookupAdapter) GetActiveByIDContext(ctx context.Context, id int) (*securitydomain.TargetRef, error) {
	target, err := adapter.repo.GetActiveByIDContext(ctx, id)
	if err != nil {
		return nil, err
	}
	return &securitydomain.TargetRef{ID: target.ID, Name: target.Name, Type: target.Type}, nil
}
