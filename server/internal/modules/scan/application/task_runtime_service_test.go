package application

import (
	"context"
	"testing"
)

type taskStoreStub struct {
	pulledTask    *TaskRecord
	releasedTask  int
	releaseCalled bool
}

func (stub *taskStoreStub) GetByID(context.Context, int) (*TaskRecord, error) {
	return nil, nil
}

func (stub *taskStoreStub) PullTask(context.Context, int) (*TaskRecord, error) {
	return stub.pulledTask, nil
}

func (stub *taskStoreStub) GetStatusCountsByScanID(context.Context, int) (int, int, int, int, int, error) {
	return 0, 0, 0, 0, 0, nil
}

func (stub *taskStoreStub) CountActiveByScanAndStage(context.Context, int, int) (int, error) {
	return 0, nil
}

func (stub *taskStoreStub) UpdateStatus(context.Context, int, string, string) error {
	return nil
}

func (stub *taskStoreStub) ReleaseTaskClaim(_ context.Context, id int) error {
	stub.releaseCalled = true
	stub.releasedTask = id
	return nil
}

func (stub *taskStoreStub) UnlockNextStage(context.Context, int, int) (int64, error) {
	return 0, nil
}

type runtimeScanStoreStub struct {
	scan *TaskScanRecord
}

func (stub *runtimeScanStoreStub) GetTaskRuntimeByID(int) (*TaskScanRecord, error) {
	return stub.scan, nil
}

func (stub *runtimeScanStoreStub) UpdateStatus(int, string, string) error {
	return nil
}

type fixedCompatibilityGate struct {
	compatible bool
}

func (gate fixedCompatibilityGate) Supports(context.Context, int, WorkflowVersionTuple) (bool, error) {
	return gate.compatible, nil
}

type taskRuntimeAgentStoreStub struct {
	agent *TaskRuntimeAgentRecord
	err   error
}

func (stub *taskRuntimeAgentStoreStub) GetTaskRuntimeAgentByID(context.Context, int) (*TaskRuntimeAgentRecord, error) {
	if stub.err != nil {
		return nil, stub.err
	}
	return stub.agent, nil
}

