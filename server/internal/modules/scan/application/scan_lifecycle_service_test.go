package application

import (
	"context"
	"errors"
	"testing"
	"time"

	scandomain "github.com/yyhuni/lunafox/server/internal/modules/scan/domain"
)

type lifecycleScanStoreStub struct {
}

func (stub *lifecycleScanStoreStub) GetLifecycleRefByID(id int) (*QueryScan, error) {
	return nil, nil
}

func (stub *lifecycleScanStoreStub) FindByIDs(ids []int) ([]QueryScan, error) {
	return nil, nil
}

func (stub *lifecycleScanStoreStub) BatchSoftDelete(ids []int) (int64, []string, error) {
	return 0, nil, nil
}

func (stub *lifecycleScanStoreStub) UpdateScanStatus(id int, status string, failure *FailureDetail) error {
	return nil
}

type lifecycleScanStopStoreStub struct {
	err          error
	outcome      *ScanStopOutcome
	batchErr     error
	batchOutcome *BatchScanStopOutcome
	calls        int
	batchCalls   int
	ctx          context.Context
	scanID       int
	batchIDs     []int
	stopped      time.Time
	onCall       func()
}

func (stub *lifecycleScanStopStoreStub) StopActiveScan(ctx context.Context, scanID int, stoppedAt time.Time) (*ScanStopOutcome, error) {
	stub.calls++
	stub.ctx = ctx
	stub.scanID = scanID
	stub.stopped = stoppedAt
	if stub.onCall != nil {
		stub.onCall()
	}
	return stub.outcome, stub.err
}

func (stub *lifecycleScanStopStoreStub) BatchStopActiveScans(ctx context.Context, scanIDs []int, stoppedAt time.Time) (*BatchScanStopOutcome, error) {
	stub.batchCalls++
	stub.ctx = ctx
	stub.batchIDs = append([]int(nil), scanIDs...)
	stub.stopped = stoppedAt
	if stub.onCall != nil {
		stub.onCall()
	}
	return stub.batchOutcome, stub.batchErr
}

type lifecycleTaskCancelNotifierStub struct {
	delivered bool
	calls     []ScanStopNotification
}

func (stub *lifecycleTaskCancelNotifierStub) TrySendTaskCancel(agentID, scanID, taskID int) bool {
	stub.calls = append(stub.calls, ScanStopNotification{AgentID: agentID, TaskID: taskID})
	return stub.delivered
}

type lifecycleFixedClock struct{ now time.Time }

func (clock lifecycleFixedClock) Now() time.Time { return clock.now }

func TestLifecycleServiceStopActiveForDeleteIgnoresCannotStop(t *testing.T) {
	store := &lifecycleScanStoreStub{}
	service := NewLifecycleService(store, &lifecycleScanStopStoreStub{err: scandomain.ErrScanCannotStop}, nil)

	before := scanDeleteStopIgnoredTotal.Value()
	_, err := service.stopActiveForDelete(context.Background(), &QueryScan{
		ID:     1001,
		Status: string(scandomain.ScanStatusRunning),
	})
	after := scanDeleteStopIgnoredTotal.Value()

	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if after != before+1 {
		t.Fatalf("expected metric +1, before=%d after=%d", before, after)
	}
}

func TestLifecycleServiceStopActiveForDeleteIgnoresInvalidStatusChange(t *testing.T) {
	store := &lifecycleScanStoreStub{}
	service := NewLifecycleService(store, &lifecycleScanStopStoreStub{err: scandomain.ErrInvalidStatusChange}, nil)

	before := scanDeleteStopIgnoredTotal.Value()
	_, err := service.stopActiveForDelete(context.Background(), &QueryScan{
		ID:     1002,
		Status: string(scandomain.ScanStatusRunning),
	})
	after := scanDeleteStopIgnoredTotal.Value()

	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if after != before+1 {
		t.Fatalf("expected metric +1, before=%d after=%d", before, after)
	}
}

func TestLifecycleServiceStopActiveForDeleteReturnsUnexpectedError(t *testing.T) {
	unexpected := errors.New("boom")
	store := &lifecycleScanStoreStub{}
	service := NewLifecycleService(store, &lifecycleScanStopStoreStub{err: unexpected}, nil)

	before := scanDeleteStopIgnoredTotal.Value()
	_, err := service.stopActiveForDelete(context.Background(), &QueryScan{
		ID:     1003,
		Status: string(scandomain.ScanStatusRunning),
	})
	after := scanDeleteStopIgnoredTotal.Value()

	if !errors.Is(err, unexpected) {
		t.Fatalf("expected unexpected error, got %v", err)
	}
	if after != before {
		t.Fatalf("expected metric unchanged, before=%d after=%d", before, after)
	}
}

