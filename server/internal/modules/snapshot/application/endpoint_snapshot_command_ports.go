package application

import (
	"context"

	snapshotdomain "github.com/yyhuni/lunafox/server/internal/modules/snapshot/domain"
)

type EndpointSnapshotCommandStore interface {
	BatchCreateContext(context.Context, []snapshotdomain.EndpointSnapshot) (int64, error)
}

type EndpointAssetSync interface {
	BatchUpsertContext(context.Context, int, []EndpointAssetUpsertItem) (int64, error)
}
