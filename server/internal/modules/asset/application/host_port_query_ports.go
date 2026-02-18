package application

import (
	"database/sql"

	assetdomain "github.com/yyhuni/lunafox/server/internal/modules/asset/domain"
)

type HostPortQueryStore interface {
	GetIPAggregation(targetID int, page, pageSize int, filter string) ([]assetdomain.IPAggregationRow, int64, error)
	GetHostsAndPortsByIP(targetID int, ip string, filter string) ([]string, []int, error)
	StreamByTargetID(targetID int) (*sql.Rows, error)
	StreamByTargetIDAndIPs(targetID int, ips []string) (*sql.Rows, error)
	CountByTargetID(targetID int) (int64, error)
	ScanRow(rows *sql.Rows) (*assetdomain.HostPort, error)
}
