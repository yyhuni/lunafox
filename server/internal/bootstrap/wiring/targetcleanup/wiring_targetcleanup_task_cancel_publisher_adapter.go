package targetcleanupwiring

import agentcontrol "github.com/yyhuni/lunafox/server/internal/grpc/agentcontrol"

type targetCleanupTaskCancelPublisherAdapter struct {
	publisher *agentcontrol.AgentControlEventPublisher
}

func newTargetCleanupTaskCancelPublisherAdapter(publisher *agentcontrol.AgentControlEventPublisher) *targetCleanupTaskCancelPublisherAdapter {
	return &targetCleanupTaskCancelPublisherAdapter{publisher: publisher}
}

func (adapter *targetCleanupTaskCancelPublisherAdapter) TrySendTaskCancel(agentID, scanID, taskID int) bool {
	return adapter.publisher.TrySendTaskCancel(agentID, scanID, taskID)
}
