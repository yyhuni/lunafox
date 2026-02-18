package application

import catalogdomain "github.com/yyhuni/lunafox/server/internal/modules/catalog/domain"

type EngineCommandStore interface {
	GetByID(id int) (*catalogdomain.ScanEngine, error)
	ExistsByName(name string, excludeID ...int) (bool, error)
	Create(engine *catalogdomain.ScanEngine) error
	Update(engine *catalogdomain.ScanEngine) error
	Delete(id int) error
}
