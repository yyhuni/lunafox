package application

import catalogdomain "github.com/yyhuni/lunafox/server/internal/modules/catalog/domain"

type EngineQueryStore interface {
	GetByID(id int) (*catalogdomain.ScanEngine, error)
	FindAll(page, pageSize int) ([]catalogdomain.ScanEngine, int64, error)
}
