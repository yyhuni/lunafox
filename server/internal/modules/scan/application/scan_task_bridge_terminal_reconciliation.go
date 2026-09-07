package application

import (
	"context"
	"errors"

	scandomain "github.com/yyhuni/lunafox/server/internal/modules/scan/domain"
)

const terminalReconciliationBatchSize = 256

// ReconcilePersistedTerminalTasks replays Server-owned workflow convergence
// for terminal commits whose Agent acknowledgement path was interrupted.
func (service *ScanTaskBridgeService) ReconcilePersistedTerminalTasks(ctx context.Context) error {
	if service == nil || service.taskStore == nil || ctx == nil {
		return ErrScanTaskInvalidUpdate
	}
	tasks, err := service.taskStore.ListTerminalTasksPendingReconciliation(ctx, 0, terminalReconciliationBatchSize)
	if err != nil {
		return err
	}
	var reconciliationErr error
	for index := range tasks {
		task := &tasks[index]
		status, ok := scandomain.ParseTaskStatus(task.Status)
		if !ok || !scandomain.IsTerminalTaskStatus(status) {
			reconciliationErr = errors.Join(reconciliationErr, ErrScanTaskInvalidTransition)
			continue
		}
		if err := service.reconcileAndClearTerminalTaskResult(ctx, task, status); err != nil {
			reconciliationErr = errors.Join(reconciliationErr, err)
		}
	}
	return reconciliationErr
}

func (service *ScanTaskBridgeService) reconcileSupersededSessionTerminalTasks(ctx context.Context, agentID int, currentSessionEpoch int64) error {
	for {
		tasks, err := service.taskStore.ListSupersededSessionTerminalTasksPendingReconciliation(ctx, agentID, currentSessionEpoch, terminalReconciliationBatchSize)
		if err != nil {
			return err
		}
		if len(tasks) == 0 {
			return nil
		}
		for index := range tasks {
			task := &tasks[index]
			status, ok := scandomain.ParseTaskStatus(task.Status)
			if !ok || !scandomain.IsTerminalTaskStatus(status) {
				return ErrScanTaskInvalidTransition
			}
			if err := service.reconcileAndClearTerminalTaskResult(ctx, task, status); err != nil {
				return err
			}
		}
	}
}

// ReconcilePersistedTerminalTasks exposes the durable recovery job boundary.
func (service *ScanTaskFacade) ReconcilePersistedTerminalTasks(ctx context.Context) error {
	return service.taskBridgeService.ReconcilePersistedTerminalTasks(ctx)
}
