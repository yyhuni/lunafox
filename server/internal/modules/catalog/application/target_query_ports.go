package application

import catalogdomain "github.com/yyhuni/lunafox/server/internal/modules/catalog/domain"

type TargetQueryStore interface {
	GetActiveByID(id int) (*catalogdomain.Target, error)
	FindAll(page, pageSize int, targetType, filter string) ([]catalogdomain.Target, int64, error)
	GetAssetCounts(targetID int) (*catalogdomain.TargetAssetCounts, error)
	GetVulnerabilityCounts(targetID int) (*catalogdomain.VulnerabilityCounts, error)
}
