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
	"github.com/yyhuni/lunafox/server/internal/modules/agent/dto"
)

type agentLocationMapHandlerServiceStub struct {
	locationMap agentapp.AgentLocationMap
	err         error
}

func (stub *agentLocationMapHandlerServiceStub) Current(context.Context) (agentapp.AgentLocationMap, error) {
	return stub.locationMap, stub.err
}

func TestAgentLocationMapHandlerPreservesUnknownServerAndRuntimePresence(t *testing.T) {
	gin.SetMode(gin.TestMode)
	generatedAt := time.Date(2026, 8, 4, 12, 0, 0, 0, time.UTC)
	zero := 0
	handler := NewAgentLocationMapHandler(&agentLocationMapHandlerServiceStub{locationMap: agentapp.AgentLocationMap{
		GeneratedAt: generatedAt,
		Agents: []agentapp.AgentLocationMapAgent{
			{
				AgentID: 1, DisplayName: "missing-runtime", Status: "online",
				Location: agentapp.AgentLocationMapLocation{State: agentapp.LocationFreshnessCurrent, Latitude: 1.123456789, Longitude: 2.987654321, SourceObservedIP: "8.8.8.8", ProviderKey: "freeipapi", ResolvedAt: generatedAt},
			},
			{
				AgentID: 2, DisplayName: "offline-zero", Status: "offline", HealthState: "healthy", TaskSlotsUsed: &zero,
				Location: agentapp.AgentLocationMapLocation{State: agentapp.LocationFreshnessExpired, Latitude: 3, Longitude: 4, SourceObservedIP: "1.1.1.1", ProviderKey: "freeipapi", ResolvedAt: generatedAt.Add(-7 * 24 * time.Hour)},
			},
		},
	}})
	router := gin.New()
	router.GET("/v1/admin/agentLocationMaps/current", handler.Current)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/v1/admin/agentLocationMaps/current", nil))
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, body=%s", recorder.Code, recorder.Body.String())
	}
	body := recorder.Body.String()
	for _, required := range []string{`"serverLocation":null`, `"taskSlotsUsed":null`, `"taskSlotsUsed":0`} {
		if !strings.Contains(body, required) {
			t.Fatalf("map response missing %s: %s", required, body)
		}
	}
	if strings.Contains(body, `"connections"`) {
		t.Fatal("backend map projection introduced sampled connection output")
	}
	var response dto.AgentLocationMapResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if response.Name != "agentLocationMaps/current" || response.ServerLocation != nil || len(response.Agents) != 2 {
		t.Fatalf("map identity/Server state = %#v", response)
	}
	if response.Agents[0].Name != "agents/1" || response.Agents[0].Location.Latitude != 1.123456789 {
		t.Fatalf("Agent coordinate projection was rounded or renamed: %#v", response.Agents[0])
	}
	if response.Agents[1].Status != "offline" || response.Agents[1].Location.State != "expired" {
		t.Fatalf("offline marker projection = %#v", response.Agents[1])
	}
}
