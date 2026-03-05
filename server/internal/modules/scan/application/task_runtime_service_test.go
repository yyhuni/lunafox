package application

import (
	"context"
	"testing"
)

type taskStoreStub struct {
	pulledTask        *TaskRecord
	pulledTasks       []*TaskRecord
	pullIndex         int
	countActive       int
	failCalled        bool
	failedTask        int
	failedMessage     string
	failedRejectCause string
	unlockCalled      bool
	unlockScanID      int
	unlockStage       int
}

func (stub *taskStoreStub) GetByID(context.Context, int) (*TaskRecord, error) {
	return nil, nil
}

func (stub *taskStoreStub) PullTask(context.Context, int) (*TaskRecord, error) {
	if len(stub.pulledTasks) > 0 {
		if stub.pullIndex >= len(stub.pulledTasks) {
			return nil, nil
		}
		item := stub.pulledTasks[stub.pullIndex]
		stub.pullIndex++
		return item, nil
	}
	return stub.pulledTask, nil
}

func (stub *taskStoreStub) GetStatusCountsByScanID(context.Context, int) (int, int, int, int, int, error) {
	return 0, 0, 0, 0, 0, nil
}

func (stub *taskStoreStub) CountActiveByScanAndStage(context.Context, int, int) (int, error) {
	return stub.countActive, nil
}

func (stub *taskStoreStub) UpdateStatus(context.Context, int, string, string) error {
	return nil
}

func (stub *taskStoreStub) FailTaskClaim(_ context.Context, id int, errorMessage string, reason string) error {
	stub.failCalled = true
	stub.failedTask = id
	stub.failedMessage = errorMessage
	stub.failedRejectCause = reason
	return nil
}

func (stub *taskStoreStub) UnlockNextStage(_ context.Context, scanID, stage int) (int64, error) {
	stub.unlockCalled = true
	stub.unlockScanID = scanID
	stub.unlockStage = stage
	return 0, nil
}

type runtimeScanStoreStub struct {
	scan              *TaskScanRecord
	lastUpdatedStatus string
	lastUpdatedError  string
	updateCalls       int
}

func (stub *runtimeScanStoreStub) GetTaskRuntimeByID(int) (*TaskScanRecord, error) {
	return stub.scan, nil
}

func (stub *runtimeScanStoreStub) UpdateStatus(_ int, status string, errorMessage string) error {
	stub.updateCalls++
	stub.lastUpdatedStatus = status
	stub.lastUpdatedError = errorMessage
	return nil
}

func TestTaskRuntimeServiceUnlockNextStageIfReady_RespectsStageDependencyWhenActive(t *testing.T) {
	taskStore := &taskStoreStub{countActive: 1}
	service := NewTaskRuntimeService(taskStore, &runtimeScanStoreStub{})

	if err := service.unlockNextStageIfReady(context.Background(), 99, 2); err != nil {
		t.Fatalf("unlockNextStageIfReady returned error: %v", err)
	}
	if taskStore.unlockCalled {
		t.Fatalf("expected next stage stay blocked while current stage still active")
	}
}

func TestTaskRuntimeServiceUnlockNextStageIfReady_UnlocksWhenNoActiveTasks(t *testing.T) {
	taskStore := &taskStoreStub{countActive: 0}
	service := NewTaskRuntimeService(taskStore, &runtimeScanStoreStub{})

	if err := service.unlockNextStageIfReady(context.Background(), 100, 3); err != nil {
		t.Fatalf("unlockNextStageIfReady returned error: %v", err)
	}
	if !taskStore.unlockCalled {
		t.Fatalf("expected next stage unlocked when current stage drained")
	}
	if taskStore.unlockScanID != 100 || taskStore.unlockStage != 3 {
		t.Fatalf("unexpected unlock args scan=%d stage=%d", taskStore.unlockScanID, taskStore.unlockStage)
	}
}

