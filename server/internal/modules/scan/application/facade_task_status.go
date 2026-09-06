package application

import "context"

// ReportTerminalTaskResult validates and records the terminal task result reported by an agent.
func (service *ScanTaskFacade) ReportTerminalTaskResult(ctx context.Context, agentID int, sessionID string, sessionEpoch int64, taskID int, result string, failure *FailureDetail) error {
	err := service.taskBridgeService.ReportTerminalTaskResult(ctx, agentID, sessionID, sessionEpoch, taskID, result, failure)
	if err != nil {
		return mapTaskStatusBoundaryError(err)
	}
	return nil
}

func (service *ScanTaskFacade) ReportTerminalTaskResultWithDiagnostics(ctx context.Context, agentID int, sessionID string, sessionEpoch int64, taskID int, result string, failure *FailureDetail, diagnostics *EngineExecutionDiagnostics) error {
	err := service.taskBridgeService.ReportTerminalTaskResultWithDiagnostics(ctx, agentID, sessionID, sessionEpoch, taskID, result, failure, diagnostics)
	if err != nil {
		return mapTaskStatusBoundaryError(err)
	}
	return nil
}
