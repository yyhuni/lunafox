package application

import (
	"database/sql"

	assetdomain "github.com/yyhuni/lunafox/server/internal/modules/asset/domain"
)

type EndpointQueryStore interface {
	FindByTargetID(targetID int, page, pageSize int, filter string) ([]assetdomain.Endpoint, int64, error)
	GetByID(id int) (*assetdomain.Endpoint, error)
	StreamByTargetID(targetID int) (*sql.Rows, error)
	CountByTargetID(targetID int) (int64, error)
	ScanRow(rows *sql.Rows) (*assetdomain.Endpoint, error)
}
