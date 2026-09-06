package application

import (
	"context"

	catalogdomain "github.com/yyhuni/lunafox/server/internal/modules/catalog/domain"
)

type TargetCommandStore interface {
	GetActiveByID(id int) (*catalogdomain.Target, error)
	ExistsByName(name string, excludeID ...int) (bool, error)
	Create(target *catalogdomain.Target) error
	Update(target *catalogdomain.Target) error
	SoftDelete(id int) error
	BatchSoftDelete(ids []int) (int64, error)
	TombstoneAndEnsureCleanup(ctx context.Context, id int) (bool, error)
	BatchTombstoneAndEnsureCleanup(ctx context.Context, ids []int) (int64, error)
	BatchCreateIgnoreConflicts(targets []catalogdomain.Target) (int, error)
	FindByNames(names []string) ([]catalogdomain.Target, error)
}

// TargetBatchCommandStore is the context-aware persistence surface used by
// the shared batch transaction. It deliberately stays narrower than the
// legacy command store so REST compatibility wrappers cannot drop context.
type TargetBatchCommandStore interface {
	BatchCreateIgnoreConflictsContext(context.Context, []catalogdomain.Target) (int, error)
	FindByNamesContext(context.Context, []string) ([]catalogdomain.Target, error)
}

// TransactionCoordinator owns the database transaction shared by target and
// organization repositories. The callback receives the transaction-bearing
// context and must perform no external side effects.
type TransactionCoordinator interface {
	WithinTransaction(context.Context, func(context.Context) error) error
}
