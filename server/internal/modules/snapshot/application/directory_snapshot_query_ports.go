package application

import (
	"database/sql"

	snapshotdomain "github.com/yyhuni/lunafox/server/internal/modules/snapshot/domain"
)

type DirectorySnapshotQueryStore interface {
	FindByScanID(scanID int, page, pageSize int, filter string) ([]snapshotdomain.DirectorySnapshot, int64, error)
	StreamByScanID(scanID int) (*sql.Rows, error)
	CountByScanID(scanID int) (int64, error)
	ScanRow(rows *sql.Rows) (*snapshotdomain.DirectorySnapshot, error)
}
