package application

import (
	"context"

	assetdomain "github.com/yyhuni/lunafox/server/internal/modules/asset/domain"
)

type DirectoryQueryStore interface {
	ListByTargetID(targetID int, page, pageSize int, filter, orderBy string) ([]assetdomain.Directory, int64, error)
	ListFilterOptionsByTargetID(targetID int, field string) ([]assetdomain.FilterOption, error)
	ForEachByTargetID(ctx context.Context, targetID int, visit func(assetdomain.Directory) error) error
	CountByTargetID(targetID int) (int64, error)
}
