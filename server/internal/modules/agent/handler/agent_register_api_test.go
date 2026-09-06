package handler

import (
	"bytes"
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
)

type handlerAgentStoreStub struct {
	created []*agentdomain.Agent
}

func (stub *handlerAgentStoreStub) Create(_ context.Context, agent *agentdomain.Agent) error {
	stub.created = append(stub.created, agent)
	agent.ID = 42
	return nil
}

func (stub *handlerAgentStoreStub) GetByID(_ context.Context, _ int) (*agentdomain.Agent, error) {
	return nil, nil
}

func (stub *handlerAgentStoreStub) List(_ context.Context, _, _ int, _, _ string) ([]*agentdomain.Agent, int64, error) {
	return nil, 0, nil
}
func (stub *handlerAgentStoreStub) ListFilterOptions(_ context.Context, _ string) ([]agentdomain.FilterOption, error) {
	return nil, nil
}

func (stub *handlerAgentStoreStub) Update(_ context.Context, _ *agentdomain.Agent) error {
	return nil
}

func (stub *handlerAgentStoreStub) Delete(_ context.Context, _ int) error {
	return nil
}

type handlerTokenStoreStub struct {
	token       *agentdomain.RegistrationToken
	resource    *agentdomain.RegistrationTokenResource
	createdID   int
	resourceErr error
}

func (stub *handlerTokenStoreStub) Create(_ context.Context, token *agentdomain.RegistrationToken) error {
	if stub.createdID > 0 {
		token.ID = stub.createdID
	}
	return nil
}

func (stub *handlerTokenStoreStub) FindValid(_ context.Context, _ string, _ time.Time) (*agentdomain.RegistrationToken, error) {
	return stub.token, nil
}

func (stub *handlerTokenStoreStub) GetResourceByID(_ context.Context, _ int) (*agentdomain.RegistrationTokenResource, error) {
	return stub.resource, stub.resourceErr
}

func (stub *handlerTokenStoreStub) DeleteNeverAttributedBefore(_ context.Context, _ time.Time) error {
	return nil
}

type handlerClockStub struct {
	now time.Time
}

func (stub handlerClockStub) NowUTC() time.Time {
	return stub.now
}

type handlerTokenGenStub struct {
	values []string
	index  int
}

func (stub *handlerTokenGenStub) GenerateHex(int) (string, error) {
	value := stub.values[stub.index]
	stub.index++
	return value, nil
}

func TestRegisterReturnsInstanceIDAndGeneratedDisplayName(t *testing.T) {
	gin.SetMode(gin.TestMode)

	agentStore := &handlerAgentStoreStub{}
	tokenStore := &handlerTokenStoreStub{
		token: &agentdomain.RegistrationToken{ID: 1, Token: "abcd1234"},
	}
	clock := handlerClockStub{now: time.Date(2026, 4, 12, 10, 0, 0, 0, time.UTC)}
	facade := agentapp.NewAgentFacade(
		agentapp.NewAgentQueryService(agentStore),
		agentapp.NewAgentCommandService(agentStore),
		agentapp.NewAgentRegistrationService(agentStore, tokenStore, clock, &handlerTokenGenStub{values: []string{"deadbeef"}}),
	)
	handler := NewAgentHandler(
		facade,
		runtimeConfigPublisherStub{},
		"v1.2.3",
		"https://public.example.com:8083",
		"http://server:9090",
		"docker.io/example/lunafox-agent:v1.2.3",
		"lunafox_data:/opt/lunafox",
		nil,
	)

	router := gin.New()
	router.POST("/v1/agents:register", handler.Register)

	body := bytes.NewBufferString(`{
		"token":"abcd1234",
		"observedHostname":"node-a",
		"agentVersion":"1.2.3"
	}`)

	request := httptest.NewRequest(http.MethodPost, "/v1/agents:register", body)
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d, body=%s", recorder.Code, recorder.Body.String())
	}

	var payload map[string]any
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if _, ok := payload["instanceId"]; !ok {
		t.Fatalf("expected registration response to include instanceId, got %v", payload)
	}
	if gotName, ok := payload["name"].(string); !ok || gotName != "agents/42" {
		t.Fatalf("expected canonical agent name, got %v", payload["name"])
	}
	gotDisplayName, ok := payload["displayName"].(string)
	if !ok || gotDisplayName == "" {
		t.Fatalf("expected generated displayName, got %v", payload["displayName"])
	}
	if !strings.HasPrefix(gotDisplayName, "node-a-") {
		t.Fatalf("expected generated displayName derived from hostname, got %q", gotDisplayName)
	}
}

