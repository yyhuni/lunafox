package application

import (
	"context"

	assetdomain "github.com/yyhuni/lunafox/server/internal/modules/asset/domain"
)

type WebsiteCommandStore interface {
	GetByID(id int) (*assetdomain.Website, error)
	BatchCreateContext(context.Context, []assetdomain.Website) (int, error)
	Delete(id int) error
	BatchDelete(ids []int) (int64, error)
	BatchUpsertContext(context.Context, []assetdomain.Website) (int64, error)
	BatchUpsertTechnologyContext(context.Context, int, []assetdomain.WebsiteTechnology) (int64, error)
}

type WebsiteStore interface {
	WebsiteQueryStore
	WebsiteCommandStore
}
