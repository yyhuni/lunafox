package application

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	cleanupdomain "github.com/yyhuni/lunafox/server/internal/modules/targetcleanup/domain"
)

type reconciliationStoreStub struct {
	steps          []string
	assetBatches   map[cleanupdomain.AssetResource][]int64
	completed      bool
	completionCall int
}

func (stub *reconciliationStoreStub) ConfirmTombstone(context.Context, int) error {
	stub.steps = append(stub.steps, "confirm")
	return nil
}

func (stub *reconciliationStoreStub) DeleteTargetControlPlane(context.Context, int) (int64, int64, error) {
	stub.steps = append(stub.steps, "control")
	return 2, 1, nil
}

func (stub *reconciliationStoreStub) DeleteCurrentAssetBatch(_ context.Context, _ int, resource cleanupdomain.AssetResource, _ int) (int64, error) {
	stub.steps = append(stub.steps, "asset:"+string(resource))
	items := stub.assetBatches[resource]
	if len(items) == 0 {
		return 0, nil
	}
	stub.assetBatches[resource] = items[1:]
	return items[0], nil
}

func (stub *reconciliationStoreStub) MarkCompletedIfClear(context.Context, int, int, time.Time) (bool, error) {
	stub.steps = append(stub.steps, "complete")
	stub.completionCall++
	return stub.completed, nil
}

type scheduleCleanerStub struct {
	steps   *[]string
	deleted int
}

func (stub scheduleCleanerStub) DeleteTargetScopedSchedules(context.Context, int) (int, error) {
	*stub.steps = append(*stub.steps, "schedules")
	return stub.deleted, nil
}

type scanCancellerStub struct {
	steps         *[]string
	cancellations []*TargetScanCancellation
}

func (stub *scanCancellerStub) CancelNextActiveScan(context.Context, int, time.Time) (*TargetScanCancellation, error) {
	*stub.steps = append(*stub.steps, "scan")
	if len(stub.cancellations) == 0 {
		return nil, nil
	}
	next := stub.cancellations[0]
	stub.cancellations = stub.cancellations[1:]
	return next, nil
}

type cancelPublisherStub struct {
	notifications []TargetTaskCancelNotification
	delivered     map[int]bool
}

type restartReconciliationStoreStub struct {
	steps                   []string
	relationships           int64
	policies                int64
	assets                  map[cleanupdomain.AssetResource]int64
	failBeforeControlOnce   bool
	failAfterCommittedAsset bool
	confirmations           int
	controlCalls            int
	assetCalls              int
	completionCalls         int
}

func (stub *restartReconciliationStoreStub) ConfirmTombstone(context.Context, int) error {
	stub.confirmations++
	stub.steps = append(stub.steps, "confirm")
	return nil
}

func (stub *restartReconciliationStoreStub) DeleteTargetControlPlane(context.Context, int) (int64, int64, error) {
	stub.controlCalls++
	stub.steps = append(stub.steps, "control")
	if stub.failBeforeControlOnce {
		stub.failBeforeControlOnce = false
		return 0, 0, errors.New("injected rollback before control-plane commit")
	}
	relationships, policies := stub.relationships, stub.policies
	stub.relationships, stub.policies = 0, 0
	return relationships, policies, nil
}

func (stub *restartReconciliationStoreStub) DeleteCurrentAssetBatch(_ context.Context, _ int, resource cleanupdomain.AssetResource, _ int) (int64, error) {
	stub.assetCalls++
	stub.steps = append(stub.steps, "asset:"+string(resource))
	if stub.assets[resource] == 0 {
		return 0, nil
	}
	stub.assets[resource]--
	if stub.failAfterCommittedAsset {
		stub.failAfterCommittedAsset = false
		return 1, errors.New("injected failure after committed asset batch")
	}
	return 1, nil
}

func (stub *restartReconciliationStoreStub) MarkCompletedIfClear(context.Context, int, int, time.Time) (bool, error) {
	stub.completionCalls++
	stub.steps = append(stub.steps, "complete")
	if stub.relationships != 0 || stub.policies != 0 {
		return false, nil
	}
	for _, remaining := range stub.assets {
		if remaining != 0 {
			return false, nil
		}
	}
	return true, nil
}

type restartScheduleCleanerStub struct {
	steps     *[]string
	remaining int
	calls     int
}

func (stub *restartScheduleCleanerStub) DeleteTargetScopedSchedules(context.Context, int) (int, error) {
	stub.calls++
	*stub.steps = append(*stub.steps, "schedules")
	deleted := stub.remaining
	stub.remaining = 0
	return deleted, nil
}

func (stub *cancelPublisherStub) TrySendTaskCancel(agentID, scanID, taskID int) bool {
	stub.notifications = append(stub.notifications, TargetTaskCancelNotification{AgentID: agentID, TaskID: taskID})
	return stub.delivered[taskID]
}

