package application

import (
	"context"
	"time"

	agentdomain "github.com/yyhuni/lunafox/server/internal/modules/agent/domain"
)

type AgentClusterSummaryStore interface {
	GetClusterAggregate(ctx context.Context, generatedAt time.Time, agentLimit int) (agentdomain.AgentClusterAggregate, error)
}
