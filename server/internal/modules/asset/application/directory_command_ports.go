package application

import (
	"context"

	assetdomain "github.com/yyhuni/lunafox/server/internal/modules/asset/domain"
)

type DirectoryCommandStore interface {
	BatchCreate(directories []assetdomain.Directory) (int, error)
	BatchDelete(ids []int) (int64, error)
	BatchUpsert(directories []assetdomain.Directory) (int64, error)
	BatchUpsertContext(context.Context, []assetdomain.Directory) (int64, error)
}

type DirectoryStore interface {
	DirectoryQueryStore
	DirectoryCommandStore
}
