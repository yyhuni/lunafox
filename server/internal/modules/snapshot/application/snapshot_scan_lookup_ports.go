package application

import snapshotdomain "github.com/yyhuni/lunafox/server/internal/modules/snapshot/domain"

type SnapshotScanRefLookup interface {
	GetScanRefByID(id int) (*snapshotdomain.ScanRef, error)
	GetTargetRefByScanID(scanID int) (*snapshotdomain.ScanTargetRef, error)
}
