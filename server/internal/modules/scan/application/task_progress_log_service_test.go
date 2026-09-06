package application

import (
	"context"
	"errors"
	"testing"
	"time"

	"gorm.io/gorm"
)

type taskProgressLogQueryCommandStoreStub struct {
	rows           []TaskProgressLogEntry
	err            error
	lastScanID     int
	lastAfterID    int64
	lastLimit      int
	batchCreateErr error
	accepted       int
	duplicates     int
	lastLogs       []TaskProgressLogEntry
	batchCreateCtx context.Context
	batchCalls     int
}

func (stub *taskProgressLogQueryCommandStoreStub) FindByScanIDWithCursor(scanID int, afterID int64, limit int) ([]TaskProgressLogEntry, error) {
	stub.lastScanID = scanID
	stub.lastAfterID = afterID
	stub.lastLimit = limit
	if stub.err != nil {
		return nil, stub.err
	}
	items := make([]TaskProgressLogEntry, len(stub.rows))
	copy(items, stub.rows)
	return items, nil
}

func (stub *taskProgressLogQueryCommandStoreStub) BatchCreateTaskProgressLogs(ctx context.Context, logs []TaskProgressLogEntry) (int, int, error) {
	stub.batchCreateCtx = ctx
	stub.batchCalls++
	stub.lastLogs = append([]TaskProgressLogEntry(nil), logs...)
	if stub.batchCreateErr != nil {
		return 0, 0, stub.batchCreateErr
	}
	return stub.accepted, stub.duplicates, nil
}

type scanLookupStub struct {
	err error
}

func (stub *scanLookupStub) GetTaskProgressLogRefByID(id int) (*TaskProgressLogScanRef, error) {
	if stub.err != nil {
		return nil, stub.err
	}
	return &TaskProgressLogScanRef{ID: id}, nil
}

func TestTaskProgressLogServiceListByAfterID(t *testing.T) {
	queryCommandStore := &taskProgressLogQueryCommandStoreStub{rows: []TaskProgressLogEntry{{ID: 10}, {ID: 11}, {ID: 12}}}
	service := NewTaskProgressLogService(queryCommandStore, queryCommandStore, &scanLookupStub{})

	items, hasMore, err := service.ListByScanID(context.Background(), 7, &TaskProgressLogListQuery{AfterID: 0, Limit: 2})
	if err != nil {
		t.Fatalf("list failed: %v", err)
	}
	if !hasMore || len(items) != 2 {
		t.Fatalf("unexpected page: hasMore=%v len=%d", hasMore, len(items))
	}
	if queryCommandStore.lastAfterID != 0 || queryCommandStore.lastLimit != 3 {
		t.Fatalf("unexpected store args afterID=%d limit=%d", queryCommandStore.lastAfterID, queryCommandStore.lastLimit)
	}

	_, _, err = service.ListByScanID(context.Background(), 7, &TaskProgressLogListQuery{AfterID: 9, Limit: 2})
	if err != nil {
		t.Fatalf("list with afterID failed: %v", err)
	}
	if queryCommandStore.lastAfterID != 9 {
		t.Fatalf("expected afterID 9, got %d", queryCommandStore.lastAfterID)
	}
}

func TestTaskProgressLogServiceNegativeAfterIDClamped(t *testing.T) {
	queryCommandStore := &taskProgressLogQueryCommandStoreStub{rows: []TaskProgressLogEntry{{ID: 1}}}
	service := NewTaskProgressLogService(queryCommandStore, queryCommandStore, &scanLookupStub{})

	_, _, err := service.ListByScanID(context.Background(), 7, &TaskProgressLogListQuery{AfterID: -10, Limit: 20})
	if err != nil {
		t.Fatalf("list failed: %v", err)
	}
	if queryCommandStore.lastAfterID != 0 {
		t.Fatalf("expected afterID clamped to 0, got %d", queryCommandStore.lastAfterID)
	}
}

func TestTaskProgressLogServiceScanNotFound(t *testing.T) {
	queryCommandStore := &taskProgressLogQueryCommandStoreStub{}
	service := NewTaskProgressLogService(queryCommandStore, queryCommandStore, &scanLookupStub{err: gorm.ErrRecordNotFound})

	_, _, err := service.ListByScanID(context.Background(), 7, &TaskProgressLogListQuery{AfterID: 0, Limit: 20})
	if !errors.Is(err, ErrScanNotFound) {
		t.Fatalf("expected ErrScanNotFound, got %v", err)
	}
}

func TestTaskProgressLogServiceBatchCreateTaskProgressLogs(t *testing.T) {
	emittedAt := time.Date(2026, 5, 7, 10, 30, 0, 0, time.UTC)
	commandStore := &taskProgressLogQueryCommandStoreStub{accepted: 1, duplicates: 1}
	service := NewTaskProgressLogService(commandStore, commandStore, &scanLookupStub{})

	accepted, duplicates, err := service.BatchCreateTaskProgressLogs(context.Background(), &TaskProgressLogBatchCreateRequest{
		ScanID:    7,
		TaskID:    101,
		RequestID: "request-1",
		Items: []TaskProgressLogCreateItem{
			{Sequence: 1, Level: "info", Content: "started", EmittedAt: emittedAt},
			{Sequence: 2, Level: "warning", Content: "retrying", EmittedAt: emittedAt.Add(time.Second)},
		},
	})
	if err != nil {
		t.Fatalf("batch create task progress logs failed: %v", err)
	}
	if accepted != 1 || duplicates != 1 {
		t.Fatalf("unexpected counts accepted=%d duplicates=%d", accepted, duplicates)
	}
	if len(commandStore.lastLogs) != 2 {
		t.Fatalf("expected 2 logs, got %d", len(commandStore.lastLogs))
	}
	if commandStore.batchCreateCtx == nil {
		t.Fatal("batch create context was not forwarded")
	}
	first := commandStore.lastLogs[0]
	if first.TaskID != 101 || first.RequestID != "request-1" {
		t.Fatalf("unexpected first log identity: %+v", first)
	}
	if first.Sequence != 1 || first.Level != "info" || first.Content != "started" || first.EmittedAt == nil || !first.EmittedAt.Equal(emittedAt) {
		t.Fatalf("unexpected first log payload: %+v", first)
	}
}

func TestTaskProgressLogServiceBatchCreatePreservesCancellation(t *testing.T) {
	commandStore := &taskProgressLogQueryCommandStoreStub{}
	service := NewTaskProgressLogService(commandStore, commandStore, &scanLookupStub{})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, _, err := service.BatchCreateTaskProgressLogs(ctx, &TaskProgressLogBatchCreateRequest{
		ScanID: 7, TaskID: 101, RequestID: "request-1",
		Items: []TaskProgressLogCreateItem{{Sequence: 1, Level: "info", Content: "started", EmittedAt: time.Now().UTC()}},
	})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("BatchCreateTaskProgressLogs() error = %v, want context canceled", err)
	}
	if commandStore.batchCalls != 0 {
		t.Fatalf("cancelled request reached command store %d times", commandStore.batchCalls)
	}
}
