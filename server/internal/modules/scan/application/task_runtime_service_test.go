package application

import (
	"context"
	"testing"
	"time"
)

type taskStoreStub struct {
	pulledTask     *TaskRecord
	pulledTasks    []*TaskRecord
	getByIDFn      func(context.Context, int) (*TaskRecord, error)
	listFailedFn   func(context.Context, int) ([]TaskRecord, error)
	pullIndex      int
	countActive    int
	pendingCount   int
	runningCount   int
	completedCount int
	failedCount    int
	cancelledCount int
	failCalled     bool
	failedTask     int
	failedFailure  *FailureDetail
	updatedFailure *FailureDetail
	updatedStatus  string
	updatedTaskID  int
	unlockCalled   bool
	unlockScanID   int
	unlockStage    int
}

func cloneFailureDetailForTest(failure *FailureDetail) *FailureDetail {
	if failure == nil {
		return nil
	}
	cloned := *failure
	return &cloned
}

func (stub *taskStoreStub) GetByID(ctx context.Context, id int) (*TaskRecord, error) {
	if stub.getByIDFn != nil {
		return stub.getByIDFn(ctx, id)
	}
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
	return stub.pendingCount, stub.runningCount, stub.completedCount, stub.failedCount, stub.cancelledCount, nil
}

func (stub *taskStoreStub) CountActiveByScanAndStage(context.Context, int, int) (int, error) {
	return stub.countActive, nil
}

func (stub *taskStoreStub) UpdateStatus(_ context.Context, id int, status string, failure *FailureDetail) error {
	stub.updatedTaskID = id
	stub.updatedStatus = status
	stub.updatedFailure = cloneFailureDetailForTest(failure)
	return nil
}

func (stub *taskStoreStub) FailTaskClaim(_ context.Context, id int, failure *FailureDetail) error {
	stub.failCalled = true
	stub.failedTask = id
	stub.failedFailure = cloneFailureDetailForTest(failure)
	return nil
}

func (stub *taskStoreStub) ListFailedByScanID(ctx context.Context, scanID int) ([]TaskRecord, error) {
	if stub.listFailedFn != nil {
		return stub.listFailedFn(ctx, scanID)
	}
	return nil, nil
}

func (stub *taskStoreStub) UnlockNextStage(_ context.Context, scanID, stage int) (int64, error) {
	stub.unlockCalled = true
	stub.unlockScanID = scanID
	stub.unlockStage = stage
	return 0, nil
}

type runtimeScanStoreStub struct {
	scan               *TaskScanRecord
	lastUpdatedStatus  string
	lastUpdatedFailure *FailureDetail
	updateCalls        int
}

func (stub *runtimeScanStoreStub) GetTaskRuntimeByID(int) (*TaskScanRecord, error) {
	return stub.scan, nil
}

func (stub *runtimeScanStoreStub) UpdateStatus(_ int, status string, failure *FailureDetail) error {
	stub.updateCalls++
	stub.lastUpdatedStatus = status
	stub.lastUpdatedFailure = cloneFailureDetailForTest(failure)
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
	taskStore := &taskStoreStub{pulledTask: &TaskRecord{
		ID:         101,
		ScanID:     9,
		Stage:      1,
		WorkflowID: "subdomain_discovery",
		WorkflowConfig: map[string]any{
			"recon": map[string]any{
				"enabled": false,
				"tools":   map[string]any{"subfinder": map[string]any{"enabled": false}},
			},
		},
		Status: "pending",
	}}
	runtimeStore := &runtimeScanStoreStub{scan: &TaskScanRecord{ID: 9, TargetID: 88, Status: "running"}}
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
	if assignment.WorkflowConfig == nil {
		t.Fatalf("expected workflow config object on assignment")
	}
}

