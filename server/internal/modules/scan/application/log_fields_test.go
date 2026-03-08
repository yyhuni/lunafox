package application

import (
	"context"
	"testing"

	scandomain "github.com/yyhuni/lunafox/server/internal/modules/scan/domain"
	pkglogger "github.com/yyhuni/lunafox/server/internal/pkg"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
)

func withObservedLogger(t *testing.T) *observer.ObservedLogs {
	t.Helper()
	core, logs := observer.New(zap.DebugLevel)
	logger := zap.New(core)
	previousLogger := pkglogger.Logger
	previousSugar := pkglogger.Sugar
	pkglogger.Logger = logger
	pkglogger.Sugar = logger.Sugar()
	t.Cleanup(func() {
		pkglogger.Logger = previousLogger
		pkglogger.Sugar = previousSugar
	})
	return logs
}

func TestScanTaskFacadePullTask_LogsSemanticFields(t *testing.T) {
	logs := withObservedLogger(t)
	taskStore := &taskStoreStub{pulledTask: &TaskRecord{
		ID:         101,
		ScanID:     9,
		Stage:      1,
		WorkflowID: "subdomain_discovery",
		WorkflowConfig: map[string]any{
			"recon": map[string]any{"enabled": false},
		},
		Status: "pending",
	}}
	runtimeStore := &runtimeScanStoreStub{scan: &TaskScanRecord{ID: 9, TargetID: 88, Status: "running"}}
	service := NewScanTaskFacade(taskStore, runtimeStore)

	assignment, err := service.PullTask(context.Background(), 7)
	if err != nil {
		t.Fatalf("PullTask returned unexpected error: %v", err)
	}
	if assignment == nil {
		t.Fatalf("expected task assignment")
	}

	entries := logs.FilterMessage("Task assigned to agent").All()
	if len(entries) != 1 {
		t.Fatalf("expected 1 task assignment log, got %d", len(entries))
	}
	ctx := entries[0].ContextMap()
	for _, key := range []string{"task.id", "agent.id", "scan.id", "workflow.id"} {
		if _, ok := ctx[key]; !ok {
			t.Fatalf("expected %s field, got %v", key, ctx)
		}
	}
	for _, key := range []string{"task_id", "agent_id", "scan_id", "workflow_id"} {
		if _, ok := ctx[key]; ok {
			t.Fatalf("expected legacy %s field removed, got %v", key, ctx)
		}
	}
}

func TestLifecycleServiceStopActiveForDelete_LogsSemanticFields(t *testing.T) {
	logs := withObservedLogger(t)
	store := &lifecycleScanStoreStub{updateStatusErr: scandomain.ErrScanCannotStop}
	service := NewLifecycleService(store, nil, nil, nil)

	_, err := service.stopActiveForDelete(context.Background(), &QueryScan{ID: 1001, Status: string(scandomain.ScanStatusRunning)})
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	entries := logs.FilterMessage("ignoring stop error during scan deletion").All()
	if len(entries) != 1 {
		t.Fatalf("expected 1 lifecycle warning log, got %d", len(entries))
	}
	ctx := entries[0].ContextMap()
	if _, ok := ctx["scan.id"]; !ok {
		t.Fatalf("expected scan.id field, got %v", ctx)
	}
	if _, ok := ctx["scan.status"]; !ok {
		t.Fatalf("expected scan.status field, got %v", ctx)
	}
	if _, ok := ctx["scan_id"]; ok {
		t.Fatalf("expected legacy scan_id field removed, got %v", ctx)
	}
	if _, ok := ctx["scan_status"]; ok {
		t.Fatalf("expected legacy scan_status field removed, got %v", ctx)
	}
}
