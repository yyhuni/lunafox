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
	"github.com/yyhuni/lunafox/server/internal/cache"
	agentapp "github.com/yyhuni/lunafox/server/internal/modules/agent/application"
	agentdomain "github.com/yyhuni/lunafox/server/internal/modules/agent/domain"
)

type agentListQueryStoreStub struct {
	agents []*agentdomain.Agent
}

func (stub *agentListQueryStoreStub) GetByID(_ context.Context, id int) (*agentdomain.Agent, error) {
	for _, agent := range stub.agents {
		if agent.ID == id {
			copy := *agent
			return &copy, nil
		}
	}
	return nil, errors.New("agent not found")
}

func (stub *agentListQueryStoreStub) List(_ context.Context, _, _ int, _, _ string) ([]*agentdomain.Agent, int64, error) {
	results := make([]*agentdomain.Agent, 0, len(stub.agents))
	for _, agent := range stub.agents {
		copy := *agent
		results = append(results, &copy)
	}
	return results, int64(len(results)), nil
}

func (stub *agentListQueryStoreStub) ListFilterOptions(context.Context, string) ([]agentdomain.FilterOption, error) {
	return nil, nil
}

type agentListHeartbeatCacheStub struct {
	data *cache.HeartbeatData
	err  error
}

func (stub *agentListHeartbeatCacheStub) Set(context.Context, int, *cache.HeartbeatData) error {
	return nil
}

func (stub *agentListHeartbeatCacheStub) Get(context.Context, int) (*cache.HeartbeatData, error) {
	return stub.data, stub.err
}

func (stub *agentListHeartbeatCacheStub) Delete(context.Context, int) error { return nil }

func TestAgentListHeartbeatProjectionUsesRedisOnly(t *testing.T) {
	gin.SetMode(gin.TestMode)
	lastHeartbeat := time.Date(2026, time.August, 9, 0, 0, 0, 0, time.UTC)
	storedAgent := &agentdomain.Agent{
		ID:            7,
		DisplayName:   "edge-7",
		Status:        "online",
		LastHeartbeat: &lastHeartbeat,
		// These persisted runtime values are intentionally distinct from a
		// heartbeat projection and must never become a cache-miss fallback.
		RunningTasks:  91,
		TaskSlotsUsed: 92,
	}

	tests := []struct {
		name          string
		cache         *agentListHeartbeatCacheStub
		wantHeartbeat bool
	}{
		{
			name: "cache hit projects realtime metrics",
			cache: &agentListHeartbeatCacheStub{data: &cache.HeartbeatData{
				CPU: 11, Mem: 22, Disk: 33, RunningTasks: 1, TaskSlotsUsed: 2, Uptime: 3, UpdatedAt: lastHeartbeat,
			}},
			wantHeartbeat: true,
		},
		{name: "cache miss omits realtime metrics", cache: &agentListHeartbeatCacheStub{}},
		{name: "cache read failure omits realtime metrics", cache: &agentListHeartbeatCacheStub{err: errors.New("redis unavailable")}},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			facade := agentapp.NewAgentFacade(
				agentapp.NewAgentQueryService(&agentListQueryStoreStub{agents: []*agentdomain.Agent{storedAgent}}),
				nil,
				nil,
			)
			handler := NewAgentHandler(facade, runtimeConfigPublisherStub{}, "", "", "", "", "", test.cache)
			router := gin.New()
			router.GET("/v1/admin/agents", handler.List)

			recorder := httptest.NewRecorder()
			router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/v1/admin/agents", nil))
			if recorder.Code != http.StatusOK {
				t.Fatalf("List status = %d, body=%s", recorder.Code, recorder.Body.String())
			}

			var response agentListResponse
			if err := readAgentListResponse(recorder, &response); err != nil {
				t.Fatal(err)
			}
			if len(response.Results) != 1 {
				t.Fatalf("result count = %d, want 1", len(response.Results))
			}
			result := response.Results[0]
			if result.LastHeartbeat == nil || !result.LastHeartbeat.Equal(lastHeartbeat) {
				t.Fatalf("lastHeartbeat = %v, want %s", result.LastHeartbeat, lastHeartbeat)
			}
			if (result.Heartbeat != nil) != test.wantHeartbeat {
				t.Fatalf("heartbeat = %#v, want present=%t", result.Heartbeat, test.wantHeartbeat)
			}
			if test.wantHeartbeat && (result.Heartbeat.RunningTasks != 1 || result.Heartbeat.TaskSlotsUsed != 2) {
				t.Fatalf("heartbeat projection = %#v, want Redis values", result.Heartbeat)
			}
		})
	}
}

func readAgentListResponse(recorder *httptest.ResponseRecorder, response *agentListResponse) error {
	return json.NewDecoder(recorder.Body).Decode(response)
}
