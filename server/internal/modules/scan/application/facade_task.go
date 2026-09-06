package application

type ScanTaskFacade struct{ taskBridgeService *ScanTaskBridgeService }

func NewScanTaskFacade(taskBridgeService *ScanTaskBridgeService) *ScanTaskFacade {
	return &ScanTaskFacade{taskBridgeService: taskBridgeService}
}
