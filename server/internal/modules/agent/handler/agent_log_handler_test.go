package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/yyhuni/lunafox/server/internal/loki"
	agentapp "github.com/yyhuni/lunafox/server/internal/modules/agent/application"
	agentdomain "github.com/yyhuni/lunafox/server/internal/modules/agent/domain"
)

type agentRepoForLogHandlerStub struct {
	agent *agentdomain.Agent
	err   error
}

func (stub *agentRepoForLogHandlerStub) Create(context.Context, *agentdomain.Agent) error { return nil }
func (stub *agentRepoForLogHandlerStub) GetByID(context.Context, int) (*agentdomain.Agent, error) {
	return stub.agent, stub.err
}
func (stub *agentRepoForLogHandlerStub) FindByAuthenticationToken(context.Context, string) (*agentdomain.Agent, error) {
	return nil, nil
}
func (stub *agentRepoForLogHandlerStub) List(context.Context, int, int, string, string) ([]*agentdomain.Agent, int64, error) {
	return nil, 0, nil
}
func (stub *agentRepoForLogHandlerStub) ListFilterOptions(context.Context, string) ([]agentdomain.FilterOption, error) {
	return nil, nil
}
func (stub *agentRepoForLogHandlerStub) FindStaleOnline(context.Context, time.Time) ([]*agentdomain.Agent, error) {
	return nil, nil
}
func (stub *agentRepoForLogHandlerStub) Update(context.Context, *agentdomain.Agent) error { return nil }
func (stub *agentRepoForLogHandlerStub) UpdateStatus(context.Context, int, string) error  { return nil }
func (stub *agentRepoForLogHandlerStub) UpdateHeartbeat(context.Context, int, agentdomain.AgentHeartbeatUpdate) error {
	return nil
}
func (stub *agentRepoForLogHandlerStub) Delete(context.Context, int) error { return nil }

type agentLookupForLogHandlerStub struct {
	agent *agentdomain.Agent
	err   error
}

func (stub *agentLookupForLogHandlerStub) GetAgent(context.Context, int) (*agentdomain.Agent, error) {
	return stub.agent, stub.err
}

type lokiClientForLogHandlerStub struct {
	results []loki.StreamResult
	err     error
}

func (stub *lokiClientForLogHandlerStub) QueryRange(context.Context, loki.QueryRangeRequest) ([]loki.StreamResult, error) {
	return stub.results, stub.err
}

type agentLogQueryServiceStub struct {
	input  agentapp.LokiLogQueryInput
	result agentapp.LokiLogQueryResult
	err    error
}

func (stub *agentLogQueryServiceStub) Query(_ context.Context, input agentapp.LokiLogQueryInput) (agentapp.LokiLogQueryResult, error) {
	stub.input = input
	return stub.result, stub.err
}

func TestNewAgentLogHandlerAcceptsQueryInterface(t *testing.T) {
	handler := NewAgentLogHandler(&agentLookupForLogHandlerStub{}, &agentLogQueryServiceStub{})
	if handler == nil {
		t.Fatalf("expected handler instance")
	}
}

