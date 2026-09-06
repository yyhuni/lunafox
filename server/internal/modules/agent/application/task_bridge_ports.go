package application

import (
	"context"

	scanapp "github.com/yyhuni/lunafox/server/internal/modules/scan/application"
)

// ScanTaskBridgePort describes the scan task execution bridge dependency for agent task service.
// ClaimNextTaskAssignment names the scan-side claim action and intentionally
// does not mirror the transport RequestTask payload one-to-one.
type ScanTaskBridgePort interface {
	ReportTerminalTaskResult(ctx context.Context, agentID int, sessionID string, sessionEpoch int64, taskID int, result string, failure *scanapp.FailureDetail) error
	ReportTerminalTaskResultWithDiagnostics(ctx context.Context, agentID int, sessionID string, sessionEpoch int64, taskID int, result string, failure *scanapp.FailureDetail, diagnostics *scanapp.EngineExecutionDiagnostics) error
}
