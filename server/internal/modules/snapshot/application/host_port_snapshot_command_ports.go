package application

import (
	"context"

	snapshotdomain "github.com/yyhuni/lunafox/server/internal/modules/snapshot/domain"
)

type HostPortSnapshotCommandStore interface {
	BatchCreateContext(context.Context, []snapshotdomain.HostPortSnapshot) (int64, error)
}

type HostPortAssetSync interface {
	BatchUpsertContext(context.Context, int, []HostPortAssetItem) (int64, error)
}
