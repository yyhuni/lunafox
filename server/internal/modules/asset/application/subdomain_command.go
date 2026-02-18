package application

import (
	"context"
	"errors"

	assetdomain "github.com/yyhuni/lunafox/server/internal/modules/asset/domain"
	"github.com/yyhuni/lunafox/server/internal/pkg/dberrors"
)

var ErrSubdomainInvalidTargetType = errors.New("target type must be domain for subdomains")

type SubdomainCommandService struct {
	store        SubdomainCommandStore
	targetLookup SubdomainTargetLookup
}

func NewSubdomainCommandService(store SubdomainCommandStore, targetLookup SubdomainTargetLookup) *SubdomainCommandService {
	return &SubdomainCommandService{store: store, targetLookup: targetLookup}
}

func (service *SubdomainCommandService) BulkCreate(ctx context.Context, targetID int, names []string) (int, error) {
	_ = ctx

	target, err := service.targetLookup.GetActiveByID(targetID)
	if err != nil {
		if dberrors.IsRecordNotFound(err) {
			return 0, ErrTargetNotFound
		}
		return 0, err
	}

	if !assetdomain.IsDomainTargetType(target.Type) {
		return 0, ErrSubdomainInvalidTargetType
	}

	subdomains := make([]assetdomain.Subdomain, 0, len(names))
	for _, name := range names {
		if assetdomain.IsSubdomainMatchTarget(name, *target) {
			subdomains = append(subdomains, assetdomain.Subdomain{
				TargetID: targetID,
				Name:     name,
			})
		}
	}

	if len(subdomains) == 0 {
		return 0, nil
	}

	return service.store.BulkCreate(subdomains)
}

func (service *SubdomainCommandService) BulkDelete(ctx context.Context, ids []int) (int64, error) {
	_ = ctx

	if len(ids) == 0 {
		return 0, nil
	}

	return service.store.BulkDelete(ids)
}
