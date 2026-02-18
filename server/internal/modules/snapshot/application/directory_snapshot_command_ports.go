package application

import snapshotdomain "github.com/yyhuni/lunafox/server/internal/modules/snapshot/domain"

type DirectorySnapshotCommandStore interface {
	BulkCreate(snapshots []snapshotdomain.DirectorySnapshot) (int64, error)
}

type DirectoryAssetSync interface {
	BulkUpsert(targetID int, items []DirectoryAssetUpsertItem) (int64, error)
}
