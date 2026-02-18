package application

import snapshotdomain "github.com/yyhuni/lunafox/server/internal/modules/snapshot/domain"

type HostPortSnapshotCommandStore interface {
	BulkCreate(snapshots []snapshotdomain.HostPortSnapshot) (int64, error)
}

type HostPortAssetSync interface {
	BulkUpsert(targetID int, items []HostPortAssetItem) (int64, error)
}
