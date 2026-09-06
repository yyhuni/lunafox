package application

import (
	"context"

	agentdomain "github.com/yyhuni/lunafox/server/internal/modules/agent/domain"
)

type AgentCommandStore interface {
	Create(ctx context.Context, agent *agentdomain.Agent) error
	GetByID(ctx context.Context, id int) (*agentdomain.Agent, error)
	Update(ctx context.Context, agent *agentdomain.Agent) error
	Delete(ctx context.Context, id int) error
}