func TestTaskRuntimeServicePullTask_CompatibleTupleReturnsAssignment(t *testing.T) {
	taskStore := &taskStoreStub{
		pulledTask: &TaskRecord{
			ID:           101,
			ScanID:       9,
			Stage:        1,
			WorkflowName: "subdomain_discovery",
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
  apiVersion: v1
  schemaVersion: 1.0.0
`,
		},
	}
	service := NewTaskRuntimeServiceWithCompatibilityGate(taskStore, runtimeStore, fixedCompatibilityGate{compatible: true})

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
	if taskStore.releaseCalled {
		t.Fatalf("compatible task should not be released")
	}
}

func TestTaskRuntimeServicePullTask_IncompatibleTupleReturnsWorkflowError(t *testing.T) {
	taskStore := &taskStoreStub{
		pulledTask: &TaskRecord{
			ID:           202,
			ScanID:       18,
			Stage:        2,
			WorkflowName: "subdomain_discovery",
			Status:       "pending",
		},
	}
	runtimeStore := &runtimeScanStoreStub{
		scan: &TaskScanRecord{
			ID:       18,
			TargetID: 99,
			Status:   "running",
			YamlConfiguration: `
subdomain_discovery:
  apiVersion: v1
  schemaVersion: 1.0.0
`,
		},
	}
	service := NewTaskRuntimeServiceWithCompatibilityGate(taskStore, runtimeStore, fixedCompatibilityGate{compatible: false})

	assignment, err := service.PullTask(context.Background(), 1001)
	if assignment != nil {
		t.Fatalf("expected no assignment on incompatible tuple")
	}
	workflowErr, ok := AsWorkflowError(err)
	if !ok {
		t.Fatalf("expected WorkflowError, got: %v", err)
	}
	if workflowErr.Code != WorkflowErrorCodeWorkerVersionIncompatible {
		t.Fatalf("unexpected code: %s", workflowErr.Code)
	}
	if workflowErr.Stage != WorkflowErrorStageSchedulerCompatibility {
		t.Fatalf("unexpected stage: %s", workflowErr.Stage)
	}
	if !taskStore.releaseCalled || taskStore.releasedTask != 202 {
		t.Fatalf("expected task claim released, got called=%v id=%d", taskStore.releaseCalled, taskStore.releasedTask)
	}
}

func TestTaskRuntimeServicePullTask_MissingVersionReturnsSchemaInvalid(t *testing.T) {
	taskStore := &taskStoreStub{
		pulledTask: &TaskRecord{
			ID:           303,
			ScanID:       21,
			Stage:        2,
			WorkflowName: "subdomain_discovery",
			Status:       "pending",
		},
	}
	runtimeStore := &runtimeScanStoreStub{
		scan: &TaskScanRecord{
			ID:       21,
			TargetID: 100,
			Status:   "running",
			YamlConfiguration: `
subdomain_discovery:
  schemaVersion: 1.0.0
`,
		},
	}
	service := NewTaskRuntimeServiceWithCompatibilityGate(taskStore, runtimeStore, fixedCompatibilityGate{compatible: true})

	assignment, err := service.PullTask(context.Background(), 1002)
	if assignment != nil {
		t.Fatalf("expected no assignment when version field missing")
	}
	workflowErr, ok := AsWorkflowError(err)
	if !ok {
		t.Fatalf("expected WorkflowError, got: %v", err)
	}
	if workflowErr.Code != WorkflowErrorCodeSchemaInvalid {
		t.Fatalf("unexpected code: %s", workflowErr.Code)
	}
	if workflowErr.Stage != WorkflowErrorStageServerSchemaGate {
		t.Fatalf("unexpected stage: %s", workflowErr.Stage)
	}
	if workflowErr.Field != "apiVersion" {
		t.Fatalf("unexpected field: %s", workflowErr.Field)
	}
	if !taskStore.releaseCalled || taskStore.releasedTask != 303 {
		t.Fatalf("expected task claim released, got called=%v id=%d", taskStore.releaseCalled, taskStore.releasedTask)
	}
}

func TestTaskRuntimeServicePullTask_InvalidAPIVersionFormatReturnsSchemaInvalid(t *testing.T) {
	taskStore := &taskStoreStub{
		pulledTask: &TaskRecord{
			ID:           333,
			ScanID:       23,
			Stage:        2,
			WorkflowName: "subdomain_discovery",
			Status:       "pending",
		},
	}
	runtimeStore := &runtimeScanStoreStub{
		scan: &TaskScanRecord{
			ID:       23,
			TargetID: 101,
			Status:   "running",
			YamlConfiguration: `
subdomain_discovery:
  apiVersion: version1
  schemaVersion: 1.0.0
`,
		},
	}
	service := NewTaskRuntimeServiceWithCompatibilityGate(taskStore, runtimeStore, fixedCompatibilityGate{compatible: true})

	assignment, err := service.PullTask(context.Background(), 1003)
	if assignment != nil {
		t.Fatalf("expected no assignment when apiVersion format invalid")
	}
	workflowErr, ok := AsWorkflowError(err)
	if !ok {
		t.Fatalf("expected WorkflowError, got: %v", err)
	}
	if workflowErr.Code != WorkflowErrorCodeSchemaInvalid {
		t.Fatalf("unexpected code: %s", workflowErr.Code)
	}
	if workflowErr.Stage != WorkflowErrorStageServerSchemaGate {
		t.Fatalf("unexpected stage: %s", workflowErr.Stage)
	}
	if workflowErr.Field != "apiVersion" {
		t.Fatalf("unexpected field: %s", workflowErr.Field)
	}
	if !taskStore.releaseCalled || taskStore.releasedTask != 333 {
		t.Fatalf("expected task claim released, got called=%v id=%d", taskStore.releaseCalled, taskStore.releasedTask)
	}
}

func TestTaskRuntimeServicePullTask_InvalidSchemaVersionFormatReturnsSchemaInvalid(t *testing.T) {
	taskStore := &taskStoreStub{
		pulledTask: &TaskRecord{
			ID:           363,
			ScanID:       26,
			Stage:        2,
			WorkflowName: "subdomain_discovery",
			Status:       "pending",
		},
	}
	runtimeStore := &runtimeScanStoreStub{
		scan: &TaskScanRecord{
			ID:       26,
			TargetID: 102,
			Status:   "running",
			YamlConfiguration: `
subdomain_discovery:
  apiVersion: v1
  schemaVersion: 1.0
`,
		},
	}
	service := NewTaskRuntimeServiceWithCompatibilityGate(taskStore, runtimeStore, fixedCompatibilityGate{compatible: true})

	assignment, err := service.PullTask(context.Background(), 1004)
	if assignment != nil {
		t.Fatalf("expected no assignment when schemaVersion format invalid")
	}
	workflowErr, ok := AsWorkflowError(err)
	if !ok {
		t.Fatalf("expected WorkflowError, got: %v", err)
	}
	if workflowErr.Code != WorkflowErrorCodeSchemaInvalid {
		t.Fatalf("unexpected code: %s", workflowErr.Code)
	}
	if workflowErr.Stage != WorkflowErrorStageServerSchemaGate {
		t.Fatalf("unexpected stage: %s", workflowErr.Stage)
	}
	if workflowErr.Field != "schemaVersion" {
		t.Fatalf("unexpected field: %s", workflowErr.Field)
	}
	if !taskStore.releaseCalled || taskStore.releasedTask != 363 {
		t.Fatalf("expected task claim released, got called=%v id=%d", taskStore.releaseCalled, taskStore.releasedTask)
	}
}

func TestTaskRuntimeServicePullTask_DynamicCapabilityByAgentVersion(t *testing.T) {
	taskStore := &taskStoreStub{
		pulledTask: &TaskRecord{
			ID:           404,
			ScanID:       30,
			Stage:        1,
			WorkflowName: "subdomain_discovery",
			Status:       "pending",
		},
	}
	runtimeStore := &runtimeScanStoreStub{
		scan: &TaskScanRecord{
			ID:       30,
			TargetID: 77,
			Status:   "running",
			YamlConfiguration: `
subdomain_discovery:
  apiVersion: v1
  schemaVersion: 1.0.0
`,
		},
	}
	agentStore := &taskRuntimeAgentStoreStub{
		agent: &TaskRuntimeAgentRecord{
			ID:      1,
			Version: "1.0.0",
			Status:  "online",
		},
	}
	service := NewTaskRuntimeServiceWithAgentStore(taskStore, runtimeStore, agentStore)

	assignment, err := service.PullTask(context.Background(), 1)
	if err != nil {
		t.Fatalf("PullTask returned unexpected error: %v", err)
	}
	if assignment == nil {
		t.Fatalf("expected assignment for compatible agent version")
	}
	if taskStore.releaseCalled {
		t.Fatalf("compatible task should not be released")
	}
}

func TestTaskRuntimeServicePullTask_DynamicCapabilityVersionMismatch(t *testing.T) {
	taskStore := &taskStoreStub{
		pulledTask: &TaskRecord{
			ID:           505,
			ScanID:       40,
			Stage:        1,
			WorkflowName: "subdomain_discovery",
			Status:       "pending",
		},
	}
	runtimeStore := &runtimeScanStoreStub{
		scan: &TaskScanRecord{
			ID:       40,
			TargetID: 18,
			Status:   "running",
			YamlConfiguration: `
subdomain_discovery:
  apiVersion: v1
  schemaVersion: 1.0.0
`,
		},
	}
	agentStore := &taskRuntimeAgentStoreStub{
		agent: &TaskRuntimeAgentRecord{
			ID:      2,
			Version: "0.9.0",
			Status:  "online",
		},
	}
	service := NewTaskRuntimeServiceWithAgentStore(taskStore, runtimeStore, agentStore)

	assignment, err := service.PullTask(context.Background(), 2)
	if assignment != nil {
		t.Fatalf("expected no assignment when agent capability mismatches")
	}
	workflowErr, ok := AsWorkflowError(err)
	if !ok {
		t.Fatalf("expected WorkflowError, got: %v", err)
	}
	if workflowErr.Code != WorkflowErrorCodeWorkerVersionIncompatible {
		t.Fatalf("unexpected code: %s", workflowErr.Code)
	}
	if !taskStore.releaseCalled || taskStore.releasedTask != 505 {
		t.Fatalf("expected task claim released, got called=%v id=%d", taskStore.releaseCalled, taskStore.releasedTask)
	}
}
