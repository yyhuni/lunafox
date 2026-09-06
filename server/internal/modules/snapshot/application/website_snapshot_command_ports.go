package application

import (
	"context"

	snapshotdomain "github.com/yyhuni/lunafox/server/internal/modules/snapshot/domain"
)

type WebsiteSnapshotCommandStore interface {
	BatchCreateContext(context.Context, []snapshotdomain.WebsiteSnapshot) (int64, error)
}

type WebsiteAssetSync interface {
	BatchUpsertContext(context.Context, int, []WebsiteAssetUpsertItem) (int64, error)
}
