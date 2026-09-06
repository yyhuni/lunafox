package snapshotwiring

import (
	scandomain "github.com/yyhuni/lunafox/server/internal/modules/scan/domain"
	snapshotdomain "github.com/yyhuni/lunafox/server/internal/modules/snapshot/domain"
)

func snapshotScanModelToDomain(item *scandomain.QueryScan) *snapshotdomain.ScanRef {
	if item == nil {
		return nil
	}
	return &snapshotdomain.ScanRef{ID: item.ID, TargetID: item.TargetID}
}

func snapshotScanTargetModelToDomain(item *scandomain.QueryTargetRef) *snapshotdomain.ScanTargetRef {
	if item == nil {
		return nil
	}
	return &snapshotdomain.ScanTargetRef{ID: item.ID, Name: item.Name, Type: item.Type, CreatedAt: item.CreatedAt}
}
