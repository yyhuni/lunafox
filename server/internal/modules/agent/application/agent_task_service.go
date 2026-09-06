package application

import (
	"context"

	agentexecutionv1 "github.com/yyhuni/lunafox/contracts/gen/lunafox/agent/execution/v1"
	agentdomain "github.com/yyhuni/lunafox/server/internal/modules/agent/domain"
	scanapp "github.com/yyhuni/lunafox/server/internal/modules/scan/application"
)

type engineExecutionClaimBridgePort interface {
	ClaimNextExecutionPlan(context.Context, int, string, int64, string, agentdomain.AgentExecutionCapabilitySnapshot) (*agentexecutionv1.ResolvedEngineExecutionPlan, error)
}

type agentSessionFenceBridgePort interface {
	FenceSupersededAgentSession(context.Context, int, int64) error
}

// AgentTaskService orchestrates saved-plan claims and terminal-task-result
// reporting for agent control-plane endpoints.
type AgentTaskService struct {
	taskBridge ScanTaskBridgePort
}

func (service *AgentTaskService) ClaimNextExecutionPlan(ctx context.Context, agentID int, sessionID string, sessionEpoch int64, requestID string, snapshot agentdomain.AgentExecutionCapabilitySnapshot) (*agentexecutionv1.ResolvedEngineExecutionPlan, error) {
	claims, ok := service.taskBridge.(engineExecutionClaimBridgePort)
	if !ok {
		return nil, scanapp.ErrEngineExecutionClaimUnavailable
	}
	return claims.ClaimNextExecutionPlan(ctx, agentID, sessionID, sessionEpoch, requestID, snapshot)
}

// FenceSupersededAgentSession closes lower-epoch leases before the control
// plane confirms a replacement session.
func (service *AgentTaskService) FenceSupersededAgentSession(ctx context.Context, agentID int, sessionEpoch int64) error {
	fence, ok := service.taskBridge.(agentSessionFenceBridgePort)
	if !ok {
		return scanapp.ErrEngineExecutionClaimUnavailable
	}
	return fence.FenceSupersededAgentSession(ctx, agentID, sessionEpoch)
}

func NewAgentTaskService(taskBridge ScanTaskBridgePort) *AgentTaskService {
	return &AgentTaskService{taskBridge: taskBridge}
}

// ReportTerminalTaskResult reports a terminal task result back through the scan-side bridge.
func (service *AgentTaskService) ReportTerminalTaskResult(ctx context.Context, agentID int, sessionID string, sessionEpoch int64, taskID int, result string, failure *scanapp.FailureDetail) error {
	return service.taskBridge.ReportTerminalTaskResult(ctx, agentID, sessionID, sessionEpoch, taskID, result, failure)
}

// ReportTerminalTaskResultWithDiagnostics preserves the required Engine
// evidence while adapting the Agent control plane to the scan task boundary.
func (service *AgentTaskService) ReportTerminalTaskResultWithDiagnostics(ctx context.Context, agentID int, sessionID string, sessionEpoch int64, taskID int, result string, failure *scanapp.FailureDetail, diagnostics *scanapp.EngineExecutionDiagnostics) error {
	return service.taskBridge.ReportTerminalTaskResultWithDiagnostics(ctx, agentID, sessionID, sessionEpoch, taskID, result, failure, diagnostics)
}
