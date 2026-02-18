package application

import snapshotdomain "github.com/yyhuni/lunafox/server/internal/modules/snapshot/domain"

type ScreenshotSnapshotQueryStore interface {
	FindByScanID(scanID int, page, pageSize int, filter string) ([]snapshotdomain.ScreenshotSnapshot, int64, error)
	FindByIDAndScanID(id int, scanID int) (*snapshotdomain.ScreenshotSnapshot, error)
}
