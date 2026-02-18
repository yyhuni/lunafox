package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	scanapp "github.com/yyhuni/lunafox/server/internal/modules/scan/application"
	"gorm.io/gorm"
)

type scanLogQueryCommandStoreForHandlerStub struct {
	rows        []scanapp.ScanLogEntry
	err         error
	lastAfterID int64
	lastLimit   int
}

func (stub *scanLogQueryCommandStoreForHandlerStub) FindByScanIDWithCursor(scanID int, afterID int64, limit int) ([]scanapp.ScanLogEntry, error) {
	_ = scanID
	stub.lastAfterID = afterID
	stub.lastLimit = limit
	if stub.err != nil {
		return nil, stub.err
	}
	items := make([]scanapp.ScanLogEntry, len(stub.rows))
	copy(items, stub.rows)
	return items, nil
}

func (stub *scanLogQueryCommandStoreForHandlerStub) BulkCreate(logs []scanapp.ScanLogEntry) error {
	_ = logs
	return nil
}

type scanLookupForHandlerStub struct {
	err error
}

func (stub *scanLookupForHandlerStub) GetScanLogRefByID(id int) (*scanapp.ScanLogScanRef, error) {
	if stub.err != nil {
		return nil, stub.err
	}
	return &scanapp.ScanLogScanRef{ID: id}, nil
}

func newScanLogHandlerForTest(
	queryStore scanapp.ScanLogQueryStore,
	commandStore scanapp.ScanLogCommandStore,
	lookup scanapp.ScanLogScanLookup,
) *ScanLogHandler {
	service := scanapp.NewScanLogService(queryStore, commandStore, lookup)
	return NewScanLogHandler(service)
}

func TestScanLogHandlerRejectCursorParam(t *testing.T) {
	gin.SetMode(gin.TestMode)
	queryCommandStore := &scanLogQueryCommandStoreForHandlerStub{}
	h := newScanLogHandlerForTest(queryCommandStore, queryCommandStore, &scanLookupForHandlerStub{})
	router := gin.New()
	router.GET("/api/scans/:id/logs", h.List)

	req := httptest.NewRequest(http.MethodGet, "/api/scans/1/logs?cursor=abc", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.Code)
	}
	if body := resp.Body.String(); !contains(body, "CURSOR_UNSUPPORTED_PARAM") {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestScanLogHandlerListWithAfterID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	queryCommandStore := &scanLogQueryCommandStoreForHandlerStub{rows: []scanapp.ScanLogEntry{{ID: 11}, {ID: 12}, {ID: 13}}}
	h := newScanLogHandlerForTest(queryCommandStore, queryCommandStore, &scanLookupForHandlerStub{})
	router := gin.New()
	router.GET("/api/scans/:id/logs", h.List)

	req := httptest.NewRequest(http.MethodGet, "/api/scans/1/logs?afterId=10&limit=2", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.Code)
	}
	if queryCommandStore.lastAfterID != 10 || queryCommandStore.lastLimit != 3 {
		t.Fatalf("unexpected paging args: afterId=%d limit=%d", queryCommandStore.lastAfterID, queryCommandStore.lastLimit)
	}
	body := resp.Body.String()
	if !contains(body, "\"hasMore\":true") {
		t.Fatalf("expected hasMore=true, got body: %s", body)
	}
	var payload map[string]any
	if err := json.Unmarshal(resp.Body.Bytes(), &payload); err != nil {
		t.Fatalf("failed to parse response body: %v", err)
	}
	nextField := "next" + "Cursor"
	if _, exists := payload[nextField]; exists {
		t.Fatalf("response should not include extra pagination field: %s", body)
	}
}

func TestScanLogHandlerScanNotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)
	queryCommandStore := &scanLogQueryCommandStoreForHandlerStub{}
	h := newScanLogHandlerForTest(
		queryCommandStore,
		queryCommandStore,
		&scanLookupForHandlerStub{err: gorm.ErrRecordNotFound},
	)
	router := gin.New()
	router.GET("/api/scans/:id/logs", h.List)

	req := httptest.NewRequest(http.MethodGet, "/api/scans/1/logs", nil)
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
