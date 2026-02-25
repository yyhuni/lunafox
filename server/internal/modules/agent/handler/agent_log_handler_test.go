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
func (stub *agentRepoForLogHandlerStub) FindByAPIKey(context.Context, string) (*agentdomain.Agent, error) {
	return nil, nil
}
func (stub *agentRepoForLogHandlerStub) List(context.Context, int, int, string) ([]*agentdomain.Agent, int64, error) {
	return nil, 0, nil
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

type lokiClientForLogHandlerStub struct {
	results []loki.StreamResult
	err     error
}

func (stub *lokiClientForLogHandlerStub) QueryRange(context.Context, loki.QueryRangeRequest) ([]loki.StreamResult, error) {
	return stub.results, stub.err
}

func TestAgentLogHandlerListFirstScreenWithoutCursor(t *testing.T) {
	gin.SetMode(gin.TestMode)

	repo := &agentRepoForLogHandlerStub{agent: &agentdomain.Agent{ID: 1, Name: "agent-1"}}
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
	router.GET("/api/admin/agents/:id/logs", handler.List)

	request := httptest.NewRequest(http.MethodGet, "/api/admin/agents/1/logs?container=lunafox-agent&limit=50", nil)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d, body=%s", recorder.Code, recorder.Body.String())
	}

	var payload struct {
		Logs []struct {
			ID string `json:"id"`
		} `json:"logs"`
		NextCursor string `json:"nextCursor"`
		HasMore    bool   `json:"hasMore"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(payload.Logs) != 1 {
		t.Fatalf("expected 1 log item, got %d", len(payload.Logs))
	}
	if payload.NextCursor == "" {
		t.Fatalf("expected nextCursor to be non-empty")
	}
	if payload.HasMore {
		t.Fatalf("expected hasMore=false")
	}
}

func TestAgentLogHandlerListInvalidCursor(t *testing.T) {
	gin.SetMode(gin.TestMode)

	repo := &agentRepoForLogHandlerStub{agent: &agentdomain.Agent{ID: 1, Name: "agent-1"}}
	client := &lokiClientForLogHandlerStub{}
	service := agentapp.NewLokiLogQueryService(client, "test-secret")
	handler := NewAgentLogHandler(repo, service)

	router := gin.New()
	router.GET("/api/admin/agents/:id/logs", handler.List)

	request := httptest.NewRequest(http.MethodGet, "/api/admin/agents/1/logs?container=lunafox-agent&cursor=invalid-cursor", nil)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d, body=%s", recorder.Code, recorder.Body.String())
	}
}

func TestAgentLogHandlerListRejectsCrossContainerCursorReuse(t *testing.T) {
	gin.SetMode(gin.TestMode)

	repo := &agentRepoForLogHandlerStub{agent: &agentdomain.Agent{ID: 1, Name: "agent-1"}}
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
	router.GET("/api/admin/agents/:id/logs", handler.List)

	firstReq := httptest.NewRequest(http.MethodGet, "/api/admin/agents/1/logs?container=lunafox-agent", nil)
	firstRec := httptest.NewRecorder()
	router.ServeHTTP(firstRec, firstReq)
	if firstRec.Code != http.StatusOK {
		t.Fatalf("expected first request 200, got %d, body=%s", firstRec.Code, firstRec.Body.String())
	}

	var firstPayload struct {
		NextCursor string `json:"nextCursor"`
	}
	if err := json.Unmarshal(firstRec.Body.Bytes(), &firstPayload); err != nil {
		t.Fatalf("decode first response: %v", err)
	}
	if firstPayload.NextCursor == "" {
		t.Fatalf("expected non-empty nextCursor from first response")
	}

	secondReq := httptest.NewRequest(http.MethodGet, "/api/admin/agents/1/logs?container=another-container&cursor="+firstPayload.NextCursor, nil)
	secondRec := httptest.NewRecorder()
	router.ServeHTTP(secondRec, secondReq)
	if secondRec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for cross-container cursor reuse, got %d, body=%s", secondRec.Code, secondRec.Body.String())
	}
}

func TestAgentLogHandlerListNoNewLogsKeepsCursorNonEmpty(t *testing.T) {
	gin.SetMode(gin.TestMode)

	repo := &agentRepoForLogHandlerStub{agent: &agentdomain.Agent{ID: 1, Name: "agent-1"}}
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
	router.GET("/api/admin/agents/:id/logs", handler.List)

	firstReq := httptest.NewRequest(http.MethodGet, "/api/admin/agents/1/logs?container=lunafox-agent", nil)
	firstRec := httptest.NewRecorder()
	router.ServeHTTP(firstRec, firstReq)
	if firstRec.Code != http.StatusOK {
		t.Fatalf("expected first request 200, got %d, body=%s", firstRec.Code, firstRec.Body.String())
	}

	var firstPayload struct {
		NextCursor string `json:"nextCursor"`
	}
	if err := json.Unmarshal(firstRec.Body.Bytes(), &firstPayload); err != nil {
		t.Fatalf("decode first response: %v", err)
	}
	if firstPayload.NextCursor == "" {
		t.Fatalf("expected non-empty first nextCursor")
	}

	secondReq := httptest.NewRequest(http.MethodGet, "/api/admin/agents/1/logs?container=lunafox-agent&cursor="+firstPayload.NextCursor, nil)
	secondRec := httptest.NewRecorder()
	router.ServeHTTP(secondRec, secondReq)
	if secondRec.Code != http.StatusOK {
		t.Fatalf("expected second request 200, got %d, body=%s", secondRec.Code, secondRec.Body.String())
	}

	var secondPayload struct {
		Logs       []json.RawMessage `json:"logs"`
		NextCursor string            `json:"nextCursor"`
	}
	if err := json.Unmarshal(secondRec.Body.Bytes(), &secondPayload); err != nil {
		t.Fatalf("decode second response: %v", err)
	}
	if len(secondPayload.Logs) != 0 {
		t.Fatalf("expected empty logs on second request, got %d", len(secondPayload.Logs))
	}
	if secondPayload.NextCursor == "" {
		t.Fatalf("expected nextCursor to remain non-empty when no new logs")
	}
}

func TestAgentLogHandlerListRejectsDeprecatedDirectionQuery(t *testing.T) {
	gin.SetMode(gin.TestMode)

	repo := &agentRepoForLogHandlerStub{agent: &agentdomain.Agent{ID: 1, Name: "agent-1"}}
	client := &lokiClientForLogHandlerStub{}
	service := agentapp.NewLokiLogQueryService(client, "test-secret")
	handler := NewAgentLogHandler(repo, service)

	router := gin.New()
	router.GET("/api/admin/agents/:id/logs", handler.List)

	request := httptest.NewRequest(http.MethodGet, "/api/admin/agents/1/logs?container=lunafox-agent&direction=forward", nil)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d, body=%s", recorder.Code, recorder.Body.String())
	}
}
