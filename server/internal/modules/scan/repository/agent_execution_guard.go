package repository

import (
	"context"
	"time"
)

// AgentExecutionGuard binds task claims and identity changes to current entitlement.
// Its transaction callback carries the same database transaction through dbtx.
type AgentExecutionGuard interface {
	WithinClaim(context.Context, func(context.Context) error) error
	WithinIdentityChange(context.Context, func(context.Context) error) error
	ExecutionEligibilityAt(context.Context, int, time.Time) (string, error)
}
