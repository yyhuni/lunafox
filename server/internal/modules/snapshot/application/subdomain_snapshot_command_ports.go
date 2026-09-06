package application

import (
	"context"

	snapshotdomain "github.com/yyhuni/lunafox/server/internal/modules/snapshot/domain"
)

type SubdomainSnapshotCommandStore interface {
	BatchCreateContext(context.Context, []snapshotdomain.SubdomainSnapshot) (int64, error)
}

type SubdomainAssetSync interface {
	BatchCreateContext(context.Context, int, []string) (int, error)
}
