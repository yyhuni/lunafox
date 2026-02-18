package domain

import (
	"context"
	"time"
)

// AgentRepository defines persistence port for agents.
type AgentRepository interface {
	Create(ctx context.Context, agent *Agent) error
	GetByID(ctx context.Context, id int) (*Agent, error)
	FindByAPIKey(ctx context.Context, apiKey string) (*Agent, error)
	List(ctx context.Context, page, pageSize int, status string) ([]*Agent, int64, error)
	FindStaleOnline(ctx context.Context, before time.Time) ([]*Agent, error)
	Update(ctx context.Context, agent *Agent) error
	UpdateStatus(ctx context.Context, id int, status string) error
	UpdateHeartbeat(ctx context.Context, id int, update AgentHeartbeatUpdate) error
	Delete(ctx context.Context, id int) error
}

// RegistrationTokenRepository defines persistence port for registration tokens.
type RegistrationTokenRepository interface {
	Create(ctx context.Context, token *RegistrationToken) error
	FindValid(ctx context.Context, token string, now time.Time) (*RegistrationToken, error)
	DeleteExpired(ctx context.Context, now time.Time) error
}
