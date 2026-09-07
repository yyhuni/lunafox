package domain

import (
	"context"
	"time"
)

// AgentRepository defines persistence port for agents.
type AgentRepository interface {
	Create(ctx context.Context, agent *Agent) error
	GetByID(ctx context.Context, id int) (*Agent, error)
	FindByAuthenticationToken(ctx context.Context, authenticationToken string) (*Agent, error)
	List(ctx context.Context, page, pageSize int, filter, orderBy string) ([]*Agent, int64, error)
	ListFilterOptions(ctx context.Context, field string) ([]FilterOption, error)
	FindStaleOnline(ctx context.Context, before time.Time) ([]*Agent, error)
	Update(ctx context.Context, agent *Agent) error
	RecordConnection(ctx context.Context, agentID int, sourceIP string, connectedAt time.Time) (AgentConnectionObservation, error)
	ClearConnectionIP(ctx context.Context, agentID int) error
	UpdateStatus(ctx context.Context, id int, status string) error
	MarkOfflineFromOnline(ctx context.Context, id int) (bool, error)
	UpdateHeartbeat(ctx context.Context, id int, update AgentHeartbeatUpdate) error
	Delete(ctx context.Context, id int) error
}

// AgentClusterSummaryRepository produces complete-fleet aggregate facts without collection traversal.
type AgentClusterSummaryRepository interface {
	GetClusterAggregate(ctx context.Context, generatedAt time.Time) (AgentClusterAggregate, error)
}

type AgentLocationMapRepository interface {
	ListLocationMapAgents(ctx context.Context) ([]AgentLocationMapRecord, error)
}

// AgentOperationalRepository is the concrete module repository surface assembled by bootstrap.
type AgentOperationalRepository interface {
	AgentRepository
	AgentClusterSummaryRepository
	AgentLocationMapRepository
}

type FilterOption struct {
	Value string
	Label string
	Count int64
}

// RegistrationTokenRepository defines persistence port for registration tokens.
type RegistrationTokenRepository interface {
	Create(ctx context.Context, token *RegistrationToken) error
	FindValid(ctx context.Context, token string, now time.Time) (*RegistrationToken, error)
	GetResourceByID(ctx context.Context, id int) (*RegistrationTokenResource, error)
	DeleteNeverAttributedBefore(ctx context.Context, expiredBefore time.Time) error
}
