package application

import "context"

// RecalculateScanStatus refreshes the scan aggregate from persisted task
// statuses after a recovery path terminalizes tasks outside agent result flow.
func (service *ScanTaskFacade) RecalculateScanStatus(ctx context.Context, scanID int) error {
	return service.taskBridgeService.recalculateScanStatus(ctx, scanID)
}
