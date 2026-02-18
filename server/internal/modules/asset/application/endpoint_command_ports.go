package application

import assetdomain "github.com/yyhuni/lunafox/server/internal/modules/asset/domain"

type EndpointCommandStore interface {
	GetByID(id int) (*assetdomain.Endpoint, error)
	BulkCreate(endpoints []assetdomain.Endpoint) (int, error)
	Delete(id int) error
	BulkDelete(ids []int) (int64, error)
	BulkUpsert(endpoints []assetdomain.Endpoint) (int64, error)
}

type EndpointStore interface {
	EndpointQueryStore
	EndpointCommandStore
}
