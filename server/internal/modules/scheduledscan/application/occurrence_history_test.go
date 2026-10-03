package application

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestDeriveOccurrenceStatusPrecedence(t *testing.T) {
	now := time.Date(2026, 10, 1, 2, 0, 0, 0, time.UTC)
	failureKind := "scan_create_failed"
	cases := []struct {
		name     string
		snapshot OccurrenceSnapshot
		want     OccurrenceStatus
	}{
		{"pending", OccurrenceSnapshot{ID: 1}, OccurrenceStatusPending},
		{"dispatching", OccurrenceSnapshot{ID: 2, AttemptedAt: &now}, OccurrenceStatusDispatching},
		{"retrying", OccurrenceSnapshot{ID: 3, AttemptedAt: &now, NextRetryAt: &now}, OccurrenceStatusRetrying},
		{"failed", OccurrenceSnapshot{ID: 4, AttemptedAt: &now, FailureKind: &failureKind, NextRetryAt: &now}, OccurrenceStatusFailed},
		{"succeeded wins over failure columns", OccurrenceSnapshot{ID: 5, AttemptedAt: &now, DispatchedAt: &now, FailureKind: &failureKind}, OccurrenceStatusSucceeded},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			if got := DeriveOccurrenceStatus(testCase.snapshot); got != testCase.want {
				t.Fatalf("DeriveOccurrenceStatus() = %s, want %s", got, testCase.want)
			}
		})
	}
}

func TestProjectOccurrenceRecordDurationAndCause(t *testing.T) {
	attempted := time.Date(2026, 10, 1, 2, 0, 1, 0, time.UTC)
	dispatched := attempted.Add(1500 * time.Millisecond)
	failureKind := "scan_create_failed"
	cause := HandoffCauseTargetUnavailable
	message := "target not found"

	succeeded := projectOccurrenceRecord(OccurrenceSnapshot{ID: 1, AttemptedAt: &attempted, DispatchedAt: &dispatched})
	if succeeded.Status != OccurrenceStatusSucceeded || succeeded.DurationMs == nil || *succeeded.DurationMs != 1500 {
		t.Fatalf("succeeded projection wrong: %+v", succeeded)
	}
	failed := projectOccurrenceRecord(OccurrenceSnapshot{ID: 2, AttemptedAt: &attempted, FailureKind: &failureKind, LastFailureCause: (*string)(&cause), FailureMessage: &message})
	if failed.Status != OccurrenceStatusFailed || failed.DurationMs != nil || failed.FailureCause == nil || *failed.FailureCause != HandoffCauseTargetUnavailable {
		t.Fatalf("failed projection wrong: %+v", failed)
	}
	if failed.FailureMessage == nil || *failed.FailureMessage != message {
		t.Fatalf("failure message lost: %+v", failed)
	}
}

type occurrenceStoreStub struct {
	snapshots []OccurrenceSnapshot
	total     int64
	counts    OccurrenceStatusCounts
}

func (stub occurrenceStoreStub) ListOccurrences(context.Context, OccurrenceHistoryQuery) ([]OccurrenceSnapshot, int64, error) {
	return stub.snapshots, stub.total, nil
}

func (stub occurrenceStoreStub) CountOccurrencesByStatus(context.Context, int) (OccurrenceStatusCounts, error) {
	return stub.counts, nil
}

type occurrenceServiceStoreStub struct {
	missing bool
}

func (stub occurrenceServiceStoreStub) GetByID(_ context.Context, id int) (*ScheduledScan, error) {
	if stub.missing || id <= 0 {
		return nil, ErrScheduledScanNotFound
	}
	return &ScheduledScan{ID: id}, nil
}

func (occurrenceServiceStoreStub) Create(context.Context, *ScheduledScanCreate) (*ScheduledScan, error) {
	return nil, errors.New("not implemented")
}
func (occurrenceServiceStoreStub) Update(context.Context, int, *ScheduledScanUpdate) (*ScheduledScan, error) {
	return nil, errors.New("not implemented")
}
func (occurrenceServiceStoreStub) BatchUpdateStatus(context.Context, []ScheduledScanStatusUpdate) (int, error) {
	return 0, errors.New("not implemented")
}
func (occurrenceServiceStoreStub) List(context.Context, ScheduledScanListQuery) ([]ScheduledScan, int64, error) {
	return nil, 0, errors.New("not implemented")
}
func (occurrenceServiceStoreStub) GetOverviewSummary(context.Context, ScheduledScanOverviewQuery) (*ScheduledScanOverviewProjection, error) {
	return nil, errors.New("not implemented")
}
func (occurrenceServiceStoreStub) Delete(context.Context, int) error {
	return errors.New("not implemented")
}

func TestListOccurrenceHistoryRequiresOccurrenceStore(t *testing.T) {
	service := NewScheduledScanService(occurrenceServiceStoreStub{}, nil)
	if _, err := service.ListOccurrenceHistory(context.Background(), OccurrenceHistoryQuery{ScheduledScanID: 7}); !errors.Is(err, ErrScheduledScanInvalidArgument) {
		t.Fatalf("expected invalid argument, got %v", err)
	}
}

func TestListOccurrenceHistoryUnknownScheduleReusesGetNotFound(t *testing.T) {
	service := NewScheduledScanService(occurrenceServiceStoreStub{missing: true}, nil).WithOccurrenceStore(occurrenceStoreStub{})
	if _, err := service.ListOccurrenceHistory(context.Background(), OccurrenceHistoryQuery{ScheduledScanID: 7}); !errors.Is(err, ErrScheduledScanNotFound) {
		t.Fatalf("expected not found, got %v", err)
	}
}

func TestListOccurrenceHistoryProjectsSnapshots(t *testing.T) {
	now := time.Date(2026, 10, 1, 2, 0, 0, 0, time.UTC)
	service := NewScheduledScanService(occurrenceServiceStoreStub{}, nil).WithOccurrenceStore(occurrenceStoreStub{
		snapshots: []OccurrenceSnapshot{{ID: 9, ScheduledFor: now, AttemptedAt: &now, DispatchedAt: &now}},
		total:     1,
		counts:    OccurrenceStatusCounts{Succeeded: 1},
	})
	history, err := service.ListOccurrenceHistory(context.Background(), OccurrenceHistoryQuery{ScheduledScanID: 7})
	if err != nil {
		t.Fatalf("ListOccurrenceHistory() error = %v", err)
	}
	if history.Total != 1 || len(history.Records) != 1 || history.Records[0].Status != OccurrenceStatusSucceeded {
		t.Fatalf("history projected wrong: %+v", history)
	}
	if history.Counts.Succeeded != 1 {
		t.Fatalf("counts projected wrong: %+v", history.Counts)
	}
}