func TestLifecycleServiceStopScanCommitsBeforeBestEffortNotification(t *testing.T) {
	stoppedAt := time.Date(2026, time.August, 9, 2, 3, 4, 0, time.FixedZone("test", 8*60*60))
	notifier := &lifecycleTaskCancelNotifierStub{delivered: true}
	store := &lifecycleScanStopStoreStub{outcome: &ScanStopOutcome{
		ScanID:             9,
		CancelledTaskCount: 2,
		NotificationCandidates: []ScanStopNotification{
			{TaskID: 14, AgentID: 27},
		},
	}}
	store.onCall = func() {
		if len(notifier.calls) != 0 {
			t.Fatal("task cancellation notification was sent before Stop transaction returned")
		}
	}
	service := NewLifecycleService(nil, store, notifier, lifecycleFixedClock{now: stoppedAt})
	contextKey := struct{}{}
	ctx := context.WithValue(context.Background(), contextKey, "request-context")

	count, err := service.StopScan(ctx, 9)
	if err != nil {
		t.Fatalf("StopScan: %v", err)
	}
	if count != 2 || store.calls != 1 || store.scanID != 9 {
		t.Fatalf("Stop outcome = count:%d calls:%d scan:%d", count, store.calls, store.scanID)
	}
	if got := store.ctx.Value(contextKey); got != "request-context" {
		t.Fatalf("Stop store did not receive caller context value: %v", got)
	}
	if !store.stopped.Equal(stoppedAt.UTC()) {
		t.Fatalf("shared stop time = %s, want %s", store.stopped, stoppedAt.UTC())
	}
	if len(notifier.calls) != 1 || notifier.calls[0] != (ScanStopNotification{TaskID: 14, AgentID: 27}) {
		t.Fatalf("post-commit notification = %+v", notifier.calls)
	}
}

func TestLifecycleServiceStopScanKeepsCommittedSuccessWhenNotificationFails(t *testing.T) {
	logs := withObservedLogger(t)
	notifier := &lifecycleTaskCancelNotifierStub{delivered: false}
	store := &lifecycleScanStopStoreStub{outcome: &ScanStopOutcome{
		ScanID:             11,
		CancelledTaskCount: 1,
		NotificationCandidates: []ScanStopNotification{
			{TaskID: 17, AgentID: 31},
		},
	}}
	service := NewLifecycleService(nil, store, notifier)
	before := scanStopCancelDeliveryFailedTotal.Value()

	count, err := service.StopScan(context.Background(), 11)
	if err != nil || count != 1 {
		t.Fatalf("StopScan with delivery failure = count:%d err:%v", count, err)
	}
	if got := scanStopCancelDeliveryFailedTotal.Value(); got != before+1 {
		t.Fatalf("delivery failure count = %d, want %d", got, before+1)
	}
	entries := logs.FilterMessage("scan task cancellation notification was not accepted").All()
	if len(entries) != 1 {
		t.Fatalf("delivery warning count = %d", len(entries))
	}
	fields := entries[0].ContextMap()
	for _, field := range []string{"agent.id", "scan.id", "task.id"} {
		if _, ok := fields[field]; !ok {
			t.Fatalf("delivery warning missing %s: %v", field, fields)
		}
	}
}

func TestLifecycleServiceStopScanDoesNotStartAfterCallerCancellation(t *testing.T) {
	store := &lifecycleScanStopStoreStub{}
	service := NewLifecycleService(nil, store, nil)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := service.StopScan(ctx, 12)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("StopScan cancellation error = %v, want context.Canceled", err)
	}
	if store.calls != 0 {
		t.Fatalf("cancelled Stop called repository %d times", store.calls)
	}
}

func TestLifecycleServiceBatchStopScansPreservesContextAndNotifiesAfterCommit(t *testing.T) {
	stoppedAt := time.Date(2026, time.August, 18, 12, 30, 0, 0, time.UTC)
	notifier := &lifecycleTaskCancelNotifierStub{delivered: true}
	store := &lifecycleScanStopStoreStub{batchOutcome: &BatchScanStopOutcome{
		StoppedCount:     2,
		SkippedCount:     1,
		RevokedTaskCount: 3,
		NotificationCandidates: []BatchScanStopNotification{{
			ScanID:  21,
			TaskID:  22,
			AgentID: 23,
		}},
	}}
	store.onCall = func() {
		if len(notifier.calls) != 0 {
			t.Fatal("batch notification was sent before repository commit")
		}
	}
	service := NewLifecycleService(nil, store, notifier, lifecycleFixedClock{now: stoppedAt})
	contextKey := struct{}{}
	ctx := context.WithValue(context.Background(), contextKey, "batch-stop-request")

	outcome, err := service.BatchStopScans(ctx, []int{22, 21})
	if err != nil {
		t.Fatalf("BatchStopScans: %v", err)
	}
	if outcome.StoppedCount != 2 || outcome.SkippedCount != 1 || outcome.RevokedTaskCount != 3 {
		t.Fatalf("unexpected outcome: %+v", outcome)
	}
	if store.batchCalls != 1 || len(store.batchIDs) != 2 || store.batchIDs[0] != 22 || store.batchIDs[1] != 21 {
		t.Fatalf("batch store input = calls:%d ids:%v", store.batchCalls, store.batchIDs)
	}
	if got := store.ctx.Value(contextKey); got != "batch-stop-request" {
		t.Fatalf("batch stop store lost request context: %v", got)
	}
	if !store.stopped.Equal(stoppedAt.UTC()) {
		t.Fatalf("batch stop time = %s, want %s", store.stopped, stoppedAt.UTC())
	}
	if len(notifier.calls) != 1 || notifier.calls[0] != (ScanStopNotification{TaskID: 22, AgentID: 23}) {
		t.Fatalf("batch notification calls = %+v", notifier.calls)
	}
}

func TestLifecycleServiceBatchStopScansDoesNotStartAfterCallerCancellation(t *testing.T) {
	store := &lifecycleScanStopStoreStub{}
	service := NewLifecycleService(nil, store, nil)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := service.BatchStopScans(ctx, []int{21})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("BatchStopScans cancellation error = %v, want context.Canceled", err)
	}
	if store.batchCalls != 0 {
		t.Fatalf("cancelled batch stop called repository %d times", store.batchCalls)
	}
}
