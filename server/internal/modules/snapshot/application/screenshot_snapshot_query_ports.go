package application

import snapshotdomain "github.com/yyhuni/lunafox/server/internal/modules/snapshot/domain"

type ScreenshotSnapshotQueryStore interface {
	ListByScanID(scanID int, page, pageSize int, filter, orderBy string) ([]snapshotdomain.ScreenshotSnapshot, int64, error)
	ListFilterOptionsByScanID(scanID int, field string) ([]snapshotdomain.FilterOption, error)
	FindByIDAndScanID(id int, scanID int) (*snapshotdomain.ScreenshotSnapshot, error)
}
