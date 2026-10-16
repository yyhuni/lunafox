package application

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

func TestScanStopRuntimeLogsKeepCommittedOutcomesAndReplayIdentity(t *testing.T) {
	for _, mode := range []string{"single", "batch", "upgrade"} {
		t.Run(mode, func(t *testing.T) {
			logs := withObservedLogger(t)
			cancelled := []ScanStopTask{{ScanID: 7, TaskID: 101}, {ScanID: 7, TaskID: 102}}
			store := &lifecycleScanStopStoreStub{
				outcome: &ScanStopOutcome{ScanID: 7, CancelledTaskCount: 2, CancelledTasks: cancelled,
					NotificationCandidates: []ScanStopNotification{{TaskID: 101, AgentID: 17}}},
				batchOutcome: &BatchScanStopOutcome{StoppedCount: 1, RevokedTaskCount: 2, CancelledTasks: cancelled,
					NotificationCandidates: []BatchScanStopNotification{{ScanID: 7, TaskID: 101, AgentID: 17}}},
				upgradeOutcome: &UpgradeScanStopOutcome{CancelledScanCount: 1, CancelledTaskCount: 2, CancelledTasks: cancelled,
					NotificationCandidates: []BatchScanStopNotification{{ScanID: 7, TaskID: 101, AgentID: 17}}},
			}
			logStore := &runtimeLogStoreStub{}
			notifier := &lifecycleTaskCancelNotifierStub{delivered: true}
			service := NewLifecycleService(nil, store, notifier).WithTaskRuntimeLogAppender(NewTaskRuntimeLogAppender(logStore))
			for attempt := range 2 {
				store.onCall = func() {
					if len(logStore.requests) != attempt*2 || len(notifier.calls) != attempt {
						t.Fatal("evidence or notification ran before stop committed")
					}
				}
				switch mode {
				case "single":
					count, err := service.StopScan(context.Background(), 7)
					if err != nil || count != 2 {
						t.Fatalf("single outcome changed: %d %v", count, err)
					}
				case "batch":
					outcome, err := service.BatchStopScans(context.Background(), []int{7})
					if err != nil || outcome != store.batchOutcome {
						t.Fatalf("batch outcome changed: %#v %v", outcome, err)
					}
				case "upgrade":
					outcome, err := service.StopAllActiveForUpgrade(context.Background(), "upgrade-operation-17")
					if err != nil || outcome != store.upgradeOutcome {
						t.Fatalf("upgrade outcome changed: %#v %v", outcome, err)
					}
				}
			}
			if len(logStore.requests) != 4 || len(notifier.calls) != 2 || logs.FilterMessage("task runtime log persistence failed").Len() != 0 {
				t.Fatal("stop evidence failure or notification was lost")
			}
			for index := range 2 {
				first, replay := logStore.requests[index], logStore.requests[index+2]
				if first.TaskID != cancelled[index].TaskID || first.ScanID != 7 || first.RequestID != replay.RequestID || !strings.Contains(first.Items[0].Content, "cancelled") {
					t.Fatalf("stop identity/content changed: %#v %#v", first, replay)
				}
			}
		})
	}
}

func TestScanStopRuntimeLogsDoNotPrecedeFailedTransaction(t *testing.T) {
	logStore := &runtimeLogStoreStub{}
	store := &lifecycleScanStopStoreStub{batchErr: errors.New("transaction rolled back")}
	service := NewLifecycleService(nil, store, nil).WithTaskRuntimeLogAppender(NewTaskRuntimeLogAppender(logStore))
	if _, err := service.BatchStopScans(context.Background(), []int{7}); err == nil || len(logStore.requests) != 0 {
		t.Fatal("failed transaction produced committed cancellation evidence")
	}
}

func TestSessionFenceRuntimeLogsFilterOwnershipAndKeepReplayIdentity(t *testing.T) {
	failed := func(id, scanID, agentID int, epoch int64, kind string) ScanTaskRecord {
		return *assignedTaskRecordForTest(ScanTaskRecord{ID: id, ScanID: scanID, Status: "failed", Failure: &FailureDetail{Kind: kind, Message: "Agent disconnected /raw/path token=canary"}}, agentID, "old-session", epoch)
	}
	committed := false
	tasks := []ScanTaskRecord{
		failed(101, 7, 17, 23, "agent_disconnected"),
		failed(102, 7, 18, 23, "agent_disconnected"),
		failed(103, 7, 17, 24, "agent_disconnected"),
		failed(104, 7, 17, 23, "engine_exit_failed"),
		failed(105, 8, 17, 23, "agent_disconnected"),
	}
	store := &taskStoreStub{pendingCount: 1,
		fenceSupersededFn: func(context.Context, int, int64) ([]int, error) {
			committed = true
			return []int{7}, nil
		},
		listFailedFn: func(ctx context.Context, scanID int) ([]ScanTaskRecord, error) {
			if !committed || scanID != 7 {
				t.Fatal("log lookup preceded the session fence commit")
			}
			if _, ok := ctx.Deadline(); !ok {
				t.Fatal("post-fence evidence lookup has no deadline")
			}
			return tasks, nil
		},
	}
	logStore := &runtimeLogStoreStub{}
	service := newScanTaskBridgeServiceForTest(store, &scanTaskRuntimeScanStoreStub{}).WithTaskRuntimeLogAppender(NewTaskRuntimeLogAppender(logStore))
	for range 2 {
		if err := service.FenceSupersededAgentSession(context.Background(), 17, 24); err != nil {
			t.Fatalf("fence failed: %v", err)
		}
	}
	if len(logStore.requests) != 2 || logStore.requests[0].RequestID != logStore.requests[1].RequestID || logStore.requests[0].TaskID != 101 {
		t.Fatalf("fence scope/replay identity = %#v", logStore.requests)
	}
	content := logStore.requests[0].Items[0].Content
	if !strings.Contains(content, "session epoch 23 was superseded") || !strings.Contains(content, tasks[0].Failure.Message) {
		t.Fatalf("fence reason/raw error missing: %q", content)
	}
}

