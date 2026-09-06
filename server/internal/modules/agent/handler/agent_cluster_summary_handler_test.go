package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	agentapp "github.com/yyhuni/lunafox/server/internal/modules/agent/application"
	"github.com/yyhuni/lunafox/server/internal/modules/agent/dto"
)

type agentClusterSummaryHandlerServiceStub struct {
	summary agentapp.AgentClusterSummary
	err     error
}

func (stub *agentClusterSummaryHandlerServiceStub) Current(context.Context) (agentapp.AgentClusterSummary, error) {
	return stub.summary, stub.err
}

func TestAgentClusterSummaryHandlerCurrentReturnsCanonicalProjection(t *testing.T) {
	gin.SetMode(gin.TestMode)
	generatedAt := time.Date(2026, 8, 4, 12, 0, 0, 0, time.UTC)
	handler := NewAgentClusterSummaryHandler(&agentClusterSummaryHandlerServiceStub{summary: agentapp.AgentClusterSummary{
		GeneratedAt:               generatedAt,
		ExecutionFreshnessSeconds: 15,
		Nodes:                     agentapp.AgentClusterNodeCounts{Total: 4, Healthy: 1, Warning: 1, Offline: 1, Unknown: 1, Stale: 2},
		ExecutionCapacity:         agentapp.AgentClusterExecutionCapacity{Configured: 12, Occupied: 3, Available: 2, Unavailable: 7, Overcommitted: 1},
		State:                     agentapp.AgentClusterStateNeedsAttention,
		ReasonCodes:               []agentapp.AgentClusterReasonCode{agentapp.AgentClusterReasonOfflineAgents, agentapp.AgentClusterReasonWarningAgents},
		LocationCoverage:          agentapp.AgentClusterLocationCoverage{Positioned: 3, Unpositioned: 1},
	}})
	router := gin.New()
	router.GET("/v1/admin/agentClusterSummaries/current", handler.Current)

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/v1/admin/agentClusterSummaries/current", nil))
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, body=%s", recorder.Code, recorder.Body.String())
	}
	var response dto.AgentClusterSummaryResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if response.Name != "agentClusterSummaries/current" || !response.GeneratedAt.Equal(generatedAt) || response.ExecutionFreshnessSeconds != 15 {
		t.Fatalf("summary identity = %#v", response)
	}
	if response.TotalNodes != 4 || response.HealthyCount != 1 || response.WarningCount != 1 || response.OfflineCount != 1 || response.UnknownCount != 1 || response.StaleAgentCount != 2 {
		t.Fatalf("summary counts = %#v", response)
	}
	if response.ExecutionCapacity.ConfiguredSlots != 12 || response.ExecutionCapacity.OccupiedSlots != 3 || response.ExecutionCapacity.AvailableSlots != 2 || response.ExecutionCapacity.UnavailableSlots != 7 || response.ExecutionCapacity.OvercommittedSlots != 1 {
		t.Fatalf("summary capacity = %#v", response.ExecutionCapacity)
	}
	if response.ClusterState != "needsAttention" || len(response.ReasonCodes) != 2 || response.LocationCoverage.PositionedCount != 3 || response.LocationCoverage.UnpositionedCount != 1 {
		t.Fatalf("summary policy/coverage = %#v", response)
	}
}

func TestAgentClusterSummaryHandlerCurrentHidesStoreFailure(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := NewAgentClusterSummaryHandler(&agentClusterSummaryHandlerServiceStub{err: errors.New("database details")})
	router := gin.New()
	router.GET("/v1/admin/agentClusterSummaries/current", handler.Current)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/v1/admin/agentClusterSummaries/current", nil))
	if recorder.Code != http.StatusInternalServerError || recorder.Body.String() == "" {
		t.Fatalf("status = %d, body=%s", recorder.Code, recorder.Body.String())
	}
	if string(recorder.Body.Bytes()) == "database details" {
		t.Fatal("handler exposed internal aggregate failure")
	}
}
