package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	blacklistapp "github.com/yyhuni/lunafox/server/internal/modules/blacklist/application"
)

type blacklistPolicyHandlerServiceStub struct {
	global           *blacklistapp.BlacklistPolicy
	target           *blacklistapp.BlacklistPolicy
	globalErr        error
	targetErr        error
	patchGlobalErr   error
	patchTargetErr   error
	patchGlobalCalls int
	patchTargetCalls int
	lastGlobalInput  blacklistapp.ReplaceBlacklistPolicyInput
	lastTargetID     int
	lastTargetInput  blacklistapp.ReplaceBlacklistPolicyInput
}

func (stub *blacklistPolicyHandlerServiceStub) GetGlobal(context.Context) (*blacklistapp.BlacklistPolicy, error) {
	return stub.global, stub.globalErr
}

func (stub *blacklistPolicyHandlerServiceStub) GetTarget(context.Context, int) (*blacklistapp.BlacklistPolicy, error) {
	return stub.target, stub.targetErr
}

func (stub *blacklistPolicyHandlerServiceStub) ReplaceGlobal(_ context.Context, input blacklistapp.ReplaceBlacklistPolicyInput) (*blacklistapp.BlacklistPolicy, error) {
	stub.patchGlobalCalls++
	stub.lastGlobalInput = input
	return stub.global, stub.patchGlobalErr
}

func (stub *blacklistPolicyHandlerServiceStub) ReplaceTarget(_ context.Context, targetID int, input blacklistapp.ReplaceBlacklistPolicyInput) (*blacklistapp.BlacklistPolicy, error) {
	stub.patchTargetCalls++
	stub.lastTargetID = targetID
	stub.lastTargetInput = input
	return stub.target, stub.patchTargetErr
}

func newBlacklistPolicyHandlerForTest(t *testing.T) (*BlacklistPolicyHandler, *blacklistPolicyHandlerServiceStub) {
	t.Helper()
	timestamp := time.Date(2026, 8, 5, 12, 0, 0, 0, time.UTC)
	stub := &blacklistPolicyHandlerServiceStub{
		global: &blacklistapp.BlacklistPolicy{Name: "blacklistPolicy", Patterns: []string{}, ETag: "global-etag", UpdateTime: timestamp},
		target: &blacklistapp.BlacklistPolicy{Name: "targets/42/blacklistPolicy", Patterns: []string{"*.example.com"}, ETag: "target-etag", UpdateTime: timestamp},
	}
	handler, err := NewBlacklistPolicyHandler(stub)
	if err != nil {
		t.Fatal(err)
	}
	return handler, stub
}

func newBlacklistPolicyRouter(handler *BlacklistPolicyHandler) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/v1/blacklistPolicy", handler.GetGlobal)
	router.PATCH("/v1/blacklistPolicy", handler.PatchGlobal)
	router.GET("/v1/targets/:target/blacklistPolicy", handler.GetTarget)
	router.PATCH("/v1/targets/:target/blacklistPolicy", handler.PatchTarget)
	return router
}

func performBlacklistPolicyRequest(router *gin.Engine, method, path, body string) *httptest.ResponseRecorder {
	request := httptest.NewRequest(method, path, strings.NewReader(body))
	if body != "" {
		request.Header.Set("Content-Type", "application/json")
	}
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
	return recorder
}

