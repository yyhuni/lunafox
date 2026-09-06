package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/yyhuni/lunafox/server/internal/loki"
	systemapp "github.com/yyhuni/lunafox/server/internal/modules/system/application"
)

type serverLogQueryServiceStub struct {
	input  systemapp.ServerLogQueryInput
	result systemapp.ServerLogQueryResult
	err    error
}

func (stub *serverLogQueryServiceStub) Query(_ context.Context, input systemapp.ServerLogQueryInput) (systemapp.ServerLogQueryResult, error) {
	stub.input = input
	return stub.result, stub.err
}

func TestServerLogHandlerListReturnsViewerMetadata(t *testing.T) {
	gin.SetMode(gin.TestMode)

	service := &serverLogQueryServiceStub{
		result: systemapp.ServerLogQueryResult{
			Logs: []systemapp.ServerLogLineItem{
				{
					ID:        "srv:lunafox-server:1740381601000000000:stdout:abc:000000",
					TS:        "2026-02-24T10:00:01Z",
					TSNs:      "1740381601000000000",
					Stream:    "stdout",
					Line:      `{"level":"info","msg":"server started"}`,
					Truncated: false,
				},
			},
			NextCursor:     "follow-token",
			PreviousCursor: "older-token",
			HasOlder:       true,
			HasNewer:       false,
			CaughtUp:       true,
			Gap:            false,
		},
	}
	handler := NewServerLogHandler(service)

	router := gin.New()
	router.GET("/v1/admin/system/logEntries", handler.List)

	request := httptest.NewRequest(http.MethodGet, "/v1/admin/system/logEntries?pageSize=50&pageToken=cursor-1&direction=newer", nil)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d, body=%s", recorder.Code, recorder.Body.String())
	}
	if service.input.Limit != 50 || service.input.Cursor != "cursor-1" || service.input.Direction != "newer" {
		t.Fatalf("unexpected service input: %+v", service.input)
	}

	var payload struct {
		Results []struct {
			ID   string `json:"id"`
			Line string `json:"line"`
		} `json:"results"`
		NextPageToken     string `json:"nextPageToken"`
		PreviousPageToken string `json:"previousPageToken"`
		HasOlder          bool   `json:"hasOlder"`
		CaughtUp          bool   `json:"caughtUp"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(payload.Results) != 1 || payload.Results[0].ID == "" || payload.Results[0].Line == "" {
		t.Fatalf("expected one log result, got %+v", payload.Results)
	}
	if payload.NextPageToken != "follow-token" || payload.PreviousPageToken != "older-token" {
		t.Fatalf("unexpected cursors: %+v", payload)
	}
	if !payload.HasOlder || !payload.CaughtUp {
		t.Fatalf("unexpected metadata: %+v", payload)
	}
}

func TestServerLogHandlerListRejectsArbitrarySources(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler := NewServerLogHandler(&serverLogQueryServiceStub{})
	router := gin.New()
	router.GET("/v1/admin/system/logEntries", handler.List)

	request := httptest.NewRequest(http.MethodGet, "/v1/admin/system/logEntries?container=lunafox-agent", nil)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d, body=%s", recorder.Code, recorder.Body.String())
	}
}

func TestServerLogHandlerMapsLokiErrors(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want int
	}{
		{name: "invalid cursor", err: systemapp.ErrServerLogCursorInvalid, want: http.StatusBadRequest},
		{name: "query mismatch", err: systemapp.ErrServerLogCursorQueryMismatch, want: http.StatusBadRequest},
		{name: "timeout", err: systemapp.ErrServerLogQueryTimeout, want: http.StatusGatewayTimeout},
		{name: "loki unavailable", err: loki.ErrLokiUnavailable, want: http.StatusServiceUnavailable},
		{name: "unknown", err: errors.New("boom"), want: http.StatusInternalServerError},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			gin.SetMode(gin.TestMode)
			handler := NewServerLogHandler(&serverLogQueryServiceStub{err: tc.err})
			router := gin.New()
			router.GET("/v1/admin/system/logEntries", handler.List)

			request := httptest.NewRequest(http.MethodGet, "/v1/admin/system/logEntries", nil)
			recorder := httptest.NewRecorder()
			router.ServeHTTP(recorder, request)

			if recorder.Code != tc.want {
				t.Fatalf("expected %d, got %d, body=%s", tc.want, recorder.Code, recorder.Body.String())
			}
		})
	}
}

func TestServerLogHandlerListUsesDefaultAndMaximumPageSize(t *testing.T) {
	gin.SetMode(gin.TestMode)
	service := &serverLogQueryServiceStub{}
	handler := NewServerLogHandler(service)
	router := gin.New()
	router.GET("/v1/admin/system/logEntries", handler.List)

	for _, testCase := range []struct {
		query string
		want  int
	}{
		{query: "", want: defaultServerLogPageSize},
		{query: "?pageSize=500", want: maxServerLogPageSize},
	} {
		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodGet, "/v1/admin/system/logEntries"+testCase.query, nil)
		router.ServeHTTP(recorder, request)
		if recorder.Code != http.StatusOK || service.input.Limit != testCase.want {
			t.Fatalf("query %q: got status=%d limit=%d", testCase.query, recorder.Code, service.input.Limit)
		}
	}
}

func TestServerLogHandlerListRejectsInvalidPageSize(t *testing.T) {
	gin.SetMode(gin.TestMode)
	service := &serverLogQueryServiceStub{}
	handler := NewServerLogHandler(service)
	router := gin.New()
	router.GET("/v1/admin/system/logEntries", handler.List)

	for _, pageSize := range []string{"501", "-1", "not-a-number"} {
		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodGet, "/v1/admin/system/logEntries?pageSize="+pageSize, nil)
		router.ServeHTTP(recorder, request)
		if recorder.Code != http.StatusBadRequest {
			t.Fatalf("pageSize=%q: expected 400, got %d", pageSize, recorder.Code)
		}
	}
}
