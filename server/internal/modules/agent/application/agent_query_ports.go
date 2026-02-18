package application

import (
	"context"

	agentdomain "github.com/yyhuni/lunafox/server/internal/modules/agent/domain"
)

type AgentQueryStore interface {
	GetByID(ctx context.Context, id int) (*agentdomain.Agent, error)
	List(ctx context.Context, page, pageSize int, status string) ([]*agentdomain.Agent, int64, error)
}
