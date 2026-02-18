package assetwiring

import (
	assetdomain "github.com/yyhuni/lunafox/server/internal/modules/asset/domain"
	catalogrepo "github.com/yyhuni/lunafox/server/internal/modules/catalog/repository"
	"github.com/yyhuni/lunafox/server/internal/pkg/timeutil"
)

type assetTargetLookupAdapter struct {
	repo *catalogrepo.TargetRepository
}

func newAssetTargetLookupAdapter(repo *catalogrepo.TargetRepository) *assetTargetLookupAdapter {
	return &assetTargetLookupAdapter{repo: repo}
}

func (adapter *assetTargetLookupAdapter) GetActiveByID(id int) (*assetdomain.TargetRef, error) {
	target, err := adapter.repo.GetActiveByID(id)
	if err != nil {
		return nil, err
	}
	return &assetdomain.TargetRef{
		ID:            target.ID,
		Name:          target.Name,
		Type:          target.Type,
		CreatedAt:     timeutil.ToUTC(target.CreatedAt),
		LastScannedAt: timeutil.ToUTCPtr(target.LastScannedAt),
		DeletedAt:     timeutil.ToUTCPtr(target.DeletedAt),
	}, nil
}
