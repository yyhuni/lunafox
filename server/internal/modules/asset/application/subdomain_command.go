package application

import (
	"context"
	"errors"
	"fmt"

	contractresults "github.com/yyhuni/lunafox/contracts/results"
	assetdomain "github.com/yyhuni/lunafox/server/internal/modules/asset/domain"
	"github.com/yyhuni/lunafox/server/internal/pkg/dberrors"
)

var ErrSubdomainInvalidTargetType = errors.New("target type must be domain for subdomains")

type SubdomainCommandService struct {
	store        SubdomainCommandStore
	targetLookup AssetCommandTargetLookup
}

func NewSubdomainCommandService(store SubdomainCommandStore, targetLookup AssetCommandTargetLookup) *SubdomainCommandService {
	return &SubdomainCommandService{store: store, targetLookup: targetLookup}
}

func (service *SubdomainCommandService) BatchCreate(ctx context.Context, targetID int, dnsNames []string) (int, error) {
	if ctx == nil {
		return 0, context.Canceled
	}
	if err := ctx.Err(); err != nil {
		return 0, err
	}

	target, err := service.targetLookup.GetActiveByIDContext(ctx, targetID)
	if err != nil {
		if dberrors.IsRecordNotFound(err) {
			return 0, ErrTargetNotFound
		}
		return 0, err
	}

	if !assetdomain.IsDomainTargetType(target.Type) {
		return 0, ErrSubdomainInvalidTargetType
	}

	subdomains := make([]assetdomain.Subdomain, 0, len(dnsNames))
	seen := make(map[string]struct{}, len(dnsNames))
	for index, dnsName := range dnsNames {
		canonical, ok := contractresults.NormalizeSubdomainDNSName(dnsName)
		if !ok || canonical != dnsName {
			return 0, fmt.Errorf("dnsNames[%d]: subdomain dnsName must be canonical", index)
		}
		if _, exists := seen[canonical]; exists {
			continue
		}
		seen[canonical] = struct{}{}
		if assetdomain.IsSubdomainMatchTarget(canonical, *target) {
			subdomains = append(subdomains, assetdomain.Subdomain{
				TargetID: targetID,
				DNSName:  dnsName,
			})
		}
	}

	if len(subdomains) == 0 {
		return 0, nil
	}

	if err := ctx.Err(); err != nil {
		return 0, err
	}
	return service.store.BatchCreateContext(ctx, subdomains)
}

func (service *SubdomainCommandService) BatchDelete(ctx context.Context, ids []int) (int64, error) {
	_ = ctx

	if len(ids) == 0 {
		return 0, nil
	}

	return service.store.BatchDelete(ids)
}
