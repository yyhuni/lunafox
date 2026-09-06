package application

import snapshotdomain "github.com/yyhuni/lunafox/server/internal/modules/snapshot/domain"

var (
	ErrSnapshotScanNotFound   = snapshotdomain.ErrSnapshotScanNotFound
	ErrSnapshotTargetMismatch = snapshotdomain.ErrSnapshotTargetMismatch
)
