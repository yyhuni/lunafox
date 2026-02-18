package application

import snapshotdomain "github.com/yyhuni/lunafox/server/internal/modules/snapshot/domain"

type SubdomainSnapshotCommandStore interface {
	BulkCreate(snapshots []snapshotdomain.SubdomainSnapshot) (int64, error)
}

type SubdomainAssetSync interface {
	BulkCreate(targetID int, names []string) (int, error)
}
