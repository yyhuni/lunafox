package application

import (
	"context"

	snapshotdomain "github.com/yyhuni/lunafox/server/internal/modules/snapshot/domain"
)

type ScreenshotSnapshotCommandStore interface {
	BatchUpsertContext(context.Context, []snapshotdomain.ScreenshotSnapshot) (int64, error)
}

type ScreenshotAssetSync interface {
	BatchUpsertContext(context.Context, int, *ScreenshotAssetUpsertRequest) (int64, error)
}
