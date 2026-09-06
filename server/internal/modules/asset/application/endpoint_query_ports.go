package application

import (
	"context"

	assetdomain "github.com/yyhuni/lunafox/server/internal/modules/asset/domain"
)

type EndpointQueryStore interface {
	ListByTargetID(targetID int, page, pageSize int, filter, orderBy string) ([]assetdomain.Endpoint, int64, error)
	ListFilterOptionsByTargetID(targetID int, field string) ([]assetdomain.FilterOption, error)
	GetByID(id int) (*assetdomain.Endpoint, error)
	ForEachByTargetID(ctx context.Context, targetID int, visit func(assetdomain.Endpoint) error) error
	CountByTargetID(targetID int) (int64, error)
}
