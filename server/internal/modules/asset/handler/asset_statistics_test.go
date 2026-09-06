package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	assetdomain "github.com/yyhuni/lunafox/server/internal/modules/asset/domain"
)

type assetStatisticsServiceStub struct {
	current      assetdomain.AssetStatistics
	history      []assetdomain.AssetStatisticsHistoryItem
	historyCalls int
	historyDays  int
}

func (stub *assetStatisticsServiceStub) GetCurrent(context.Context) (assetdomain.AssetStatistics, error) {
	return stub.current, nil
}

func (stub *assetStatisticsServiceStub) ListHistory(_ context.Context, days int) ([]assetdomain.AssetStatisticsHistoryItem, error) {
	stub.historyCalls++
	stub.historyDays = days
	return stub.history, nil
}

func TestAssetStatisticsHandlerReturnsCurrentProjection(t *testing.T) {
	gin.SetMode(gin.TestMode)
	updatedAt := time.Date(2026, 8, 7, 12, 0, 0, 0, time.UTC)
	service := &assetStatisticsServiceStub{current: assetdomain.AssetStatistics{
		TotalAssets: 6,
		UpdatedAt:   updatedAt,
		VulnsBySeverity: assetdomain.VulnerabilitySeverityCounts{
			Critical: 1,
		},
	}}
	router := gin.New()
	router.GET("/assetStatistics", NewAssetStatisticsHandler(service).GetCurrent)

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/assetStatistics", nil))
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
	if got := recorder.Body.String(); !containsAll(got, `"totalAssets":6`, `"updatedAt":"2026-08-07T12:00:00Z"`, `"critical":1`) {
		t.Fatalf("unexpected body: %s", got)
	}
}

func TestAssetStatisticsHandlerRequiresOneBoundedHistoryWindow(t *testing.T) {
	gin.SetMode(gin.TestMode)
	service := &assetStatisticsServiceStub{}
	router := gin.New()
	router.GET("/assetStatistics/history", NewAssetStatisticsHandler(service).ListHistory)

	for _, target := range []string{
		"/assetStatistics/history",
		"/assetStatistics/history?days=7&days=8",
		"/assetStatistics/history?days=0",
		"/assetStatistics/history?days=31",
		"/assetStatistics/history?days=seven",
	} {
		recorder := httptest.NewRecorder()
		router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, target, nil))
		if recorder.Code != http.StatusBadRequest {
			t.Fatalf("%s status = %d, want 400; body = %s", target, recorder.Code, recorder.Body.String())
		}
	}
	if service.historyCalls != 0 {
		t.Fatalf("invalid requests called history service %d times", service.historyCalls)
	}
}

func TestAssetStatisticsHandlerMapsHistoryDate(t *testing.T) {
	gin.SetMode(gin.TestMode)
	service := &assetStatisticsServiceStub{history: []assetdomain.AssetStatisticsHistoryItem{{
		Date:        time.Date(2026, 8, 7, 0, 0, 0, 0, time.UTC),
		TotalAssets: 6,
	}}}
	router := gin.New()
	router.GET("/assetStatistics/history", NewAssetStatisticsHandler(service).ListHistory)

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/assetStatistics/history?days=7", nil))
	if recorder.Code != http.StatusOK || service.historyDays != 7 {
		t.Fatalf("status = %d, days = %d, body = %s", recorder.Code, service.historyDays, recorder.Body.String())
	}
	if got := recorder.Body.String(); !containsAll(got, `"date":"2026-08-07"`, `"totalAssets":6`) {
		t.Fatalf("unexpected body: %s", got)
	}
}

func containsAll(value string, parts ...string) bool {
	for _, part := range parts {
		if !strings.Contains(value, part) {
			return false
		}
	}
	return true
}
