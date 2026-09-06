package router

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	agentapp "github.com/yyhuni/lunafox/server/internal/modules/agent/application"
	agenthandler "github.com/yyhuni/lunafox/server/internal/modules/agent/handler"
)

type agentClusterSummaryRouteServiceStub struct{}

func (agentClusterSummaryRouteServiceStub) Current(context.Context) (agentapp.AgentClusterSummary, error) {
	return agentapp.AgentClusterSummary{
		GeneratedAt:               time.Now().UTC(),
		ExecutionFreshnessSeconds: 15,
		State:                     agentapp.AgentClusterStateEmpty,
		ReasonCodes:               []agentapp.AgentClusterReasonCode{agentapp.AgentClusterReasonNoAgents},
	}, nil
}

func TestAgentFixedViewRoutesAreExactAndProtected(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	api := router.Group("/v1")
	protected := api.Group("")
	protected.Use(func(c *gin.Context) {
		if c.GetHeader("Authorization") != "Bearer test" {
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}
		c.Next()
	})
	RegisterAgentRoutes(
		api,
		protected,
		&agenthandler.AgentHandler{},
		&agenthandler.AgentLogHandler{},
		agenthandler.NewAgentClusterSummaryHandler(agentClusterSummaryRouteServiceStub{}),
		agenthandler.NewAgentLocationMapHandler(agentLocationMapRouteServiceStub{}),
	)

	for _, path := range []string{
		"/v1/admin/agentClusterSummaries/current",
		"/v1/admin/agentLocationMaps/current",
	} {
		unauthorized := httptest.NewRecorder()
		router.ServeHTTP(unauthorized, httptest.NewRequest(http.MethodGet, path, nil))
		if unauthorized.Code != http.StatusUnauthorized {
			t.Fatalf("%s unauthorized status = %d", path, unauthorized.Code)
		}

		authorizedRequest := httptest.NewRequest(http.MethodGet, path, nil)
		authorizedRequest.Header.Set("Authorization", "Bearer test")
		authorized := httptest.NewRecorder()
		router.ServeHTTP(authorized, authorizedRequest)
		if authorized.Code != http.StatusOK {
			t.Fatalf("%s authorized status = %d, body=%s", path, authorized.Code, authorized.Body.String())
		}

		wrongPath := httptest.NewRecorder()
		wrongRequest := httptest.NewRequest(http.MethodGet, path[:len(path)-len("/current")], nil)
		wrongRequest.Header.Set("Authorization", "Bearer test")
		router.ServeHTTP(wrongPath, wrongRequest)
		if wrongPath.Code != http.StatusNotFound {
			t.Fatalf("%s collection alias unexpectedly registered: %d", path, wrongPath.Code)
		}
	}
}

func TestRegistrationTokenRoutesUseCanonicalResourceAndHardCutNestedCreation(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	api := router.Group("/v1")
	protected := api.Group("")
	protected.Use(func(c *gin.Context) {
		c.AbortWithStatus(http.StatusUnauthorized)
	})
	RegisterAgentRoutes(
		api,
		protected,
		&agenthandler.AgentHandler{},
		&agenthandler.AgentLogHandler{},
		agenthandler.NewAgentClusterSummaryHandler(agentClusterSummaryRouteServiceStub{}),
		agenthandler.NewAgentLocationMapHandler(agentLocationMapRouteServiceStub{}),
	)

	routes := make(map[string]bool)
	for _, route := range router.Routes() {
		routes[route.Method+" "+route.Path] = true
	}
	for _, required := range []string{
		"POST /v1/admin/agentRegistrationTokens",
		"GET /v1/admin/agentRegistrationTokens/:registrationToken",
	} {
		if !routes[required] {
			t.Errorf("missing canonical registration-token route %q", required)
		}
	}
	if routes["POST /v1/admin/agents/registrationTokens"] {
		t.Fatal("legacy nested registration-token creation route remains registered")
	}

	for _, request := range []*http.Request{
		httptest.NewRequest(http.MethodPost, "/v1/admin/agentRegistrationTokens", nil),
		httptest.NewRequest(http.MethodGet, "/v1/admin/agentRegistrationTokens/1", nil),
	} {
		recorder := httptest.NewRecorder()
		router.ServeHTTP(recorder, request)
		if recorder.Code != http.StatusUnauthorized {
			t.Errorf("%s %s status = %d, want protected route", request.Method, request.URL.Path, recorder.Code)
		}
	}

	legacy := httptest.NewRecorder()
	router.ServeHTTP(legacy, httptest.NewRequest(http.MethodPost, "/v1/admin/agents/registrationTokens", nil))
	if legacy.Code != http.StatusNotFound {
		t.Fatalf("legacy nested creation status = %d, want 404", legacy.Code)
	}
}

type agentLocationMapRouteServiceStub struct{}

func (agentLocationMapRouteServiceStub) Current(context.Context) (agentapp.AgentLocationMap, error) {
	return agentapp.AgentLocationMap{GeneratedAt: time.Now().UTC(), Agents: []agentapp.AgentLocationMapAgent{}}, nil
}
