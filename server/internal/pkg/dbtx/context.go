// Package dbtx keeps GORM transaction handles inside infrastructure code while
// allowing repository calls that already accept context to share one database
// transaction. Application code only passes context.Context.
package dbtx

import (
	"context"

	"gorm.io/gorm"
)

type transactionKey struct{}

// WithTransaction attaches an infrastructure-owned transaction to ctx.
func WithTransaction(ctx context.Context, tx *gorm.DB) context.Context {
	return context.WithValue(ctx, transactionKey{}, tx)
}

// Resolve returns the transaction attached to ctx, or root when none exists.
func Resolve(ctx context.Context, root *gorm.DB) *gorm.DB {
	if ctx != nil {
		if tx, ok := ctx.Value(transactionKey{}).(*gorm.DB); ok && tx != nil {
			return tx
		}
	}
	return root
}
