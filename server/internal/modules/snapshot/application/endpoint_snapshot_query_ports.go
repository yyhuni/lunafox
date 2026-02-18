package application

import (
	"database/sql"

	snapshotdomain "github.com/yyhuni/lunafox/server/internal/modules/snapshot/domain"
)

type EndpointSnapshotQueryStore interface {
	FindByScanID(scanID int, page, pageSize int, filter string) ([]snapshotdomain.EndpointSnapshot, int64, error)
	StreamByScanID(scanID int) (*sql.Rows, error)
	CountByScanID(scanID int) (int64, error)
	ScanRow(rows *sql.Rows) (*snapshotdomain.EndpointSnapshot, error)
}
