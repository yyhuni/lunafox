package application

import (
	"context"
	"testing"

	"github.com/yyhuni/lunafox/contracts/runtimecontract"
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

type fixedCompatibilityGate struct {
	compatible bool
}

func (gate fixedCompatibilityGate) Supports(context.Context, int, WorkflowVersionTuple) (bool, error) {
	return gate.compatible, nil
}

type sequenceCompatibilityGate struct {
	results []bool
	index   int
}

func (gate *sequenceCompatibilityGate) Supports(context.Context, int, WorkflowVersionTuple) (bool, error) {
	if gate == nil || len(gate.results) == 0 {
		return false, nil
	}
	if gate.index >= len(gate.results) {
		return gate.results[len(gate.results)-1], nil
	}
	result := gate.results[gate.index]
	gate.index++
	return result, nil
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

func TestTaskRuntimeServiceUnlockNextStageIfReady_RespectsStageDependencyWhenActive(t *testing.T) {
	taskStore := &taskStoreStub{countActive: 1}
	service := NewTaskRuntimeServiceWithCompatibilityGate(taskStore, &runtimeScanStoreStub{}, fixedCompatibilityGate{compatible: true})

	if err := service.unlockNextStageIfReady(context.Background(), 99, 2); err != nil {
		t.Fatalf("unlockNextStageIfReady returned error: %v", err)
	}
	if taskStore.unlockCalled {
		t.Fatalf("expected next stage stay blocked while current stage still active")
	}
}

func TestTaskRuntimeServiceUnlockNextStageIfReady_UnlocksWhenNoActiveTasks(t *testing.T) {
	taskStore := &taskStoreStub{countActive: 0}
	service := NewTaskRuntimeServiceWithCompatibilityGate(taskStore, &runtimeScanStoreStub{}, fixedCompatibilityGate{compatible: true})

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

func TestTaskRuntimeServicePullTask_CompatibleTupleReturnsAssignment(t *testing.T) {
	taskStore := &taskStoreStub{
		pulledTask: &TaskRecord{
			ID:           101,
			ScanID:       9,
			Stage:        1,
			WorkflowName: "subdomain_discovery",
			Config:       "apiVersion: v1\nschemaVersion: 1.0.0\n",
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
}

func TestTaskRuntimeServicePullTask_IncompatibleTupleReturnsWorkflowError(t *testing.T) {
	taskStore := &taskStoreStub{
		pulledTask: &TaskRecord{
			ID:           202,
			ScanID:       18,
			Stage:        2,
			WorkflowName: "subdomain_discovery",
			Config:       "apiVersion: v1\nschemaVersion: 1.0.0\n",
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
	if !taskStore.failCalled || taskStore.failedTask != 202 {
		t.Fatalf("expected task claim failed, got called=%v id=%d", taskStore.failCalled, taskStore.failedTask)
	}
	if taskStore.failedRejectCause != "worker_incompatible" {
		t.Fatalf("expected worker_incompatible fail reason, got %q", taskStore.failedRejectCause)
	}
}

func TestTaskRuntimeServicePullTask_MissingVersionReturnsSchemaInvalid(t *testing.T) {
	taskStore := &taskStoreStub{
		pulledTask: &TaskRecord{
			ID:           303,
			ScanID:       21,
			Stage:        2,
			WorkflowName: "subdomain_discovery",
			Config:       "schemaVersion: 1.0.0\n",
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
	if !taskStore.failCalled || taskStore.failedTask != 303 {
		t.Fatalf("expected task claim failed, got called=%v id=%d", taskStore.failCalled, taskStore.failedTask)
	}
}

func TestTaskRuntimeServicePullTask_InvalidAPIVersionFormatReturnsSchemaInvalid(t *testing.T) {
	taskStore := &taskStoreStub{
		pulledTask: &TaskRecord{
			ID:           333,
			ScanID:       23,
			Stage:        2,
			WorkflowName: "subdomain_discovery",
			Config:       "apiVersion: version1\nschemaVersion: 1.0.0\n",
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
	if !taskStore.failCalled || taskStore.failedTask != 333 {
		t.Fatalf("expected task claim failed, got called=%v id=%d", taskStore.failCalled, taskStore.failedTask)
	}
}

func TestTaskRuntimeServicePullTask_InvalidSchemaVersionFormatReturnsSchemaInvalid(t *testing.T) {
	taskStore := &taskStoreStub{
		pulledTask: &TaskRecord{
			ID:           363,
			ScanID:       26,
			Stage:        2,
			WorkflowName: "subdomain_discovery",
			Config:       "apiVersion: v1\nschemaVersion: 1.0\n",
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
	if !taskStore.failCalled || taskStore.failedTask != 363 {
		t.Fatalf("expected task claim failed, got called=%v id=%d", taskStore.failCalled, taskStore.failedTask)
	}
}

func TestTaskRuntimeServicePullTask_MissingSupportedWorkflowSnapshotFailsClosedEvenWhenVersionMatches(t *testing.T) {
	taskStore := &taskStoreStub{
		pulledTask: &TaskRecord{
			ID:                    404,
			ScanID:                30,
			Stage:                 1,
			WorkflowName:          "subdomain_discovery",
			WorkflowAPIVersion:    "v1",
			WorkflowSchemaVersion: "1.0.0",
			Config:                "apiVersion: v1\nschemaVersion: 1.0.0\n",
			Status:                "pending",
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
			AgentVersion:  "9.9.9",
			ID:            1,
			WorkerVersion: "1.0.0",
			Status:        "online",
		},
	}
	service := NewTaskRuntimeServiceWithAgentStore(taskStore, runtimeStore, agentStore)

	assignment, err := service.PullTask(context.Background(), 1)
	if assignment != nil {
		t.Fatalf("expected no assignment when capability snapshot is missing")
	}
	workflowErr, ok := AsWorkflowError(err)
	if !ok {
		t.Fatalf("expected WorkflowError, got: %v", err)
	}
	if workflowErr.Code != WorkflowErrorCodeWorkerVersionIncompatible {
		t.Fatalf("unexpected code: %s", workflowErr.Code)
	}
	if !taskStore.failCalled || taskStore.failedTask != 404 {
		t.Fatalf("expected task claim failed, got called=%v id=%d", taskStore.failCalled, taskStore.failedTask)
	}
}

func TestTaskRuntimeServicePullTask_DynamicCapabilityVersionMismatch(t *testing.T) {
	taskStore := &taskStoreStub{
		pulledTask: &TaskRecord{
			ID:                    505,
			ScanID:                40,
			Stage:                 1,
			WorkflowName:          "subdomain_discovery",
			WorkflowAPIVersion:    "v1",
			WorkflowSchemaVersion: "1.0.0",
			Config:                "apiVersion: v1\nschemaVersion: 1.0.0\n",
			Status:                "pending",
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
			AgentVersion:  "1.0.0",
			ID:            2,
			WorkerVersion: "0.9.0",
			Status:        "online",
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
	if !taskStore.failCalled || taskStore.failedTask != 505 {
		t.Fatalf("expected task claim failed, got called=%v id=%d", taskStore.failCalled, taskStore.failedTask)
	}
}

func TestTaskRuntimeServicePullTask_MissingWorkerCapabilityFailsClosed(t *testing.T) {
	taskStore := &taskStoreStub{
		pulledTask: &TaskRecord{
			ID:                    606,
			ScanID:                41,
			Stage:                 1,
			WorkflowName:          "subdomain_discovery",
			WorkflowAPIVersion:    "v1",
			WorkflowSchemaVersion: "1.0.0",
			Config:                "apiVersion: v1\nschemaVersion: 1.0.0\n",
			Status:                "pending",
		},
	}
	runtimeStore := &runtimeScanStoreStub{
		scan: &TaskScanRecord{
			ID:                41,
			TargetID:          19,
			Status:            "running",
			YamlConfiguration: "subdomain_discovery:\n  apiVersion: v1\n  schemaVersion: 1.0.0\n",
		},
	}
	agentStore := &taskRuntimeAgentStoreStub{
		agent: &TaskRuntimeAgentRecord{
			ID:           3,
			Status:       "online",
			AgentVersion: "1.0.0",
		},
	}
	service := NewTaskRuntimeServiceWithAgentStore(taskStore, runtimeStore, agentStore)

	assignment, err := service.PullTask(context.Background(), 3)
	if assignment != nil {
		t.Fatalf("expected no assignment when capability snapshot is missing")
	}
	workflowErr, ok := AsWorkflowError(err)
	if !ok || workflowErr.Code != WorkflowErrorCodeWorkerVersionIncompatible {
		t.Fatalf("expected worker version incompatible error, got: %v", err)
	}
	if !taskStore.failCalled || taskStore.failedTask != 606 {
		t.Fatalf("expected task claim failed")
	}
}

func TestTaskRuntimeServicePullTask_SupportedWorkflowSnapshotWins(t *testing.T) {
	taskStore := &taskStoreStub{
		pulledTask: &TaskRecord{
			ID:                    707,
			ScanID:                42,
			Stage:                 1,
			WorkflowName:          "subdomain_discovery",
			WorkflowAPIVersion:    "v1",
			WorkflowSchemaVersion: "1.0.0",
			Config:                "apiVersion: v1\nschemaVersion: 1.0.0\n",
			Status:                "pending",
		},
	}
	runtimeStore := &runtimeScanStoreStub{
		scan: &TaskScanRecord{
			ID:                42,
			TargetID:          20,
			Status:            "running",
			YamlConfiguration: "subdomain_discovery:\n  apiVersion: v1\n  schemaVersion: 1.0.0\n",
		},
	}
	agentStore := &taskRuntimeAgentStoreStub{
		agent: &TaskRuntimeAgentRecord{
			ID:            4,
			WorkerVersion: "0.0.1",
			Status:        "online",
			SupportedWorkflows: []WorkflowVersionTuple{
				{
					Workflow:      "subdomain_discovery",
					APIVersion:    "v1",
					SchemaVersion: "1.0.0",
				},
			},
		},
	}
	service := NewTaskRuntimeServiceWithAgentStore(taskStore, runtimeStore, agentStore)

	assignment, err := service.PullTask(context.Background(), 4)
	if err != nil {
		t.Fatalf("expected compatible assignment from supported workflow snapshot, got: %v", err)
	}
	if assignment == nil {
		t.Fatalf("expected task assignment")
	}
}

func TestTaskRuntimeServicePullTask_IncompatibleDoesNotPromotePendingScan(t *testing.T) {
	taskStore := &taskStoreStub{
		pulledTask: &TaskRecord{
			ID:                    808,
			ScanID:                43,
			Stage:                 1,
			WorkflowName:          "subdomain_discovery",
			WorkflowAPIVersion:    "v1",
			WorkflowSchemaVersion: "1.0.0",
			Config:                "apiVersion: v1\nschemaVersion: 1.0.0\n",
			Status:                "pending",
		},
	}
	runtimeStore := &runtimeScanStoreStub{
		scan: &TaskScanRecord{
			ID:                43,
			TargetID:          21,
			Status:            "pending",
			YamlConfiguration: "subdomain_discovery:\n  apiVersion: v1\n  schemaVersion: 1.0.0\n",
		},
	}
	agentStore := &taskRuntimeAgentStoreStub{
		agent: &TaskRuntimeAgentRecord{
			ID:            5,
			WorkerVersion: "0.9.0",
			Status:        "online",
		},
	}
	service := NewTaskRuntimeServiceWithAgentStore(taskStore, runtimeStore, agentStore)

	assignment, err := service.PullTask(context.Background(), 5)
	if assignment != nil {
		t.Fatalf("expected no assignment")
	}
	if _, ok := AsWorkflowError(err); !ok {
		t.Fatalf("expected workflow error, got %v", err)
	}
	if runtimeStore.lastUpdatedStatus == "running" {
		t.Fatalf("incompatible task should not promote scan to running")
	}
}

func TestTaskRuntimeServicePullTask_SchemaInvalidFailsScan(t *testing.T) {
	taskStore := &taskStoreStub{
		pulledTask: &TaskRecord{
			ID:           909,
			ScanID:       44,
			Stage:        1,
			WorkflowName: "subdomain_discovery",
			Config:       "schemaVersion: 1.0.0\n",
			Status:       "pending",
		},
	}
	runtimeStore := &runtimeScanStoreStub{
		scan: &TaskScanRecord{
			ID:       44,
			TargetID: 22,
			Status:   "pending",
			YamlConfiguration: `
subdomain_discovery:
  schemaVersion: 1.0.0
`,
		},
	}
	service := NewTaskRuntimeServiceWithCompatibilityGate(taskStore, runtimeStore, fixedCompatibilityGate{compatible: true})

	assignment, err := service.PullTask(context.Background(), 7)
	if assignment != nil {
		t.Fatalf("expected no assignment")
	}
	if _, ok := AsWorkflowError(err); !ok {
		t.Fatalf("expected workflow error")
	}
	if runtimeStore.lastUpdatedStatus != "failed" {
		t.Fatalf("expected scan failed status, got %q", runtimeStore.lastUpdatedStatus)
	}
	if !taskStore.failCalled {
		t.Fatalf("expected task fail path")
	}
}

func TestTaskRuntimeServicePullTask_MissingTaskConfigSliceFailsFast(t *testing.T) {
	taskStore := &taskStoreStub{
		pulledTask: &TaskRecord{
			ID:                    1001,
			ScanID:                45,
			Stage:                 1,
			WorkflowName:          "subdomain_discovery",
			WorkflowAPIVersion:    "v1",
			WorkflowSchemaVersion: "1.0.0",
			Config:                "",
			Status:                "pending",
		},
	}
	runtimeStore := &runtimeScanStoreStub{
		scan: &TaskScanRecord{
			ID:                45,
			TargetID:          23,
			Status:            "pending",
			YamlConfiguration: "subdomain_discovery:\n  apiVersion: v1\n  schemaVersion: 1.0.0\n",
		},
	}
	service := NewTaskRuntimeServiceWithCompatibilityGate(taskStore, runtimeStore, fixedCompatibilityGate{compatible: true})

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

func TestTaskRuntimeServicePullTask_IncompatibleHeadFailsFastWithoutPickingNextCandidate(t *testing.T) {
	taskStore := &taskStoreStub{
		pulledTasks: []*TaskRecord{
			{
				ID:                    3001,
				ScanID:                50,
				Stage:                 2,
				WorkflowName:          "subdomain_discovery",
				WorkflowAPIVersion:    "v1",
				WorkflowSchemaVersion: "1.0.0",
				Config:                "apiVersion: v1\nschemaVersion: 1.0.0\n",
				Status:                "pending",
			},
			{
				ID:                    3002,
				ScanID:                51,
				Stage:                 1,
				WorkflowName:          "subdomain_discovery",
				WorkflowAPIVersion:    "v1",
				WorkflowSchemaVersion: "1.0.0",
				Config:                "apiVersion: v1\nschemaVersion: 1.0.0\n",
				Status:                "pending",
			},
		},
	}
	runtimeStore := &runtimeScanStoreStub{
		scan: &TaskScanRecord{
			ID:                50,
			TargetID:          8,
			Status:            "running",
			YamlConfiguration: "subdomain_discovery:\n  apiVersion: v1\n  schemaVersion: 1.0.0\n",
		},
	}
	gate := &sequenceCompatibilityGate{results: []bool{false, true}}
	service := NewTaskRuntimeServiceWithCompatibilityGate(taskStore, runtimeStore, gate)

	assignment, err := service.PullTask(context.Background(), 12)
	if assignment != nil {
		t.Fatalf("expected no assignment")
	}
	workflowErr, ok := AsWorkflowError(err)
	if !ok {
		t.Fatalf("expected workflow error, got %v", err)
	}
	if workflowErr.Code != WorkflowErrorCodeWorkerVersionIncompatible {
		t.Fatalf("unexpected code: %s", workflowErr.Code)
	}
	if taskStore.pullIndex != 1 {
		t.Fatalf("fail-fast should stop after first incompatible candidate, got pullIndex=%d", taskStore.pullIndex)
	}
	if !taskStore.failCalled || taskStore.failedTask != 3001 {
		t.Fatalf("expected first candidate failed")
	}
}

func TestBuildTaskWorkflowTuple_PrefersPersistedTupleOverConfigText(t *testing.T) {
	task := &TaskRecord{
		ID:                    2001,
		WorkflowName:          "subdomain_discovery",
		WorkflowAPIVersion:    "v1",
		WorkflowSchemaVersion: "1.0.0",
	}
	// Config text drifts to another version, but scheduler should trust persisted tuple.
	configYAML := "apiVersion: v9\nschemaVersion: 9.9.9\n"
	tuple, err := buildTaskWorkflowTuple(task, configYAML)
	if err != nil {
		t.Fatalf("buildTaskWorkflowTuple returned error: %v", err)
	}
	if tuple.APIVersion != "v1" || tuple.SchemaVersion != "1.0.0" {
		t.Fatalf("unexpected tuple %+v", tuple)
	}
}

func TestTaskRuntimeServicePullTask_InvalidPersistedAPIVersionUsesStableSchemaError(t *testing.T) {
	taskStore := &taskStoreStub{
		pulledTask: &TaskRecord{
			ID:                    4001,
			ScanID:                61,
			Stage:                 1,
			WorkflowName:          "subdomain_discovery",
			WorkflowAPIVersion:    "version1",
			WorkflowSchemaVersion: "1.0.0",
			Config:                "apiVersion: v1\nschemaVersion: 1.0.0\n",
			Status:                "pending",
		},
	}
	runtimeStore := &runtimeScanStoreStub{
		scan: &TaskScanRecord{
			ID:                61,
			TargetID:          31,
			Status:            "running",
			YamlConfiguration: "subdomain_discovery:\n  apiVersion: v1\n  schemaVersion: 1.0.0\n",
		},
	}
	service := NewTaskRuntimeServiceWithCompatibilityGate(taskStore, runtimeStore, fixedCompatibilityGate{compatible: true})

	assignment, err := service.PullTask(context.Background(), 1005)
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
	if workflowErr.Stage != WorkflowErrorStageServerSchemaGate {
		t.Fatalf("unexpected stage: %s", workflowErr.Stage)
	}
	if workflowErr.Field != "workflow_api_version" {
		t.Fatalf("unexpected field: %s", workflowErr.Field)
	}
	if workflowErr.Message != runtimecontract.APIVersionFieldMessage("workflow_api_version") {
		t.Fatalf("unexpected message: %s", workflowErr.Message)
	}
	if !taskStore.failCalled {
		t.Fatalf("expected task fail path for non-recoverable scheduler schema error")
	}
}

func TestTaskRuntimeServicePullTask_InvalidPersistedSchemaVersionUsesStableSchemaError(t *testing.T) {
	taskStore := &taskStoreStub{
		pulledTask: &TaskRecord{
			ID:                    4002,
			ScanID:                62,
			Stage:                 1,
			WorkflowName:          "subdomain_discovery",
			WorkflowAPIVersion:    "v1",
			WorkflowSchemaVersion: "1.0",
			Config:                "apiVersion: v1\nschemaVersion: 1.0.0\n",
			Status:                "pending",
		},
	}
	runtimeStore := &runtimeScanStoreStub{
		scan: &TaskScanRecord{
			ID:                62,
			TargetID:          32,
			Status:            "running",
			YamlConfiguration: "subdomain_discovery:\n  apiVersion: v1\n  schemaVersion: 1.0.0\n",
		},
	}
	service := NewTaskRuntimeServiceWithCompatibilityGate(taskStore, runtimeStore, fixedCompatibilityGate{compatible: true})

	assignment, err := service.PullTask(context.Background(), 1006)
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
	if workflowErr.Stage != WorkflowErrorStageServerSchemaGate {
		t.Fatalf("unexpected stage: %s", workflowErr.Stage)
	}
	if workflowErr.Field != "workflow_schema_version" {
		t.Fatalf("unexpected field: %s", workflowErr.Field)
	}
	if workflowErr.Message != runtimecontract.SchemaVersionFieldMessage("workflow_schema_version") {
		t.Fatalf("unexpected message: %s", workflowErr.Message)
	}
	if !taskStore.failCalled {
		t.Fatalf("expected task fail path for non-recoverable scheduler schema error")
	}
}