func TestSessionFenceRuntimeLogFailuresDoNotChangeRecovery(t *testing.T) {
	for _, stage := range []string{"lookup", "write"} {
		t.Run(stage, func(t *testing.T) {
			logs := withObservedLogger(t)
			store := &taskStoreStub{pendingCount: 1,
				fenceSupersededFn: func(context.Context, int, int64) ([]int, error) { return []int{7}, nil },
				listFailedFn: func(context.Context, int) ([]ScanTaskRecord, error) {
					if stage == "lookup" {
						return nil, errors.New("raw lookup error /srv/private")
					}
					return []ScanTaskRecord{*assignedTaskRecordForTest(ScanTaskRecord{ID: 101, ScanID: 7, Status: "failed", Failure: &FailureDetail{Kind: "agent_disconnected", Message: "Agent disconnected"}}, 17, "old-session", 23)}, nil
				},
			}
			logStore := &runtimeLogStoreStub{err: errors.New("log DB unavailable")}
			service := newScanTaskBridgeServiceForTest(store, &scanTaskRuntimeScanStoreStub{}).WithTaskRuntimeLogAppender(NewTaskRuntimeLogAppender(logStore))
			if err := service.FenceSupersededAgentSession(context.Background(), 17, 24); err != nil {
				t.Fatalf("evidence %s failure changed fence result: %v", stage, err)
			}
			message := "task runtime log persistence failed"
			if stage == "lookup" {
				message = "session fence runtime log task lookup failed"
			}
			if logs.FilterMessage(message).Len() != 1 {
				t.Fatalf("evidence %s failure is not observable", stage)
			}
		})
	}
}

func TestScanStopRuntimeLogBudgetDoesNotDelayCancellationDelivery(t *testing.T) {
	logs := withObservedLogger(t)
	cancelled := make([]ScanStopTask, 100)
	for index := range cancelled {
		cancelled[index] = ScanStopTask{ScanID: 7, TaskID: 101 + index}
	}
	store := &lifecycleScanStopStoreStub{batchOutcome: &BatchScanStopOutcome{
		StoppedCount: 1, RevokedTaskCount: len(cancelled), CancelledTasks: cancelled,
		NotificationCandidates: []BatchScanStopNotification{{ScanID: 7, TaskID: 101, AgentID: 17}},
	}}
	notifier := &lifecycleTaskCancelNotifierStub{delivered: true}
	logStore := &runtimeLogStoreStub{onWrite: func(ctx context.Context, _ *TaskProgressLogBatchCreateRequest) error {
		if len(notifier.calls) != 1 {
			t.Fatal("cancellation delivery was delayed by runtime evidence")
		}
		if _, ok := ctx.Deadline(); !ok {
			t.Fatal("bulk stop has no bounded runtime evidence budget")
		}
		return errors.New("log database unavailable")
	}}
	service := NewLifecycleService(nil, store, notifier).WithTaskRuntimeLogAppender(NewTaskRuntimeLogAppender(logStore))
	outcome, err := service.BatchStopScans(context.Background(), []int{7})
	if err != nil || outcome != store.batchOutcome || len(logStore.requests) != 1 || logs.FilterMessage("task runtime log persistence failed").Len() != 1 {
		t.Fatalf("logging fault changed committed stop: %#v %v", outcome, err)
	}
}

func TestSessionFenceLookupsAndWritesShareOneTotalEvidenceBudget(t *testing.T) {
	logs := withObservedLogger(t)
	var deadline time.Time
	lookups := 0
	store := &taskStoreStub{pendingCount: 1,
		fenceSupersededFn: func(context.Context, int, int64) ([]int, error) { return []int{7, 8}, nil },
		listFailedFn: func(ctx context.Context, scanID int) ([]ScanTaskRecord, error) {
			got, ok := ctx.Deadline()
			if !ok || (lookups > 0 && !got.Equal(deadline)) {
				t.Fatal("session fence renewed the evidence budget for each Scan")
			}
			deadline = got
			lookups++
			return []ScanTaskRecord{*assignedTaskRecordForTest(ScanTaskRecord{ID: 100 + scanID, ScanID: scanID, Status: "failed", Failure: &FailureDetail{Kind: "agent_disconnected", Message: "Agent disconnected"}}, 17, "old-session", 23)}, nil
		},
	}
	logStore := &runtimeLogStoreStub{onWrite: func(ctx context.Context, _ *TaskProgressLogBatchCreateRequest) error {
		got, ok := ctx.Deadline()
		if !ok || !got.Equal(deadline) {
			t.Fatal("session fence renewed the evidence budget between lookup and write")
		}
		return errors.New("log database unavailable")
	}}
	service := newScanTaskBridgeServiceForTest(store, &scanTaskRuntimeScanStoreStub{}).WithTaskRuntimeLogAppender(NewTaskRuntimeLogAppender(logStore))
	if err := service.FenceSupersededAgentSession(context.Background(), 17, 24); err != nil || lookups != 2 || len(logStore.requests) != 1 || logs.FilterMessage("task runtime log persistence failed").Len() != 1 {
		t.Fatalf("fence evidence budget/failure isolation changed: lookups=%d writes=%d err=%v", lookups, len(logStore.requests), err)
	}
}
