package application

import (
	"context"

	snapshotdomain "github.com/yyhuni/lunafox/server/internal/modules/snapshot/domain"
)

type SnapshotScanRefLookup interface {
	GetScanRefByID(id int) (*snapshotdomain.ScanRef, error)
	GetTargetRefByScanID(scanID int) (*snapshotdomain.ScanTargetRef, error)
}

// SnapshotCommandScanRefLookup preserves the result-ingest caller context
// through scan/target authorization lookups before any snapshot write.
type SnapshotCommandScanRefLookup interface {
	GetScanRefByIDContext(ctx context.Context, id int) (*snapshotdomain.ScanRef, error)
	GetTargetRefByScanIDContext(ctx context.Context, scanID int) (*snapshotdomain.ScanTargetRef, error)
}

type SnapshotApplicationScanRefLookup interface {
	SnapshotScanRefLookup
	SnapshotCommandScanRefLookup
}
