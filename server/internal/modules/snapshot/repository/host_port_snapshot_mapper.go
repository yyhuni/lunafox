package repository

import (
	snapshotdomain "github.com/yyhuni/lunafox/server/internal/modules/snapshot/domain"
	model "github.com/yyhuni/lunafox/server/internal/modules/snapshot/repository/persistence"
	"github.com/yyhuni/lunafox/server/internal/pkg/timeutil"
)

func hostPortSnapshotModelToDomain(item *model.HostPortSnapshot) *snapshotdomain.HostPortSnapshot {
	if item == nil {
		return nil
	}
	return &snapshotdomain.HostPortSnapshot{ID: item.ID, ScanID: item.ScanID, Host: item.Host, IP: item.IP, Port: item.Port, CreatedAt: timeutil.ToUTC(item.CreatedAt)}
}

func hostPortSnapshotDomainToModel(item *snapshotdomain.HostPortSnapshot) *model.HostPortSnapshot {
	if item == nil {
		return nil
	}
	return &model.HostPortSnapshot{ID: item.ID, ScanID: item.ScanID, Host: item.Host, IP: item.IP, Port: item.Port, CreatedAt: timeutil.ToUTC(item.CreatedAt)}
}

func hostPortSnapshotDomainListToModel(items []snapshotdomain.HostPortSnapshot) []model.HostPortSnapshot {
	results := make([]model.HostPortSnapshot, 0, len(items))
	for index := range items {
		results = append(results, *hostPortSnapshotDomainToModel(&items[index]))
	}
	return results
}

func hostPortSnapshotModelListToDomain(items []model.HostPortSnapshot) []snapshotdomain.HostPortSnapshot {
	results := make([]snapshotdomain.HostPortSnapshot, 0, len(items))
	for index := range items {
		results = append(results, *hostPortSnapshotModelToDomain(&items[index]))
	}
	return results
}
