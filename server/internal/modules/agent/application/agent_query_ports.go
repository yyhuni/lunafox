package application

import (
	"context"

	agentdomain "github.com/yyhuni/lunafox/server/internal/modules/agent/domain"
)

type AgentQueryStore interface {
	GetByID(ctx context.Context, id int) (*agentdomain.Agent, error)
	List(ctx context.Context, page, pageSize int, filter, orderBy string) ([]*agentdomain.Agent, int64, error)
	ListFilterOptions(ctx context.Context, field string) ([]agentdomain.FilterOption, error)
}