func TestTaskRuntimeServicePullTask_MissingTaskConfigObjectFailsFast(t *testing.T) {
	taskStore := &taskStoreStub{pulledTask: &TaskRecord{
		ID:         1001,
		ScanID:     45,
		Stage:      1,
		WorkflowID: "subdomain_discovery",
		Status:     "pending",
	}}
	runtimeStore := &runtimeScanStoreStub{scan: &TaskScanRecord{ID: 45, TargetID: 23, Status: "pending"}}
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

func TestTaskRuntimeServicePullTask_EmptyWorkflowIDFailsAsSchemaInvalid(t *testing.T) {
	taskStore := &taskStoreStub{pulledTask: &TaskRecord{
		ID:         1201,
		ScanID:     46,
		Stage:      1,
		WorkflowID: "",
		WorkflowConfig: map[string]any{
			"recon": map[string]any{"enabled": false},
		},
		Status: "pending",
	}}
	runtimeStore := &runtimeScanStoreStub{scan: &TaskScanRecord{ID: 46, TargetID: 24, Status: "pending"}}
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
	if taskStore.failedFailure == nil || taskStore.failedFailure.Kind != "schema_invalid" {
		t.Fatalf("expected failure kind %q, got %+v", "schema_invalid", taskStore.failedFailure)
	}
}

func TestTaskRuntimeServiceUpdateStatus_RejectsFailedWithoutFailureMessage(t *testing.T) {
	service := NewTaskRuntimeService(&taskStoreStub{}, &runtimeScanStoreStub{})

	if err := service.UpdateStatus(context.Background(), 7, 303, "failed", &FailureDetail{Kind: "runtime_error"}); err != ErrTaskInvalidUpdate {
		t.Fatalf("expected ErrTaskInvalidUpdate, got %v", err)
	}
}

func TestTaskRuntimeServiceUpdateStatus_RejectsNonFailedWithFailure(t *testing.T) {
	service := NewTaskRuntimeService(&taskStoreStub{}, &runtimeScanStoreStub{})

	if err := service.UpdateStatus(context.Background(), 7, 303, "completed", &FailureDetail{Kind: "runtime_error", Message: "boom"}); err != ErrTaskInvalidUpdate {
		t.Fatalf("expected ErrTaskInvalidUpdate, got %v", err)
	}
}

func TestTaskRuntimeServiceUpdateStatus_PropagatesFailureObject(t *testing.T) {
	agentID := 7
	taskOwner := agentID
	taskStore := &taskStoreStub{pendingCount: 1}
	service := NewTaskRuntimeService(taskStore, &runtimeScanStoreStub{})
	taskStore.getByIDFn = func(context.Context, int) (*TaskRecord, error) {
		return &TaskRecord{ID: 303, ScanID: 99, Stage: 1, WorkflowID: "subdomain_discovery", Status: "running", AgentID: &taskOwner}, nil
	}

	failure := &FailureDetail{Kind: "runtime_error", Message: "boom"}
	err := service.UpdateStatus(context.Background(), agentID, 303, "failed", failure)
	if err != nil {
		t.Fatalf("UpdateStatus returned error: %v", err)
	}
	if taskStore.updatedFailure == nil || taskStore.updatedFailure.Kind != "runtime_error" || taskStore.updatedFailure.Message != "boom" {
		t.Fatalf("expected propagated failure, got %+v", taskStore.updatedFailure)
	}
}

func TestTaskRuntimeServiceRecalculateScanStatus_ProjectsCanonicalFailureByPriority(t *testing.T) {
	timeoutAt := time.Date(2026, 3, 9, 10, 0, 0, 0, time.UTC)
	schemaAt := timeoutAt.Add(1 * time.Minute)
	taskStore := &taskStoreStub{
		failedCount: 2,
		listFailedFn: func(context.Context, int) ([]TaskRecord, error) {
			return []TaskRecord{
				{ID: 2, ScanID: 88, Stage: 2, Status: "failed", CompletedAt: &timeoutAt, Failure: &FailureDetail{Kind: "task_timeout", Message: "task timed out"}},
				{ID: 1, ScanID: 88, Stage: 1, Status: "failed", CompletedAt: &schemaAt, Failure: &FailureDetail{Kind: "schema_invalid", Message: "schema invalid"}},
			}, nil
		},
	}
	runtimeStore := &runtimeScanStoreStub{}
	service := NewTaskRuntimeService(taskStore, runtimeStore)

	if err := service.recalculateScanStatus(context.Background(), 88); err != nil {
		t.Fatalf("recalculateScanStatus returned error: %v", err)
	}
	if runtimeStore.lastUpdatedStatus != "failed" {
		t.Fatalf("expected failed scan status, got %q", runtimeStore.lastUpdatedStatus)
	}
	if runtimeStore.lastUpdatedFailure == nil || runtimeStore.lastUpdatedFailure.Kind != "schema_invalid" {
		t.Fatalf("expected schema_invalid canonical failure, got %+v", runtimeStore.lastUpdatedFailure)
	}
}

func TestTaskRuntimeServiceRecalculateScanStatus_ProjectsCanonicalFailureStableTieBreak(t *testing.T) {
	later := time.Date(2026, 3, 9, 11, 1, 0, 0, time.UTC)
	earlier := time.Date(2026, 3, 9, 11, 0, 0, 0, time.UTC)
	taskStore := &taskStoreStub{
		failedCount: 3,
		listFailedFn: func(context.Context, int) ([]TaskRecord, error) {
			return []TaskRecord{
				{ID: 5, ScanID: 89, Stage: 3, Status: "failed", CompletedAt: &later, Failure: &FailureDetail{Kind: "runtime_error", Message: "later stage"}},
				{ID: 4, ScanID: 89, Stage: 2, Status: "failed", CompletedAt: &later, Failure: &FailureDetail{Kind: "runtime_error", Message: "earlier stage"}},
				{ID: 3, ScanID: 89, Stage: 2, Status: "failed", CompletedAt: &earlier, Failure: &FailureDetail{Kind: "runtime_error", Message: "earliest completion"}},
			}, nil
		},
	}
	runtimeStore := &runtimeScanStoreStub{}
	service := NewTaskRuntimeService(taskStore, runtimeStore)

	if err := service.recalculateScanStatus(context.Background(), 89); err != nil {
		t.Fatalf("recalculateScanStatus returned error: %v", err)
	}
	if runtimeStore.lastUpdatedFailure == nil || runtimeStore.lastUpdatedFailure.Message != "earliest completion" {
		t.Fatalf("expected stable canonical failure, got %+v", runtimeStore.lastUpdatedFailure)
	}
}

func TestWorkflowErrorCodeToFailureKind_MapsDomainCodes(t *testing.T) {
	if got := workflowErrorCodeToFailureKind(WorkflowErrorCodeSchemaInvalid); got != "schema_invalid" {
		t.Fatalf("expected schema_invalid, got %q", got)
	}
}

func TestTaskRuntimeServicePullTask_CompatiblePendingScanPromotesToRunning(t *testing.T) {
	taskStore := &taskStoreStub{pulledTask: &TaskRecord{
		ID:         1401,
		ScanID:     47,
		Stage:      1,
		WorkflowID: "subdomain_discovery",
		WorkflowConfig: map[string]any{
			"recon": map[string]any{"enabled": false},
		},
		Status: "pending",
	}}
	runtimeStore := &runtimeScanStoreStub{scan: &TaskScanRecord{ID: 47, TargetID: 25, Status: "pending"}}
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
	if runtimeStore.lastUpdatedFailure != nil {
		t.Fatalf("expected running scan to clear failure, got %+v", runtimeStore.lastUpdatedFailure)
	}
}
