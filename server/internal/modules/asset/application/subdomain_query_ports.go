package application

import (
	"context"

	assetdomain "github.com/yyhuni/lunafox/server/internal/modules/asset/domain"
)

type SubdomainQueryStore interface {
	ListByTargetID(targetID int, page, pageSize int, filter, orderBy string) ([]assetdomain.Subdomain, int64, error)
	ForEachByTargetID(ctx context.Context, targetID int, visit func(assetdomain.Subdomain) error) error
	CountByTargetID(targetID int) (int64, error)
}
