package catalogwiring

import (
	"context"
	"fmt"

	catalogapp "github.com/yyhuni/lunafox/server/internal/modules/catalog/application"
	"github.com/yyhuni/lunafox/server/internal/pkg/dbtx"
	"gorm.io/gorm"
)

// catalogTransactionCoordinator owns the same-database commit boundary for a
// target batch and its organization relationships.
type catalogTransactionCoordinator struct {
	db *gorm.DB
}

func newCatalogTransactionCoordinator(db *gorm.DB) catalogapp.TransactionCoordinator {
	if db == nil {
		panic("catalog transaction database is required")
	}
	return &catalogTransactionCoordinator{db: db}
}

func (coordinator *catalogTransactionCoordinator) WithinTransaction(ctx context.Context, callback func(context.Context) error) error {
	if ctx == nil {
		return context.Canceled
	}
	if callback == nil {
		return fmt.Errorf("catalog transaction callback is required")
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	return coordinator.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		txContext := dbtx.WithTransaction(ctx, tx)
		if err := txContext.Err(); err != nil {
			return err
		}
		if err := callback(txContext); err != nil {
			return err
		}
		// Do not let GORM commit after cancellation raced with the final write.
		return txContext.Err()
	})
}
