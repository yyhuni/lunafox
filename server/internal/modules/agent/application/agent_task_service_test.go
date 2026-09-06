package application

import (
	"context"
	"testing"

	agentexecutionv1 "github.com/yyhuni/lunafox/contracts/gen/lunafox/agent/execution/v1"
	agentdomain "github.com/yyhuni/lunafox/server/internal/modules/agent/domain"
	scanapp "github.com/yyhuni/lunafox/server/internal/modules/scan/application"
)

type scanTaskBridgeForAgentTaskServiceStub struct {
	claimCalls    int
	claimAgentID  int
	claimSession  string
	claimEpoch    int64
	claimRequest  string
	claimSnapshot agentdomain.AgentExecutionCapabilitySnapshot
	claimPlan     *agentexecutionv1.ResolvedEngineExecutionPlan
	claimErr      error
	reportCalls   int
	agentID       int
	sessionID     string
	sessionEpoch  int64
	taskID        int
	result        string
	failure       *scanapp.FailureDetail
	diagnostics   *scanapp.EngineExecutionDiagnostics
}

func (stub *scanTaskBridgeForAgentTaskServiceStub) ClaimNextExecutionPlan(_ context.Context, agentID int, sessionID string, sessionEpoch int64, requestID string, snapshot agentdomain.AgentExecutionCapabilitySnapshot) (*agentexecutionv1.ResolvedEngineExecutionPlan, error) {
	stub.claimCalls++
	stub.claimAgentID = agentID
	stub.claimSession = sessionID
	stub.claimEpoch = sessionEpoch
	stub.claimRequest = requestID
	stub.claimSnapshot = snapshot.Clone()
	return stub.claimPlan, stub.claimErr
}

func (stub *scanTaskBridgeForAgentTaskServiceStub) ReportTerminalTaskResult(_ context.Context, agentID int, sessionID string, sessionEpoch int64, taskID int, status string, failure *scanapp.FailureDetail) error {
	stub.reportCalls++
	stub.agentID = agentID
	stub.sessionID = sessionID
	stub.sessionEpoch = sessionEpoch
	stub.taskID = taskID
	stub.result = status
	stub.failure = failure
	return nil
}

func (stub *scanTaskBridgeForAgentTaskServiceStub) ReportTerminalTaskResultWithDiagnostics(_ context.Context, agentID int, sessionID string, sessionEpoch int64, taskID int, status string, failure *scanapp.FailureDetail, diagnostics *scanapp.EngineExecutionDiagnostics) error {
	stub.reportCalls++
	stub.agentID = agentID
	stub.sessionID = sessionID
	stub.sessionEpoch = sessionEpoch
	stub.taskID = taskID
	stub.result = status
	stub.failure = failure
	stub.diagnostics = diagnostics
	return nil
}

func TestAgentTaskServiceClaimNextExecutionPlanDelegatesSessionAndSnapshot(t *testing.T) {
	plan := &agentexecutionv1.ResolvedEngineExecutionPlan{Execution: "executions/7"}
	bridge := &scanTaskBridgeForAgentTaskServiceStub{claimPlan: plan}
	service := NewAgentTaskService(bridge)
	snapshot := agentdomain.AgentExecutionCapabilitySnapshot{
		AgentVersion:             "agent-test",
		ContainerRuntimeReady:    true,
		SupportedEngineAPIMajors: []uint32{2},
		RunningTasks:             1,
		TaskSlotsUsed:            2,
	}

	got, err := service.ClaimNextExecutionPlan(context.Background(), 7, "session-7", 11, "request-7", snapshot)
	if err != nil {
		t.Fatalf("ClaimNextExecutionPlan error: %v", err)
	}
	if got != plan {
		t.Fatalf("expected persisted plan pointer from bridge, got %#v", got)
	}
	if bridge.claimCalls != 1 || bridge.claimAgentID != 7 || bridge.claimSession != "session-7" || bridge.claimEpoch != 11 || bridge.claimRequest != "request-7" {
		t.Fatalf("unexpected claim scope: %+v", bridge)
	}
	if bridge.claimSnapshot.ContainerRuntimeReady != snapshot.ContainerRuntimeReady || len(bridge.claimSnapshot.SupportedEngineAPIMajors) != 1 || bridge.claimSnapshot.SupportedEngineAPIMajors[0] != 2 {
		t.Fatalf("unexpected capability snapshot: %+v", bridge.claimSnapshot)
	}
}

func TestAgentTaskServiceReportTerminalTaskResultDelegatesToBridge(t *testing.T) {
	bridge := &scanTaskBridgeForAgentTaskServiceStub{}
	service := NewAgentTaskService(bridge)

	failure := &scanapp.FailureDetail{Kind: "runtime_error", Message: "boom"}
	if err := service.ReportTerminalTaskResult(context.Background(), 7, "session-7", 11, 99, "failed", failure); err != nil {
		t.Fatalf("ReportTerminalTaskResult error: %v", err)
	}
	if bridge.reportCalls != 1 {
		t.Fatalf("expected one terminal task result report, got %d", bridge.reportCalls)
	}
	if bridge.agentID != 7 || bridge.sessionID != "session-7" || bridge.sessionEpoch != 11 || bridge.taskID != 99 || bridge.result != "failed" {
		t.Fatalf("unexpected report payload: %+v", bridge)
	}
	if bridge.failure == nil || bridge.failure.Kind != "runtime_error" {
		t.Fatalf("unexpected failure payload: %+v", bridge.failure)
	}
}

func TestAgentTaskServiceReportTerminalTaskResultWithDiagnosticsDelegatesToBridge(t *testing.T) {
	bridge := &scanTaskBridgeForAgentTaskServiceStub{}
	service := NewAgentTaskService(bridge)
	diagnostics := &scanapp.EngineExecutionDiagnostics{
		CompatibilityRevision: "engine-execution-diagnostics-r1",
		Availability:          "available",
		ResultState:           "complete",
	}

	if err := service.ReportTerminalTaskResultWithDiagnostics(context.Background(), 7, "session-7", 11, 99, "succeeded", nil, diagnostics); err != nil {
		t.Fatalf("ReportTerminalTaskResultWithDiagnostics error: %v", err)
	}
	if bridge.reportCalls != 1 {
		t.Fatalf("expected one diagnostic terminal task result report, got %d", bridge.reportCalls)
	}
	if bridge.agentID != 7 || bridge.sessionID != "session-7" || bridge.sessionEpoch != 11 || bridge.taskID != 99 || bridge.result != "succeeded" {
		t.Fatalf("unexpected diagnostic report payload: %+v", bridge)
	}
	if bridge.failure != nil || bridge.diagnostics != diagnostics {
		t.Fatalf("unexpected diagnostic terminal payload: failure=%+v diagnostics=%+v", bridge.failure, bridge.diagnostics)
	}
}
