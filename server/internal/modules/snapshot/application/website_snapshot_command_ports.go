package application

import snapshotdomain "github.com/yyhuni/lunafox/server/internal/modules/snapshot/domain"

type WebsiteSnapshotCommandStore interface {
	BulkCreate(snapshots []snapshotdomain.WebsiteSnapshot) (int64, error)
}

type WebsiteAssetSync interface {
	BulkUpsert(targetID int, items []WebsiteAssetUpsertItem) (int64, error)
}
