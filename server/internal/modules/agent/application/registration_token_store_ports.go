package application

import (
	"context"
	"time"

	agentdomain "github.com/yyhuni/lunafox/server/internal/modules/agent/domain"
)

type RegistrationTokenStore interface {
	Create(ctx context.Context, token *agentdomain.RegistrationToken) error
	FindValid(ctx context.Context, token string, now time.Time) (*agentdomain.RegistrationToken, error)
	DeleteExpired(ctx context.Context, now time.Time) error
}