func TestAgentLogHandlerListFirstScreenWithAIPPagination(t *testing.T) {
	gin.SetMode(gin.TestMode)

	repo := &agentLookupForLogHandlerStub{agent: &agentdomain.Agent{ID: 1, DisplayName: "agent-1"}}
	client := &lokiClientForLogHandlerStub{
		results: []loki.StreamResult{
			{
				Stream: map[string]string{"source": "stdout"},
				Values: []loki.StreamValue{
					{TsNs: "1740381601000000000", Line: "hello"},
				},
			},
		},
	}
	service := agentapp.NewLokiLogQueryService(client, "test-secret")
	handler := NewAgentLogHandler(repo, service)

	router := gin.New()
	router.GET("/v1/admin/agents/:agent/logEntries", handler.List)

	request := httptest.NewRequest(http.MethodGet, "/v1/admin/agents/1/logEntries?container=lunafox-agent&pageSize=50", nil)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d, body=%s", recorder.Code, recorder.Body.String())
	}

	var payload struct {
		Results []struct {
			ID string `json:"id"`
		} `json:"results"`
		NextPageToken     string `json:"nextPageToken"`
		PreviousPageToken string `json:"previousPageToken"`
		HasOlder          bool   `json:"hasOlder"`
		HasNewer          bool   `json:"hasNewer"`
		CaughtUp          bool   `json:"caughtUp"`
		Gap               bool   `json:"gap"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(payload.Results) != 1 {
		t.Fatalf("expected 1 log item, got %d", len(payload.Results))
	}
	if payload.NextPageToken == "" {
		t.Fatalf("expected nextPageToken to be non-empty")
	}
	if payload.PreviousPageToken != "" {
		t.Fatalf("expected no previousPageToken for first screen without older history, got %q", payload.PreviousPageToken)
	}
	if payload.HasOlder {
		t.Fatalf("expected hasOlder=false for first screen with no older history")
	}
	if payload.HasNewer {
		t.Fatalf("expected hasNewer=false for first screen")
	}
	if !payload.CaughtUp {
		t.Fatalf("expected caughtUp=true for first screen")
	}
	if payload.Gap {
		t.Fatalf("expected gap=false for first screen")
	}
}

func TestAgentLogHandlerListReturnsViewerMetadata(t *testing.T) {
	gin.SetMode(gin.TestMode)

	repo := &agentLookupForLogHandlerStub{agent: &agentdomain.Agent{ID: 1, DisplayName: "agent-1"}}
	service := &agentLogQueryServiceStub{
		result: agentapp.LokiLogQueryResult{
			Logs: []agentapp.LokiLogLineItem{
				{
					ID:        "agt_1:lunafox-agent:1740381601000000000:stdout:abc:000000",
					TS:        "2026-02-24T10:00:01Z",
					TSNs:      "1740381601000000000",
					Stream:    "stdout",
					Line:      "hello",
					Truncated: false,
				},
			},
			NextCursor:     "follow-token",
			PreviousCursor: "older-token",
			HasOlder:       true,
			HasNewer:       true,
			CaughtUp:       false,
			Gap:            true,
			GapReason:      "query_limit",
		},
	}
	handler := NewAgentLogHandler(repo, service)

	router := gin.New()
	router.GET("/v1/admin/agents/:agent/logEntries", handler.List)

	request := httptest.NewRequest(http.MethodGet, "/v1/admin/agents/1/logEntries?container=lunafox-agent&pageSize=50", nil)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d, body=%s", recorder.Code, recorder.Body.String())
	}

	var payload struct {
		PreviousPageToken string `json:"previousPageToken"`
		NextPageToken     string `json:"nextPageToken"`
		HasOlder          bool   `json:"hasOlder"`
		HasNewer          bool   `json:"hasNewer"`
		CaughtUp          bool   `json:"caughtUp"`
		Gap               bool   `json:"gap"`
		GapReason         string `json:"gapReason"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload.PreviousPageToken != "older-token" {
		t.Fatalf("expected previousPageToken, got %q", payload.PreviousPageToken)
	}
	if payload.NextPageToken != "follow-token" {
		t.Fatalf("expected nextPageToken, got %q", payload.NextPageToken)
	}
	if !payload.HasOlder || !payload.HasNewer {
		t.Fatalf("expected older/newer metadata, got %+v", payload)
	}
	if payload.CaughtUp {
		t.Fatalf("expected caughtUp=false")
	}
	if !payload.Gap || payload.GapReason != "query_limit" {
		t.Fatalf("expected gap metadata, got %+v", payload)
	}
}

func TestAgentLogHandlerListInvalidPageToken(t *testing.T) {
	gin.SetMode(gin.TestMode)

	repo := &agentLookupForLogHandlerStub{agent: &agentdomain.Agent{ID: 1, DisplayName: "agent-1"}}
	client := &lokiClientForLogHandlerStub{}
	service := agentapp.NewLokiLogQueryService(client, "test-secret")
	handler := NewAgentLogHandler(repo, service)

	router := gin.New()
	router.GET("/v1/admin/agents/:agent/logEntries", handler.List)

	request := httptest.NewRequest(http.MethodGet, "/v1/admin/agents/1/logEntries?container=lunafox-agent&pageToken=invalid-token", nil)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d, body=%s", recorder.Code, recorder.Body.String())
	}
}

