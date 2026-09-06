package application

type ScanTaskBridgeService struct {
	taskStore                ScanTaskStore
	scanTaskRuntimeScanStore ScanTaskRuntimeScanStore
	engineExecutionClaims    EngineExecutionClaimStore
}

func NewScanTaskBridgeService(
	taskStore ScanTaskStore,
	scanTaskRuntimeScanStore ScanTaskRuntimeScanStore,
) *ScanTaskBridgeService {
	if taskStore == nil {
		panic("scan task store is required")
	}
	if scanTaskRuntimeScanStore == nil {
		panic("scan runtime scan store is required")
	}
	return &ScanTaskBridgeService{taskStore: taskStore, scanTaskRuntimeScanStore: scanTaskRuntimeScanStore}
}
