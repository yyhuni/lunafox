package application

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	agentexecutionv1 "github.com/yyhuni/lunafox/contracts/gen/lunafox/agent/execution/v1"
	agentdomain "github.com/yyhuni/lunafox/server/internal/modules/agent/domain"
	scandomain "github.com/yyhuni/lunafox/server/internal/modules/scan/domain"
)

type runtimeLogStoreStub struct {
	requests []*TaskProgressLogBatchCreateRequest
	contexts []context.Context
	err      error
	ack      *[2]int
	onWrite  func(context.Context, *TaskProgressLogBatchCreateRequest) error
}

type runtimeLogClaimStore struct {
	plan *agentexecutionv1.ResolvedEngineExecutionPlan
}

func (store runtimeLogClaimStore) ClaimNextCompatibleSavedExecutionPlan(context.Context, int, string, int64, string, []uint32) (*agentexecutionv1.ResolvedEngineExecutionPlan, error) {
	return store.plan, nil
}

func TestTaskAssignmentRuntimeLogKeepsClaimReplayIdentity(t *testing.T) {
	logStore := &runtimeLogStoreStub{}
	plan := &agentexecutionv1.ResolvedEngineExecutionPlan{Task: "scans/7/tasks/101"}
	service := newScanTaskBridgeServiceForTest(&taskStoreStub{}, &scanTaskRuntimeScanStoreStub{}).
		WithEngineExecutionClaimStore(runtimeLogClaimStore{plan: plan}).
		WithTaskRuntimeLogAppender(NewTaskRuntimeLogAppender(logStore))
	for range 2 {
		got, err := service.ClaimNextExecutionPlan(context.Background(), 17, "session-17", 23, "550e8400-e29b-41d4-a716-446655440000", agentdomain.AgentExecutionCapabilitySnapshot{
			ContainerRuntimeReady: true, SupportedEngineAPIMajors: []uint32{2},
		})
		if err != nil || got != plan {
			t.Fatalf("claim result changed: %v %v", got, err)
		}
	}
	if len(logStore.requests) != 2 || logStore.requests[0].RequestID != logStore.requests[1].RequestID || logStore.requests[0].TaskID != 101 || !strings.Contains(logStore.requests[0].Items[0].Content, "state=running") {
		t.Fatalf("assignment replay event = %#v", logStore.requests)
	}
}

func (store *runtimeLogStoreStub) BatchCreateTaskRuntimeLogs(ctx context.Context, request *TaskProgressLogBatchCreateRequest) (int, int, error) {
	store.requests = append(store.requests, request)
	store.contexts = append(store.contexts, ctx)
	if store.onWrite != nil {
		if err := store.onWrite(ctx, request); err != nil {
			return 0, 0, err
		}
	}
	if err := ctx.Err(); err != nil {
		return 0, 0, err
	}
	if store.ack != nil {
		return store.ack[0], store.ack[1], store.err
	}
	return len(request.Items), 0, store.err
}

func TestTaskRuntimeLogAppenderPreservesRawUTF8ChunksAndReplayIdentity(t *testing.T) {
	store := &runtimeLogStoreStub{}
	appender := NewTaskRuntimeLogAppender(store)
	content := strings.Repeat(" ", taskRuntimeLogContentBytes) + "token=raw-canary /srv/private/config\n\x1b[31m" + strings.Repeat("诊断", taskRuntimeLogContentBytes)
	event := TaskRuntimeLogEvent{ScanID: 7, TaskID: 101, EventID: "terminal:23:failed", Level: "error", Content: content}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	appender.AppendTaskRuntimeLog(ctx, event)
	appender.AppendTaskRuntimeLog(ctx, event)
	if len(store.requests) != 2 {
		t.Fatalf("write count = %d", len(store.requests))
	}
	first, replay := store.requests[0], store.requests[1]
	if first.RequestID == "" || first.RequestID != replay.RequestID || len(first.RequestID) > 100 {
		t.Fatalf("replay identity = %q / %q", first.RequestID, replay.RequestID)
	}
	var reconstructed strings.Builder
	for index, item := range first.Items {
		if item.Sequence != int64(index+1) || len(item.Content) > taskRuntimeLogContentBytes || !utf8.ValidString(item.Content) {
			t.Fatalf("invalid raw chunk = %#v", item)
		}
		reconstructed.WriteString(item.Content)
	}
	if reconstructed.String() != content || first.Items[0].Content != strings.Repeat(" ", taskRuntimeLogContentBytes) {
		t.Fatal("raw content or whitespace-only chunk changed")
	}
	if first.Items[0].EmittedAt.Location() != time.UTC {
		t.Fatal("event time must be UTC")
	}
	if _, ok := store.contexts[0].Deadline(); !ok {
		t.Fatal("runtime log write must have a separate deadline")
	}
}

func TestTaskRuntimeLogAppenderReportsPersistenceFailure(t *testing.T) {
	logs := withObservedLogger(t)
	store := &runtimeLogStoreStub{err: errors.New("raw DB error /srv/data token=canary")}
	NewTaskRuntimeLogAppender(store).AppendTaskRuntimeLog(context.Background(), TaskRuntimeLogEvent{
		ScanID: 7, TaskID: 101, EventID: "result:failed", Level: "error", Content: "raw result failure",
	})
	entries := logs.FilterMessage("task runtime log persistence failed").All()
	if len(entries) != 1 || entries[0].ContextMap()["error"] != store.err.Error() {
		t.Fatalf("persistence error is not observable: %#v", entries)
	}
}

