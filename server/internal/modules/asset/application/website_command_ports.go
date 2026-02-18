package application

import assetdomain "github.com/yyhuni/lunafox/server/internal/modules/asset/domain"

type WebsiteCommandStore interface {
	GetByID(id int) (*assetdomain.Website, error)
	BulkCreate(websites []assetdomain.Website) (int, error)
	Delete(id int) error
	BulkDelete(ids []int) (int64, error)
	BulkUpsert(websites []assetdomain.Website) (int64, error)
}

type WebsiteStore interface {
	WebsiteQueryStore
	WebsiteCommandStore
}
