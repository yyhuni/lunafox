package application

import (
	"context"
)

// AdminPasswordResetStore owns the password/token-version transaction.
type AdminPasswordResetStore interface {
	ResetAdminPassword(ctx context.Context, hashedPassword string) error
}
