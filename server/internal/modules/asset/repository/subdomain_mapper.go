package repository

import (
	assetdomain "github.com/yyhuni/lunafox/server/internal/modules/asset/domain"
	model "github.com/yyhuni/lunafox/server/internal/modules/asset/repository/persistence"
	"github.com/yyhuni/lunafox/server/internal/pkg/timeutil"
)

func subdomainModelToDomain(item *model.Subdomain) *assetdomain.Subdomain {
	if item == nil {
		return nil
	}
	return &assetdomain.Subdomain{
		ID:        item.ID,
		TargetID:  item.TargetID,
		Name:      item.Name,
		CreatedAt: timeutil.ToUTC(item.CreatedAt),
	}
}

func subdomainDomainToModel(item *assetdomain.Subdomain) *model.Subdomain {
	if item == nil {
		return nil
	}
	return &model.Subdomain{
		ID:        item.ID,
		TargetID:  item.TargetID,
		Name:      item.Name,
		CreatedAt: timeutil.ToUTC(item.CreatedAt),
	}
}

func subdomainDomainListToModel(items []assetdomain.Subdomain) []model.Subdomain {
	results := make([]model.Subdomain, 0, len(items))
	for index := range items {
		results = append(results, *subdomainDomainToModel(&items[index]))
	}
	return results
}

func subdomainModelListToDomain(items []model.Subdomain) []assetdomain.Subdomain {
	results := make([]assetdomain.Subdomain, 0, len(items))
	for index := range items {
		results = append(results, *subdomainModelToDomain(&items[index]))
	}
	return results
}
