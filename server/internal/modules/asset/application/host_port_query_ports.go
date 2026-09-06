package application

import (
	"context"

	assetdomain "github.com/yyhuni/lunafox/server/internal/modules/asset/domain"
)

type HostPortQueryStore interface {
	GetIPAggregation(targetID int, page, pageSize int, filter, orderBy string) ([]assetdomain.IPAggregationRow, int64, error)
	GetHostsAndPortsByIP(targetID int, ip string, filter string) ([]string, []int, error)
	ListPortOptionsByTargetID(targetID int) ([]assetdomain.FilterOption, error)
	ForEachByTargetID(ctx context.Context, targetID int, visit func(assetdomain.HostPort) error) error
	ForEachByTargetIDAndIPs(ctx context.Context, targetID int, ips []string, visit func(assetdomain.HostPort) error) error
	CountByTargetID(targetID int) (int64, error)
}
