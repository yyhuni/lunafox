package application

import (
	"context"
	"errors"
	"testing"

	"gorm.io/gorm"
)

type scanLogQueryCommandStoreStub struct {
	rows          []ScanLogEntry
	err           error
	lastScanID    int
	lastAfterID   int64
	lastLimit     int
	bulkCreateErr error
}

func (stub *scanLogQueryCommandStoreStub) FindByScanIDWithCursor(scanID int, afterID int64, limit int) ([]ScanLogEntry, error) {
	stub.lastScanID = scanID
	stub.lastAfterID = afterID
	stub.lastLimit = limit
	if stub.err != nil {
		return nil, stub.err
	}
	items := make([]ScanLogEntry, len(stub.rows))
	copy(items, stub.rows)
	return items, nil
}

func (stub *scanLogQueryCommandStoreStub) BulkCreate(logs []ScanLogEntry) error {
	_ = logs
	return stub.bulkCreateErr
}

type scanLookupStub struct {
	err error
}

func (stub *scanLookupStub) GetScanLogRefByID(id int) (*ScanLogScanRef, error) {
	if stub.err != nil {
		return nil, stub.err
	}
	return &ScanLogScanRef{ID: id}, nil
}

func TestScanLogServiceListByAfterID(t *testing.T) {
	queryCommandStore := &scanLogQueryCommandStoreStub{rows: []ScanLogEntry{{ID: 10}, {ID: 11}, {ID: 12}}}
	service := NewScanLogService(queryCommandStore, queryCommandStore, &scanLookupStub{})

	items, hasMore, err := service.ListByScanID(context.Background(), 7, &ScanLogListQuery{AfterID: 0, Limit: 2})
	if err != nil {
		t.Fatalf("list failed: %v", err)
	}
	if !hasMore || len(items) != 2 {
		t.Fatalf("unexpected page: hasMore=%v len=%d", hasMore, len(items))
	}
	if queryCommandStore.lastAfterID != 0 || queryCommandStore.lastLimit != 3 {
		t.Fatalf("unexpected store args afterID=%d limit=%d", queryCommandStore.lastAfterID, queryCommandStore.lastLimit)
	}

	_, _, err = service.ListByScanID(context.Background(), 7, &ScanLogListQuery{AfterID: 9, Limit: 2})
	if err != nil {
		t.Fatalf("list with afterID failed: %v", err)
	}
	if queryCommandStore.lastAfterID != 9 {
		t.Fatalf("expected afterID 9, got %d", queryCommandStore.lastAfterID)
	}
}

func TestScanLogServiceNegativeAfterIDClamped(t *testing.T) {
	queryCommandStore := &scanLogQueryCommandStoreStub{rows: []ScanLogEntry{{ID: 1}}}
	service := NewScanLogService(queryCommandStore, queryCommandStore, &scanLookupStub{})

	_, _, err := service.ListByScanID(context.Background(), 7, &ScanLogListQuery{AfterID: -10, Limit: 20})
	if err != nil {
		t.Fatalf("list failed: %v", err)
	}
	if queryCommandStore.lastAfterID != 0 {
		t.Fatalf("expected afterID clamped to 0, got %d", queryCommandStore.lastAfterID)
	}
}

func TestScanLogServiceScanNotFound(t *testing.T) {
	queryCommandStore := &scanLogQueryCommandStoreStub{}
	service := NewScanLogService(queryCommandStore, queryCommandStore, &scanLookupStub{err: gorm.ErrRecordNotFound})

	_, _, err := service.ListByScanID(context.Background(), 7, &ScanLogListQuery{AfterID: 0, Limit: 20})
	if !errors.Is(err, ErrScanNotFound) {
		t.Fatalf("expected ErrScanNotFound, got %v", err)
	}
}
