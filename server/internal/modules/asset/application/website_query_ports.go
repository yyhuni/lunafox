package application

import (
	"database/sql"

	assetdomain "github.com/yyhuni/lunafox/server/internal/modules/asset/domain"
)

type WebsiteQueryStore interface {
	FindByTargetID(targetID int, page, pageSize int, filter string) ([]assetdomain.Website, int64, error)
	StreamByTargetID(targetID int) (*sql.Rows, error)
	CountByTargetID(targetID int) (int64, error)
	ScanRow(rows *sql.Rows) (*assetdomain.Website, error)
}
