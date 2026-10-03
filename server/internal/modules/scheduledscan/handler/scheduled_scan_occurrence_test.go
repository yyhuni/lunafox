package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	scheduledapp "github.com/yyhuni/lunafox/server/internal/modules/scheduledscan/application"
)

type scheduledScanOccurrenceStoreStub struct {
	snapshots []scheduledapp.OccurrenceSnapshot
	total     int64
	counts    scheduledapp.OccurrenceStatusCounts
	query     *scheduledapp.OccurrenceHistoryQuery
}

func (stub *scheduledScanOccurrenceStoreStub) ListOccurrences(_ context.Context, query scheduledapp.OccurrenceHistoryQuery) ([]scheduledapp.OccurrenceSnapshot, int64, error) {
	stub.query = &query
	return stub.snapshots, stub.total, nil
}

func (stub *scheduledScanOccurrenceStoreStub) CountOccurrencesByStatus(context.Context, int) (scheduledapp.OccurrenceStatusCounts, error) {
	return stub.counts, nil
}

func TestScheduledScanOccurrenceListProjectsStatusesAndCounts(t *testing.T) {
	gin.SetMode(gin.TestMode)
	failedKind := "scan_create_failed"
	failedCause := "ENGINE_UNAVAILABLE"
	failedMessage := "Scan creation did not complete."
	attemptedAt := time.Date(2026, 10, 1, 2, 0, 1, 0, time.UTC)
	dispatchedAt := time.Date(2026, 10, 1, 2, 0, 2, 0, time.UTC)
	nextRetryAt := time.Date(2026, 10, 1, 2, 1, 0, 0, time.UTC)
	occurrenceStub := &scheduledScanOccurrenceStoreStub{
		total:  4,
		counts: scheduledapp.OccurrenceStatusCounts{Succeeded: 1, Failed: 1, Retrying: 1, Dispatching: 1},
		snapshots: []scheduledapp.OccurrenceSnapshot{
			{ID: 41, ScheduledFor: time.Date(2026, 10, 1, 14, 0, 0, 0, time.UTC),
				AttemptedAt: &attemptedAt, FailureKind: &failedKind, LastFailureCause: &failedCause, FailureMessage: &failedMessage, RetryCount: 3},
			{ID: 40, ScheduledFor: time.Date(2026, 10, 1, 2, 0, 0, 0, time.UTC),
				AttemptedAt: &attemptedAt, DispatchedAt: &dispatchedAt},
			{ID: 39, ScheduledFor: time.Date(2026, 9, 30, 14, 0, 0, 0, time.UTC),
				AttemptedAt: &attemptedAt, RetryCount: 2, NextRetryAt: &nextRetryAt},
			{ID: 38, ScheduledFor: time.Date(2026, 9, 30, 2, 0, 0, 0, time.UTC),
				AttemptedAt: &attemptedAt},
		},
	}
	store := &scheduledScanHandlerStore{}
	handler := NewScheduledScanHandler(newScheduledScanServiceForHandlerTest(store).WithOccurrenceStore(occurrenceStub))
	recorder := performScheduledScanOccurrenceRequest(handler.ListOccurrences, "/v1/scheduledScans/7/occurrences", "7")

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", recorder.Code, recorder.Body.String())
	}
	var body struct {
		Occurrences []struct {
			Name           string  `json:"name"`
			ID             int64   `json:"id"`
			Status         string  `json:"status"`
			FailureCause   *string `json:"failureCause"`
			RetryCount     int     `json:"retryCount"`
			NextRetryAt    *string `json:"nextRetryAt"`
			DurationMs     *int64  `json:"durationMs"`
		} `json:"occurrences"`
		StatusCounts struct {
			Pending     int64 `json:"pending"`
			Dispatching int64 `json:"dispatching"`
			Retrying    int64 `json:"retrying"`
			Succeeded   int64 `json:"succeeded"`
			Failed      int64 `json:"failed"`
		} `json:"statusCounts"`
		TotalSize int64 `json:"totalSize"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(body.Occurrences) != 4 {
		t.Fatalf("expected 4 occurrences, got %d body=%s", len(body.Occurrences), recorder.Body.String())
	}
	first := body.Occurrences[0]
	if first.ID != 41 || first.Name != "scheduledScans/7/occurrences/41" || first.Status != "FAILED" || first.FailureCause == nil || *first.FailureCause != "ENGINE_UNAVAILABLE" || first.RetryCount != 3 {
		t.Fatalf("newest failed occurrence projected wrong: %+v", first)
	}
	if first.DurationMs != nil {
		t.Fatalf("failed occurrence must not carry durationMs: %+v", first)
	}
	second := body.Occurrences[1]
	if second.ID != 40 || second.Status != "SUCCEEDED" || second.DurationMs == nil || *second.DurationMs != 1000 {
		t.Fatalf("succeeded occurrence projected wrong: %+v", second)
	}
	if body.Occurrences[2].Status != "RETRYING" || body.Occurrences[2].NextRetryAt == nil {
		t.Fatalf("retrying occurrence projected wrong: %+v", body.Occurrences[2])
	}
	if body.Occurrences[3].Status != "DISPATCHING" {
		t.Fatalf("dispatching occurrence projected wrong: %+v", body.Occurrences[3])
	}
	if body.StatusCounts.Succeeded != 1 || body.StatusCounts.Failed != 1 || body.StatusCounts.Retrying != 1 || body.StatusCounts.Dispatching != 1 || body.StatusCounts.Pending != 0 {
		t.Fatalf("status counts projected wrong: %+v", body.StatusCounts)
	}
	if body.TotalSize != 4 {
		t.Fatalf("expected totalSize 4, got %d", body.TotalSize)
	}
	if occurrenceStub.query == nil || occurrenceStub.query.ScheduledScanID != 7 {
		t.Fatalf("occurrence query not routed: %+v", occurrenceStub.query)
	}
}

func TestScheduledScanOccurrenceListReturnsNotFoundForUnknownSchedule(t *testing.T) {
	gin.SetMode(gin.TestMode)
	service := scheduledapp.NewScheduledScanService(scheduledScanHandlerMissingStore{&scheduledScanHandlerStore{}}, scheduledScanWorkflowStoreStub{}).
		WithConfigResourceValidator(scheduledScanConfigResourceValidatorStub{}).
		WithOccurrenceStore(&scheduledScanOccurrenceStoreStub{})
	handler := NewScheduledScanHandler(service)
	recorder := performScheduledScanOccurrenceRequest(handler.ListOccurrences, "/v1/scheduledScans/999/occurrences", "999")

	if recorder.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d body=%s", recorder.Code, recorder.Body.String())
	}
}

func TestScheduledScanOccurrenceListRejectsInvalidScheduleID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	store := &scheduledScanHandlerStore{}
	handler := NewScheduledScanHandler(newScheduledScanServiceForHandlerTest(store).WithOccurrenceStore(&scheduledScanOccurrenceStoreStub{}))
	recorder := performScheduledScanOccurrenceRequest(handler.ListOccurrences, "/v1/scheduledScans/not-an-id/occurrences", "not-an-id")

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d body=%s", recorder.Code, recorder.Body.String())
	}
}

func TestScheduledScanOccurrenceListRequiresOccurrenceStore(t *testing.T) {
	gin.SetMode(gin.TestMode)
	store := &scheduledScanHandlerStore{}
	handler := NewScheduledScanHandler(newScheduledScanServiceForHandlerTest(store))
	recorder := performScheduledScanOccurrenceRequest(handler.ListOccurrences, "/v1/scheduledScans/1/occurrences", "1")

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for unconfigured occurrence store, got %d body=%s", recorder.Code, recorder.Body.String())
	}
}

// scheduledScanHandlerMissingStore makes GetByID resolve not-found so the
// occurrence history endpoint reuses the Scheduled Scan Get semantics.
type scheduledScanHandlerMissingStore struct {
	*scheduledScanHandlerStore
}

func (scheduledScanHandlerMissingStore) GetByID(context.Context, int) (*scheduledapp.ScheduledScan, error) {
	return nil, scheduledapp.ErrScheduledScanNotFound
}

func performScheduledScanOccurrenceRequest(handler gin.HandlerFunc, target, scheduledScanParam string) *httptest.ResponseRecorder {
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	req := httptest.NewRequest(http.MethodGet, target, nil)
	req.Header.Set("Content-Type", "application/json")
	c.Request = req
	c.Params = gin.Params{{Key: "scheduledScan", Value: scheduledScanParam}}
	handler(c)
	return recorder
}
