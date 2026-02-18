package securitywiring

import (
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
	target, err := adapter.repo.GetActiveByID(id)
	if err != nil {
		return nil, err
	}
	return &securitydomain.TargetRef{ID: target.ID}, nil
}