func TestRegisterRejectsExplicitDisplayNameInput(t *testing.T) {
	gin.SetMode(gin.TestMode)

	agentStore := &handlerAgentStoreStub{}
	tokenStore := &handlerTokenStoreStub{
		token: &agentdomain.RegistrationToken{ID: 1, Token: "abcd1234"},
	}
	clock := handlerClockStub{now: time.Date(2026, 4, 12, 10, 0, 0, 0, time.UTC)}
	facade := agentapp.NewAgentFacade(
		agentapp.NewAgentQueryService(agentStore),
		agentapp.NewAgentCommandService(agentStore),
		agentapp.NewAgentRegistrationService(agentStore, tokenStore, clock, &handlerTokenGenStub{values: []string{"deadbeef"}}),
	)
	handler := NewAgentHandler(
		facade,
		runtimeConfigPublisherStub{},
		"v1.2.3",
		"https://public.example.com:8083",
		"http://server:9090",
		"docker.io/example/lunafox-agent:v1.2.3",
		"lunafox_data:/opt/lunafox",
		nil,
	)

	router := gin.New()
	router.POST("/v1/agents:register", handler.Register)

	body := bytes.NewBufferString(`{
		"token":"abcd1234",
		"displayName":"scanner-a",
		"observedHostname":"node-a",
		"agentVersion":"1.2.3"
	}`)

	request := httptest.NewRequest(http.MethodPost, "/v1/agents:register", body)
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d, body=%s", recorder.Code, recorder.Body.String())
	}
	if !strings.Contains(recorder.Body.String(), "Invalid request body") {
		t.Fatalf("expected invalid request body response, got %s", recorder.Body.String())
	}
}

func TestRegistrationTokenCreateAndReadUseCanonicalNonSecretResource(t *testing.T) {
	gin.SetMode(gin.TestMode)
	now := time.Now().UTC()
	heartbeatAt := now.Add(-time.Second)
	tokenStore := &handlerTokenStoreStub{
		createdID: 23,
		resource: &agentdomain.RegistrationTokenResource{
			ID:        23,
			ExpiresAt: now.Add(time.Hour),
			Agents: []*agentdomain.Agent{
				{ID: 7, InstanceID: "agent-7", DisplayName: "Node 7", Status: "online", ConnectionIP: "172.20.0.5", ObservedSourceIP: "8.8.8.8", LastHeartbeat: &heartbeatAt, CreatedAt: now.Add(-time.Minute)},
				{ID: 8, InstanceID: "agent-8", DisplayName: "Node 8", Status: "offline", CreatedAt: now},
			},
		},
	}
	agentStore := &handlerAgentStoreStub{}
	facade := agentapp.NewAgentFacade(
		agentapp.NewAgentQueryService(agentStore),
		agentapp.NewAgentCommandService(agentStore),
		agentapp.NewAgentRegistrationService(agentStore, tokenStore, handlerClockStub{now: now}, &handlerTokenGenStub{values: []string{"abcd1234"}}),
	)
	handler := NewAgentHandler(facade, runtimeConfigPublisherStub{}, "1.0.0", "https://example.com", "http://server:9090", "example/agent:1", "data:/data", nil)
	router := gin.New()
	router.POST("/v1/admin/agentRegistrationTokens", handler.CreateRegistrationToken)
	router.GET("/v1/admin/agentRegistrationTokens/:registrationToken", handler.GetRegistrationToken)

	createRecorder := httptest.NewRecorder()
	router.ServeHTTP(createRecorder, httptest.NewRequest(http.MethodPost, "/v1/admin/agentRegistrationTokens", nil))
	if createRecorder.Code != http.StatusCreated {
		t.Fatalf("create status = %d, body=%s", createRecorder.Code, createRecorder.Body.String())
	}
	var createResponse map[string]any
	if err := json.Unmarshal(createRecorder.Body.Bytes(), &createResponse); err != nil {
		t.Fatalf("decode create response: %v", err)
	}
	if createResponse["name"] != "agentRegistrationTokens/23" || createResponse["token"] != "abcd1234" {
		t.Fatalf("create response = %#v", createResponse)
	}

	getRecorder := httptest.NewRecorder()
	router.ServeHTTP(getRecorder, httptest.NewRequest(http.MethodGet, "/v1/admin/agentRegistrationTokens/23", nil))
	if getRecorder.Code != http.StatusOK {
		t.Fatalf("get status = %d, body=%s", getRecorder.Code, getRecorder.Body.String())
	}
	var getResponse map[string]any
	if err := json.Unmarshal(getRecorder.Body.Bytes(), &getResponse); err != nil {
		t.Fatalf("decode get response: %v", err)
	}
	if getResponse["name"] != "agentRegistrationTokens/23" || getResponse["state"] != "active" {
		t.Fatalf("get response identity/state = %#v", getResponse)
	}
	if _, exposed := getResponse["token"]; exposed || strings.Contains(getRecorder.Body.String(), "abcd1234") {
		t.Fatalf("status response exposed bearer secret: %s", getRecorder.Body.String())
	}
	agents, ok := getResponse["agents"].([]any)
	if !ok || len(agents) != 2 {
		t.Fatalf("complete attributed Agent projection = %#v", getResponse["agents"])
	}
	first, ok := agents[0].(map[string]any)
	if !ok || first["name"] != "agents/7" || first["displayName"] != "Node 7" || first["status"] != "online" || first["connectionIp"] != "172.20.0.5" || first["lastHeartbeat"] == nil {
		t.Fatalf("current attributed Agent projection = %#v", agents[0])
	}
	if _, exists := first["ipAddress"]; exists {
		t.Fatalf("current attributed Agent projection retained legacy ipAddress: %#v", agents[0])
	}
}
