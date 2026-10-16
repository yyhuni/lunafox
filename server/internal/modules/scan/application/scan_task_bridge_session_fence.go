package application

import (
	"context"
	"fmt"

	"github.com/yyhuni/lunafox/server/internal/pkg"
	"go.uber.org/zap"
)

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
	service.appendSessionFenceRuntimeLogs(scanIDs, agentID, currentSessionEpoch)
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

func (service *ScanTaskBridgeService) appendSessionFenceRuntimeLogs(scanIDs []int, agentID int, currentSessionEpoch int64) {
	if service.runtimeLogs == nil {
		return
	}
	// The old Agent has lost upload authority. All committed-task lookups and
	// writes share one detached budget before session recovery continues.
	logCtx, cancel := context.WithTimeout(context.Background(), taskRuntimeLogWriteTimeout)
	defer cancel()
	events := make([]TaskRuntimeLogEvent, 0)
	for _, scanID := range scanIDs {
		if err := logCtx.Err(); err != nil {
			pkg.Error("session fence runtime log task lookup failed", zap.Int("scan.id", scanID), zap.Int("agent.id", agentID), zap.Error(err))
			return
		}
		tasks, err := service.taskStore.ListFailedByScanID(logCtx, scanID)
		if err != nil {
			pkg.Error("session fence runtime log task lookup failed", zap.Int("scan.id", scanID), zap.Int("agent.id", agentID), zap.Error(err))
			return
		}
		for index := range tasks {
			task := &tasks[index]
			if task.ScanID != scanID || task.Status != "failed" || task.AssignedAgentID == nil || *task.AssignedAgentID != agentID ||
				task.AssignedSessionEpoch == nil || *task.AssignedSessionEpoch <= 0 || *task.AssignedSessionEpoch >= currentSessionEpoch || task.Failure == nil || task.Failure.Kind != "agent_disconnected" {
				continue
			}
			epoch := *task.AssignedSessionEpoch
			content := fmt.Sprintf("[server] task state committed: failed; Agent %d assigned session epoch %d was superseded: %s", agentID, epoch, task.Failure.Message)
			if task.Failure.DisplayMessage != "" {
				content += "\n" + task.Failure.DisplayMessage
			}
			events = append(events, TaskRuntimeLogEvent{ScanID: scanID, TaskID: task.ID, EventID: fmt.Sprintf("session-fence:%d", epoch), Level: "error", Content: content})
		}
	}
	service.runtimeLogs.AppendTaskRuntimeLogs(logCtx, events)
}
