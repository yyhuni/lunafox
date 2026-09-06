package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	scanapp "github.com/yyhuni/lunafox/server/internal/modules/scan/application"
	"gorm.io/gorm"
)

type taskProgressLogQueryCommandStoreForHandlerStub struct {
	rows        []scanapp.TaskProgressLogEntry
	err         error
	lastAfterID int64
	lastLimit   int
}

func (stub *taskProgressLogQueryCommandStoreForHandlerStub) FindByScanIDWithCursor(scanID int, afterID int64, limit int) ([]scanapp.TaskProgressLogEntry, error) {
	_ = scanID
	stub.lastAfterID = afterID
	stub.lastLimit = limit
	if stub.err != nil {
		return nil, stub.err
	}
	items := make([]scanapp.TaskProgressLogEntry, len(stub.rows))
	copy(items, stub.rows)
	return items, nil
}

func (stub *taskProgressLogQueryCommandStoreForHandlerStub) BatchCreateTaskProgressLogs(_ context.Context, logs []scanapp.TaskProgressLogEntry) (int, int, error) {
	return len(logs), 0, nil
}

type scanLookupForHandlerStub struct {
	err error
}

func (stub *scanLookupForHandlerStub) GetTaskProgressLogRefByID(id int) (*scanapp.TaskProgressLogScanRef, error) {
	if stub.err != nil {
		return nil, stub.err
	}
	return &scanapp.TaskProgressLogScanRef{ID: id}, nil
}

func newTaskProgressLogHandlerForTest(
	queryStore scanapp.TaskProgressLogQueryStore,
	commandStore scanapp.TaskProgressLogCommandStore,
	lookup scanapp.TaskProgressLogScanLookup,
) *TaskProgressLogHandler {
	service := scanapp.NewTaskProgressLogService(queryStore, commandStore, lookup)
	return NewTaskProgressLogHandler(service)
}

func TestTaskProgressLogHandlerRejectLegacyPagingParams(t *testing.T) {
	gin.SetMode(gin.TestMode)
	queryCommandStore := &taskProgressLogQueryCommandStoreForHandlerStub{}
	h := newTaskProgressLogHandlerForTest(queryCommandStore, queryCommandStore, &scanLookupForHandlerStub{})
	router := gin.New()
	router.GET("/v1/scans/:scan/taskProgressLogs", h.List)

	for _, query := range []string{"cursor=abc", "afterId=10", "limit=2"} {
		req := httptest.NewRequest(http.MethodGet, "/v1/scans/1/taskProgressLogs?"+query, nil)
		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)

		if resp.Code != http.StatusBadRequest {
			t.Fatalf("expected 400 for %s, got %d", query, resp.Code)
		}
		if body := resp.Body.String(); !contains(body, "AIP_PAGING_REQUIRED") {
			t.Fatalf("unexpected response body for %s: %s", query, body)
		}
	}
}

func TestTaskProgressLogHandlerListWithAIPPageToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	queryCommandStore := &taskProgressLogQueryCommandStoreForHandlerStub{rows: []scanapp.TaskProgressLogEntry{
		{ID: 11, TaskID: 7001},
		{ID: 12, TaskID: 7001},
		{ID: 13, TaskID: 7001},
	}}
	h := newTaskProgressLogHandlerForTest(queryCommandStore, queryCommandStore, &scanLookupForHandlerStub{})
	router := gin.New()
	router.GET("/v1/scans/:scan/taskProgressLogs", h.List)

	req := httptest.NewRequest(http.MethodGet, "/v1/scans/1/taskProgressLogs?pageSize=2", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.Code)
	}
	if queryCommandStore.lastAfterID != 0 || queryCommandStore.lastLimit != 3 {
		t.Fatalf("unexpected paging args: afterId=%d limit=%d", queryCommandStore.lastAfterID, queryCommandStore.lastLimit)
	}
	body := resp.Body.String()
	var payload map[string]any
	if err := json.Unmarshal(resp.Body.Bytes(), &payload); err != nil {
		t.Fatalf("failed to parse response body: %v", err)
	}
	if _, exists := payload["hasMore"]; exists {
		t.Fatalf("response should not include hasMore: %s", body)
	}
	if _, exists := payload["nextCursor"]; exists {
		t.Fatalf("response should not include nextCursor: %s", body)
	}
	nextPageToken, ok := payload["nextPageToken"].(string)
	if !ok || nextPageToken == "" {
		t.Fatalf("expected non-empty nextPageToken, got body: %s", body)
	}
	results, ok := payload["results"].([]any)
	if !ok || len(results) != 2 {
		t.Fatalf("expected 2 results, got body: %s", body)
	}
	firstResult, ok := results[0].(map[string]any)
	if !ok {
		t.Fatalf("expected first result object, got body: %s", body)
	}
	if got := int(firstResult["taskId"].(float64)); got != 7001 {
		t.Fatalf("expected taskId %d, got body: %s", 7001, body)
	}
	if got := int(firstResult["scanId"].(float64)); got != 1 {
		t.Fatalf("expected parent scanId 1, got body: %s", body)
	}

	queryCommandStore.rows = []scanapp.TaskProgressLogEntry{{ID: 13}}
	nextReq := httptest.NewRequest(http.MethodGet, "/v1/scans/1/taskProgressLogs?pageSize=2&pageToken="+nextPageToken, nil)
	nextResp := httptest.NewRecorder()
	router.ServeHTTP(nextResp, nextReq)
	if nextResp.Code != http.StatusOK {
		t.Fatalf("expected next request 200, got %d, body=%s", nextResp.Code, nextResp.Body.String())
	}
	if queryCommandStore.lastAfterID != 12 || queryCommandStore.lastLimit != 3 {
		t.Fatalf("unexpected next paging args: afterId=%d limit=%d", queryCommandStore.lastAfterID, queryCommandStore.lastLimit)
	}
	var nextPayload map[string]any
	if err := json.Unmarshal(nextResp.Body.Bytes(), &nextPayload); err != nil {
		t.Fatalf("failed to parse next response body: %v", err)
	}
	if nextPageToken, exists := nextPayload["nextPageToken"].(string); exists && nextPageToken != "" {
		t.Fatalf("terminal page should not include nextPageToken, got body: %s", nextResp.Body.String())
	}
}

func TestTaskProgressLogHandlerScanNotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)
	queryCommandStore := &taskProgressLogQueryCommandStoreForHandlerStub{}
	h := newTaskProgressLogHandlerForTest(
		queryCommandStore,
		queryCommandStore,
		&scanLookupForHandlerStub{err: gorm.ErrRecordNotFound},
	)
	router := gin.New()
	router.GET("/v1/scans/:scan/taskProgressLogs", h.List)

	req := httptest.NewRequest(http.MethodGet, "/v1/scans/1/taskProgressLogs", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", resp.Code)
	}
}

func contains(haystack, needle string) bool {
	return len(needle) == 0 || (func() bool {
		for index := 0; index+len(needle) <= len(haystack); index++ {
			if haystack[index:index+len(needle)] == needle {
				return true
			}
		}
		return false
	})()
}
