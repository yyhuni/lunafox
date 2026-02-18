package application

import catalogdomain "github.com/yyhuni/lunafox/server/internal/modules/catalog/domain"

type TargetCommandStore interface {
	GetActiveByID(id int) (*catalogdomain.Target, error)
	ExistsByName(name string, excludeID ...int) (bool, error)
	Create(target *catalogdomain.Target) error
	Update(target *catalogdomain.Target) error
	SoftDelete(id int) error
	BulkSoftDelete(ids []int) (int64, error)
	BulkCreateIgnoreConflicts(targets []catalogdomain.Target) (int, error)
	FindByNames(names []string) ([]catalogdomain.Target, error)
}