func TestAgentLogHandlerListRejectsCrossContainerCursorReuse(t *testing.T) {
	gin.SetMode(gin.TestMode)

	repo := &agentLookupForLogHandlerStub{agent: &agentdomain.Agent{ID: 1, DisplayName: "agent-1"}}
	client := &lokiClientForLogHandlerStub{
		results: []loki.StreamResult{
			{
				Stream: map[string]string{"source": "stdout"},
				Values: []loki.StreamValue{
					{TsNs: "1740381601000000000", Line: "line-1"},
				},
			},
		},
	}
	service := agentapp.NewLokiLogQueryService(client, "test-secret")
	handler := NewAgentLogHandler(repo, service)

	router := gin.New()
	router.GET("/v1/admin/agents/:agent/logEntries", handler.List)

	firstReq := httptest.NewRequest(http.MethodGet, "/v1/admin/agents/1/logEntries?container=lunafox-agent", nil)
	firstRec := httptest.NewRecorder()
	router.ServeHTTP(firstRec, firstReq)
	if firstRec.Code != http.StatusOK {
		t.Fatalf("expected first request 200, got %d, body=%s", firstRec.Code, firstRec.Body.String())
	}

	var firstPayload struct {
		NextPageToken string `json:"nextPageToken"`
	}
	if err := json.Unmarshal(firstRec.Body.Bytes(), &firstPayload); err != nil {
		t.Fatalf("decode first response: %v", err)
	}
	if firstPayload.NextPageToken == "" {
		t.Fatalf("expected non-empty nextPageToken from first response")
	}
	secondReq := httptest.NewRequest(http.MethodGet, "/v1/admin/agents/1/logEntries?container=another-container&pageToken="+firstPayload.NextPageToken, nil)
	secondRec := httptest.NewRecorder()
	router.ServeHTTP(secondRec, secondReq)
	if secondRec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for cross-container cursor reuse, got %d, body=%s", secondRec.Code, secondRec.Body.String())
	}
}

func TestAgentLogHandlerListNoNewLogsKeepsCursorNonEmpty(t *testing.T) {
	gin.SetMode(gin.TestMode)

	repo := &agentLookupForLogHandlerStub{agent: &agentdomain.Agent{ID: 1, DisplayName: "agent-1"}}
	client := &lokiClientForLogHandlerStub{
		results: []loki.StreamResult{
			{
				Stream: map[string]string{"source": "stdout"},
				Values: []loki.StreamValue{
					{TsNs: "1740381601000000000", Line: "line-1"},
				},
			},
		},
	}
	service := agentapp.NewLokiLogQueryService(client, "test-secret")
	handler := NewAgentLogHandler(repo, service)

	router := gin.New()
	router.GET("/v1/admin/agents/:agent/logEntries", handler.List)

	firstReq := httptest.NewRequest(http.MethodGet, "/v1/admin/agents/1/logEntries?container=lunafox-agent", nil)
	firstRec := httptest.NewRecorder()
	router.ServeHTTP(firstRec, firstReq)
	if firstRec.Code != http.StatusOK {
		t.Fatalf("expected first request 200, got %d, body=%s", firstRec.Code, firstRec.Body.String())
	}

	var firstPayload struct {
		NextPageToken string `json:"nextPageToken"`
	}
	if err := json.Unmarshal(firstRec.Body.Bytes(), &firstPayload); err != nil {
		t.Fatalf("decode first response: %v", err)
	}
	if firstPayload.NextPageToken == "" {
		t.Fatalf("expected non-empty first nextPageToken")
	}
	secondReq := httptest.NewRequest(http.MethodGet, "/v1/admin/agents/1/logEntries?container=lunafox-agent&pageToken="+firstPayload.NextPageToken, nil)
	secondRec := httptest.NewRecorder()
	router.ServeHTTP(secondRec, secondReq)
	if secondRec.Code != http.StatusOK {
		t.Fatalf("expected second request 200, got %d, body=%s", secondRec.Code, secondRec.Body.String())
	}

	var secondPayload struct {
		Results       []json.RawMessage `json:"results"`
		NextPageToken string            `json:"nextPageToken"`
	}
	if err := json.Unmarshal(secondRec.Body.Bytes(), &secondPayload); err != nil {
		t.Fatalf("decode second response: %v", err)
	}
	if len(secondPayload.Results) != 0 {
		t.Fatalf("expected empty logs on second request, got %d", len(secondPayload.Results))
	}
	if secondPayload.NextPageToken == "" {
		t.Fatalf("expected nextPageToken to remain non-empty when no new logs")
	}
}

