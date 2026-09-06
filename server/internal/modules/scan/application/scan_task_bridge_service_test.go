package application

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	scandomain "github.com/yyhuni/lunafox/server/internal/modules/scan/domain"
)

type taskStoreStub struct {
	getByIDFn           func(context.Context, int) (*ScanTaskRecord, error)
	listFailedFn        func(context.Context, int) ([]ScanTaskRecord, error)
	countActive         int
	pendingCount        int
	runningCount        int
	completedCount      int
	failedCount         int
	cancelledCount      int
	skippedCount        int
	skippedUnstarted    bool
	skippedScanID       int
	skippedReason       string
	cancelledUnstarted  bool
	cancelledScanID     int
	failCalled          bool
	failedTask          int
	failedFailure       *FailureDetail
	updatedFailure      *FailureDetail
	updatedStatus       string
	updatedTaskID       int
	requireSessionFn    func(context.Context, int, string, int64) error
	commitSessionFn     func(context.Context, int, int, string, int64, string, *FailureDetail) (bool, error)
	pendingTerminalFn   func(context.Context, int, int) ([]ScanTaskRecord, error)
	pendingSupersededFn func(context.Context, int, int64, int) ([]ScanTaskRecord, error)
	clearTerminalFn     func(context.Context, int) error
	clearedTerminalIDs  []int
	unlockCalled        bool
	unlockScanID        int
	unlockStage         int
	fenceSupersededFn   func(context.Context, int, int64) ([]int, error)
	ops                 *[]string
}

func cloneFailureDetailForTest(failure *FailureDetail) *FailureDetail {
	if failure == nil {
		return nil
	}
	cloned := *failure
	return &cloned
}

func assignedTaskRecordForTest(task ScanTaskRecord, agentID int, sessionID string, sessionEpoch int64) *ScanTaskRecord {
	task.AgentID = &agentID
	task.AssignedAgentID = &agentID
	task.AssignedSessionID = &sessionID
	task.AssignedSessionEpoch = &sessionEpoch
	return &task
}

func TestFailClaimedTaskPersistsFixedSafeSchedulerRejection(t *testing.T) {
	store := &taskStoreStub{}
	service := newScanTaskBridgeServiceForTest(store, &scanTaskRuntimeScanStoreStub{})
	raw := errors.New("privatePath=/srv/private/provider.yaml token=canary")

	err := service.failClaimedTask(context.Background(), &ScanTaskRecord{ID: 303}, nil, "scheduler_rejected", raw)
	if !errors.Is(err, raw) {
		t.Fatalf("failClaimedTask() error = %v, want original local error", err)
	}
	if !store.failCalled || store.failedTask != 303 || store.failedFailure == nil {
		t.Fatalf("persisted scheduler rejection = %#v", store.failedFailure)
	}
	if store.failedFailure.Kind != "scheduler_rejected" || store.failedFailure.Message != "The Server rejected the task assignment." {
		t.Fatalf("persisted scheduler rejection = %#v", store.failedFailure)
	}
	if strings.Contains(store.failedFailure.Message, "canary") || strings.Contains(store.failedFailure.Message, "/srv/private") {
		t.Fatalf("scheduler rejection persisted raw error detail: %#v", store.failedFailure)
	}

	schemaFailure := failureFromReason("schema_invalid")
	if schemaFailure.Message != "The task execution schema is invalid." {
		t.Fatalf("schema rejection message = %#v", schemaFailure)
	}
}

func workflowStepExecutionTaskConfigForBridgeTest(target string) map[string]any {
	return map[string]any{
		"workflowStepExecution": map[string]any{
			"schemaVersion":  1,
			"scanWorkflowId": "subdomain_discovery",
			"step": map[string]any{
				"stageId":      "discovery",
				"stepId":       "subdomain_discovery",
				"engineId":     "engine.lunafox.subdomain_discovery",
				"engineConfig": map[string]any{"enabled": true},
				"input": map[string]any{
					"kind":       "targetSeed",
					"target":     target,
					"targetType": "domain",
					"source": map[string]any{
						"seedType": "domain",
						"seed":     "example.com",
					},
				},
			},
		},
	}
}

func (stub *taskStoreStub) GetByID(ctx context.Context, id int) (*ScanTaskRecord, error) {
	if stub.getByIDFn != nil {
		return stub.getByIDFn(ctx, id)
	}
	return nil, nil
}

func (stub *taskStoreStub) CountByStatusForScanID(context.Context, int) (int, int, int, int, int, int, error) {
	return stub.pendingCount, stub.runningCount, stub.completedCount, stub.failedCount, stub.cancelledCount, stub.skippedCount, nil
}

func (stub *taskStoreStub) CountActiveByScanAndStageOrder(context.Context, int, int) (int, error) {
	return stub.countActive, nil
}

func (stub *taskStoreStub) RequireCurrentAgentExecutionSession(ctx context.Context, agentID int, sessionID string, sessionEpoch int64) error {
	if stub.requireSessionFn != nil {
		return stub.requireSessionFn(ctx, agentID, sessionID, sessionEpoch)
	}
	return nil
}

func (stub *taskStoreStub) CommitScanTaskTerminalStatusForSession(ctx context.Context, id, agentID int, sessionID string, sessionEpoch int64, status string, failure *FailureDetail) (bool, error) {
	stub.updatedTaskID = id
	stub.updatedStatus = status
	stub.updatedFailure = cloneFailureDetailForTest(failure)
	if stub.commitSessionFn != nil {
		return stub.commitSessionFn(ctx, id, agentID, sessionID, sessionEpoch, status, failure)
	}
	return true, nil
}

