package application

type ScanTaskBridgeService struct {
	taskStore                ScanTaskStore
	scanTaskRuntimeScanStore ScanTaskRuntimeScanStore
	engineExecutionClaims    EngineExecutionClaimStore
	runtimeLogs              TaskRuntimeLogAppender
}

// WithTaskRuntimeLogAppender adds Server task evidence outside state commits.
func (service *ScanTaskBridgeService) WithTaskRuntimeLogAppender(appender TaskRuntimeLogAppender) *ScanTaskBridgeService {
	if appender == nil {
		panic("task runtime log appender is required")
	}
	service.runtimeLogs = appender
	return service
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
