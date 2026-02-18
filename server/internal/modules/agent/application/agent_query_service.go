package application

import (
	"context"

	agentdomain "github.com/yyhuni/lunafox/server/internal/modules/agent/domain"
	"github.com/yyhuni/lunafox/server/internal/pkg/dberrors"
)

// AgentQueryService handles read-side use cases for agents.
type AgentQueryService struct {
	agentRepo AgentQueryStore
}

func NewAgentQueryService(agentRepo AgentQueryStore) *AgentQueryService {
	return &AgentQueryService{agentRepo: agentRepo}
}

func (service *AgentQueryService) ListAgents(ctx context.Context, page, pageSize int, status string) ([]*agentdomain.Agent, int64, error) {
	return service.agentRepo.List(ctx, page, pageSize, status)
}

func (service *AgentQueryService) GetAgent(ctx context.Context, id int) (*agentdomain.Agent, error) {
	agent, err := service.agentRepo.GetByID(ctx, id)
	if err != nil {
		if dberrors.IsRecordNotFound(err) {
			return nil, ErrAgentNotFound
		}
		return nil, err
	}
	return agent, nil
}
