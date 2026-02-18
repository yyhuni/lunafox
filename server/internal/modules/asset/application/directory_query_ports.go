package application

import (
	"database/sql"

	assetdomain "github.com/yyhuni/lunafox/server/internal/modules/asset/domain"
)

type DirectoryQueryStore interface {
	FindByTargetID(targetID int, page, pageSize int, filter string) ([]assetdomain.Directory, int64, error)
	StreamByTargetID(targetID int) (*sql.Rows, error)
	CountByTargetID(targetID int) (int64, error)
	ScanRow(rows *sql.Rows) (*assetdomain.Directory, error)
}