func TestTaskRuntimeServicePullTask_CompatibleWorkflowReturnsAssignment(t *testing.T) {
	taskStore := &taskStoreStub{
		pulledTask: &TaskRecord{
			ID:           101,
			ScanID:       9,
			Stage:        1,
			WorkflowName: "subdomain_discovery",
			Config:       "recon:\n  enabled: false\n  tools:\n    subfinder:\n      enabled: false\n",
			Status:       "pending",
		},
	}
	runtimeStore := &runtimeScanStoreStub{
		scan: &TaskScanRecord{
			ID:       9,
			TargetID: 88,
			Status:   "running",
			YamlConfiguration: `
subdomain_discovery:
  recon:
    enabled: false
    tools:
      subfinder:
        enabled: false
  bruteforce:
    enabled: false
    tools:
      subdomain-bruteforce:
        enabled: false
  permutation:
    enabled: false
    tools:
      subdomain-permutation-resolve:
        enabled: false
  resolve:
    enabled: false
    tools:
      subdomain-resolve:
        enabled: false
`,
		},
	}
	service := NewTaskRuntimeService(taskStore, runtimeStore)

	assignment, err := service.PullTask(context.Background(), 1)
	if err != nil {
		t.Fatalf("PullTask returned unexpected error: %v", err)
	}
	if assignment == nil {
		t.Fatalf("expected task assignment")
	}
	if assignment.TaskID != 101 {
		t.Fatalf("unexpected task id: %d", assignment.TaskID)
	}
}

func TestTaskRuntimeServicePullTask_MissingTaskConfigSliceFailsFast(t *testing.T) {
	taskStore := &taskStoreStub{
		pulledTask: &TaskRecord{
			ID:           1001,
			ScanID:       45,
			Stage:        1,
			WorkflowName: "subdomain_discovery",
			Config:       "",
			Status:       "pending",
		},
	}
	runtimeStore := &runtimeScanStoreStub{
		scan: &TaskScanRecord{ID: 45, TargetID: 23, Status: "pending"},
	}
	service := NewTaskRuntimeService(taskStore, runtimeStore)

	assignment, err := service.PullTask(context.Background(), 9)
	if assignment != nil {
		t.Fatalf("expected no assignment")
	}
	workflowErr, ok := AsWorkflowError(err)
	if !ok {
		t.Fatalf("expected workflow error, got %v", err)
	}
	if workflowErr.Code != WorkflowErrorCodeWorkflowConfigInvalid {
		t.Fatalf("unexpected workflow code: %s", workflowErr.Code)
	}
	if !taskStore.failCalled {
		t.Fatalf("expected non-recoverable path to fail task claim")
	}
}

func TestTaskRuntimeServicePullTask_EmptyWorkflowNameFailsAsSchemaInvalid(t *testing.T) {
	taskStore := &taskStoreStub{
		pulledTask: &TaskRecord{
			ID:           1201,
			ScanID:       46,
			Stage:        1,
			WorkflowName: "",
			Config:       "recon:\n  enabled: false\n  tools:\n    subfinder:\n      enabled: false\n",
			Status:       "pending",
		},
	}
	runtimeStore := &runtimeScanStoreStub{
		scan: &TaskScanRecord{ID: 46, TargetID: 24, Status: "pending"},
	}
	service := NewTaskRuntimeService(taskStore, runtimeStore)

	assignment, err := service.PullTask(context.Background(), 10)
	if assignment != nil {
		t.Fatalf("expected no assignment")
	}
	workflowErr, ok := AsWorkflowError(err)
	if !ok {
		t.Fatalf("expected WorkflowError, got: %v", err)
	}
	if workflowErr.Code != WorkflowErrorCodeSchemaInvalid {
		t.Fatalf("unexpected code: %s", workflowErr.Code)
	}
	if workflowErr.Field != "workflow" {
		t.Fatalf("unexpected field: %s", workflowErr.Field)
	}
}

func TestTaskRuntimeServicePullTask_CompatiblePendingScanPromotesToRunning(t *testing.T) {
	taskStore := &taskStoreStub{
		pulledTask: &TaskRecord{
			ID:           1401,
			ScanID:       47,
			Stage:        1,
			WorkflowName: "subdomain_discovery",
			Config:       "recon:\n  enabled: false\n  tools:\n    subfinder:\n      enabled: false\n",
			Status:       "pending",
		},
	}
	runtimeStore := &runtimeScanStoreStub{
		scan: &TaskScanRecord{ID: 47, TargetID: 25, Status: "pending"},
	}
	service := NewTaskRuntimeService(taskStore, runtimeStore)

	assignment, err := service.PullTask(context.Background(), 13)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if assignment == nil {
		t.Fatalf("expected assignment")
	}
	if runtimeStore.lastUpdatedStatus != "running" {
		t.Fatalf("expected scan promoted to running, got %q", runtimeStore.lastUpdatedStatus)
	}
}
