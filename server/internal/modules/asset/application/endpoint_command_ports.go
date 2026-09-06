package application

import assetdomain "github.com/yyhuni/lunafox/server/internal/modules/asset/domain"

type EndpointCommandStore interface {
	GetByID(id int) (*assetdomain.Endpoint, error)
	BatchCreate(endpoints []assetdomain.Endpoint) (int, error)
	Delete(id int) error
	BatchDelete(ids []int) (int64, error)
	BatchUpsert(endpoints []assetdomain.Endpoint) (int64, error)
}

type EndpointStore interface {
	EndpointQueryStore
	EndpointCommandStore
}
