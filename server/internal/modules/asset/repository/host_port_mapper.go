package repository

import (
	assetdomain "github.com/yyhuni/lunafox/server/internal/modules/asset/domain"
	model "github.com/yyhuni/lunafox/server/internal/modules/asset/repository/persistence"
	"github.com/yyhuni/lunafox/server/internal/pkg/timeutil"
)

func hostPortModelToDomain(item *model.HostPort) *assetdomain.HostPort {
	if item == nil {
		return nil
	}
	return &assetdomain.HostPort{
		ID:        item.ID,
		TargetID:  item.TargetID,
		Host:      item.Host,
		IP:        item.IP,
		Port:      item.Port,
		CreatedAt: timeutil.ToUTC(item.CreatedAt),
	}
}

func hostPortDomainToModel(item *assetdomain.HostPort) *model.HostPort {
	if item == nil {
		return nil
	}
	return &model.HostPort{
		ID:        item.ID,
		TargetID:  item.TargetID,
		Host:      item.Host,
		IP:        item.IP,
		Port:      item.Port,
		CreatedAt: timeutil.ToUTC(item.CreatedAt),
	}
}

func hostPortDomainListToModel(items []assetdomain.HostPort) []model.HostPort {
	results := make([]model.HostPort, 0, len(items))
	for index := range items {
		results = append(results, *hostPortDomainToModel(&items[index]))
	}
	return results
}
