package application

import snapshotdomain "github.com/yyhuni/lunafox/server/internal/modules/snapshot/domain"

type ScreenshotSnapshotCommandStore interface {
	BulkUpsert(snapshots []snapshotdomain.ScreenshotSnapshot) (int64, error)
}

type ScreenshotAssetSync interface {
	BulkUpsert(targetID int, req *ScreenshotAssetUpsertRequest) (int64, error)
}