func (stub *taskStoreStub) ListTerminalTasksPendingReconciliation(ctx context.Context, afterTaskID, limit int) ([]ScanTaskRecord, error) {
	if stub.pendingTerminalFn != nil {
		return stub.pendingTerminalFn(ctx, afterTaskID, limit)
	}
	return nil, nil
}

func (stub *taskStoreStub) ListSupersededSessionTerminalTasksPendingReconciliation(ctx context.Context, agentID int, currentSessionEpoch int64, limit int) ([]ScanTaskRecord, error) {
	if stub.pendingSupersededFn != nil {
		return stub.pendingSupersededFn(ctx, agentID, currentSessionEpoch, limit)
	}
	return nil, nil
}

func (stub *taskStoreStub) ClearTerminalTaskReconciliationPending(ctx context.Context, taskID int) error {
	stub.clearedTerminalIDs = append(stub.clearedTerminalIDs, taskID)
	if stub.clearTerminalFn != nil {
		return stub.clearTerminalFn(ctx, taskID)
	}
	return nil
}

func (stub *taskStoreStub) FailClaimedTask(_ context.Context, id int, failure *FailureDetail) error {
	stub.failCalled = true
	stub.failedTask = id
	stub.failedFailure = cloneFailureDetailForTest(failure)
	return nil
}

func (stub *taskStoreStub) SkipUnstartedTasksByScanID(_ context.Context, scanID int, reason string) (int64, error) {
	stub.recordOperation("skip-unstarted")
	stub.skippedUnstarted = true
	stub.skippedScanID = scanID
	stub.skippedReason = reason
	return 0, nil
}

func (stub *taskStoreStub) CancelUnstartedTasksByScanID(_ context.Context, scanID int) (int64, error) {
	stub.recordOperation("cancel-unstarted")
	stub.cancelledUnstarted = true
	stub.cancelledScanID = scanID
	return 0, nil
}

func (stub *taskStoreStub) recordOperation(name string) {
	if stub.ops == nil {
		return
	}
	*stub.ops = append(*stub.ops, name)
}

func (stub *taskStoreStub) ListFailedByScanID(ctx context.Context, scanID int) ([]ScanTaskRecord, error) {
	if stub.listFailedFn != nil {
		return stub.listFailedFn(ctx, scanID)
	}
	return nil, nil
}

func (stub *taskStoreStub) UnlockNextStageOrder(_ context.Context, scanID, stage int) (int64, error) {
	stub.unlockCalled = true
	stub.unlockScanID = scanID
	stub.unlockStage = stage
	return 0, nil
}

func (stub *taskStoreStub) FailTasksForSupersededAgentSession(ctx context.Context, agentID int, currentSessionEpoch int64) ([]int, error) {
	if stub.fenceSupersededFn != nil {
		return stub.fenceSupersededFn(ctx, agentID, currentSessionEpoch)
	}
	return nil, nil
}

type scanTaskRuntimeScanStoreStub struct {
	scan               *ScanTaskRuntimeScanRecord
	lastUpdatedStatus  string
	lastUpdatedFailure *FailureDetail
	updateCalls        int
	ops                *[]string
	updateErrors       []error
}

func newScanTaskBridgeServiceForTest(taskStore ScanTaskStore, scanTaskRuntimeScanStore ScanTaskRuntimeScanStore) *ScanTaskBridgeService {
	service := NewScanTaskBridgeService(taskStore, scanTaskRuntimeScanStore)
	if claims, ok := taskStore.(EngineExecutionClaimStore); ok {
		service.WithEngineExecutionClaimStore(claims)
	}
	return service
}

func (stub *scanTaskRuntimeScanStoreStub) GetScanForScanTask(int) (*ScanTaskRuntimeScanRecord, error) {
	return stub.scan, nil
}

func (stub *scanTaskRuntimeScanStoreStub) UpdateScanStatus(_ int, status string, failure *FailureDetail) error {
	if stub.ops != nil {
		*stub.ops = append(*stub.ops, "update-scan")
	}
	stub.updateCalls++
	stub.lastUpdatedStatus = status
	stub.lastUpdatedFailure = cloneFailureDetailForTest(failure)
	if len(stub.updateErrors) > 0 {
		err := stub.updateErrors[0]
		stub.updateErrors = stub.updateErrors[1:]
		return err
	}
	return nil
}

func TestScanTaskBridgeServiceUnlockNextStageIfReady_RespectsStageDependencyWhenActive(t *testing.T) {
	taskStore := &taskStoreStub{countActive: 1}
	service := newScanTaskBridgeServiceForTest(taskStore, &scanTaskRuntimeScanStoreStub{})

	if err := service.unlockNextStageIfReady(context.Background(), 99, 2); err != nil {
		t.Fatalf("unlockNextStageIfReady returned error: %v", err)
	}
	if taskStore.unlockCalled {
		t.Fatalf("expected next stage stay blocked while current stage still active")
	}
}

