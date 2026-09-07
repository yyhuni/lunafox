package application

import "context"

// FenceSupersededAgentSession synchronously closes every lower-epoch running
// lease before the replacement Agent session can receive SessionReady.
func (service *ScanTaskBridgeService) FenceSupersededAgentSession(ctx context.Context, agentID int, currentSessionEpoch int64) error {
	if service == nil || service.taskStore == nil || ctx == nil || agentID <= 0 || currentSessionEpoch <= 0 {
		return ErrScanTaskInvalidUpdate
	}
	scanIDs, err := service.taskStore.FailTasksForSupersededAgentSession(ctx, agentID, currentSessionEpoch)
	if err != nil {
		return err
	}
	for _, scanID := range scanIDs {
		if err := service.recalculateScanStatus(ctx, scanID); err != nil {
			return err
		}
	}
	// A superseded process may have committed a terminal task and crashed before
	// its process-local outbox received acknowledgement. Replay durable markers
	// before the replacement session is allowed to claim work.
	return service.reconcileSupersededSessionTerminalTasks(ctx, agentID, currentSessionEpoch)
}

// FenceSupersededAgentSession exposes the session-fencing use case through the
// scan task facade used by the Agent control plane.
func (service *ScanTaskFacade) FenceSupersededAgentSession(ctx context.Context, agentID int, currentSessionEpoch int64) error {
	return service.taskBridgeService.FenceSupersededAgentSession(ctx, agentID, currentSessionEpoch)
}