func TestAgentLogHandlerListRejectsLegacyPagingParams(t *testing.T) {
	gin.SetMode(gin.TestMode)

	repo := &agentLookupForLogHandlerStub{agent: &agentdomain.Agent{ID: 1, DisplayName: "agent-1"}}
	service := &agentLogQueryServiceStub{}
	handler := NewAgentLogHandler(repo, service)

	router := gin.New()
	router.GET("/v1/admin/agents/:agent/logEntries", handler.List)

	for _, query := range []string{"limit=50", "cursor=abc"} {
		request := httptest.NewRequest(http.MethodGet, "/v1/admin/agents/1/logEntries?container=lunafox-agent&"+query, nil)
		recorder := httptest.NewRecorder()
		router.ServeHTTP(recorder, request)

		if recorder.Code != http.StatusBadRequest {
			t.Fatalf("expected 400 for %s, got %d, body=%s", query, recorder.Code, recorder.Body.String())
		}
	}
}

func TestAgentLogHandlerListRejectsDeprecatedDirectionQuery(t *testing.T) {
	gin.SetMode(gin.TestMode)

	repo := &agentLookupForLogHandlerStub{agent: &agentdomain.Agent{ID: 1, DisplayName: "agent-1"}}
	client := &lokiClientForLogHandlerStub{}
	service := agentapp.NewLokiLogQueryService(client, "test-secret")
	handler := NewAgentLogHandler(repo, service)

	router := gin.New()
	router.GET("/v1/admin/agents/:agent/logEntries", handler.List)

	request := httptest.NewRequest(http.MethodGet, "/v1/admin/agents/1/logEntries?container=lunafox-agent&direction=forward", nil)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d, body=%s", recorder.Code, recorder.Body.String())
	}
}

func TestAgentLogHandlerListAcceptsOlderDirection(t *testing.T) {
	gin.SetMode(gin.TestMode)

	repo := &agentLookupForLogHandlerStub{agent: &agentdomain.Agent{ID: 1, DisplayName: "agent-1"}}
	service := &agentLogQueryServiceStub{}
	handler := NewAgentLogHandler(repo, service)

	router := gin.New()
	router.GET("/v1/admin/agents/:agent/logEntries", handler.List)

	request := httptest.NewRequest(http.MethodGet, "/v1/admin/agents/1/logEntries?container=lunafox-agent&pageToken=foo&direction=older", nil)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	if recorder.Code == http.StatusBadRequest {
		t.Fatalf("expected older direction to be accepted, got 400 body=%s", recorder.Body.String())
	}
}

func TestAgentLogHandlerListUsesDefaultAndMaximumPageSize(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := &agentLookupForLogHandlerStub{agent: &agentdomain.Agent{ID: 1, DisplayName: "agent-1"}}
	service := &agentLogQueryServiceStub{}
	handler := NewAgentLogHandler(repo, service)
	router := gin.New()
	router.GET("/v1/admin/agents/:agent/logEntries", handler.List)

	for _, testCase := range []struct {
		query string
		want  int
	}{
		{query: "", want: defaultAgentLogLimit},
		{query: "&pageSize=500", want: maxAgentLogLimit},
	} {
		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodGet, "/v1/admin/agents/1/logEntries?container=lunafox-agent"+testCase.query, nil)
		router.ServeHTTP(recorder, request)
		if recorder.Code != http.StatusOK || service.input.Limit != testCase.want {
			t.Fatalf("query %q: got status=%d limit=%d", testCase.query, recorder.Code, service.input.Limit)
		}
	}
}

func TestAgentLogHandlerListRejectsInvalidPageSize(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := &agentLookupForLogHandlerStub{agent: &agentdomain.Agent{ID: 1, DisplayName: "agent-1"}}
	service := &agentLogQueryServiceStub{}
	handler := NewAgentLogHandler(repo, service)
	router := gin.New()
	router.GET("/v1/admin/agents/:agent/logEntries", handler.List)

	for _, pageSize := range []string{"501", "-1", "not-a-number"} {
		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodGet, "/v1/admin/agents/1/logEntries?container=lunafox-agent&pageSize="+pageSize, nil)
		router.ServeHTTP(recorder, request)
		if recorder.Code != http.StatusBadRequest {
			t.Fatalf("pageSize=%q: expected 400, got %d", pageSize, recorder.Code)
		}
	}
}