func TestScanTaskBridgeServiceUnlockNextStageIfReady_UnlocksWhenNoActiveTasks(t *testing.T) {
	taskStore := &taskStoreStub{countActive: 0}
	service := newScanTaskBridgeServiceForTest(taskStore, &scanTaskRuntimeScanStoreStub{})

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

func TestScanTaskBridgeServiceReportTerminalTaskResult_RejectsFailedWithoutFailureMessage(t *testing.T) {
	service := newScanTaskBridgeServiceForTest(&taskStoreStub{}, &scanTaskRuntimeScanStoreStub{})

	if err := service.ReportTerminalTaskResult(context.Background(), 7, "session-7", 1, 303, "failed", &FailureDetail{Kind: "runtime_error"}); err != ErrScanTaskInvalidUpdate {
		t.Fatalf("expected ErrScanTaskInvalidUpdate, got %v", err)
	}
}

func TestScanTaskBridgeServiceReportTerminalTaskResult_RejectsFailedWithoutFailureKind(t *testing.T) {
	service := newScanTaskBridgeServiceForTest(&taskStoreStub{}, &scanTaskRuntimeScanStoreStub{})

	if err := service.ReportTerminalTaskResult(context.Background(), 7, "session-7", 1, 303, "failed", &FailureDetail{Message: "boom"}); err != ErrScanTaskInvalidUpdate {
		t.Fatalf("expected ErrScanTaskInvalidUpdate, got %v", err)
	}
}

func TestScanTaskBridgeServiceReportTerminalTaskResult_RejectsNonFailedWithFailure(t *testing.T) {
	service := newScanTaskBridgeServiceForTest(&taskStoreStub{}, &scanTaskRuntimeScanStoreStub{})

	if err := service.ReportTerminalTaskResult(context.Background(), 7, "session-7", 1, 303, "succeeded", &FailureDetail{Kind: "runtime_error", Message: "boom"}); err != ErrScanTaskInvalidUpdate {
		t.Fatalf("expected ErrScanTaskInvalidUpdate, got %v", err)
	}
}

func TestScanTaskBridgeServiceReportTerminalTaskResult_PropagatesFailureObject(t *testing.T) {
	sessionEpoch := int64(11)
	sessionID := "session-7"
	agentID := 7
	taskStore := &taskStoreStub{pendingCount: 1}
	service := newScanTaskBridgeServiceForTest(taskStore, &scanTaskRuntimeScanStoreStub{})
	taskStore.getByIDFn = func(context.Context, int) (*ScanTaskRecord, error) {
		return assignedTaskRecordForTest(ScanTaskRecord{ID: 303, ScanID: 99, ScanWorkflowStageOrder: 1, ScanWorkflowID: "subdomain_discovery", Status: "running"}, agentID, sessionID, sessionEpoch), nil
	}

	failure := &FailureDetail{Kind: "runtime_error", Message: "boom", DisplayMessage: "Restore Agent storage permissions."}
	err := service.ReportTerminalTaskResult(context.Background(), agentID, sessionID, sessionEpoch, 303, "failed", failure)
	if err != nil {
		t.Fatalf("ReportTerminalTaskResult returned error: %v", err)
	}
	if taskStore.updatedFailure == nil || taskStore.updatedFailure.Kind != "runtime_error" || taskStore.updatedFailure.Message != "boom" || taskStore.updatedFailure.DisplayMessage != "Restore Agent storage permissions." {
		t.Fatalf("expected propagated failure, got %+v", taskStore.updatedFailure)
	}
}

func TestScanTaskBridgeServiceReportTerminalTaskResultRejectsUnsafeFailureDetail(t *testing.T) {
	service := newScanTaskBridgeServiceForTest(&taskStoreStub{}, &scanTaskRuntimeScanStoreStub{})
	for _, displayMessage := range []string{" leading whitespace", "line one\nline two", string(make([]byte, 501))} {
		if err := service.ReportTerminalTaskResult(context.Background(), 7, "session-7", 1, 303, "failed", &FailureDetail{Kind: "runtime_error", Message: "boom", DisplayMessage: displayMessage}); err != ErrScanTaskInvalidUpdate {
			t.Fatalf("display message %q error = %v, want %v", displayMessage, err, ErrScanTaskInvalidUpdate)
		}
	}
}

func TestScanTaskBridgeServiceReportTerminalTaskResult_DoesNotUnlockNextStageOnFailure(t *testing.T) {
	sessionEpoch := int64(11)
	sessionID := "session-7"
	agentID := 7
	taskStore := &taskStoreStub{countActive: 0, failedCount: 1, listFailedFn: func(context.Context, int) ([]ScanTaskRecord, error) {
		completedAt := time.Now().UTC()
		return []ScanTaskRecord{{ID: 303, ScanID: 99, ScanWorkflowStageOrder: 1, Status: "failed", CompletedAt: &completedAt, Failure: &FailureDetail{Kind: "runtime_error", Message: "boom"}}}, nil
	}}
	service := newScanTaskBridgeServiceForTest(taskStore, &scanTaskRuntimeScanStoreStub{})
	taskStore.getByIDFn = func(context.Context, int) (*ScanTaskRecord, error) {
		return assignedTaskRecordForTest(ScanTaskRecord{ID: 303, ScanID: 99, ScanWorkflowStageOrder: 1, ScanWorkflowID: "subdomain_discovery", Status: "running"}, agentID, sessionID, sessionEpoch), nil
	}

	err := service.ReportTerminalTaskResult(context.Background(), agentID, sessionID, sessionEpoch, 303, "failed", &FailureDetail{Kind: "runtime_error", Message: "boom"})
	if err != nil {
		t.Fatalf("ReportTerminalTaskResult returned error: %v", err)
	}
	if taskStore.unlockCalled {
		t.Fatalf("failed stage must not unlock later blocked stages")
	}
}

func TestScanTaskBridgeServiceReportTerminalTaskResult_WaitsForEveryActiveTaskBeforeUnlock(t *testing.T) {
	sessionEpoch := int64(11)
	sessionID := "session-7"
	agentID := 7
	taskStore := &taskStoreStub{countActive: 1, pendingCount: 1}
	service := newScanTaskBridgeServiceForTest(taskStore, &scanTaskRuntimeScanStoreStub{})
	taskStore.getByIDFn = func(context.Context, int) (*ScanTaskRecord, error) {
		return assignedTaskRecordForTest(ScanTaskRecord{ID: 303, ScanID: 99, ScanWorkflowStageOrder: 1, ScanWorkflowID: "subdomain_discovery", Status: "running"}, agentID, sessionID, sessionEpoch), nil
	}

	err := service.ReportTerminalTaskResult(context.Background(), agentID, sessionID, sessionEpoch, 303, "succeeded", nil)
	if err != nil {
		t.Fatalf("ReportTerminalTaskResult returned error: %v", err)
	}
	if taskStore.unlockCalled {
		t.Fatalf("next stage must stay blocked while current stage still has active tasks")
	}
}

func TestScanTaskBridgeServiceReportTerminalTaskResult_RejectsMismatchedAssignedSessionEpoch(t *testing.T) {
	agentID := 7
	sessionID := "session-7"
	assignedSessionEpoch := int64(11)
	service := newScanTaskBridgeServiceForTest(&taskStoreStub{
		getByIDFn: func(context.Context, int) (*ScanTaskRecord, error) {
			return assignedTaskRecordForTest(ScanTaskRecord{
				ID:                     404,
				ScanID:                 101,
				ScanWorkflowStageOrder: 1,
				ScanWorkflowID:         "subdomain_discovery",
				Status:                 "running",
			}, agentID, sessionID, assignedSessionEpoch), nil
		},
	}, &scanTaskRuntimeScanStoreStub{})

	err := service.ReportTerminalTaskResult(context.Background(), agentID, sessionID, 12, 404, "succeeded", nil)
	if err != ErrScanTaskNotOwned {
		t.Fatalf("expected ErrScanTaskNotOwned for mismatched session epoch, got %v", err)
	}
}

func TestScanTaskBridgeServiceReportTerminalTaskResultRequiresExactAssignedSessionTuple(t *testing.T) {
	agentID := 7
	assignedAgentID := agentID
	sessionID := "session-7"
	epoch := int64(11)
	for _, test := range []struct {
		name              string
		reportedSessionID string
		assignedAgentID   *int
		assignedSessionID *string
		wantErr           error
	}{
		{name: "exact tuple", reportedSessionID: sessionID, assignedAgentID: &assignedAgentID, assignedSessionID: &sessionID},
		{name: "same epoch wrong session", reportedSessionID: "session-other", assignedAgentID: &assignedAgentID, assignedSessionID: &sessionID, wantErr: ErrScanTaskNotOwned},
		{name: "missing assigned agent", reportedSessionID: sessionID, assignedSessionID: &sessionID, wantErr: ErrScanTaskNotOwned},
		{name: "missing assigned session", reportedSessionID: sessionID, assignedAgentID: &assignedAgentID, wantErr: ErrScanTaskNotOwned},
	} {
		t.Run(test.name, func(t *testing.T) {
			store := &taskStoreStub{
				completedCount: 1,
				getByIDFn: func(context.Context, int) (*ScanTaskRecord, error) {
					return &ScanTaskRecord{
						ID: 303, ScanID: 44, ScanWorkflowStageOrder: 2, Status: "running", AgentID: &agentID,
						AssignedAgentID: test.assignedAgentID, AssignedSessionID: test.assignedSessionID, AssignedSessionEpoch: &epoch,
					}, nil
				},
				commitSessionFn: func(_ context.Context, _ int, gotAgentID int, gotSessionID string, gotEpoch int64, _ string, _ *FailureDetail) (bool, error) {
					if gotAgentID != agentID || gotSessionID != sessionID || gotEpoch != epoch {
						t.Fatalf("terminal CAS tuple = (%d, %q, %d)", gotAgentID, gotSessionID, gotEpoch)
					}
					return true, nil
				},
			}
			service := newScanTaskBridgeServiceForTest(store, &scanTaskRuntimeScanStoreStub{})
			err := service.ReportTerminalTaskResult(context.Background(), agentID, test.reportedSessionID, epoch, 303, "succeeded", nil)
			if !errors.Is(err, test.wantErr) {
				t.Fatalf("ReportTerminalTaskResult() error = %v, want %v", err, test.wantErr)
			}
			if test.wantErr != nil && store.updatedTaskID != 0 {
				t.Fatalf("rejected terminal tuple reached CAS for task %d", store.updatedTaskID)
			}
		})
	}
}

func TestScanTaskBridgeServiceDuplicateTerminalResultRejectsPersistedSessionTakeover(t *testing.T) {
	agentID := 7
	sessionID := "session-7"
	epoch := int64(11)
	store := &taskStoreStub{
		requireSessionFn: func(context.Context, int, string, int64) error {
			return scandomain.ErrAgentExecutionSessionFenced
		},
		getByIDFn: func(context.Context, int) (*ScanTaskRecord, error) {
			t.Fatal("fenced duplicate must be rejected before reading or reconciling task state")
			return nil, nil
		},
	}
	service := newScanTaskBridgeServiceForTest(store, &scanTaskRuntimeScanStoreStub{})
	err := service.ReportTerminalTaskResult(context.Background(), agentID, sessionID, epoch, 303, "succeeded", nil)
	if err != ErrScanTaskNotOwned {
		t.Fatalf("fenced duplicate error = %v, want ErrScanTaskNotOwned", err)
	}
}

func TestScanTaskBridgeServiceDurablyReplaysTerminalReconciliationAfterAgentCrash(t *testing.T) {
	agentID := 7
	assignedAgentID := agentID
	sessionID := "session-7"
	epoch := int64(11)
	terminalTask := ScanTaskRecord{
		ID: 303, ScanID: 44, ScanWorkflowStageOrder: 2, Status: "succeeded", AgentID: &agentID,
		AssignedAgentID: &assignedAgentID, AssignedSessionID: &sessionID, AssignedSessionEpoch: &epoch,
	}
	store := &taskStoreStub{
		completedCount: 1,
		getByIDFn: func(context.Context, int) (*ScanTaskRecord, error) {
			running := terminalTask
			running.Status = "running"
			return &running, nil
		},
		pendingTerminalFn: func(_ context.Context, afterTaskID, limit int) ([]ScanTaskRecord, error) {
			if afterTaskID != 0 || limit != terminalReconciliationBatchSize {
				t.Fatalf("terminal replay cursor = after %d limit %d", afterTaskID, limit)
			}
			return []ScanTaskRecord{terminalTask}, nil
		},
	}
	scanStore := &scanTaskRuntimeScanStoreStub{updateErrors: []error{errors.New("scan persistence unavailable"), nil}}
	service := newScanTaskBridgeServiceForTest(store, scanStore)

	err := service.ReportTerminalTaskResult(context.Background(), agentID, sessionID, epoch, 303, "succeeded", nil)
	if err == nil || len(store.clearedTerminalIDs) != 0 {
		t.Fatalf("failed reconciliation error=%v cleared=%v; marker must remain durable", err, store.clearedTerminalIDs)
	}
	// No duplicate Agent delivery follows: a restarted process has no old outbox.
	if err := service.ReconcilePersistedTerminalTasks(context.Background()); err != nil {
		t.Fatalf("Server-owned terminal replay error = %v", err)
	}
	if !reflect.DeepEqual(store.clearedTerminalIDs, []int{303}) {
		t.Fatalf("terminal marker clears = %v, want task 303 after convergence", store.clearedTerminalIDs)
	}
	if scanStore.updateCalls != 2 {
		t.Fatalf("scan reconciliation attempts = %d, want failed delivery + durable replay", scanStore.updateCalls)
	}
}

func TestScanTaskBridgeServiceTerminalReconciliationContinuesAfterEarlierRecordFails(t *testing.T) {
	tasks := []ScanTaskRecord{
		{ID: 301, ScanID: 41, ScanWorkflowStageOrder: 1, Status: "succeeded"},
		{ID: 302, ScanID: 42, ScanWorkflowStageOrder: 1, Status: "succeeded"},
	}
	store := &taskStoreStub{
		completedCount: 1,
		pendingTerminalFn: func(_ context.Context, afterTaskID, limit int) ([]ScanTaskRecord, error) {
			if afterTaskID != 0 || limit != terminalReconciliationBatchSize {
				t.Fatalf("terminal replay cursor = after %d limit %d", afterTaskID, limit)
			}
			return tasks, nil
		},
	}
	scanStore := &scanTaskRuntimeScanStoreStub{updateErrors: []error{errors.New("first scan unavailable"), nil}}
	service := newScanTaskBridgeServiceForTest(store, scanStore)

	if err := service.ReconcilePersistedTerminalTasks(context.Background()); err == nil {
		t.Fatal("batch must report the first reconciliation failure")
	}
	if scanStore.updateCalls != 2 {
		t.Fatalf("batch stopped after first record: scan updates = %d", scanStore.updateCalls)
	}
	if !reflect.DeepEqual(store.clearedTerminalIDs, []int{302}) {
		t.Fatalf("marker clears = %v, want only the later successful task", store.clearedTerminalIDs)
	}
}

func TestScanTaskBridgeServiceDuplicateTerminalResultRetriesWorkflowReconciliationBeforeAck(t *testing.T) {
	agentID := 7
	sessionID := "session-7"
	epoch := int64(11)
	taskStore := &taskStoreStub{
		completedCount: 1,
		getByIDFn: func(context.Context, int) (*ScanTaskRecord, error) {
			return assignedTaskRecordForTest(ScanTaskRecord{ID: 303, ScanID: 44, ScanWorkflowStageOrder: 2, Status: "succeeded"}, agentID, sessionID, epoch), nil
		},
	}
	scanStore := &scanTaskRuntimeScanStoreStub{}
	service := newScanTaskBridgeServiceForTest(taskStore, scanStore)

	if err := service.ReportTerminalTaskResult(context.Background(), agentID, sessionID, epoch, 303, "succeeded", nil); err != nil {
		t.Fatalf("duplicate terminal result reconciliation error = %v", err)
	}
	if taskStore.updatedTaskID != 0 {
		t.Fatalf("duplicate terminal result rewrote task state: %d", taskStore.updatedTaskID)
	}
	if !taskStore.unlockCalled || scanStore.updateCalls != 1 || scanStore.lastUpdatedStatus != "succeeded" {
		t.Fatalf("duplicate terminal result did not retry reconciliation: unlock=%v updates=%d status=%q", taskStore.unlockCalled, scanStore.updateCalls, scanStore.lastUpdatedStatus)
	}
}

func TestScanTaskBridgeServiceDuplicateFailedResultRejectsDifferentFailureSnapshot(t *testing.T) {
	agentID := 7
	sessionID := "session-7"
	epoch := int64(11)
	taskStore := &taskStoreStub{
		failedCount: 1,
		getByIDFn: func(context.Context, int) (*ScanTaskRecord, error) {
			return assignedTaskRecordForTest(ScanTaskRecord{
				ID: 303, ScanID: 44, ScanWorkflowStageOrder: 2, Status: "failed",
				Failure: &FailureDetail{Kind: "runtime_error", Message: "first snapshot"},
			}, agentID, sessionID, epoch), nil
		},
	}
	scanStore := &scanTaskRuntimeScanStoreStub{}
	service := newScanTaskBridgeServiceForTest(taskStore, scanStore)

	err := service.ReportTerminalTaskResult(context.Background(), agentID, sessionID, epoch, 303, "failed", &FailureDetail{Kind: "runtime_error", Message: "different snapshot"})
	if err != ErrScanTaskInvalidTransition {
		t.Fatalf("different duplicate failure error = %v, want ErrScanTaskInvalidTransition", err)
	}
	if scanStore.updateCalls != 0 {
		t.Fatalf("conflicting duplicate must not reconcile scan: updates=%d", scanStore.updateCalls)
	}
}

func TestScanTaskBridgeServiceDuplicateFailedResultRejectsDifferentFailureDetail(t *testing.T) {
	agentID := 7
	sessionID := "session-7"
	epoch := int64(11)
	taskStore := &taskStoreStub{
		failedCount: 1,
		getByIDFn: func(context.Context, int) (*ScanTaskRecord, error) {
			return assignedTaskRecordForTest(ScanTaskRecord{ID: 303, ScanID: 44, ScanWorkflowStageOrder: 2, Status: "failed", Failure: &FailureDetail{Kind: "runtime_error", Message: "first snapshot", DisplayMessage: "First diagnostic."}}, agentID, sessionID, epoch), nil
		},
	}
	service := newScanTaskBridgeServiceForTest(taskStore, &scanTaskRuntimeScanStoreStub{})
	err := service.ReportTerminalTaskResult(context.Background(), agentID, sessionID, epoch, 303, "failed", &FailureDetail{Kind: "runtime_error", Message: "first snapshot", DisplayMessage: "Different diagnostic."})
	if err != ErrScanTaskInvalidTransition {
		t.Fatalf("different duplicate failure detail error = %v, want %v", err, ErrScanTaskInvalidTransition)
	}
}

func TestScanTaskBridgeServiceConcurrentConflictingTerminalResultCannotOverwriteWinner(t *testing.T) {
	agentID := 7
	sessionID := "session-7"
	epoch := int64(11)
	reads := 0
	taskStore := &taskStoreStub{
		commitSessionFn: func(context.Context, int, int, string, int64, string, *FailureDetail) (bool, error) {
			return false, nil
		},
		getByIDFn: func(context.Context, int) (*ScanTaskRecord, error) {
			reads++
			status := "running"
			if reads > 1 {
				status = "failed"
			}
			return assignedTaskRecordForTest(ScanTaskRecord{ID: 303, ScanID: 44, ScanWorkflowStageOrder: 2, Status: status}, agentID, sessionID, epoch), nil
		},
	}
	service := newScanTaskBridgeServiceForTest(taskStore, &scanTaskRuntimeScanStoreStub{})

	err := service.ReportTerminalTaskResult(context.Background(), agentID, sessionID, epoch, 303, "succeeded", nil)
	if err != ErrScanTaskInvalidTransition {
		t.Fatalf("conflicting terminal result error = %v, want ErrScanTaskInvalidTransition", err)
	}
}

func TestScanTaskBridgeServiceConcurrentDuplicateTerminalResultRetriesReconciliation(t *testing.T) {
	agentID := 7
	sessionID := "session-7"
	epoch := int64(11)
	reads := 0
	taskStore := &taskStoreStub{
		completedCount: 1,
		commitSessionFn: func(context.Context, int, int, string, int64, string, *FailureDetail) (bool, error) {
			return false, nil
		},
		getByIDFn: func(context.Context, int) (*ScanTaskRecord, error) {
			reads++
			status := "running"
			if reads > 1 {
				status = "succeeded"
			}
			return assignedTaskRecordForTest(ScanTaskRecord{ID: 303, ScanID: 44, ScanWorkflowStageOrder: 2, Status: status}, agentID, sessionID, epoch), nil
		},
	}
	scanStore := &scanTaskRuntimeScanStoreStub{}
	service := newScanTaskBridgeServiceForTest(taskStore, scanStore)

	if err := service.ReportTerminalTaskResult(context.Background(), agentID, sessionID, epoch, 303, "succeeded", nil); err != nil {
		t.Fatalf("concurrent duplicate terminal result error = %v", err)
	}
	if !taskStore.unlockCalled || scanStore.updateCalls != 1 || scanStore.lastUpdatedStatus != "succeeded" {
		t.Fatalf("concurrent duplicate did not reconcile: unlock=%v updates=%d status=%q", taskStore.unlockCalled, scanStore.updateCalls, scanStore.lastUpdatedStatus)
	}
	if !reflect.DeepEqual(taskStore.clearedTerminalIDs, []int{303}) {
		t.Fatalf("concurrent duplicate left durable marker uncleared: %v", taskStore.clearedTerminalIDs)
	}
}

func TestScanTaskBridgeServiceConcurrentFailedResultRejectsDifferentFailureSnapshot(t *testing.T) {
	agentID := 7
	sessionID := "session-7"
	epoch := int64(11)
	reads := 0
	taskStore := &taskStoreStub{
		commitSessionFn: func(context.Context, int, int, string, int64, string, *FailureDetail) (bool, error) {
			return false, nil
		},
		getByIDFn: func(context.Context, int) (*ScanTaskRecord, error) {
			reads++
			task := assignedTaskRecordForTest(ScanTaskRecord{ID: 303, ScanID: 44, ScanWorkflowStageOrder: 2, Status: "running"}, agentID, sessionID, epoch)
			if reads > 1 {
				task.Status = "failed"
				task.Failure = &FailureDetail{Kind: "runtime_error", Message: "first snapshot"}
			}
			return task, nil
		},
	}
	service := newScanTaskBridgeServiceForTest(taskStore, &scanTaskRuntimeScanStoreStub{})

	err := service.ReportTerminalTaskResult(context.Background(), agentID, sessionID, epoch, 303, "failed", &FailureDetail{Kind: "runtime_error", Message: "different snapshot"})
	if err != ErrScanTaskInvalidTransition {
		t.Fatalf("concurrent different failure error = %v, want ErrScanTaskInvalidTransition", err)
	}
}

func TestScanTaskBridgeServiceRecalculateScanStatus_ProjectsCanonicalFailureByPriority(t *testing.T) {
	timeoutAt := time.Date(2026, 3, 9, 10, 0, 0, 0, time.UTC)
	schemaAt := timeoutAt.Add(1 * time.Minute)
	taskStore := &taskStoreStub{
		failedCount: 2,
		listFailedFn: func(context.Context, int) ([]ScanTaskRecord, error) {
			return []ScanTaskRecord{
				{ID: 2, ScanID: 88, ScanWorkflowStageOrder: 2, Status: "failed", CompletedAt: &timeoutAt, Failure: &FailureDetail{Kind: "task_timeout", Message: "task timed out"}},
				{ID: 1, ScanID: 88, ScanWorkflowStageOrder: 1, Status: "failed", CompletedAt: &schemaAt, Failure: &FailureDetail{Kind: "schema_invalid", Message: "schema invalid"}},
			}, nil
		},
	}
	scanTaskRuntimeScanStore := &scanTaskRuntimeScanStoreStub{}
	service := newScanTaskBridgeServiceForTest(taskStore, scanTaskRuntimeScanStore)

	if err := service.recalculateScanStatus(context.Background(), 88); err != nil {
		t.Fatalf("recalculateScanStatus returned error: %v", err)
	}
	if scanTaskRuntimeScanStore.lastUpdatedStatus != "failed" {
		t.Fatalf("expected failed scan status, got %q", scanTaskRuntimeScanStore.lastUpdatedStatus)
	}
	if scanTaskRuntimeScanStore.lastUpdatedFailure == nil || scanTaskRuntimeScanStore.lastUpdatedFailure.Kind != "schema_invalid" {
		t.Fatalf("expected schema_invalid canonical failure, got %+v", scanTaskRuntimeScanStore.lastUpdatedFailure)
	}
}

func TestScanTaskBridgeServiceRecalculateScanStatus_CancelsUnstartedTasksBeforeFailingScan(t *testing.T) {
	completedAt := time.Date(2026, 7, 3, 10, 0, 0, 0, time.UTC)
	ops := []string{}
	taskStore := &taskStoreStub{
		ops:         &ops,
		failedCount: 1,
		listFailedFn: func(context.Context, int) ([]ScanTaskRecord, error) {
			return []ScanTaskRecord{{ID: 303, ScanID: 99, ScanWorkflowStageOrder: 1, Status: "failed", CompletedAt: &completedAt, Failure: &FailureDetail{Kind: "runtime_error", Message: "boom"}}}, nil
		},
	}
	scanTaskRuntimeScanStore := &scanTaskRuntimeScanStoreStub{ops: &ops}
	service := newScanTaskBridgeServiceForTest(taskStore, scanTaskRuntimeScanStore)

	if err := service.recalculateScanStatus(context.Background(), 99); err != nil {
		t.Fatalf("recalculateScanStatus returned error: %v", err)
	}
	if !taskStore.cancelledUnstarted || taskStore.cancelledScanID != 99 {
		t.Fatalf("expected unstarted tasks cancelled for failed scan, called=%v scanID=%d", taskStore.cancelledUnstarted, taskStore.cancelledScanID)
	}
	if taskStore.skippedUnstarted {
		t.Fatal("technical failure must not turn unstarted tasks into planning-time skipped")
	}
	if scanTaskRuntimeScanStore.lastUpdatedStatus != "failed" {
		t.Fatalf("expected failed scan status, got %q", scanTaskRuntimeScanStore.lastUpdatedStatus)
	}
	if !reflect.DeepEqual(ops, []string{"cancel-unstarted", "update-scan"}) {
		t.Fatalf("expected unstarted tasks closed before scan update, got ops=%v", ops)
	}
}

func TestScanTaskBridgeServiceRecalculateScanStatus_CancelsUnstartedTasksBeforeCancellingScan(t *testing.T) {
	ops := []string{}
	taskStore := &taskStoreStub{cancelledCount: 1, ops: &ops}
	scanTaskRuntimeScanStore := &scanTaskRuntimeScanStoreStub{ops: &ops}
	service := newScanTaskBridgeServiceForTest(taskStore, scanTaskRuntimeScanStore)

	if err := service.recalculateScanStatus(context.Background(), 100); err != nil {
		t.Fatalf("recalculateScanStatus returned error: %v", err)
	}
	if !taskStore.cancelledUnstarted || taskStore.cancelledScanID != 100 {
		t.Fatalf("expected unstarted tasks cancelled for cancelled scan, called=%v scanID=%d", taskStore.cancelledUnstarted, taskStore.cancelledScanID)
	}
	if scanTaskRuntimeScanStore.lastUpdatedStatus != "cancelled" {
		t.Fatalf("expected cancelled scan status, got %q", scanTaskRuntimeScanStore.lastUpdatedStatus)
	}
	if !reflect.DeepEqual(ops, []string{"cancel-unstarted", "update-scan"}) {
		t.Fatalf("expected unstarted tasks closed before scan update, got ops=%v", ops)
	}
}

func TestScanTaskBridgeServiceFenceSupersededSessionRetriesPartialReconciliation(t *testing.T) {
	completedAt := time.Date(2026, 7, 22, 10, 0, 0, 0, time.UTC)
	fenceCalls := 0
	taskStore := &taskStoreStub{
		failedCount: 1,
		fenceSupersededFn: func(_ context.Context, agentID int, epoch int64) ([]int, error) {
			fenceCalls++
			if agentID != 7 || epoch != 12 {
				t.Fatalf("fence scope = agent %d epoch %d", agentID, epoch)
			}
			return []int{99}, nil
		},
		listFailedFn: func(context.Context, int) ([]ScanTaskRecord, error) {
			return []ScanTaskRecord{{ID: 303, ScanID: 99, Status: "failed", CompletedAt: &completedAt, Failure: &FailureDetail{Kind: "agent_disconnected", Message: "Agent disconnected"}}}, nil
		},
	}
	scanStore := &scanTaskRuntimeScanStoreStub{updateErrors: []error{errors.New("temporary scan persistence failure"), nil}}
	service := newScanTaskBridgeServiceForTest(taskStore, scanStore)

	if err := service.FenceSupersededAgentSession(context.Background(), 7, 12); err == nil {
		t.Fatal("first fence must surface reconciliation failure")
	}
	if err := service.FenceSupersededAgentSession(context.Background(), 7, 12); err != nil {
		t.Fatalf("replayed fence failed: %v", err)
	}
	if fenceCalls != 2 || scanStore.updateCalls != 2 || !taskStore.cancelledUnstarted {
		t.Fatalf("fence replay did not finish reconciliation: fence=%d scan_updates=%d cancelled=%v", fenceCalls, scanStore.updateCalls, taskStore.cancelledUnstarted)
	}
}

func TestScanTaskBridgeServiceRecalculateScanStatus_ProjectsCanonicalFailureStableTieBreak(t *testing.T) {
	later := time.Date(2026, 3, 9, 11, 1, 0, 0, time.UTC)
	earlier := time.Date(2026, 3, 9, 11, 0, 0, 0, time.UTC)
	taskStore := &taskStoreStub{
		failedCount: 3,
		listFailedFn: func(context.Context, int) ([]ScanTaskRecord, error) {
			return []ScanTaskRecord{
				{ID: 5, ScanID: 89, ScanWorkflowStageOrder: 3, Status: "failed", CompletedAt: &later, Failure: &FailureDetail{Kind: "runtime_error", Message: "later stage"}},
				{ID: 4, ScanID: 89, ScanWorkflowStageOrder: 2, Status: "failed", CompletedAt: &later, Failure: &FailureDetail{Kind: "runtime_error", Message: "earlier stage"}},
				{ID: 3, ScanID: 89, ScanWorkflowStageOrder: 2, Status: "failed", CompletedAt: &earlier, Failure: &FailureDetail{Kind: "runtime_error", Message: "earliest completion"}},
			}, nil
		},
	}
	scanTaskRuntimeScanStore := &scanTaskRuntimeScanStoreStub{}
	service := newScanTaskBridgeServiceForTest(taskStore, scanTaskRuntimeScanStore)

	if err := service.recalculateScanStatus(context.Background(), 89); err != nil {
		t.Fatalf("recalculateScanStatus returned error: %v", err)
	}
	if scanTaskRuntimeScanStore.lastUpdatedFailure == nil || scanTaskRuntimeScanStore.lastUpdatedFailure.Message != "earliest completion" {
		t.Fatalf("expected stable canonical failure, got %+v", scanTaskRuntimeScanStore.lastUpdatedFailure)
	}
}

func TestTaskExecutionErrorCodeToFailureKind_MapsDomainCodes(t *testing.T) {
	if got := taskExecutionErrorCodeToFailureKind(TaskExecutionErrorCodeSchemaInvalid); got != "schema_invalid" {
		t.Fatalf("expected schema_invalid, got %q", got)
	}
}
