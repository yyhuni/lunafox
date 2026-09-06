package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	agentapp "github.com/yyhuni/lunafox/server/internal/modules/agent/application"
	agentdomain "github.com/yyhuni/lunafox/server/internal/modules/agent/domain"
	"github.com/yyhuni/lunafox/server/internal/modules/agent/dto"
)

type agentHandlerStoreStub struct {
	agent       *agentdomain.Agent
	listItems   []*agentdomain.Agent
	options     []agentdomain.FilterOption
	updated     *agentdomain.Agent
	lastFilter  string
	lastOrderBy string
	updateErr   error
}

func (stub *agentHandlerStoreStub) GetByID(ctx context.Context, id int) (*agentdomain.Agent, error) {
	_ = ctx
	if stub.agent == nil || stub.agent.ID != id {
		return nil, agentapp.ErrAgentNotFound
	}
	copyAgent := *stub.agent
	return &copyAgent, nil
}

func (stub *agentHandlerStoreStub) List(ctx context.Context, page, pageSize int, filter, orderBy string) ([]*agentdomain.Agent, int64, error) {
	_ = ctx
	_, _ = page, pageSize
	stub.lastFilter = filter
	stub.lastOrderBy = orderBy
	if stub.listItems == nil {
		return nil, 0, nil
	}
	return stub.listItems, int64(len(stub.listItems)), nil
}

func (stub *agentHandlerStoreStub) ListFilterOptions(ctx context.Context, field string) ([]agentdomain.FilterOption, error) {
	_ = ctx
	_ = field
	return stub.options, nil
}

func (stub *agentHandlerStoreStub) Create(ctx context.Context, agent *agentdomain.Agent) error {
	_ = ctx
	_ = agent
	return nil
}

func (stub *agentHandlerStoreStub) Update(ctx context.Context, agent *agentdomain.Agent) error {
	_ = ctx
	if stub.updateErr != nil {
		return stub.updateErr
	}
	copyAgent := *agent
	stub.updated = &copyAgent
	return nil
}

func (stub *agentHandlerStoreStub) Delete(ctx context.Context, id int) error {
	_ = ctx
	_ = id
	return nil
}

func newAgentHandlerForTest(store *agentHandlerStoreStub) *AgentHandler {
	clock := fixedAgentClock{now: time.Date(2026, 5, 13, 10, 0, 0, 0, time.UTC)}
	return NewAgentHandler(
		agentapp.NewAgentFacade(
			agentapp.NewAgentQueryService(store),
			agentapp.NewAgentCommandService(store),
			agentapp.NewAgentRegistrationService(store, agentRegistrationTokenStoreStub{}, clock, fixedTokenGenerator{}),
		),
		nil,
		"",
		"",
		"",
		"",
		"",
		nil,
	)
}

type fixedTokenGenerator struct{}

func (fixedTokenGenerator) GenerateHex(byteLen int) (string, error) {
	_ = byteLen
	return "fixed-token", nil
}

type agentRegistrationTokenStoreStub struct{}

func (agentRegistrationTokenStoreStub) Create(ctx context.Context, token *agentdomain.RegistrationToken) error {
	_ = ctx
	_ = token
	return nil
}

func (agentRegistrationTokenStoreStub) FindValid(ctx context.Context, token string, now time.Time) (*agentdomain.RegistrationToken, error) {
	_ = ctx
	_ = token
	_ = now
	return &agentdomain.RegistrationToken{ID: 1}, nil
}

func (agentRegistrationTokenStoreStub) GetResourceByID(context.Context, int) (*agentdomain.RegistrationTokenResource, error) {
	return nil, nil
}

func (agentRegistrationTokenStoreStub) DeleteNeverAttributedBefore(ctx context.Context, now time.Time) error {
	_ = ctx
	_ = now
	return nil
}

type fixedAgentClock struct {
	now time.Time
}

func (clock fixedAgentClock) NowUTC() time.Time {
	return clock.now
}

func performAgentRequest(t *testing.T, handler gin.HandlerFunc, method, target, body string) *httptest.ResponseRecorder {
	t.Helper()
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Handle(method, "/v1/admin/agents/:agent", handler)

	req := httptest.NewRequest(method, target, strings.NewReader(body))
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	return resp
}