func TestBlacklistPolicyHandlerReturnsExactEmptyGlobalResource(t *testing.T) {
	handler, _ := newBlacklistPolicyHandlerForTest(t)
	response := performBlacklistPolicyRequest(newBlacklistPolicyRouter(handler), http.MethodGet, "/v1/blacklistPolicy", "")
	if response.Code != http.StatusOK {
		t.Fatalf("GET status = %d body=%s", response.Code, response.Body.String())
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(response.Body.Bytes(), &fields); err != nil {
		t.Fatal(err)
	}
	if len(fields) != 4 {
		t.Fatalf("response must contain exactly four fields: %s", response.Body.String())
	}
	var patterns []string
	if err := json.Unmarshal(fields["patterns"], &patterns); err != nil || patterns == nil || len(patterns) != 0 {
		t.Fatalf("empty patterns must be []: %s", response.Body.String())
	}
}

func TestBlacklistPolicyHandlerPatchesOnlyExactReplacement(t *testing.T) {
	handler, stub := newBlacklistPolicyHandlerForTest(t)
	router := newBlacklistPolicyRouter(handler)
	body := `{"name":"blacklistPolicy","patterns":[],"etag":"global-etag","updateTime":"1900-01-01T00:00:00Z"}`
	response := performBlacklistPolicyRequest(router, http.MethodPatch, "/v1/blacklistPolicy?updateMask=patterns", body)
	if response.Code != http.StatusOK {
		t.Fatalf("PATCH status = %d body=%s", response.Code, response.Body.String())
	}
	if stub.patchGlobalCalls != 1 || stub.lastGlobalInput.ETag != "global-etag" || stub.lastGlobalInput.Patterns == nil || len(stub.lastGlobalInput.Patterns) != 0 {
		t.Fatalf("unexpected global update input: calls=%d input=%#v", stub.patchGlobalCalls, stub.lastGlobalInput)
	}

	for _, test := range []struct {
		path string
		body string
	}{
		{path: "/v1/blacklistPolicy", body: body},
		{path: "/v1/blacklistPolicy?updateMask=patterns,etag", body: body},
		{path: "/v1/blacklistPolicy?updateMask=patterns&legacy=true", body: body},
		{path: "/v1/blacklistPolicy?updateMask=patterns", body: `{"name":"blacklistPolicy","etag":"global-etag"}`},
		{path: "/v1/blacklistPolicy?updateMask=patterns", body: `{"name":"blacklistPolicy","patterns":null,"etag":"global-etag"}`},
		{path: "/v1/blacklistPolicy?updateMask=patterns", body: `{"name":"targets/42/blacklistPolicy","patterns":[],"etag":"global-etag"}`},
		{path: "/v1/blacklistPolicy?updateMask=patterns", body: `{"name":"blacklistPolicy","patterns":[]}`},
		{path: "/v1/blacklistPolicy?updateMask=patterns", body: `{"name":"blacklistPolicy","patterns":[],"etag":"global-etag","legacyPatterns":[]}`},
	} {
		response = performBlacklistPolicyRequest(router, http.MethodPatch, test.path, test.body)
		if response.Code != http.StatusBadRequest {
			t.Fatalf("invalid PATCH %s body=%s status=%d response=%s", test.path, test.body, response.Code, response.Body.String())
		}
	}
	if stub.patchGlobalCalls != 1 {
		t.Fatalf("invalid requests must not call application: %d", stub.patchGlobalCalls)
	}
}

func TestBlacklistPolicyHandlerMapsConflictAndTargetPath(t *testing.T) {
	handler, stub := newBlacklistPolicyHandlerForTest(t)
	router := newBlacklistPolicyRouter(handler)
	stub.patchGlobalErr = blacklistapp.ErrBlacklistPolicyConflict
	response := performBlacklistPolicyRequest(router, http.MethodPatch, "/v1/blacklistPolicy?updateMask=patterns", `{"name":"blacklistPolicy","patterns":[],"etag":"stale"}`)
	if response.Code != http.StatusConflict {
		t.Fatalf("stale patch status = %d body=%s", response.Code, response.Body.String())
	}
	var errorBody struct {
		Error struct {
			Status string `json:"status"`
		}
	}
	if err := json.Unmarshal(response.Body.Bytes(), &errorBody); err != nil {
		t.Fatal(err)
	}
	if errorBody.Error.Status != "ABORTED" {
		t.Fatalf("conflict status = %#v", errorBody)
	}

	stub.patchGlobalErr = nil
	response = performBlacklistPolicyRequest(router, http.MethodPatch, "/v1/targets/42/blacklistPolicy?updateMask=patterns", `{"name":"targets/42/blacklistPolicy","patterns":["EXAMPLE.com"],"etag":"target-etag"}`)
	if response.Code != http.StatusOK {
		t.Fatalf("target patch status = %d body=%s", response.Code, response.Body.String())
	}
	if stub.patchTargetCalls != 1 || stub.lastTargetID != 42 || !reflect.DeepEqual(stub.lastTargetInput.Patterns, []string{"EXAMPLE.com"}) {
		t.Fatalf("target update input = id:%d input:%#v", stub.lastTargetID, stub.lastTargetInput)
	}

	stub.targetErr = blacklistapp.ErrBlacklistPolicyNotFound
	response = performBlacklistPolicyRequest(router, http.MethodGet, "/v1/targets/42/blacklistPolicy", "")
	if response.Code != http.StatusNotFound {
		t.Fatalf("missing target policy status = %d body=%s", response.Code, response.Body.String())
	}
	if !errors.Is(stub.targetErr, blacklistapp.ErrBlacklistPolicyNotFound) {
		t.Fatal("test setup lost target error")
	}
}

func TestNewBlacklistPolicyHandlerRequiresService(t *testing.T) {
	if _, err := NewBlacklistPolicyHandler(nil); !errors.Is(err, errBlacklistPolicyHandlerDependency) {
		t.Fatalf("nil service error = %v", err)
	}
}
