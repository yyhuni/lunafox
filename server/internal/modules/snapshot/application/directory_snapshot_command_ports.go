package application

import (
	"context"

	snapshotdomain "github.com/yyhuni/lunafox/server/internal/modules/snapshot/domain"
)

type DirectorySnapshotCommandStore interface {
	BatchCreate(snapshots []snapshotdomain.DirectorySnapshot) (int64, error)
	BatchCreateContext(context.Context, []snapshotdomain.DirectorySnapshot) (int64, error)
}

type DirectoryAssetSync interface {
	BatchUpsert(targetID int, items []DirectoryAssetUpsertItem) (int64, error)
	BatchUpsertContext(context.Context, int, []DirectoryAssetUpsertItem) (int64, error)
}
