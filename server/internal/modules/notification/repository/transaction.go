package repository

import (
	"context"
	"fmt"

	"github.com/yyhuni/lunafox/server/internal/pkg/dbtx"
	"gorm.io/gorm"
)

// TransactionCoordinator supplies the materializer with one database-only
// transaction boundary. Network effects deliberately cannot enter this API.
type TransactionCoordinator struct {
	db *gorm.DB
}

// NewTransactionCoordinator creates a notification materialization boundary.
func NewTransactionCoordinator(db *gorm.DB) *TransactionCoordinator {
	if db == nil {
		panic("notification transaction database is required")
	}
	return &TransactionCoordinator{db: db}
}

// WithinTransaction attaches the transaction to context so every repository
// participating in materialization resolves the same commit boundary.
func (coordinator *TransactionCoordinator) WithinTransaction(ctx context.Context, callback func(context.Context) error) error {
	if callback == nil {
		return fmt.Errorf("notification transaction callback is required")
	}
	return coordinator.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return callback(dbtx.WithTransaction(ctx, tx))
	})
}