func TestTargetCleanupReconciliationServiceConvergesInFixedOrder(t *testing.T) {
	now := time.Date(2026, 8, 6, 12, 0, 0, 0, time.UTC)
	store := &reconciliationStoreStub{
		assetBatches: map[cleanupdomain.AssetResource][]int64{
			cleanupdomain.AssetResourceSubdomain:     {2, 1, 0},
			cleanupdomain.AssetResourceVulnerability: {1, 0},
		},
		completed: true,
	}
	schedules := scheduleCleanerStub{steps: &store.steps, deleted: 3}
	scans := &scanCancellerStub{steps: &store.steps, cancellations: []*TargetScanCancellation{{
		ScanID: 71, CancelledTaskCount: 2,
		NotificationCandidates: []TargetTaskCancelNotification{{TaskID: 701, AgentID: 9}, {TaskID: 702, AgentID: 10}},
	}}}
	publisher := &cancelPublisherStub{delivered: map[int]bool{701: true, 702: false}}
	service, err := NewTargetCleanupReconciliationService(store, schedules, scans, publisher)
	if err != nil {
		t.Fatalf("NewTargetCleanupReconciliationService(): %v", err)
	}
	service.WithClock(func() time.Time { return now })

	result, err := service.Reconcile(context.Background(), cleanupdomain.CleanupJob{ID: 1, TargetID: 7}, TargetCleanupRunOptions{
		AssetBatchSize: 2, MaxAssetBatches: 10, MaxRunDuration: time.Minute,
	})
	if err != nil || !result.Completed || result.Deferred {
		t.Fatalf("Reconcile() = %+v, %v", result, err)
	}
	if result.AssetBatches != 3 || result.Counts.Schedules != 3 || result.Counts.OrganizationRelations != 2 || result.Counts.TargetPolicies != 1 || result.Counts.Scans != 1 || result.Counts.Tasks != 2 || result.Counts.AgentNotifications != 2 || result.Counts.AgentNotificationFails != 1 {
		t.Fatalf("unexpected reconciliation counts: %+v", result)
	}
	if result.Counts.AssetRows[cleanupdomain.AssetResourceSubdomain] != 3 || result.Counts.AssetRows[cleanupdomain.AssetResourceVulnerability] != 1 {
		t.Fatalf("asset counts = %+v", result.Counts.AssetRows)
	}
	if got := fmt.Sprint(store.steps[:4]); got != "[confirm schedules control scan]" {
		t.Fatalf("control-plane order = %s", got)
	}
	if store.steps[len(store.steps)-1] != "complete" || store.completionCall != 1 {
		t.Fatalf("completion was not the final operation: %+v", store.steps)
	}
	if len(publisher.notifications) != 2 {
		t.Fatalf("post-commit notification candidates = %+v", publisher.notifications)
	}
}

func TestTargetCleanupReconciliationServiceDefersAtAssetBudgetWithoutCompletion(t *testing.T) {
	now := time.Date(2026, 8, 6, 12, 0, 0, 0, time.UTC)
	store := &reconciliationStoreStub{
		assetBatches: map[cleanupdomain.AssetResource][]int64{
			cleanupdomain.AssetResourceSubdomain: {1, 1},
		},
		completed: true,
	}
	schedules := scheduleCleanerStub{steps: &store.steps}
	scans := &scanCancellerStub{steps: &store.steps}
	service, err := NewTargetCleanupReconciliationService(store, schedules, scans, &cancelPublisherStub{})
	if err != nil {
		t.Fatalf("NewTargetCleanupReconciliationService(): %v", err)
	}
	service.WithClock(func() time.Time { return now })

	result, err := service.Reconcile(context.Background(), cleanupdomain.CleanupJob{ID: 1, TargetID: 7}, TargetCleanupRunOptions{
		AssetBatchSize: 1, MaxAssetBatches: 1, MaxRunDuration: time.Minute,
	})
	if err != nil || !result.Deferred || result.Completed || result.AssetBatches != 1 {
		t.Fatalf("Reconcile() = %+v, %v", result, err)
	}
	if store.completionCall != 0 {
		t.Fatalf("budget exhaustion must not force completion: %+v", store.steps)
	}
}

