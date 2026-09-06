package search

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	service "github.com/yyhuni/lunafox/server/internal/modules/asset/application"
	assetdomain "github.com/yyhuni/lunafox/server/internal/modules/asset/domain"
)

type globalAssetSearchHandlerServiceStub struct {
	result *service.GlobalAssetSearchResult
	err    error
	input  service.GlobalAssetSearchInput
	calls  int
}

func (stub *globalAssetSearchHandlerServiceStub) Search(_ context.Context, input service.GlobalAssetSearchInput) (*service.GlobalAssetSearchResult, error) {
	stub.calls++
	stub.input = input
	if stub.err != nil {
		return nil, stub.err
	}
	return stub.result, nil
}

func TestGlobalAssetSearchHandlerUsesCanonicalQueryAndReturnsCompleteAssetPayload(t *testing.T) {
	gin.SetMode(gin.TestMode)
	now := time.Date(2026, 8, 7, 9, 0, 0, 0, time.UTC)
	status := 200
	stub := &globalAssetSearchHandlerServiceStub{result: &service.GlobalAssetSearchResult{
		AssetType: service.GlobalAssetSearchAssetTypeWebsite,
		Websites: []assetdomain.Website{{
			ID:              7,
			TargetID:        3,
			URL:             "https://api.example.test/admin",
			Host:            "api.example.test",
			Title:           "Admin",
			StatusCode:      &status,
			Tech:            []string{"nginx"},
			ResponseHeaders: "server: nginx",
			ResponseBody:    "<html>admin</html>",
			CreatedAt:       now,
		}},
		NextPageToken: "next-token",
	}}
	handler := NewGlobalAssetSearchHandler(stub)
	recorder := performGlobalAssetSearchRequest(t, handler, "/v1/assets:search?q=example&assetType=website")
	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", recorder.Code, recorder.Body.String())
	}
	if stub.input.Query != "example" || stub.input.AssetType != service.GlobalAssetSearchAssetTypeWebsite || stub.input.PageSize != nil {
		t.Fatalf("handler must preserve canonical query shape: %+v", stub.input)
	}
	body := recorder.Body.String()
	for _, expected := range []string{
		`"results"`, `"nextPageToken":"next-token"`, `"name":"targets/3/websites/7"`, `"tech":["nginx"]`, `"responseHeaders":"server: nginx"`, `"responseBody"`, "admin",
	} {
		if !strings.Contains(body, expected) {
			t.Fatalf("missing %q in response: %s", expected, body)
		}
	}
	for _, forbidden := range []string{`"total"`, `"totalSize"`, `"totalPages"`, `"vulnerabilities"`} {
		if strings.Contains(body, forbidden) {
			t.Fatalf("response must not include %q: %s", forbidden, body)
		}
	}
}

func TestGlobalAssetSearchHandlerRejectsInvalidArgumentsAndMapsTimeout(t *testing.T) {
	gin.SetMode(gin.TestMode)
	stub := &globalAssetSearchHandlerServiceStub{err: service.ErrInvalidGlobalAssetSearchQuery}
	handler := NewGlobalAssetSearchHandler(stub)

	invalid := performGlobalAssetSearchRequest(t, handler, "/v1/assets:search?q=bad&assetType=website")
	if invalid.Code != http.StatusBadRequest || !strings.Contains(invalid.Body.String(), `"status":"INVALID_ARGUMENT"`) {
		t.Fatalf("invalid query must map to INVALID_ARGUMENT: code=%d body=%s", invalid.Code, invalid.Body.String())
	}

	pageSize := performGlobalAssetSearchRequest(t, handler, "/v1/assets:search?q=example&assetType=website&pageSize=101")
	if pageSize.Code != http.StatusBadRequest {
		t.Fatalf("pageSize above the bounded range must be rejected, got %d", pageSize.Code)
	}
	zeroPageSize := performGlobalAssetSearchRequest(t, handler, "/v1/assets:search?q=example&assetType=website&pageSize=0")
	if zeroPageSize.Code != http.StatusBadRequest {
		t.Fatalf("explicit pageSize=0 must be rejected, got %d", zeroPageSize.Code)
	}

	stub.err = service.ErrGlobalAssetSearchTimeout
	timeout := performGlobalAssetSearchRequest(t, handler, "/v1/assets:search?q=example&assetType=website")
	if timeout.Code != http.StatusGatewayTimeout || !strings.Contains(timeout.Body.String(), `"status":"DEADLINE_EXCEEDED"`) {
		t.Fatalf("timeout must use canonical deadline response: code=%d body=%s", timeout.Code, timeout.Body.String())
	}

	stub.err = errors.New("database internals")
	internal := performGlobalAssetSearchRequest(t, handler, "/v1/assets:search?q=example&assetType=website")
	if internal.Code != http.StatusInternalServerError || strings.Contains(internal.Body.String(), "database internals") {
		t.Fatalf("unexpected internal error response: code=%d body=%s", internal.Code, internal.Body.String())
	}
}

func TestGlobalAssetSearchHandlerPreservesEndpointEvidenceAndTruncationFlags(t *testing.T) {
	gin.SetMode(gin.TestMode)
	stub := &globalAssetSearchHandlerServiceStub{result: &service.GlobalAssetSearchResult{
		AssetType: service.GlobalAssetSearchAssetTypeEndpoint,
		Endpoints: []assetdomain.Endpoint{{
			ID:                       8,
			TargetID:                 4,
			URL:                      "https://api.example.test/v1",
			Tech:                     []string{},
			ResponseHeaders:          "header-evidence",
			ResponseHeadersTruncated: true,
			ResponseBody:             "body-evidence",
			ResponseBodyTruncated:    true,
		}},
	}}
	recorder := performGlobalAssetSearchRequest(t, NewGlobalAssetSearchHandler(stub), "/v1/assets:search?q=example&assetType=endpoint")
	if recorder.Code != http.StatusOK {
		t.Fatalf("expected endpoint response, got %d: %s", recorder.Code, recorder.Body.String())
	}
	for _, expected := range []string{`"responseHeaders":"header-evidence"`, `"responseHeadersTruncated":true`, `"responseBody":"body-evidence"`, `"responseBodyTruncated":true`} {
		if !strings.Contains(recorder.Body.String(), expected) {
			t.Fatalf("missing endpoint evidence field %q: %s", expected, recorder.Body.String())
		}
	}
}

func performGlobalAssetSearchRequest(t *testing.T, handler *GlobalAssetSearchHandler, target string) *httptest.ResponseRecorder {
	t.Helper()
	engine := gin.New()
	engine.GET("/v1/assets:search", handler.Search)
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, target, nil)
	engine.ServeHTTP(recorder, request)
	return recorder
}
