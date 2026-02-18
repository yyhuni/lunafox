package application

import (
	"context"
	"errors"
)

// UpdateStatus validates and updates task status for an agent.
func (service *ScanTaskFacade) UpdateStatus(ctx context.Context, agentID, taskID int, status, errorMessage string) error {
	err := service.runtimeService.UpdateStatus(ctx, agentID, taskID, status, errorMessage)
	if err != nil {
		if errors.Is(err, ErrTaskNotFound) {
			return ErrScanTaskNotFound
		}
		if errors.Is(err, ErrTaskNotOwned) {
			return ErrScanTaskNotOwned
		}
		if errors.Is(err, ErrTaskInvalidTransition) {
			return ErrScanTaskInvalidTransition
		}
		if errors.Is(err, ErrTaskInvalidUpdate) {
			return ErrScanTaskInvalidUpdate
		}
		return err
	}
	return nil
}
