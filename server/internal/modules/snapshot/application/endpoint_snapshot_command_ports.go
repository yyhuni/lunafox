package application

import snapshotdomain "github.com/yyhuni/lunafox/server/internal/modules/snapshot/domain"

type EndpointSnapshotCommandStore interface {
	BulkCreate(snapshots []snapshotdomain.EndpointSnapshot) (int64, error)
}

type EndpointAssetSync interface {
	BulkUpsert(targetID int, items []EndpointAssetUpsertItem) (int64, error)
}