func TestTaskRuntimeLogAppenderRejectsIncompleteAcknowledgement(t *testing.T) {
	for _, ack := range [][2]int{{0, 0}, {-1, 2}, {2, -1}} {
		t.Run(fmt.Sprintf("%d-%d", ack[0], ack[1]), func(t *testing.T) {
			logs := withObservedLogger(t)
			store := &runtimeLogStoreStub{ack: &ack}
			NewTaskRuntimeLogAppender(store).AppendTaskRuntimeLog(context.Background(), TaskRuntimeLogEvent{
				ScanID: 7, TaskID: 101, EventID: "result:committed", Level: "info", Content: "committed",
			})
			if logs.FilterMessage("task runtime log persistence failed").Len() != 1 {
				t.Fatal("invalid write acknowledgement was accepted")
			}
		})
	}
}

func TestTaskRuntimeLogAppenderPreservesBinaryErrorBytes(t *testing.T) {
	for _, raw := range []string{string([]byte{'a', 0xff, 'b'}), "original\x00NUL"} {
		store := &runtimeLogStoreStub{}
		NewTaskRuntimeLogAppender(store).AppendTaskRuntimeLog(context.Background(), TaskRuntimeLogEvent{
			ScanID: 7, TaskID: 101, EventID: "result:binary-error", Level: "error", Content: raw,
		})
		if len(store.requests) != 1 {
			t.Fatal("binary error evidence was dropped")
		}
		encoded, ok := strings.CutPrefix(store.requests[0].Items[0].Content, "[binary/base64] ")
		decoded, err := base64.StdEncoding.DecodeString(encoded)
		if !ok || err != nil || string(decoded) != raw {
			t.Fatalf("binary error cannot be recovered: %q %v", encoded, err)
		}
	}
}

func TestTaskTerminalLogFailureDoesNotChangeAcknowledgement(t *testing.T) {
	logs := withObservedLogger(t)
	task := assignedTaskRecordForTest(ScanTaskRecord{ID: 101, ScanID: 7, Status: "running"}, 17, "session-17", 23)
	store := &taskStoreStub{getByIDFn: func(context.Context, int) (*ScanTaskRecord, error) { return task, nil }}
	logStore := &runtimeLogStoreStub{err: errors.New("DB unavailable")}
	service := newScanTaskBridgeServiceForTest(store, &scanTaskRuntimeScanStoreStub{}).
		WithTaskRuntimeLogAppender(NewTaskRuntimeLogAppender(logStore))
	if err := service.ReportTerminalTaskResult(context.Background(), 17, "session-17", 23, 101, "succeeded", nil); err != nil {
		t.Fatalf("log failure changed terminal acknowledgement: %v", err)
	}
	if store.updatedStatus != "succeeded" || len(store.clearedTerminalIDs) != 1 || len(logStore.requests) != 1 || logs.FilterMessage("task runtime log persistence failed").Len() != 1 {
		t.Fatal("terminal commit or best-effort log boundary did not run")
	}
}

func TestTaskTerminalLogsRejectUnownedTaskAndPreserveFailureText(t *testing.T) {
	task := assignedTaskRecordForTest(ScanTaskRecord{ID: 101, ScanID: 7, Status: "running"}, 17, "session-17", 23)
	store := &taskStoreStub{getByIDFn: func(context.Context, int) (*ScanTaskRecord, error) { return task, nil }}
	logStore := &runtimeLogStoreStub{}
	service := newScanTaskBridgeServiceForTest(store, &scanTaskRuntimeScanStoreStub{}).
		WithTaskRuntimeLogAppender(NewTaskRuntimeLogAppender(logStore))
	failure := &FailureDetail{Kind: "engine_exit_failed", Message: "raw error /workspace/targets.txt token=canary"}
	if err := service.ReportTerminalTaskResult(context.Background(), 18, "session-17", 23, 101, "failed", failure); !errors.Is(err, ErrScanTaskNotOwned) {
		t.Fatalf("unexpected ownership result: %v", err)
	}
	if len(logStore.requests) != 0 {
		t.Fatal("unowned terminal request injected a runtime log")
	}
	// This focused assertion exercises the evidence projection independently of
	// scan-level canonical-failure reconciliation.
	service.appendTerminalRuntimeLog(context.Background(), task, scandomain.TaskStatusFailed, failure)
	if len(logStore.requests) != 1 || !strings.Contains(logStore.requests[0].Items[0].Content, failure.Message) {
		t.Fatal("terminal raw failure text was not preserved")
	}
}

func TestTaskRuntimeLogBatchSharesOneBudgetAndStopsAfterFirstWriteFailure(t *testing.T) {
	for _, blocked := range []bool{false, true} {
		t.Run(fmt.Sprint(blocked), func(t *testing.T) {
			logs := withObservedLogger(t)
			store := &runtimeLogStoreStub{onWrite: func(ctx context.Context, _ *TaskProgressLogBatchCreateRequest) error {
				if blocked {
					<-ctx.Done()
					return ctx.Err()
				}
				return errors.New("database unavailable")
			}}
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
			defer cancel()
			events := make([]TaskRuntimeLogEvent, 100)
			for index := range events {
				events[index] = TaskRuntimeLogEvent{ScanID: 7, TaskID: 101 + index, EventID: "stop", Level: "info", Content: "cancelled"}
			}
			started := time.Now()
			NewTaskRuntimeLogAppender(store).AppendTaskRuntimeLogs(ctx, events)
			if time.Since(started) > 500*time.Millisecond || len(store.requests) != 1 || logs.FilterMessage("task runtime log persistence failed").Len() != 1 {
				t.Fatal("batch renewed its budget or retried later tasks after database failure")
			}
		})
	}
}
