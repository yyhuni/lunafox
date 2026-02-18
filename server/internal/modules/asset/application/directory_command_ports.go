package application

import assetdomain "github.com/yyhuni/lunafox/server/internal/modules/asset/domain"

type DirectoryCommandStore interface {
	BulkCreate(directories []assetdomain.Directory) (int, error)
	BulkDelete(ids []int) (int64, error)
	BulkUpsert(directories []assetdomain.Directory) (int64, error)
}

type DirectoryStore interface {
	DirectoryQueryStore
	DirectoryCommandStore
}
