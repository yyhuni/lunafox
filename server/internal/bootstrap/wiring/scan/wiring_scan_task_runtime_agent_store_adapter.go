package scanwiring

import (
	"context"

	agentdomain "github.com/yyhuni/lunafox/server/internal/modules/agent/domain"
	scanapp "github.com/yyhuni/lunafox/server/internal/modules/scan/application"
)

type scanTaskRuntimeAgentStoreAdapter struct{ repo agentdomain.AgentRepository }

func newScanTaskRuntimeAgentStoreAdapter(repo agentdomain.AgentRepository) *scanTaskRuntimeAgentStoreAdapter {
	return &scanTaskRuntimeAgentStoreAdapter{repo: repo}
}

func (adapter *scanTaskRuntimeAgentStoreAdapter) GetTaskRuntimeAgentByID(ctx context.Context, id int) (*scanapp.TaskRuntimeAgentRecord, error) {
	agent, err := adapter.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if agent == nil {
		return nil, nil
	}
	return &scanapp.TaskRuntimeAgentRecord{
		ID:            agent.ID,
		AgentVersion:  agent.AgentVersion,
		WorkerVersion: agent.WorkerVersion,
		Status:        agent.Status,
	}, nil
}