func performAgentListRequest(t *testing.T, handler gin.HandlerFunc, target string) *httptest.ResponseRecorder {
	t.Helper()
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/v1/admin/agents", handler)
	router.GET("/v1/admin/agents/filterOptions", handler)

	req := httptest.NewRequest(http.MethodGet, target, nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	return resp
}

func TestAgentHandlerListUsesCanonicalParamsAndRejectsLegacy(t *testing.T) {
	store := &agentHandlerStoreStub{listItems: []*agentdomain.Agent{{ID: 1, DisplayName: "edge-01", Status: "online", CreatedAt: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)}}}
	handler := newAgentHandlerForTest(store)

	recorder := performAgentListRequest(t, handler.List, `/v1/admin/agents?pageSize=25&filter=displayName%3D%22edge%22&orderBy=createdAt%20desc`)
	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", recorder.Code, recorder.Body.String())
	}
	if store.lastFilter != `displayName="edge"` || store.lastOrderBy != "createdAt desc" {
		t.Fatalf("unexpected query shape filter=%q orderBy=%q", store.lastFilter, store.lastOrderBy)
	}
	var body struct {
		Results   []dto.AgentResponse `json:"results"`
		TotalSize int64               `json:"totalSize"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal list response: %v", err)
	}
	if body.TotalSize != 1 || len(body.Results) != 1 {
		t.Fatalf("unexpected list response: %+v", body)
	}

	for _, legacyParam := range []string{"page", "status", "include", "sort", "sortBy", "sortOrder", "keyword"} {
		recorder = performAgentListRequest(t, handler.List, "/v1/admin/agents?"+legacyParam+"=x")
		if recorder.Code != http.StatusBadRequest {
			t.Fatalf("expected 400 for legacy param %s, got %d", legacyParam, recorder.Code)
		}
	}
}

func TestAgentHandlerFilterOptions(t *testing.T) {
	store := &agentHandlerStoreStub{options: []agentdomain.FilterOption{{Value: "online", Label: "online", Count: 3}}}
	handler := newAgentHandlerForTest(store)

	recorder := performAgentListRequest(t, handler.FilterOptions, "/v1/admin/agents/filterOptions?field=status")
	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", recorder.Code, recorder.Body.String())
	}
	if !strings.Contains(recorder.Body.String(), "online") {
		t.Fatalf("expected option body, got %s", recorder.Body.String())
	}
}

func TestUpdateAgentConfigRequiresNameAndUpdateMask(t *testing.T) {
	handler := newAgentHandlerForTest(&agentHandlerStoreStub{
		agent: &agentdomain.Agent{ID: 8, DisplayName: "agent-8", MaxTasks: 5, CPUThreshold: 50, MemThreshold: 60, DiskThreshold: 70},
	})

	recorder := performAgentRequest(
		t,
		handler.UpdateAgentConfig,
		http.MethodPatch,
		"/v1/admin/agents/8",
		`{"name":"agents/8","updateMask":"maxTasks","maxTasks":7}`,
	)
	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", recorder.Code, recorder.Body.String())
	}

	var body dto.AgentResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal agent response: %v", err)
	}
	if body.ID != 8 || body.Name != "agents/8" || body.MaxTasks != 7 {
		t.Fatalf("unexpected agent response: %+v", body)
	}
}

func TestUpdateAgentConfigRejectsMissingOrWrongUpdateMask(t *testing.T) {
	handler := newAgentHandlerForTest(&agentHandlerStoreStub{
		agent: &agentdomain.Agent{ID: 8, DisplayName: "agent-8", MaxTasks: 5, CPUThreshold: 50, MemThreshold: 60, DiskThreshold: 70},
	})

	recorder := performAgentRequest(
		t,
		handler.UpdateAgentConfig,
		http.MethodPatch,
		"/v1/admin/agents/8",
		`{"name":"agents/8","maxTasks":7}`,
	)
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for missing updateMask, got %d body=%s", recorder.Code, recorder.Body.String())
	}

	recorder = performAgentRequest(
		t,
		handler.UpdateAgentConfig,
		http.MethodPatch,
		"/v1/admin/agents/8",
		`{"name":"agents/9","updateMask":"maxTasks","maxTasks":7}`,
	)
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for mismatched name, got %d body=%s", recorder.Code, recorder.Body.String())
	}

	recorder = performAgentRequest(
		t,
		handler.UpdateAgentConfig,
		http.MethodPatch,
		"/v1/admin/agents/8",
		`{"name":"agents/8","updateMask":"maxTasks,unknownField","maxTasks":7}`,
	)
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for unsupported updateMask field, got %d body=%s", recorder.Code, recorder.Body.String())
	}

	recorder = performAgentRequest(
		t,
		handler.UpdateAgentConfig,
		http.MethodPatch,
		"/v1/admin/agents/8",
		`{"name":"agents/8","updateMask":"maxTasks,cpuThreshold","maxTasks":7}`,
	)
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for updateMask/body mismatch, got %d body=%s", recorder.Code, recorder.Body.String())
	}
}
