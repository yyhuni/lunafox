package repository

import (
	snapshotdomain "github.com/yyhuni/lunafox/server/internal/modules/snapshot/domain"
	model "github.com/yyhuni/lunafox/server/internal/modules/snapshot/repository/persistence"
	"github.com/yyhuni/lunafox/server/internal/pkg/timeutil"
)

func subdomainSnapshotModelToDomain(item *model.SubdomainSnapshot) *snapshotdomain.SubdomainSnapshot {
	if item == nil {
		return nil
	}
	return &snapshotdomain.SubdomainSnapshot{ID: item.ID, ScanID: item.ScanID, Name: item.Name, CreatedAt: timeutil.ToUTC(item.CreatedAt)}
}

func subdomainSnapshotDomainToModel(item *snapshotdomain.SubdomainSnapshot) *model.SubdomainSnapshot {
	if item == nil {
		return nil
	}
	return &model.SubdomainSnapshot{ID: item.ID, ScanID: item.ScanID, Name: item.Name, CreatedAt: timeutil.ToUTC(item.CreatedAt)}
}

func subdomainSnapshotDomainListToModel(items []snapshotdomain.SubdomainSnapshot) []model.SubdomainSnapshot {
	results := make([]model.SubdomainSnapshot, 0, len(items))
	for index := range items {
		results = append(results, *subdomainSnapshotDomainToModel(&items[index]))
	}
	return results
}

func subdomainSnapshotModelListToDomain(items []model.SubdomainSnapshot) []snapshotdomain.SubdomainSnapshot {
	results := make([]snapshotdomain.SubdomainSnapshot, 0, len(items))
	for index := range items {
		results = append(results, *subdomainSnapshotModelToDomain(&items[index]))
	}
	return results
}
