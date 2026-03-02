package application

import (
	"context"

	agentdomain "github.com/yyhuni/lunafox/server/internal/modules/agent/domain"
	"github.com/yyhuni/lunafox/server/internal/pkg/dberrors"
)

// AgentCommandService handles write-side use cases for agents.
type AgentCommandService struct {
	agentRepo AgentCommandStore
}

func NewAgentCommandService(agentRepo AgentCommandStore) *AgentCommandService {
	return &AgentCommandService{agentRepo: agentRepo}
}

func (service *AgentCommandService) UpdateAgentConfig(ctx context.Context, id int, update agentdomain.AgentConfigUpdate) (*agentdomain.Agent, error) {
	agent, err := service.agentRepo.GetByID(ctx, id)
	if err != nil {
		if dberrors.IsRecordNotFound(err) {
			return nil, ErrAgentNotFound
		}
		return nil, err
	}

	agent.ApplyConfigUpdate(update)

	if err := service.agentRepo.Update(ctx, agent); err != nil {
		return nil, err
	}
	return agent, nil
}

func (service *AgentCommandService) DeleteAgent(ctx context.Context, id int) error {
	if err := service.agentRepo.Delete(ctx, id); err != nil {
		if dberrors.IsRecordNotFound(err) {
			return ErrAgentNotFound
		}
		return err
	}
	return nil
}

func BuildConfigUpdatePayload(agent *agentdomain.Agent) agentdomain.ConfigUpdatePayload {
	maxTasks := agent.MaxTasks
	cpu := agent.CPUThreshold
	mem := agent.MemThreshold
	disk := agent.DiskThreshold
	return agentdomain.ConfigUpdatePayload{
		MaxTasks:      &maxTasks,
		CPUThreshold:  &cpu,
		MemThreshold:  &mem,
		DiskThreshold: &disk,
	}
}
