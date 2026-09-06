package application

import (
	"context"
	"testing"

	agentexecutionv1 "github.com/yyhuni/lunafox/contracts/gen/lunafox/agent/execution/v1"
	agentdomain "github.com/yyhuni/lunafox/server/internal/modules/agent/domain"
	scanapp "github.com/yyhuni/lunafox/server/internal/modules/scan/application"
)

type scanTaskBridgePortStub struct{}

func (scanTaskBridgePortStub) ClaimNextExecutionPlan(context.Context, int, string, int64, string, agentdomain.AgentExecutionCapabilitySnapshot) (*agentexecutionv1.ResolvedEngineExecutionPlan, error) {
	return nil, nil
}

func (scanTaskBridgePortStub) ReportTerminalTaskResult(context.Context, int, string, int64, int, string, *scanapp.FailureDetail) error {
	return nil
}

func (scanTaskBridgePortStub) ReportTerminalTaskResultWithDiagnostics(context.Context, int, string, int64, int, string, *scanapp.FailureDetail, *scanapp.EngineExecutionDiagnostics) error {
	return nil
}

func TestScanTaskBridgePortNamingContract(t *testing.T) {
	var _ ScanTaskBridgePort = scanTaskBridgePortStub{}
	var _ engineExecutionClaimBridgePort = scanTaskBridgePortStub{}
}
