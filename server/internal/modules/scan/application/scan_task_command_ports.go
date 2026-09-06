package application

import "context"

type ScanTaskCommandStore interface {
	RequireCurrentAgentExecutionSession(ctx context.Context, agentID int, sessionID string, sessionEpoch int64) error
	CommitScanTaskTerminalStatusForSession(ctx context.Context, id, agentID int, sessionID string, sessionEpoch int64, status string, failure *FailureDetail) (bool, error)
	ListTerminalTasksPendingReconciliation(ctx context.Context, afterTaskID, limit int) ([]ScanTaskRecord, error)
	ListSupersededSessionTerminalTasksPendingReconciliation(ctx context.Context, agentID int, currentSessionEpoch int64, limit int) ([]ScanTaskRecord, error)
	ClearTerminalTaskReconciliationPending(ctx context.Context, taskID int) error
	FailClaimedTask(ctx context.Context, id int, failure *FailureDetail) error
	SkipUnstartedTasksByScanID(ctx context.Context, scanID int, reason string) (int64, error)
	CancelUnstartedTasksByScanID(ctx context.Context, scanID int) (int64, error)
	UnlockNextStageOrder(ctx context.Context, scanID, scanWorkflowStageOrder int) (int64, error)
	FailTasksForSupersededAgentSession(ctx context.Context, agentID int, currentSessionEpoch int64) ([]int, error)
}

// EngineDiagnosticTerminalTaskStore is the strict Agent-control terminal
// write capability. Keeping it separate makes Server-originated lifecycle
// transitions explicitly persist an unavailable snapshot instead of pretending
// they came from the Engine side channel.
type EngineDiagnosticTerminalTaskStore interface {
	CommitScanTaskTerminalStatusWithDiagnosticsForSession(ctx context.Context, id, agentID int, sessionID string, sessionEpoch int64, status string, failure *FailureDetail, diagnostics *EngineExecutionDiagnostics) (bool, error)
}