func TestTargetCleanupReconciliationServiceRestartRechecksCommittedSteps(t *testing.T) {
	now := time.Date(2026, 8, 6, 12, 0, 0, 0, time.UTC)
	store := &restartReconciliationStoreStub{
		relationships:           1,
		policies:                1,
		assets:                  map[cleanupdomain.AssetResource]int64{cleanupdomain.AssetResourceSubdomain: 2},
		failAfterCommittedAsset: true,
	}
	schedules := &restartScheduleCleanerStub{steps: &store.steps, remaining: 1}
	first, err := NewTargetCleanupReconciliationService(store, schedules, &scanCancellerStub{steps: &store.steps}, &cancelPublisherStub{})
	if err != nil {
		t.Fatalf("NewTargetCleanupReconciliationService(first): %v", err)
	}
	first.WithClock(func() time.Time { return now })
	if _, err := first.Reconcile(context.Background(), cleanupdomain.CleanupJob{ID: 1, TargetID: 7}, TargetCleanupRunOptions{AssetBatchSize: 1, MaxAssetBatches: 10, MaxRunDuration: time.Minute}); err == nil {
		t.Fatal("first reconciliation must surface the injected post-commit failure")
	}
	if store.relationships != 0 || store.policies != 0 || schedules.remaining != 0 || store.assets[cleanupdomain.AssetResourceSubdomain] != 1 {
		t.Fatalf("committed work did not survive failure as database truth: relations=%d policies=%d schedules=%d assets=%d", store.relationships, store.policies, schedules.remaining, store.assets[cleanupdomain.AssetResourceSubdomain])
	}

	restarted, err := NewTargetCleanupReconciliationService(store, schedules, &scanCancellerStub{steps: &store.steps}, &cancelPublisherStub{})
	if err != nil {
		t.Fatalf("NewTargetCleanupReconciliationService(restarted): %v", err)
	}
	restarted.WithClock(func() time.Time { return now.Add(time.Minute) })
	result, err := restarted.Reconcile(context.Background(), cleanupdomain.CleanupJob{ID: 1, TargetID: 7}, TargetCleanupRunOptions{AssetBatchSize: 1, MaxAssetBatches: 10, MaxRunDuration: time.Minute})
	if err != nil || !result.Completed {
		t.Fatalf("restarted Reconcile() = %+v, %v", result, err)
	}
	if store.confirmations != 2 || schedules.calls != 2 || store.controlCalls != 2 || store.assets[cleanupdomain.AssetResourceSubdomain] != 0 || store.completionCalls != 1 {
		t.Fatalf("restart did not begin from reconciliation step one: confirmations=%d schedules=%d controls=%d assets=%d completions=%d", store.confirmations, schedules.calls, store.controlCalls, store.assets[cleanupdomain.AssetResourceSubdomain], store.completionCalls)
	}
}

func TestTargetCleanupReconciliationServiceRestartRediscoversRolledBackStep(t *testing.T) {
	now := time.Date(2026, 8, 6, 12, 0, 0, 0, time.UTC)
	store := &restartReconciliationStoreStub{
		relationships:         1,
		policies:              1,
		assets:                map[cleanupdomain.AssetResource]int64{cleanupdomain.AssetResourceSubdomain: 1},
		failBeforeControlOnce: true,
	}
	schedules := &restartScheduleCleanerStub{steps: &store.steps}
	service, err := NewTargetCleanupReconciliationService(store, schedules, &scanCancellerStub{steps: &store.steps}, &cancelPublisherStub{})
	if err != nil {
		t.Fatalf("NewTargetCleanupReconciliationService(): %v", err)
	}
	service.WithClock(func() time.Time { return now })
	if _, err := service.Reconcile(context.Background(), cleanupdomain.CleanupJob{ID: 1, TargetID: 7}, TargetCleanupRunOptions{AssetBatchSize: 1, MaxAssetBatches: 10, MaxRunDuration: time.Minute}); err == nil {
		t.Fatal("first reconciliation must surface the injected pre-commit failure")
	}
	if store.relationships != 1 || store.policies != 1 || store.assets[cleanupdomain.AssetResourceSubdomain] != 1 {
		t.Fatalf("rolled-back control-plane state drifted: relations=%d policies=%d assets=%d", store.relationships, store.policies, store.assets[cleanupdomain.AssetResourceSubdomain])
	}

	result, err := service.Reconcile(context.Background(), cleanupdomain.CleanupJob{ID: 1, TargetID: 7}, TargetCleanupRunOptions{AssetBatchSize: 1, MaxAssetBatches: 10, MaxRunDuration: time.Minute})
	if err != nil || !result.Completed {
		t.Fatalf("retry Reconcile() = %+v, %v", result, err)
	}
	if store.relationships != 0 || store.policies != 0 || store.assets[cleanupdomain.AssetResourceSubdomain] != 0 || store.controlCalls != 2 {
		t.Fatalf("retry did not rediscover rolled-back state: relations=%d policies=%d assets=%d controlCalls=%d", store.relationships, store.policies, store.assets[cleanupdomain.AssetResourceSubdomain], store.controlCalls)
	}
}
